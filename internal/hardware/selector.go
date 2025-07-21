// Package hardware provides GPU detection and selection for transcoding.
package hardware

import (
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/savid/iptv-proxy/internal/types"
)

const (
	// Video codec constants.
	codecH264 = "h264"
	codecH265 = "h265"
	codecHEVC = "hevc"
	codecCopy = "copy"

	// Audio codec constants.
	codecAAC = "aac"
	codecMP3 = "mp3"

	// Common string constants.
	adaptive = "adaptive"
)

var (
	// ErrNoHardware is returned when no hardware acceleration is available.
	ErrNoHardware = errors.New("no hardware acceleration available")
	// ErrNoSuitableHardware is returned when no suitable hardware found.
	ErrNoSuitableHardware = errors.New("no suitable hardware found")
	// ErrDeviceNotFound is returned when specified device is not found.
	ErrDeviceNotFound = errors.New("specified device not found")
)

// Selector chooses the best available hardware for transcoding.
type Selector struct {
	detector      *Detector
	preferred     types.HardwareType
	availableGPUs []types.HardwareInfo
	logger        *log.Logger
}

// NewSelector creates a new hardware selector instance.
func NewSelector(detector *Detector, preferred types.HardwareType, logger *log.Logger) *Selector {
	return &Selector{
		detector:  detector,
		preferred: preferred,
		logger:    logger,
	}
}

// Initialize detects available hardware and prepares the selector.
func (s *Selector) Initialize() error {
	s.availableGPUs = s.detector.DetectGPUs()

	if len(s.availableGPUs) == 0 {
		return ErrNoHardware
	}

	s.logger.Printf("Available hardware acceleration:")
	for _, gpu := range s.availableGPUs {
		s.logger.Printf("  - %s: %v", gpu.Type, gpu.Capabilities)
	}

	return nil
}

// SelectHardware chooses the best hardware for the given profile.
func (s *Selector) SelectHardware(deviceType string, deviceID int) (types.HardwareInfo, error) {
	if len(s.availableGPUs) == 0 {
		return types.HardwareInfo{}, ErrNoHardware
	}

	// Handle specific device selection (e.g., nvidia:0)
	if deviceType != "auto" && deviceType != "none" && deviceType != "" {
		for _, gpu := range s.availableGPUs {
			if string(gpu.Type) == deviceType && gpu.DeviceID == deviceID && gpu.Available {
				s.logger.Printf("Selected specific device: %s:%d", gpu.Type, gpu.DeviceID)
				return gpu, nil
			}
		}
		// If specific device not found, log error and return
		s.logger.Printf("Device %s:%d not found", deviceType, deviceID)
		return types.HardwareInfo{}, ErrDeviceNotFound
	}

	// Handle "none" - force CPU encoding
	if deviceType == "none" {
		for _, gpu := range s.availableGPUs {
			if gpu.Type == types.HardwareCPU {
				return gpu, nil
			}
		}
	}

	// Handle specific hardware preference
	if s.preferred != types.HardwareAuto {
		for _, gpu := range s.availableGPUs {
			if gpu.Type == s.preferred && gpu.Available {
				return gpu, nil
			}
		}
		// If preferred hardware not available, log warning and fall through to auto selection
		s.logger.Printf("Preferred hardware %s not available, using auto selection", s.preferred)
	}

	// Auto selection: prefer GPU over CPU
	// Priority order: NVIDIA > Intel > AMD > CPU
	priority := []types.HardwareType{
		types.HardwareNVIDIA,
		types.HardwareIntel,
		types.HardwareAMD,
		types.HardwareCPU,
	}

	for _, hwType := range priority {
		for _, gpu := range s.availableGPUs {
			if gpu.Type == hwType && gpu.Available {
				s.logger.Printf("Selected hardware: %s", gpu.Type)
				return gpu, nil
			}
		}
	}

	return types.HardwareInfo{}, ErrNoSuitableHardware
}

// GetFFmpegArgs returns FFmpeg arguments for the selected hardware.
func (s *Selector) GetFFmpegArgs(hw types.HardwareInfo, profile types.TranscodingProfile) []string {
	args := []string{}

	// Add video codec arguments
	videoArgs := s.getVideoCodecArgs(hw, profile.VideoCodec)
	args = append(args, videoArgs...)

	// Add audio codec arguments
	audioArgs := s.getAudioCodecArgs(profile.AudioCodec)
	args = append(args, audioArgs...)

	// Add bitrate settings
	bitrateArgs := s.getBitrateArgs(profile)
	args = append(args, bitrateArgs...)

	// Add container format
	if profile.Container != "" {
		args = append(args, "-f", profile.Container)
	}

	// Add any extra arguments from the profile
	args = append(args, profile.ExtraArgs...)

	return args
}

// getVideoCodecArgs returns video codec specific arguments.
func (s *Selector) getVideoCodecArgs(hw types.HardwareInfo, videoCodec string) []string {
	if videoCodec == codecCopy {
		return []string{"-c:v", "copy"}
	}

	switch hw.Type {
	case types.HardwareNVIDIA:
		return s.getNVIDIAVideoArgs(hw, videoCodec)
	case types.HardwareIntel:
		return s.getIntelVideoArgs(hw, videoCodec)
	case types.HardwareAMD:
		return s.getAMDVideoArgs(hw, videoCodec)
	case types.HardwareCPU, types.HardwareAuto:
		return s.getCPUVideoArgs(videoCodec)
	default:
		return s.getCPUVideoArgs(videoCodec)
	}
}

