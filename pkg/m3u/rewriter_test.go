package m3u

import (
	"strings"
	"testing"
)

func TestRewrite(t *testing.T) {
	channels := []Channel{
		{
			Name:     "US: ESPN",
			URL:      "https://somewhere.co/abc123/efg890/200163456",
			TVGName:  "US: ESPN",
			TVGLogo:  "http://logo.png",
			Group:    "US Sports",
			Original: `#EXTINF:-1 tvg-id="" tvg-name="US: ESPN" tvg-logo="http://logo.png" group-title="US Sports",US: ESPN`,
		},
		{
			Name:     "AU: FOX SPORTS 502",
			URL:      "https://somewhere.co/abc123/efg890/600002905",
			TVGName:  "AU: FOX SPORTS 502",
			TVGLogo:  "",
			Group:    "Australia",
			Original: `#EXTINF:-1 tvg-id="" tvg-name="AU: FOX SPORTS 502" tvg-logo="" group-title="Australia",AU: FOX SPORTS 502`,
		},
	}

	baseURL := "http://localhost:8080"
	result := Rewrite(channels, baseURL)
	resultStr := string(result)

	// Check M3U header
	if !strings.HasPrefix(resultStr, "#EXTM3U\n") {
		t.Error("Result should start with #EXTM3U header")
	}

	// Check first channel
	expectedURL1 := "http://localhost:8080/stream/https%3A%2F%2Fsomewhere.co%2Fabc123%2Fefg890%2F200163456"
	if !strings.Contains(resultStr, expectedURL1) {
		t.Errorf("Expected rewritten URL '%s' not found", expectedURL1)
	}

	// Check second channel
	expectedURL2 := "http://localhost:8080/stream/https%3A%2F%2Fsomewhere.co%2Fabc123%2Fefg890%2F600002905"
	if !strings.Contains(resultStr, expectedURL2) {
		t.Errorf("Expected rewritten URL '%s' not found", expectedURL2)
	}

	// Check original EXTINF lines are preserved
	if !strings.Contains(resultStr, channels[0].Original) {
		t.Error("Original EXTINF line should be preserved")
	}
	if !strings.Contains(resultStr, channels[1].Original) {
		t.Error("Original EXTINF line should be preserved")
	}
}

func TestRewriteURL(t *testing.T) {
	tests := []struct {
		name        string
		originalURL string
		baseURL     string
		expected    string
	}{
		{
			name:        "basic URL",
			originalURL: "http://example.com/stream",
			baseURL:     "http://localhost:8080",
			expected:    "http://localhost:8080/stream/http%3A%2F%2Fexample.com%2Fstream",
		},
		{
			name:        "URL with query params",
			originalURL: "http://example.com/stream?token=abc123&user=test",
			baseURL:     "http://localhost:8080",
			expected:    "http://localhost:8080/stream/http%3A%2F%2Fexample.com%2Fstream%3Ftoken%3Dabc123%26user%3Dtest",
		},
		{
			name:        "empty URL",
			originalURL: "",
			baseURL:     "http://localhost:8080",
			expected:    "",
		},
		{
			name:        "base URL with trailing slash",
			originalURL: "http://example.com/stream",
			baseURL:     "http://localhost:8080/",
			expected:    "http://localhost:8080//stream/http%3A%2F%2Fexample.com%2Fstream",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := rewriteURL(tt.originalURL, tt.baseURL)
			if result != tt.expected {
				t.Errorf("rewriteURL() = %v, want %v", result, tt.expected)
			}
		})
	}
}
