// Package config provides configuration management for the IPTV proxy server.
package config

import (
	"errors"
	"flag"
	"fmt"
	"net/url"
	"strings"
	"time"
)

var (
	// ErrM3UURLRequired is returned when M3U URL is not provided.
	ErrM3UURLRequired = errors.New("m3u URL is required")
	// ErrEPGURLRequired is returned when EPG URL is not provided.
	ErrEPGURLRequired = errors.New("epg URL is required")
	// ErrBaseURLRequired is returned when base URL is not provided.
	ErrBaseURLRequired = errors.New("base URL is required")
	// ErrInvalidPort is returned when port number is invalid.
	ErrInvalidPort = errors.New("invalid port number")
	// ErrRefreshIntervalPositive is returned when refresh interval is not positive.
	ErrRefreshIntervalPositive = errors.New("refresh interval must be positive")
	// ErrInvalidLogLevel is returned when log level is invalid.
	ErrInvalidLogLevel = errors.New("invalid log level")
	// ErrInvalidHardwareAccel is returned when hardware acceleration value is invalid.
	ErrInvalidHardwareAccel = errors.New("invalid hardware acceleration")
	// ErrBufferSizeTooSmall is returned when buffer size is less than 1MB.
	ErrBufferSizeTooSmall = errors.New("buffer size must be at least 1MB")
	// ErrInvalidPrefetchRatio is returned when buffer prefetch ratio is out of range.
	ErrInvalidPrefetchRatio = errors.New("buffer prefetch ratio must be between 0.0 and 1.0")
	// ErrInvalidTestChannelPort is returned when test channel port is invalid.
	ErrInvalidTestChannelPort = errors.New("invalid test channel port")
	// ErrInvalidVideoCodec is returned when video codec is invalid.
	ErrInvalidVideoCodec = errors.New("invalid video codec")
	// ErrInvalidAudioCodec is returned when audio codec is invalid.
	ErrInvalidAudioCodec = errors.New("invalid audio codec")
	// ErrCodecHardwareIncompatible is returned when codec is not compatible with hardware acceleration.
	ErrCodecHardwareIncompatible = errors.New("codec not compatible with selected hardware acceleration")
	// ErrInvalidTranscodeMode is returned when transcode mode is invalid.
	ErrInvalidTranscodeMode = errors.New("invalid transcode mode")
	// ErrInvalidVideoQuality is returned when video quality is invalid.
	ErrInvalidVideoQuality = errors.New("invalid video quality")
	// ErrInvalidAudioQuality is returned when audio quality is invalid.
	ErrInvalidAudioQuality = errors.New("invalid audio quality")
	// ErrCustomBitrateRequired is returned when custom quality is selected but no bitrate provided.
	ErrCustomBitrateRequired = errors.New("custom bitrate required when quality is 'custom'")
	// ErrInvalidHardwareDeviceFormat is returned when hardware device format is invalid.
	ErrInvalidHardwareDeviceFormat = errors.New("invalid hardware device format (must be auto, none, or device ID like nvidia:0)")
	// ErrInvalidDeviceID is returned when device ID is not a valid number.
	ErrInvalidDeviceID = errors.New("invalid device ID")
)

// Config holds the application configuration.
type Config struct {
	M3UURL          string
	EPGURL          string
	BaseURL         string
	BindAddr        string
	Port            int
	LogLevel        string
	RefreshInterval time.Duration
	TunerCount      int
	// New transcoding fields
	TranscodeMode      string `mapstructure:"transcode_mode"`
	HardwareDevice     string `mapstructure:"hardware_device"`
	VideoCodec         string `mapstructure:"video_codec"`
	AudioCodec         string `mapstructure:"audio_codec"`
	VideoQuality       string `mapstructure:"video_quality"`
	AudioQuality       string `mapstructure:"audio_quality"`
	CustomVideoBitrate string `mapstructure:"custom_video_bitrate"`
	CustomAudioBitrate string `mapstructure:"custom_audio_bitrate"`
	// Buffer settings
	BufferSize          int           `mapstructure:"buffer_size"`
	BufferDuration      time.Duration `mapstructure:"buffer_duration"`
	BufferPrefetchRatio float64       `mapstructure:"buffer_prefetch_ratio"`
	// Test settings
	EnableTestChannels bool `mapstructure:"enable_test_channels"`
	TestChannelPort    int  `mapstructure:"test_channel_port"`
}

