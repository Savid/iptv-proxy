// Package config provides configuration management for the IPTV proxy server.
package config

import (
	"errors"
	"flag"
	"fmt"
	"net/url"
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
)

// Config holds the application configuration.
type Config struct {
	M3UURL              string
	EPGURL              string
	BaseURL             string
	BindAddr            string
	Port                int
	LogLevel            string
	RefreshInterval     time.Duration
	TunerCount          int
	VideoCodec          string        `mapstructure:"video_codec"`
	AudioCodec          string        `mapstructure:"audio_codec"`
	VideoBitrate        string        `mapstructure:"video_bitrate"`
	AudioBitrate        string        `mapstructure:"audio_bitrate"`
	HardwareAccel       string        `mapstructure:"hardware_accel"`
	BufferSize          int           `mapstructure:"buffer_size"`
	BufferDuration      time.Duration `mapstructure:"buffer_duration"`
	BufferPrefetchRatio float64       `mapstructure:"buffer_prefetch_ratio"`
	EnableTestChannels  bool          `mapstructure:"enable_test_channels"`
	TestChannelPort     int           `mapstructure:"test_channel_port"`
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
	flag.StringVar(&cfg.VideoCodec, "video-codec", "mpeg2", "Video codec: copy, h264, h265, mpeg2")
	flag.StringVar(&cfg.AudioCodec, "audio-codec", "mp2", "Audio codec: copy, aac, mp3, mp2")
	flag.StringVar(&cfg.VideoBitrate, "video-bitrate", "6000k", "Video bitrate (e.g., 6000k, 8M)")
	flag.StringVar(&cfg.AudioBitrate, "audio-bitrate", "224k", "Audio bitrate (e.g., 192k, 224k)")
	flag.StringVar(&cfg.HardwareAccel, "hardware-accel", "auto", "Hardware acceleration: auto, none, nvidia, intel, amd")
	flag.IntVar(&cfg.BufferSize, "buffer-size", 10, "Buffer size in MB")
	flag.DurationVar(&cfg.BufferDuration, "buffer-duration", 10*time.Second, "Buffer duration")
	flag.Float64Var(&cfg.BufferPrefetchRatio, "buffer-prefetch-ratio", 0.8, "Buffer prefetch ratio (0.0-1.0)")
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

	// Validate video codec
	validVideoCodecs := map[string]bool{
		"copy":  true,
		"h264":  true,
		"h265":  true,
		"mpeg2": true,
	}
	if !validVideoCodecs[c.VideoCodec] {
		return fmt.Errorf("%w: %s (must be copy, h264, h265, or mpeg2)", ErrInvalidVideoCodec, c.VideoCodec)
	}

	// Validate audio codec
	validAudioCodecs := map[string]bool{
		"copy": true,
		"aac":  true,
		"mp3":  true,
		"mp2":  true,
	}
	if !validAudioCodecs[c.AudioCodec] {
		return fmt.Errorf("%w: %s (must be copy, aac, mp3, or mp2)", ErrInvalidAudioCodec, c.AudioCodec)
	}

	// Validate hardware acceleration
	validHardware := map[string]bool{
		"auto":   true,
		"none":   true,
		"nvidia": true,
		"intel":  true,
		"amd":    true,
	}
	if !validHardware[c.HardwareAccel] {
		return fmt.Errorf("%w: %s (must be auto, none, nvidia, intel, or amd)", ErrInvalidHardwareAccel, c.HardwareAccel)
	}

	// Validate codec compatibility with hardware acceleration
	if c.HardwareAccel != "none" && c.HardwareAccel != "auto" {
		// GPU acceleration only supports h264 and h265
		if c.VideoCodec != "copy" && c.VideoCodec != "h264" && c.VideoCodec != "h265" {
			return fmt.Errorf("%w: %s codec cannot be hardware accelerated", ErrCodecHardwareIncompatible, c.VideoCodec)
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
