// Package hardware provides GPU detection and selection for transcoding.
package hardware

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/savid/iptv-proxy/internal/types"
)

var (
	// ErrNoNVIDIAGPU is returned when no NVIDIA GPUs are found.
	ErrNoNVIDIAGPU = errors.New("no NVIDIA GPUs found")
	// ErrNVIDIASMIFormat is returned when nvidia-smi output format is unexpected.
	ErrNVIDIASMIFormat = errors.New("unexpected nvidia-smi output format")
	// ErrNVENCNotAvailable is returned when NVIDIA GPU found but NVENC not available.
	ErrNVENCNotAvailable = errors.New("NVIDIA GPU found but NVENC not available")
	// ErrNoRenderNodes is returned when no render nodes are found.
	ErrNoRenderNodes = errors.New("no render nodes found")
	// ErrNoIntelGPU is returned when no Intel GPU with video acceleration found.
	ErrNoIntelGPU = errors.New("no Intel GPU with video acceleration found")
	// ErrNoAMDGPU is returned when no AMD GPU with video acceleration found.
	ErrNoAMDGPU = errors.New("no AMD GPU with video acceleration found")
)

// Detector identifies available hardware acceleration devices.
type Detector struct {
	logger *log.Logger
}

// NewDetector creates a new hardware detector instance.
func NewDetector(logger *log.Logger) *Detector {
	return &Detector{
		logger: logger,
	}
}

// DetectGPUs scans the system for available GPU hardware.
func (d *Detector) DetectGPUs() []types.HardwareInfo {
	var gpus []types.HardwareInfo

	// Always add CPU as a fallback option
	gpus = append(gpus, types.HardwareInfo{
		Type:         types.HardwareCPU,
		DevicePath:   "",
		Capabilities: []string{codecH264, codecH265, "vp8", "vp9"},
		Available:    true,
	})

	// Check for NVIDIA GPU
	if nvidia, err := d.CheckNVIDIA(); err == nil && nvidia != nil {
		gpus = append(gpus, *nvidia)
	}

	// Check for Intel GPU
	if intel, err := d.CheckIntel(); err == nil && intel != nil {
		gpus = append(gpus, *intel)
	}

	// Check for AMD GPU
	if amd, err := d.CheckAMD(); err == nil && amd != nil {
		gpus = append(gpus, *amd)
	}

	return gpus
}

// CheckNVIDIA detects NVIDIA GPU availability using nvidia-smi.
func (d *Detector) CheckNVIDIA() (*types.HardwareInfo, error) {
	// Check if nvidia-smi exists
	cmd := exec.Command("nvidia-smi", "--query-gpu=name,uuid", "--format=csv,noheader")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("nvidia-smi not available: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 0 {
		return nil, ErrNoNVIDIAGPU
	}

	// Use the first available GPU
	parts := strings.Split(lines[0], ", ")
	if len(parts) < 2 {
		return nil, ErrNVIDIASMIFormat
	}

	d.logger.Printf("Detected NVIDIA GPU: %s", parts[0])

	// Test NVENC availability
	capabilities := []string{}
	if d.TestHardwareCodec(types.HardwareInfo{Type: types.HardwareNVIDIA}, "h264_nvenc") {
		capabilities = append(capabilities, codecH264)
	}
	if d.TestHardwareCodec(types.HardwareInfo{Type: types.HardwareNVIDIA}, "hevc_nvenc") {
		capabilities = append(capabilities, codecH265)
	}

	if len(capabilities) == 0 {
		return nil, ErrNVENCNotAvailable
	}

	return &types.HardwareInfo{
		Type:         types.HardwareNVIDIA,
		DevicePath:   parts[1], // GPU UUID
		Capabilities: capabilities,
		Available:    true,
	}, nil
}

// CheckIntel detects Intel GPU availability through VA-API.
func (d *Detector) CheckIntel() (*types.HardwareInfo, error) {
	// Check for Intel GPU render nodes
	renderNodes, err := filepath.Glob("/dev/dri/renderD*")
	if err != nil || len(renderNodes) == 0 {
		return nil, ErrNoRenderNodes
	}

	// Try to find Intel GPU using vainfo
	for _, node := range renderNodes {
		hwInfo := d.checkIntelNode(node)
		if hwInfo != nil {
			return hwInfo, nil
		}
	}

	return nil, ErrNoIntelGPU
}

