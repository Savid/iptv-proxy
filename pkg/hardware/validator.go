// Package hardware provides GPU detection and selection for transcoding.
package hardware

import (
	"errors"
	"fmt"
	"log"

	"github.com/savid/iptv-proxy/pkg/types"
)

var (
	// ErrCodecNotSupported is returned when a codec is not supported by the hardware.
	ErrCodecNotSupported = errors.New("codec not supported by hardware")
)

// Validator validates codec and hardware compatibility.
type Validator struct {
	logger *log.Logger
}

// NewValidator creates a new codec/hardware validator instance.
func NewValidator(logger *log.Logger) *Validator {
	return &Validator{
		logger: logger,
	}
}

// ValidateCodecHardware validates that the given video codec is compatible with the hardware.
func (v *Validator) ValidateCodecHardware(videoCodec string, hw types.HardwareInfo) error {
	supportedCodecs := v.GetSupportedCodecs(hw)

	for _, supported := range supportedCodecs {
		if supported == videoCodec {
			return nil
		}
	}

	return fmt.Errorf("%w: codec %s on %s", ErrCodecNotSupported, videoCodec, hw.Type)
}

// GetSupportedCodecs returns the list of codecs supported by the given hardware.
func (v *Validator) GetSupportedCodecs(hw types.HardwareInfo) []string {
	switch hw.Type {
	case types.HardwareNVIDIA:
		// NVIDIA supports h264 and h265 via NVENC
		return []string{codecH264, codecH265}

	case types.HardwareIntel:
		// Intel Quick Sync Video supports h264, h265, and sometimes vp9
		codecs := []string{codecH264, codecH265}
		// Check if VP9 is in capabilities (detected from vainfo)
		for _, cap := range hw.Capabilities {
			if cap == "vp9" {
				codecs = append(codecs, "vp9")
				break
			}
		}
		return codecs

	case types.HardwareAMD:
		// AMD VCE/VCN supports h264 and h265
		return []string{codecH264, codecH265}

	case types.HardwareCPU:
		// CPU supports all codecs
		return []string{codecH264, codecH265, "vp9", "mpeg2"}

	case types.HardwareAuto:
		// Auto mode - return all possible codecs, actual selection happens at runtime
		return []string{codecH264, codecH265, "vp9", "mpeg2"}

	default:
		// Unknown hardware type - return empty list
		return []string{}
	}
}

// CanHardwareEncodeCodec checks if the given hardware can encode the specified codec.
func (v *Validator) CanHardwareEncodeCodec(hw types.HardwareInfo, codec string) bool {
	// Special case: mpeg2 is always CPU-only
	if codec == "mpeg2" {
		return hw.Type == types.HardwareCPU
	}

	// Check against hardware capabilities
	for _, cap := range hw.Capabilities {
		if cap == codec {
			return true
		}
	}

	return false
}