// getNVIDIAVideoArgs returns NVIDIA specific video encoding arguments.
func (s *Selector) getNVIDIAVideoArgs(hw types.HardwareInfo, videoCodec string) []string {
	args := []string{}

	// Add GPU index if specified (for multi-GPU systems)
	if hw.DeviceID >= 0 {
		args = append(args, "-gpu", fmt.Sprintf("%d", hw.DeviceID))
	}

	switch videoCodec {
	case codecH264:
		args = append(args, "-c:v", "h264_nvenc")
		args = append(args, "-preset", "p4") // Balanced preset
		args = append(args, "-tune", "hq")
		args = append(args, "-rc", "vbr")
		args = append(args, "-rc-lookahead", "20")
		args = append(args, "-b_ref_mode", "middle")
	case codecH265, codecHEVC:
		args = append(args, "-c:v", "hevc_nvenc")
		args = append(args, "-preset", "p4")
		args = append(args, "-tune", "hq")
		args = append(args, "-rc", "vbr")
	}
	return args
}

// getIntelVideoArgs returns Intel Quick Sync specific video encoding arguments.
func (s *Selector) getIntelVideoArgs(hw types.HardwareInfo, videoCodec string) []string {
	args := []string{}
	if hw.DevicePath != "" {
		args = append(args, "-init_hw_device", fmt.Sprintf("vaapi=va:%s", hw.DevicePath))
		args = append(args, "-filter_hw_device", "va")
	}

	switch videoCodec {
	case codecH264:
		args = append(args, "-c:v", "h264_vaapi")
		args = append(args, "-vaapi_device", hw.DevicePath)
	case codecH265, codecHEVC:
		args = append(args, "-c:v", "hevc_vaapi")
		args = append(args, "-vaapi_device", hw.DevicePath)
	}
	return args
}

// getAMDVideoArgs returns AMD specific video encoding arguments.
func (s *Selector) getAMDVideoArgs(hw types.HardwareInfo, videoCodec string) []string {
	if strings.Contains(hw.DevicePath, "/dev/dri") {
		return s.getAMDVAAPIArgs(hw, videoCodec)
	}
	return s.getAMDAMFArgs(videoCodec)
}

// getAMDVAAPIArgs returns AMD VA-API specific arguments.
func (s *Selector) getAMDVAAPIArgs(hw types.HardwareInfo, videoCodec string) []string {
	args := []string{}
	if hw.DevicePath != "" {
		args = append(args, "-init_hw_device", fmt.Sprintf("vaapi=va:%s", hw.DevicePath))
		args = append(args, "-filter_hw_device", "va")
	}

	switch videoCodec {
	case codecH264:
		args = append(args, "-c:v", "h264_vaapi")
		args = append(args, "-vaapi_device", hw.DevicePath)
	case codecH265, codecHEVC:
		args = append(args, "-c:v", "hevc_vaapi")
		args = append(args, "-vaapi_device", hw.DevicePath)
	}
	return args
}

// getAMDAMFArgs returns AMD AMF specific arguments.
func (s *Selector) getAMDAMFArgs(videoCodec string) []string {
	args := []string{}
	switch videoCodec {
	case codecH264:
		args = append(args, "-c:v", "h264_amf")
		args = append(args, "-usage", "transcoding")
		args = append(args, "-quality", "balanced")
	case codecH265, codecHEVC:
		args = append(args, "-c:v", "hevc_amf")
		args = append(args, "-usage", "transcoding")
		args = append(args, "-quality", "balanced")
	}
	return args
}

// getCPUVideoArgs returns CPU-based video encoding arguments.
func (s *Selector) getCPUVideoArgs(videoCodec string) []string {
	args := []string{}
	switch videoCodec {
	case codecH264:
		args = append(args, "-c:v", "libx264")
	case codecH265, codecHEVC:
		args = append(args, "-c:v", "libx265")
	}
	return args
}

// getAudioCodecArgs returns audio codec specific arguments.
func (s *Selector) getAudioCodecArgs(audioCodec string) []string {
	args := []string{}
	switch audioCodec {
	case codecAAC:
		args = append(args, "-c:a", "aac")
	case codecMP3:
		args = append(args, "-c:a", "libmp3lame")
	case codecCopy:
		args = append(args, "-c:a", "copy")
	}
	return args
}

// getBitrateArgs returns bitrate specific arguments.
func (s *Selector) getBitrateArgs(profile types.TranscodingProfile) []string {
	args := []string{}

	// Add video bitrate if specified
	if profile.VideoBitrate != "" && profile.VideoBitrate != adaptive && profile.VideoCodec != codecCopy {
		args = append(args, "-b:v", profile.VideoBitrate)
	}

	// Add audio bitrate if specified
	if profile.AudioBitrate != "" && profile.AudioBitrate != adaptive && profile.AudioCodec != codecCopy {
		args = append(args, "-b:a", profile.AudioBitrate)
	}

	return args
}
