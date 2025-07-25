// Package handlers contains HTTP request handlers.
package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/savid/iptv-proxy/config"
	"github.com/savid/iptv-proxy/pkg/streaming/proxy"
	"github.com/savid/iptv-proxy/pkg/streaming/transcode"
	"github.com/savid/iptv-proxy/pkg/utils"
)

const (
	codecCopy = "copy"
	modeAuto  = "auto"
	modeNone  = "none"
)

// StreamV2Handler handles streaming requests with transcoding support.
type StreamV2Handler struct {
	transcoder *proxy.StreamTranscoder
	logger     *log.Logger
}

// getVideoBitrate returns the video bitrate based on quality settings.
func getVideoBitrate(cfg *config.Config, mapper *transcode.QualityMapper) string {
	if cfg.VideoQuality == "custom" {
		return cfg.CustomVideoBitrate
	}
	return mapper.GetVideoBitrate(cfg.VideoQuality, cfg.VideoCodec)
}

// getAudioBitrate returns the audio bitrate based on quality settings.
func getAudioBitrate(cfg *config.Config, mapper *transcode.QualityMapper) string {
	if cfg.AudioQuality == "custom" {
		return cfg.CustomAudioBitrate
	}
	return mapper.GetAudioBitrate(cfg.AudioQuality, cfg.AudioCodec)
}

// NewStreamV2Handler creates a new stream handler with transcoding support.
func NewStreamV2Handler(cfg *config.Config, logger *log.Logger) (*StreamV2Handler, error) {
	// Create quality mapper
	qualityMapper := transcode.NewQualityMapper()

	// Determine video and audio codecs based on transcode mode
	videoCodec := cfg.VideoCodec
	audioCodec := cfg.AudioCodec
	videoBitrate := ""
	audioBitrate := ""

	if cfg.TranscodeMode == codecCopy {
		videoCodec = codecCopy
		audioCodec = codecCopy
	} else {
		videoBitrate = getVideoBitrate(cfg, qualityMapper)
		audioBitrate = getAudioBitrate(cfg, qualityMapper)
	}

	// Parse hardware device
	hardwareAccel := modeAuto
	if cfg.HardwareDevice == modeNone {
		hardwareAccel = modeNone
	} else if cfg.HardwareDevice != modeAuto {
		// For specific devices, still use auto and let the selector handle it
		hardwareAccel = modeAuto
	}

	// Create transcoder configuration
	transcoderConfig := &proxy.TranscoderConfig{
		VideoCodec:          videoCodec,
		AudioCodec:          audioCodec,
		VideoBitrate:        videoBitrate,
		AudioBitrate:        audioBitrate,
		HardwareAccel:       hardwareAccel,
		BufferSize:          cfg.BufferSize * 1024 * 1024, // Convert MB to bytes
		BufferPrefetchRatio: cfg.BufferPrefetchRatio,
		MinThreshold:        64 * 1024, // 64KB
		MaxRetries:          3,
		RetryDelay:          time.Second,
	}

	// Create transcoder
	transcoder, err := proxy.NewStreamTranscoder(transcoderConfig, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create transcoder: %w", err)
	}

	return &StreamV2Handler{
		transcoder: transcoder,
		logger:     logger,
	}, nil
}

// ServeHTTP handles HTTP requests for stream transcoding.
func (h *StreamV2Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Extract encoded URL from path
	// Expected format: /stream/{encodedURL}
	path := strings.TrimPrefix(r.URL.Path, "/stream/")
	if path == "" {
		http.Error(w, "Missing stream URL", http.StatusBadRequest)
		return
	}

	// The URL should already be encoded
	var targetURL string

	// Check if this looks like a URL (contains ://)
	if strings.Contains(path, "://") {
		// Raw URL passed
		targetURL = path
	} else {
		// Encoded URL
		decodedURL, err := utils.DecodeURL(path)
		if err != nil {
			http.Error(w, "Invalid encoded URL", http.StatusBadRequest)
			return
		}
		targetURL = decodedURL
	}

	h.logger.Printf("Streaming request - url: %s", targetURL)

	// Stream with transcoding
	if err := h.transcoder.TranscodeStream(w, r, targetURL); err != nil {
		h.logger.Printf("Stream error: %v", err)
		// Don't write error to response as headers may already be sent
	}
}
