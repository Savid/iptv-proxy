// Package proxy provides HTTP stream proxying functionality for IPTV streams.
package proxy

import (
	"time"

	"github.com/savid/iptv-proxy/pkg/types"
)

// DefaultBufferConfig returns the default buffer configuration.
func DefaultBufferConfig() types.BufferConfig {
	return types.BufferConfig{
		Size:          10 * 1024 * 1024, // 10MB
		PrefetchRatio: 0.8,              // Keep buffer 80% full
		MinThreshold:  64 * 1024,        // 64KB minimum before reading
		MaxRetries:    3,
		RetryDelay:    time.Second,
	}
}

// DefaultTranscoderConfig returns the default transcoder configuration.
func DefaultTranscoderConfig() *TranscoderConfig {
	bufConfig := DefaultBufferConfig()
	return &TranscoderConfig{
		VideoCodec:          "copy",
		AudioCodec:          "copy",
		VideoBitrate:        "copy",
		AudioBitrate:        "copy",
		HardwareAccel:       "auto",
		BufferSize:          bufConfig.Size,
		BufferPrefetchRatio: bufConfig.PrefetchRatio,
		MinThreshold:        bufConfig.MinThreshold,
		MaxRetries:          bufConfig.MaxRetries,
		RetryDelay:          bufConfig.RetryDelay,
	}
}
