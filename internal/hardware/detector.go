// Package hardware provides GPU detection and selection for transcoding.
package hardware

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/savid/iptv-proxy/internal/types"
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
		Capabilities: []string{"h264", "h265", "vp8", "vp9"},
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
		return nil, fmt.Errorf("no NVIDIA GPUs found")
	}

	// Use the first available GPU
	parts := strings.Split(lines[0], ", ")
	if len(parts) < 2 {
		return nil, fmt.Errorf("unexpected nvidia-smi output format")
	}

	d.logger.Printf("Detected NVIDIA GPU: %s", parts[0])

	// Test NVENC availability
	capabilities := []string{}
	if d.TestHardwareCodec(types.HardwareInfo{Type: types.HardwareNVIDIA}, "h264_nvenc") {
		capabilities = append(capabilities, "h264")
	}
	if d.TestHardwareCodec(types.HardwareInfo{Type: types.HardwareNVIDIA}, "hevc_nvenc") {
		capabilities = append(capabilities, "h265")
	}

	if len(capabilities) == 0 {
		return nil, fmt.Errorf("NVIDIA GPU found but NVENC not available")
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
		return nil, fmt.Errorf("no render nodes found")
	}

	// Try to find Intel GPU using vainfo
	for _, node := range renderNodes {
		cmd := exec.Command("vainfo", "--display", "drm", "--device", node)
		output, err := cmd.CombinedOutput()
		if err != nil {
			continue
		}

		outputStr := string(output)
		if strings.Contains(outputStr, "Intel") || strings.Contains(outputStr, "i965") || strings.Contains(outputStr, "iHD") {
			d.logger.Printf("Detected Intel GPU at %s", node)

			// Test codec support
			capabilities := []string{}
			if strings.Contains(outputStr, "H264") || strings.Contains(outputStr, "AVC") {
				capabilities = append(capabilities, "h264")
			}
			if strings.Contains(outputStr, "H265") || strings.Contains(outputStr, "HEVC") {
				capabilities = append(capabilities, "h265")
			}
			if strings.Contains(outputStr, "VP8") {
				capabilities = append(capabilities, "vp8")
			}
			if strings.Contains(outputStr, "VP9") {
				capabilities = append(capabilities, "vp9")
			}

			if len(capabilities) > 0 {
				return &types.HardwareInfo{
					Type:         types.HardwareIntel,
					DevicePath:   node,
					Capabilities: capabilities,
					Available:    true,
				}, nil
			}
		}
	}

	return nil, fmt.Errorf("no Intel GPU with video acceleration found")
}

// CheckAMD detects AMD GPU availability through VA-API or AMF.
func (d *Detector) CheckAMD() (*types.HardwareInfo, error) {
	// Check for AMD GPU render nodes
	renderNodes, err := filepath.Glob("/dev/dri/renderD*")
	if err != nil || len(renderNodes) == 0 {
		return nil, fmt.Errorf("no render nodes found")
	}

	// Try to find AMD GPU using vainfo
	for _, node := range renderNodes {
		cmd := exec.Command("vainfo", "--display", "drm", "--device", node)
		output, err := cmd.CombinedOutput()
		if err != nil {
			continue
		}

		outputStr := string(output)
		if strings.Contains(outputStr, "AMD") || strings.Contains(outputStr, "radeonsi") {
			d.logger.Printf("Detected AMD GPU at %s", node)

			// Test codec support
			capabilities := []string{}
			if strings.Contains(outputStr, "H264") || strings.Contains(outputStr, "AVC") {
				capabilities = append(capabilities, "h264")
			}
			if strings.Contains(outputStr, "H265") || strings.Contains(outputStr, "HEVC") {
				capabilities = append(capabilities, "h265")
			}
			if strings.Contains(outputStr, "VP8") {
				capabilities = append(capabilities, "vp8")
			}
			if strings.Contains(outputStr, "VP9") {
				capabilities = append(capabilities, "vp9")
			}

			if len(capabilities) > 0 {
				return &types.HardwareInfo{
					Type:         types.HardwareAMD,
					DevicePath:   node,
					Capabilities: capabilities,
					Available:    true,
				}, nil
			}
		}
	}

	// Check for AMD AMF on Windows
	if strings.Contains(strings.ToLower(os.Getenv("OS")), "windows") {
		// Test AMF availability
		if d.TestHardwareCodec(types.HardwareInfo{Type: types.HardwareAMD}, "h264_amf") {
			return &types.HardwareInfo{
				Type:         types.HardwareAMD,
				DevicePath:   "",
				Capabilities: []string{"h264", "h265"},
				Available:    true,
			}, nil
		}
	}

	return nil, fmt.Errorf("no AMD GPU with video acceleration found")
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
	}

	// Output to null
	args = append(args, "-f", "null", "-")

	cmd := exec.Command("ffmpeg", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		d.logger.Printf("Hardware codec %s test failed: %v", codec, err)
		return false
	}

	return true
}