// checkIntelNode checks if a specific node is an Intel GPU.
func (d *Detector) checkIntelNode(node string) *types.HardwareInfo {
	cmd := exec.Command("vainfo", "--display", "drm", "--device", node) // #nosec G204 - node comes from filepath.Glob
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil
	}

	outputStr := string(output)
	if !d.isIntelGPU(outputStr) {
		return nil
	}

	d.logger.Printf("Detected Intel GPU at %s", node)
	capabilities := d.extractCodecCapabilities(outputStr)

	if len(capabilities) == 0 {
		return nil
	}

	return &types.HardwareInfo{
		Type:         types.HardwareIntel,
		DevicePath:   node,
		Capabilities: capabilities,
		Available:    true,
	}
}

// isIntelGPU checks if the vainfo output indicates an Intel GPU.
func (d *Detector) isIntelGPU(output string) bool {
	return strings.Contains(output, "Intel") ||
		strings.Contains(output, "i965") ||
		strings.Contains(output, "iHD")
}

// extractCodecCapabilities extracts supported codecs from vainfo output.
func (d *Detector) extractCodecCapabilities(output string) []string {
	capabilities := []string{}

	if strings.Contains(output, "H264") || strings.Contains(output, "AVC") {
		capabilities = append(capabilities, codecH264)
	}
	if strings.Contains(output, "H265") || strings.Contains(output, "HEVC") {
		capabilities = append(capabilities, codecH265)
	}
	if strings.Contains(output, "VP8") {
		capabilities = append(capabilities, "vp8")
	}
	if strings.Contains(output, "VP9") {
		capabilities = append(capabilities, "vp9")
	}

	return capabilities
}

// CheckAMD detects AMD GPU availability through VA-API or AMF.
func (d *Detector) CheckAMD() (*types.HardwareInfo, error) {
	// Check for AMD GPU render nodes
	renderNodes, err := filepath.Glob("/dev/dri/renderD*")
	if err != nil || len(renderNodes) == 0 {
		return nil, ErrNoRenderNodes
	}

	// Try to find AMD GPU using vainfo
	for _, node := range renderNodes {
		hwInfo := d.checkAMDNode(node)
		if hwInfo != nil {
			return hwInfo, nil
		}
	}

	// Check for AMD AMF on Windows
	if d.isWindowsAMFAvailable() {
		return &types.HardwareInfo{
			Type:         types.HardwareAMD,
			DevicePath:   "",
			Capabilities: []string{codecH264, codecH265},
			Available:    true,
		}, nil
	}

	return nil, ErrNoAMDGPU
}

// checkAMDNode checks if a specific node is an AMD GPU.
func (d *Detector) checkAMDNode(node string) *types.HardwareInfo {
	cmd := exec.Command("vainfo", "--display", "drm", "--device", node) // #nosec G204 - node comes from filepath.Glob
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil
	}

	outputStr := string(output)
	if !d.isAMDGPU(outputStr) {
		return nil
	}

	d.logger.Printf("Detected AMD GPU at %s", node)
	capabilities := d.extractCodecCapabilities(outputStr)

	if len(capabilities) == 0 {
		return nil
	}

	return &types.HardwareInfo{
		Type:         types.HardwareAMD,
		DevicePath:   node,
		Capabilities: capabilities,
		Available:    true,
	}
}

// isAMDGPU checks if the vainfo output indicates an AMD GPU.
func (d *Detector) isAMDGPU(output string) bool {
	return strings.Contains(output, "AMD") || strings.Contains(output, "radeonsi")
}

// isWindowsAMFAvailable checks if AMD AMF is available on Windows.
func (d *Detector) isWindowsAMFAvailable() bool {
	if !strings.Contains(strings.ToLower(os.Getenv("OS")), "windows") {
		return false
	}
	return d.TestHardwareCodec(types.HardwareInfo{Type: types.HardwareAMD}, "h264_amf")
}

// TestHardwareCodec tests if a specific hardware codec is available.
func (d *Detector) TestHardwareCodec(hw types.HardwareInfo, codec string) bool {
	// Create a small test encoding command
	args := []string{
		"-f", "lavfi",
		"-i", "testsrc=duration=1:size=320x240:rate=1",
		"-c:v", codec,
	}

	// Add hardware-specific options
	switch hw.Type {
	case types.HardwareNVIDIA:
		// NVIDIA doesn't need special input options for testing
	case types.HardwareIntel, types.HardwareAMD:
		if hw.DevicePath != "" {
			args = append([]string{"-vaapi_device", hw.DevicePath}, args...)
		}
	case types.HardwareAuto, types.HardwareCPU:
		// No special options needed for auto or CPU
	}

	// Output to null
	args = append(args, "-f", "null", "-")

	cmd := exec.Command("ffmpeg", args...) // #nosec G204 - args are internally constructed
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		d.logger.Printf("Hardware codec %s test failed: %v", codec, err)
		return false
	}

	return true
}
