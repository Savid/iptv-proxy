// Package hardware provides GPU detection and selection for transcoding.
package hardware

import (
	"fmt"
	"log"
	"strings"

	"github.com/savid/iptv-proxy/internal/types"
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
		return fmt.Errorf("no hardware acceleration available")
	}

	s.logger.Printf("Available hardware acceleration:")
	for _, gpu := range s.availableGPUs {
		s.logger.Printf("  - %s: %v", gpu.Type, gpu.Capabilities)
	}

	return nil
}

// SelectHardware chooses the best hardware for the given profile.
func (s *Selector) SelectHardware(profile string) (types.HardwareInfo, error) {
	if len(s.availableGPUs) == 0 {
		return types.HardwareInfo{}, fmt.Errorf("no hardware available")
	}

	// plex-gump uses MPEG-2 which requires CPU encoding
	for _, gpu := range s.availableGPUs {
		if gpu.Type == types.HardwareCPU {
			return gpu, nil
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

	return types.HardwareInfo{}, fmt.Errorf("no suitable hardware found")
}

// GetFFmpegArgs returns FFmpeg arguments for the selected hardware.
func (s *Selector) GetFFmpegArgs(hw types.HardwareInfo, profile types.TranscodingProfile) []string {
	args := []string{}

	switch hw.Type {
	case types.HardwareNVIDIA:
		// NVIDIA hardware acceleration
		if profile.VideoCodec == "h264" {
			args = append(args, "-c:v", "h264_nvenc")
			args = append(args, "-preset", "p4") // Balanced preset
			args = append(args, "-tune", "hq")
			args = append(args, "-rc", "vbr")
			args = append(args, "-rc-lookahead", "20")
			args = append(args, "-b_ref_mode", "middle")
		} else if profile.VideoCodec == "h265" || profile.VideoCodec == "hevc" {
			args = append(args, "-c:v", "hevc_nvenc")
			args = append(args, "-preset", "p4")
			args = append(args, "-tune", "hq")
			args = append(args, "-rc", "vbr")
		}

	case types.HardwareIntel:
		// Intel Quick Sync
		if hw.DevicePath != "" {
			args = append(args, "-init_hw_device", fmt.Sprintf("vaapi=va:%s", hw.DevicePath))
			args = append(args, "-filter_hw_device", "va")
		}

		if profile.VideoCodec == "h264" {
			args = append(args, "-c:v", "h264_vaapi")
			args = append(args, "-vaapi_device", hw.DevicePath)
		} else if profile.VideoCodec == "h265" || profile.VideoCodec == "hevc" {
			args = append(args, "-c:v", "hevc_vaapi")
			args = append(args, "-vaapi_device", hw.DevicePath)
		}

	case types.HardwareAMD:
		// AMD VCE/VCN
		if strings.Contains(hw.DevicePath, "/dev/dri") {
			// Linux VA-API path
			if hw.DevicePath != "" {
				args = append(args, "-init_hw_device", fmt.Sprintf("vaapi=va:%s", hw.DevicePath))
				args = append(args, "-filter_hw_device", "va")
			}

			if profile.VideoCodec == "h264" {
				args = append(args, "-c:v", "h264_vaapi")
				args = append(args, "-vaapi_device", hw.DevicePath)
			} else if profile.VideoCodec == "h265" || profile.VideoCodec == "hevc" {
				args = append(args, "-c:v", "hevc_vaapi")
				args = append(args, "-vaapi_device", hw.DevicePath)
			}
		} else {
			// Windows AMF path
			if profile.VideoCodec == "h264" {
				args = append(args, "-c:v", "h264_amf")
				args = append(args, "-usage", "transcoding")
				args = append(args, "-quality", "balanced")
			} else if profile.VideoCodec == "h265" || profile.VideoCodec == "hevc" {
				args = append(args, "-c:v", "hevc_amf")
				args = append(args, "-usage", "transcoding")
				args = append(args, "-quality", "balanced")
			}
		}

	case types.HardwareCPU:
		// Software encoding
		if profile.VideoCodec == "h264" {
			args = append(args, "-c:v", "libx264")
		} else if profile.VideoCodec == "h265" || profile.VideoCodec == "hevc" {
			args = append(args, "-c:v", "libx265")
		}
	}

	// Handle copy codec for all hardware types
	if profile.VideoCodec == "copy" {
		args = append(args, "-c:v", "copy")
	}

	// Add audio codec
	if profile.AudioCodec == "aac" {
		args = append(args, "-c:a", "aac")
	} else if profile.AudioCodec == "mp3" {
		args = append(args, "-c:a", "libmp3lame")
	} else if profile.AudioCodec == "copy" {
		args = append(args, "-c:a", "copy")
	}

	// Add bitrate settings if specified
	if profile.VideoBitrate != "" && profile.VideoBitrate != "adaptive" && profile.VideoCodec != "copy" {
		args = append(args, "-b:v", profile.VideoBitrate)
	}
	if profile.AudioBitrate != "" && profile.AudioBitrate != "adaptive" && profile.AudioCodec != "copy" {
		args = append(args, "-b:a", profile.AudioBitrate)
	}

	// Add container format
	if profile.Container != "" {
		args = append(args, "-f", profile.Container)
	}

	// Add any extra arguments from the profile
	args = append(args, profile.ExtraArgs...)

	return args
}
