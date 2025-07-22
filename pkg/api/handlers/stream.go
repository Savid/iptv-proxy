package handlers

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/savid/iptv-proxy/pkg/streaming/proxy"
	"github.com/savid/iptv-proxy/pkg/utils"
	"github.com/sirupsen/logrus"
)

var (
	// ErrInvalidPathFormat is returned when the request path doesn't match expected format.
	ErrInvalidPathFormat = errors.New("invalid path format")
	// ErrMissingEncodedURL is returned when the encoded URL is missing from the path.
	ErrMissingEncodedURL = errors.New("missing encoded URL")
)

// StreamHandler handles HTTP requests to proxy IPTV streams.
type StreamHandler struct {
	logger *logrus.Logger
}

// NewStreamHandler creates a new stream handler instance.
func NewStreamHandler(logger *logrus.Logger) *StreamHandler {
	return &StreamHandler{
		logger: logger,
	}
}

func (h *StreamHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	encodedURL, err := extractEncodedURL(r.URL.Path)
	if err != nil {
		h.logger.WithError(err).Error("Failed to extract URL from path")
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	targetURL, err := utils.DecodeURL(encodedURL)
	if err != nil {
		h.logger.WithError(err).Error("Failed to decode URL")
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}

	h.logger.WithField("url", targetURL).Debug("Proxying stream")

	if err := proxy.Stream(w, r, targetURL); err != nil {
		// Don't log context canceled errors - these are normal when clients disconnect
		if !errors.Is(err, context.Canceled) {
			h.logger.WithError(err).Error("Failed to proxy stream")
		}
		// If we haven't written headers yet, we can send an error response
		// This typically happens for validation errors before the stream starts
		if errors.Is(err, proxy.ErrUnsupportedScheme) ||
			errors.Is(err, proxy.ErrMissingHost) {
			http.Error(w, "Bad Gateway", http.StatusBadGateway)
			return
		}
		// For other errors, headers may have already been sent when streaming started
		// The client will handle the connection drop.
	}
}

func extractEncodedURL(path string) (string, error) {
	prefix := "/stream/"
	if !strings.HasPrefix(path, prefix) {
		return "", ErrInvalidPathFormat
	}

	encodedURL := strings.TrimPrefix(path, prefix)
	if encodedURL == "" {
		return "", ErrMissingEncodedURL
	}

	return encodedURL, nil
}