// New creates a new configuration instance by parsing command-line flags.
func New() (*Config, error) {
	cfg := &Config{}

	flag.StringVar(&cfg.M3UURL, "m3u", "", "URL of the M3U playlist (required)")
	flag.StringVar(&cfg.EPGURL, "epg", "", "URL of the EPG XML file (required)")
	flag.StringVar(&cfg.BaseURL, "base", "", "Base URL for rewritten stream URLs (e.g., http://localhost:8080) (required)")
	flag.StringVar(&cfg.BindAddr, "bind", "0.0.0.0", "IP address to bind the server to")
	flag.IntVar(&cfg.Port, "port", 8080, "Port to listen on")
	flag.StringVar(&cfg.LogLevel, "log-level", "info", "Log level (debug, info, warn, error)")
	flag.DurationVar(&cfg.RefreshInterval, "refresh-interval", 30*time.Minute, "Interval between data refreshes")
	flag.IntVar(&cfg.TunerCount, "tuner-count", 2, "Number of tuners to advertise")
	// New transcoding flags
	flag.StringVar(&cfg.TranscodeMode, "transcode-mode", "transcode", "Transcoding mode: copy or transcode")
	flag.StringVar(&cfg.HardwareDevice, "hardware-device", "auto", "Hardware device: auto, none, or device ID (e.g., nvidia:0, intel:0)")
	flag.StringVar(&cfg.VideoCodec, "video-codec", "h264", "Video codec when transcoding: h264, h265, vp9, mpeg2")
	flag.StringVar(&cfg.AudioCodec, "audio-codec", "aac", "Audio codec when transcoding: aac, mp3, mp2, opus")
	flag.StringVar(&cfg.VideoQuality, "video-quality", "medium", "Video quality: low, medium, high, or custom")
	flag.StringVar(&cfg.AudioQuality, "audio-quality", "medium", "Audio quality: low, medium, high, or custom")
	flag.StringVar(&cfg.CustomVideoBitrate, "custom-video-bitrate", "", "Custom video bitrate when quality is 'custom'")
	flag.StringVar(&cfg.CustomAudioBitrate, "custom-audio-bitrate", "", "Custom audio bitrate when quality is 'custom'")
	// Buffer flags
	flag.IntVar(&cfg.BufferSize, "buffer-size", 10, "Buffer size in MB")
	flag.DurationVar(&cfg.BufferDuration, "buffer-duration", 10*time.Second, "Buffer duration")
	flag.Float64Var(&cfg.BufferPrefetchRatio, "buffer-prefetch-ratio", 0.8, "Buffer prefetch ratio (0.0-1.0)")
	// Test flags
	flag.BoolVar(&cfg.EnableTestChannels, "test-channels", false, "Enable test channels")
	flag.IntVar(&cfg.TestChannelPort, "test-port", 8889, "Port for test channel server")

	flag.Parse()

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate checks if the configuration is valid.
func (c *Config) Validate() error {
	if c.M3UURL == "" {
		return ErrM3UURLRequired
	}

	if c.EPGURL == "" {
		return ErrEPGURLRequired
	}

	if c.BaseURL == "" {
		return ErrBaseURLRequired
	}

	if _, err := url.Parse(c.M3UURL); err != nil {
		return fmt.Errorf("invalid M3U URL: %w", err)
	}

	if _, err := url.Parse(c.EPGURL); err != nil {
		return fmt.Errorf("invalid EPG URL: %w", err)
	}

	if _, err := url.Parse(c.BaseURL); err != nil {
		return fmt.Errorf("invalid base URL: %w", err)
	}

	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("%w: %d", ErrInvalidPort, c.Port)
	}

	if c.RefreshInterval <= 0 {
		return ErrRefreshIntervalPositive
	}

	validLogLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !validLogLevels[c.LogLevel] {
		return fmt.Errorf("%w: %s (must be debug, info, warn, or error)", ErrInvalidLogLevel, c.LogLevel)
	}

	// Validate transcode mode
	validTranscodeModes := map[string]bool{
		"copy":      true,
		"transcode": true,
	}
	if !validTranscodeModes[c.TranscodeMode] {
		return fmt.Errorf("%w: %s (must be copy or transcode)", ErrInvalidTranscodeMode, c.TranscodeMode)
	}

	// If transcode mode is copy, we don't need to validate codecs
	if c.TranscodeMode != "transcode" {
		return nil
	}

	// Validate video codec
	validVideoCodecs := map[string]bool{
		"h264":  true,
		"h265":  true,
		"vp9":   true,
		"mpeg2": true,
	}
	if !validVideoCodecs[c.VideoCodec] {
		return fmt.Errorf("%w: %s (must be h264, h265, vp9, or mpeg2)", ErrInvalidVideoCodec, c.VideoCodec)
	}

	// Validate audio codec
	validAudioCodecs := map[string]bool{
		"aac":  true,
		"mp3":  true,
		"mp2":  true,
		"opus": true,
	}
	if !validAudioCodecs[c.AudioCodec] {
		return fmt.Errorf("%w: %s (must be aac, mp3, mp2, or opus)", ErrInvalidAudioCodec, c.AudioCodec)
	}

	// Validate video quality
	validVideoQualities := map[string]bool{
		"low":    true,
		"medium": true,
		"high":   true,
		"custom": true,
	}
	if !validVideoQualities[c.VideoQuality] {
		return fmt.Errorf("%w: %s (must be low, medium, high, or custom)", ErrInvalidVideoQuality, c.VideoQuality)
	}

	// Validate audio quality
	validAudioQualities := map[string]bool{
		"low":    true,
		"medium": true,
		"high":   true,
		"custom": true,
	}
	if !validAudioQualities[c.AudioQuality] {
		return fmt.Errorf("%w: %s (must be low, medium, high, or custom)", ErrInvalidAudioQuality, c.AudioQuality)
	}

	// Validate custom bitrates when quality is custom
	if c.VideoQuality == "custom" && c.CustomVideoBitrate == "" {
		return fmt.Errorf("%w: video bitrate", ErrCustomBitrateRequired)
	}
	if c.AudioQuality == "custom" && c.CustomAudioBitrate == "" {
		return fmt.Errorf("%w: audio bitrate", ErrCustomBitrateRequired)
	}

	// Validate hardware device - basic validation, more detailed validation happens at runtime
	if c.HardwareDevice != "auto" && c.HardwareDevice != "none" {
		// Device format should be type:id (e.g., nvidia:0, intel:0)
		if !strings.Contains(c.HardwareDevice, ":") {
			return fmt.Errorf("%w: %s", ErrInvalidHardwareDeviceFormat, c.HardwareDevice)
		}
	}

	// Validate buffer size (now in MB)
	if c.BufferSize < 1 { // 1MB minimum
		return ErrBufferSizeTooSmall
	}

	// Validate buffer prefetch ratio
	if c.BufferPrefetchRatio < 0.0 || c.BufferPrefetchRatio > 1.0 {
		return ErrInvalidPrefetchRatio
	}

	// Validate test channel port
	if c.EnableTestChannels && (c.TestChannelPort < 1 || c.TestChannelPort > 65535) {
		return fmt.Errorf("%w: %d", ErrInvalidTestChannelPort, c.TestChannelPort)
	}

	return nil
}

// ParseHardwareDevice parses a hardware device string like "nvidia:0" into type and ID.
func (c *Config) ParseHardwareDevice() (deviceType string, deviceID int, err error) {
	if c.HardwareDevice == "auto" || c.HardwareDevice == "none" {
		return c.HardwareDevice, 0, nil
	}

	parts := strings.Split(c.HardwareDevice, ":")
	if len(parts) != 2 {
		return "", 0, fmt.Errorf("%w: %s", ErrInvalidHardwareDeviceFormat, c.HardwareDevice)
	}

	deviceType = parts[0]
	if _, err := fmt.Sscanf(parts[1], "%d", &deviceID); err != nil {
		return "", 0, fmt.Errorf("%w: %s", ErrInvalidDeviceID, parts[1])
	}

	return deviceType, deviceID, nil
}
