// Package transcode handles video and audio transcoding operations.
package transcode

import (
	"context"
	"io"
	"log"

	"github.com/savid/iptv-proxy/pkg/hardware"
	"github.com/savid/iptv-proxy/pkg/types"
)

// Transcoder defines the interface for media transcoding.
type Transcoder interface {
	Start(ctx context.Context) error
	io.ReadWriteCloser
}

// NewTranscoder creates a new transcoder instance based on the provided configuration.
func NewTranscoder(
	videoCodec, audioCodec string,
	videoBitrate, audioBitrate string,
	hardware types.HardwareInfo,
	buffer types.BufferConfig,
	selector *hardware.Selector,
	inputURL string,
	logger *log.Logger,
) (Transcoder, error) {
	// Create profile from codec settings
	prof := CreateProfile(videoCodec, audioCodec, videoBitrate, audioBitrate)

	// Apply hardware settings to profile
	prof = ApplyHardware(prof, hardware)

	// For now, we only support FFmpeg transcoding
	return NewFFmpegTranscoder(prof, hardware, buffer, selector, inputURL, logger), nil
}
