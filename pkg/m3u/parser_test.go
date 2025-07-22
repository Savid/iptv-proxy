package m3u

import (
	"os"
	"testing"
)

func TestParse(t *testing.T) {
	data, err := os.ReadFile("testdata/example.m3u")
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}

	channels, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Test total channel count
	expectedCount := 24
	if len(channels) != expectedCount {
		t.Errorf("Expected %d channels, got %d", expectedCount, len(channels))
	}

	// Test first channel
	if len(channels) > 0 {
		first := channels[0]
		if first.Name != "US: ESPN" {
			t.Errorf("Expected first channel name 'US: ESPN', got '%s'", first.Name)
		}
		if first.TVGName != "US: ESPN" {
			t.Errorf("Expected first channel tvg-name 'US: ESPN', got '%s'", first.TVGName)
		}
		if first.Group != "US Sports" {
			t.Errorf("Expected first channel group 'US Sports', got '%s'", first.Group)
		}
		if first.URL != "https://somewhere.co/abc123/efg890/200163456" {
			t.Errorf("Expected first channel URL 'https://somewhere.co/abc123/efg890/200163456', got '%s'", first.URL)
		}
	}

	// Test channel with different prefix (AU: vs AUS:)
	auChannels := 0
	for _, ch := range channels {
		if ch.Group == "Australia" {
			auChannels++
		}
	}
	if auChannels != 21 {
		t.Errorf("Expected 21 Australian channels, got %d", auChannels)
	}

	// Test extractAttribute function
	testLine := `#EXTINF:-1 tvg-id="test123" tvg-name="Test Channel" tvg-logo="http://logo.png" group-title="Test Group",Test Channel Name`

	if id := extractAttribute(testLine, "tvg-id"); id != "test123" {
		t.Errorf("Expected tvg-id 'test123', got '%s'", id)
	}

	if name := extractAttribute(testLine, "tvg-name"); name != "Test Channel" {
		t.Errorf("Expected tvg-name 'Test Channel', got '%s'", name)
	}

	if logo := extractAttribute(testLine, "tvg-logo"); logo != "http://logo.png" {
		t.Errorf("Expected tvg-logo 'http://logo.png', got '%s'", logo)
	}

	if group := extractAttribute(testLine, "group-title"); group != "Test Group" {
		t.Errorf("Expected group-title 'Test Group', got '%s'", group)
	}
}

func TestParseErrors(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		errMsg  string
	}{
		{
			name: "missing URL",
			input: `#EXTM3U
#EXTINF:-1 tvg-name="Test Channel",Test Channel
#EXTINF:-1 tvg-name="Test Channel 2",Test Channel 2`,
			wantErr: true,
			errMsg:  "found #EXTINF without URL for previous channel",
		},
		{
			name: "trailing EXTINF",
			input: `#EXTM3U
#EXTINF:-1 tvg-name="Test Channel",Test Channel
http://test.com/stream
#EXTINF:-1 tvg-name="Test Channel 2",Test Channel 2`,
			wantErr: true,
			errMsg:  "found #EXTINF without URL at end of file",
		},
		{
			name: "valid playlist",
			input: `#EXTM3U
#EXTINF:-1 tvg-name="Test Channel",Test Channel
http://test.com/stream`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse([]byte(tt.input))
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errMsg != "" && err.Error() != tt.errMsg {
				t.Errorf("Parse() error message = %v, want %v", err.Error(), tt.errMsg)
			}
		})
	}
}
