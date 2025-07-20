// Package handlers provides tests for HTTP request handlers.
package handlers

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/savid/iptv-proxy/config"
	"github.com/savid/iptv-proxy/internal/data"
	"github.com/sirupsen/logrus"
)

func TestM3UHandlerNoData(t *testing.T) {
	// Create empty store
	store := data.NewStore()
	logger := logrus.New()
	logger.SetOutput(os.Stderr)

	// Create test config
	cfg := &config.Config{
		BaseURL: "http://localhost:8080",
	}

	handler := NewM3UHandler(store, cfg, logger)

	// Create test request
	req := httptest.NewRequest("GET", "/iptv.m3u", nil)
	w := httptest.NewRecorder()

	// Handle request
	handler.ServeHTTP(w, req)

	// Check response
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503, got %d", w.Code)
	}

	// Check error message
	body := w.Body.String()
	if body != "M3U data not available\n" {
		t.Errorf("Expected 'M3U data not available\\n', got %q", body)
	}
}

func TestEPGHandlerNoData(t *testing.T) {
	// Create empty store
	store := data.NewStore()
	logger := logrus.New()
	logger.SetOutput(os.Stderr)

	// Create test config
	cfg := &config.Config{
		BaseURL: "http://localhost:8080",
	}

	handler := NewEPGHandler(store, cfg, logger)

	// Create test request
	req := httptest.NewRequest("GET", "/epg.xml", nil)
	w := httptest.NewRecorder()

	// Handle request
	handler.ServeHTTP(w, req)

	// Check response
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503, got %d", w.Code)
	}

	// Check error message
	body := w.Body.String()
	if body != "EPG data not available\n" {
		t.Errorf("Expected 'EPG data not available\\n', got %q", body)
	}
}
