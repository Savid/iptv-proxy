// Package types contains shared type definitions for the IPTV transcoding system.
package types

import "time"

// HardwareType represents the type of hardware acceleration available.
type HardwareType string

const (
	// HardwareAuto automatically selects the best available hardware.
	HardwareAuto HardwareType = "auto"
	// HardwareCPU uses software encoding on the CPU.
	HardwareCPU HardwareType = "cpu"
	// HardwareNVIDIA uses NVIDIA GPU acceleration (NVENC).
	HardwareNVIDIA HardwareType = "nvidia"
	// HardwareIntel uses Intel Quick Sync Video.
	HardwareIntel HardwareType = "intel"
	// HardwareAMD uses AMD VCE/VCN acceleration.
	HardwareAMD HardwareType = "amd"
)

// QualityPreset defines the quality level for transcoding.
type QualityPreset string

const (
	// QualityLow uses lower bitrates for smaller files.
	QualityLow QualityPreset = "low"
	// QualityMedium uses balanced bitrates for good quality and file size.
	QualityMedium QualityPreset = "medium"
	// QualityHigh uses higher bitrates for best quality.
	QualityHigh QualityPreset = "high"
	// QualityCustom allows manual bitrate specification.
	QualityCustom QualityPreset = "custom"
)

// TranscodeMode defines whether to copy streams or transcode them.
type TranscodeMode string

const (
	// TranscodeModeCopy copies streams without re-encoding.
	TranscodeModeCopy TranscodeMode = "copy"
	// TranscodeModeTranscode re-encodes streams with specified settings.
	TranscodeModeTranscode TranscodeMode = "transcode"
)

// TranscodingProfile defines the parameters for a transcoding operation.
type TranscodingProfile struct {
	Name          string
	VideoCodec    string
	AudioCodec    string
	HardwareAccel HardwareType
	VideoBitrate  string
	AudioBitrate  string
	Container     string
	ExtraArgs     []string
}

// HardwareInfo contains information about available hardware acceleration.
type HardwareInfo struct {
	Type         HardwareType
	DevicePath   string
	DeviceID     int    // Device index for multi-GPU systems
	DeviceName   string // Human-readable device name
	Capabilities []string
	Available    bool
}

// TranscodeSession tracks an active transcoding session.
type TranscodeSession struct {
	ID           string
	Profile      string
	Hardware     HardwareType
	StartTime    time.Time
	BytesRead    int64
	BytesWritten int64
}
