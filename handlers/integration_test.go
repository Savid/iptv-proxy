package handlers

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/savid/iptv-proxy/config"
	"github.com/savid/iptv-proxy/internal/data"
	"github.com/savid/iptv-proxy/internal/epg"
	"github.com/savid/iptv-proxy/internal/m3u"
	"github.com/sirupsen/logrus"
)

// setupTestEnvironment creates a test environment with mock servers and initialized store.
func setupTestEnvironment(t *testing.T) (*data.Store, func()) {
	// Create a mock M3U server
	m3uServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		data, _ := os.ReadFile("testdata/example.m3u")
		_, _ = w.Write(data)
	}))

	// Create a mock EPG server
	epgServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		data, _ := os.ReadFile("testdata/small_epg.xml")
		_, _ = w.Write(data)
	}))

	// Create config
	cfg := &config.Config{
		M3UURL:          m3uServer.URL,
		EPGURL:          epgServer.URL,
		BaseURL:         "http://localhost:8080",
		RefreshInterval: 30 * time.Minute,
	}

	// Create logger
	logger := logrus.New()
	logger.SetOutput(os.Stderr)

	// Create store and fetcher
	store := data.NewStore()
	fetcher := data.NewFetcher(cfg, logger)

	// Perform initial fetch
	result, err := fetcher.FetchAll()
	if err != nil {
		t.Fatalf("Failed to fetch initial data: %v", err)
	}
	store.SetM3U(result.M3U.Raw, result.M3U.Channels)
	store.SetEPG(result.EPG.Raw, result.EPG.Filtered)

	cleanup := func() {
		m3uServer.Close()
		epgServer.Close()
	}

	return store, cleanup
}

func TestIntegrationWithExampleFiles(t *testing.T) {
	// Read test files
	m3uData, err := os.ReadFile("testdata/example.m3u")
	if err != nil {
		t.Fatalf("Failed to read M3U test file: %v", err)
	}

	epgData, err := os.ReadFile("testdata/small_epg.xml")
	if err != nil {
		t.Fatalf("Failed to read EPG test file: %v", err)
	}

	// Parse M3U to find matching channels
	channels, err := m3u.Parse(m3uData)
	if err != nil {
		t.Fatalf("Failed to parse M3U: %v", err)
	}

	// Parse EPG
	epgReader := strings.NewReader(string(epgData))
	tv, err := epg.ParseStream(epgReader)
	if err != nil {
		t.Fatalf("Failed to parse EPG: %v", err)
	}

	// Test filter with real data
	filtered, channelMap := epg.Filter(tv, channels)

	// With direct matching, "AU: FOX SPORTS 502" won't match "FOX SPORTS 502"
	// So we expect 0 matches
	expectedMatches := 0
	if len(filtered.Channels) != expectedMatches {
		t.Errorf("Expected %d matched channels, got %d", expectedMatches, len(filtered.Channels))

		// Debug: show what channels we have
		t.Log("M3U channels with tvg-name:")
		for _, ch := range channels {
			if strings.Contains(ch.TVGName, "FOX SPORTS 502") {
				t.Logf("  - tvg-name: %q, name: %q", ch.TVGName, ch.Name)
			}
		}

		t.Log("EPG channels:")
		for _, ch := range tv.Channels {
			t.Logf("  - id: %q, display-name: %q", ch.ID, ch.DisplayName)
		}
	}

	// Test that programs are filtered correctly (should be 0 since no channels match)
	if len(filtered.Programs) != 0 {
		t.Errorf("Expected 0 filtered programs, got %d", len(filtered.Programs))
	}

	// Verify channel mapping
	if len(channelMap) != expectedMatches {
		t.Errorf("Expected %d channel mappings, got %d", expectedMatches, len(channelMap))
	}
}

func TestM3UHandlerWithMockServer(t *testing.T) {
	store, cleanup := setupTestEnvironment(t)
	defer cleanup()

	// Create handler
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
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Check content type
	contentType := w.Header().Get("Content-Type")
	if contentType != "application/vnd.apple.mpegurl" {
		t.Errorf("Expected content type 'application/vnd.apple.mpegurl', got %q", contentType)
	}

	// Check body contains rewritten URLs
	body := w.Body.String()
	if !strings.Contains(body, "http://localhost:8080/stream/") {
		t.Error("Response should contain rewritten URLs")
	}
}

func TestStreamHandler(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(os.Stderr)
	handler := NewStreamHandler(logger)

	tests := []struct {
		name       string
		path       string
		wantStatus int
	}{
		{
			name:       "missing URL",
			path:       "/stream/",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid path",
			path:       "/notstream/test",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid encoded URL",
			path:       "/stream/invalid%2Furl",
			wantStatus: http.StatusBadGateway,
		},
		{
			name:       "localhost URL (connection refused)",
			path:       "/stream/" + url.QueryEscape("http://localhost/stream"),
			wantStatus: http.StatusOK, // Headers written before connection error
		},
		{
			name:       "internal IP (TLS certificate error)",
			path:       "/stream/" + url.QueryEscape("https://192.168.1.1/stream"),
			wantStatus: http.StatusOK, // Headers written before TLS error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("Expected status %d, got %d", tt.wantStatus, w.Code)
			}
		})
	}
}

func TestEPGHandlerWithMockServer(t *testing.T) {
	store, cleanup := setupTestEnvironment(t)
	defer cleanup()

	// Create handler
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
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Check content type
	contentType := w.Header().Get("Content-Type")
	if contentType != "application/xml; charset=utf-8" {
		t.Errorf("Expected content type 'application/xml; charset=utf-8', got %q", contentType)
	}

	// Check body is valid XML
	body := w.Body.String()
	if !strings.HasPrefix(body, "<?xml") {
		t.Error("Response should start with XML declaration")
	}
}
