// Package proxy provides HTTP stream proxying functionality for IPTV streams.
package proxy

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/savid/iptv-proxy/internal/buffer"
	"github.com/savid/iptv-proxy/internal/hardware"
	"github.com/savid/iptv-proxy/internal/transcode"
	"github.com/savid/iptv-proxy/internal/types"
)

// StreamTranscoder handles transcoding and proxying of IPTV streams.
type StreamTranscoder struct {
	selector *hardware.Selector
	config   *TranscoderConfig
	logger   *log.Logger
}

// TranscoderConfig holds configuration for the stream transcoder.
type TranscoderConfig struct {
	VideoCodec          string
	AudioCodec          string
	VideoBitrate        string
	AudioBitrate        string
	HardwareAccel       string
	BufferSize          int
	BufferPrefetchRatio float64
	MinThreshold        int
	MaxRetries          int
	RetryDelay          time.Duration
}

// NewStreamTranscoder creates a new stream transcoder instance.
func NewStreamTranscoder(cfg *TranscoderConfig, logger *log.Logger) (*StreamTranscoder, error) {
	// Initialize hardware detector and selector
	detector := hardware.NewDetector(logger)
	selector := hardware.NewSelector(detector, types.HardwareType(cfg.HardwareAccel), logger)

	if err := selector.Initialize(); err != nil {
		return nil, fmt.Errorf("failed to initialize hardware selector: %w", err)
	}

	return &StreamTranscoder{
		selector: selector,
		config:   cfg,
		logger:   logger,
	}, nil
}

// TranscodeStream handles transcoding of a stream from the given URL.
func (st *StreamTranscoder) TranscodeStream(w http.ResponseWriter, r *http.Request, targetURL string) error {
	ctx := r.Context()

	// Select hardware for the video codec
	hw, err := st.selector.SelectHardware(st.config.VideoCodec)
	if err != nil {
		return fmt.Errorf("failed to select hardware: %w", err)
	}

	st.logger.Printf("Transcoding stream with video=%s, audio=%s, hardware=%s", st.config.VideoCodec, st.config.AudioCodec, hw.Type)

	// Create buffer configuration
	bufferConfig := types.BufferConfig{
		Size:          st.config.BufferSize,
		PrefetchRatio: st.config.BufferPrefetchRatio,
		MinThreshold:  st.config.MinThreshold,
		MaxRetries:    st.config.MaxRetries,
		RetryDelay:    st.config.RetryDelay,
	}

	// plex-gump is the only supported profile

	// Probe the stream to get information
	streamInfo, err := transcode.ProbeStream(targetURL)
	if err != nil {
		st.logger.Printf("Failed to probe stream, using defaults: %v", err)
		// Continue with defaults
	}

	// Get video and audio bitrates
	videoBitrate := st.config.VideoBitrate
	audioBitrate := st.config.AudioBitrate

	// Apply adaptive bitrate if configured
	if videoBitrate == "adaptive" || audioBitrate == "adaptive" {
		adaptiveVideoBitrate, adaptiveAudioBitrate := transcode.CalculateAdaptiveBitrate(streamInfo)
		if videoBitrate == "adaptive" {
			videoBitrate = adaptiveVideoBitrate
		}
		if audioBitrate == "adaptive" {
			audioBitrate = adaptiveAudioBitrate
		}
	}

	// Create transcoder
	transcoder, err := transcode.NewTranscoder(
		st.config.VideoCodec,
		st.config.AudioCodec,
		videoBitrate,
		audioBitrate,
		hw,
		bufferConfig,
		st.selector,
		targetURL,
		st.logger,
	)
	if err != nil {
		return fmt.Errorf("failed to create transcoder: %w", err)
	}

	// Start transcoding
	if err := transcoder.Start(ctx); err != nil {
		return fmt.Errorf("failed to start transcoder: %w", err)
	}
	defer transcoder.Close()

	// Create buffer manager
	bufferManager := buffer.NewBufferManager(bufferConfig, st.logger)

	// Start buffering from transcoder output
	if err := bufferManager.Start(ctx, transcoder); err != nil {
		return fmt.Errorf("failed to start buffer manager: %w", err)
	}
	defer bufferManager.Close()

	// Set response headers
	w.Header().Set("Content-Type", "video/mp2t")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Video-Codec", st.config.VideoCodec)
	w.Header().Set("X-Audio-Codec", st.config.AudioCodec)
	w.Header().Set("X-Hardware-Acceleration", string(hw.Type))

	// Stream to client
	_, err = io.Copy(w, bufferManager)
	if err != nil && err != io.EOF {
		st.logger.Printf("Error streaming to client: %v", err)
		return err
	}

	// Log final statistics
	stats := bufferManager.Stats()
	st.logger.Printf("Stream completed - bytes: %d, underruns: %d, retries: %d",
		stats.BytesConsumed, stats.Underruns, stats.Retries)

	return nil
}
