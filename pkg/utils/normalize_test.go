package utils

import (
	"testing"
)

func TestNormalizeChannelName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "country code removal",
			input:    "US: ESPN",
			expected: "espn",
		},
		{
			name:     "three letter country code",
			input:    "AUS: FOX CRICKET",
			expected: "foxcricket",
		},
		{
			name:     "spaces and case",
			input:    "FOX SPORTS 502",
			expected: "foxsports502",
		},
		{
			name:     "special characters",
			input:    "ESPN & Sports Plus",
			expected: "espnandsportsplus",
		},
		{
			name:     "dots and dashes",
			input:    "Racing.com-HD",
			expected: "racingcomhd",
		},
		{
			name:     "underscores",
			input:    "Channel_Name_123",
			expected: "channelname123",
		},
		{
			name:     "plus sign",
			input:    "Disney+",
			expected: "disneyplus",
		},
		{
			name:     "trim spaces",
			input:    "  Channel Name  ",
			expected: "channelname",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeChannelName(tt.input)
			if result != tt.expected {
				t.Errorf("NormalizeChannelName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestExtractChannelName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "with parentheses",
			input:    "Channel Name (HD)",
			expected: "Channel Name",
		},
		{
			name:     "with brackets",
			input:    "Channel Name [Region]",
			expected: "Channel Name",
		},
		{
			name:     "with both",
			input:    "Channel Name (HD) [Region]",
			expected: "Channel Name",
		},
		{
			name:     "no extras",
			input:    "Channel Name",
			expected: "Channel Name",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "spaces to trim",
			input:    "  Channel Name  (HD)  ",
			expected: "Channel Name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractChannelName(tt.input)
			if result != tt.expected {
				t.Errorf("ExtractChannelName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestEncodeDecodeURL(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{
			name: "basic URL",
			url:  "http://example.com/stream",
		},
		{
			name: "URL with query params",
			url:  "http://example.com/stream?token=abc123&user=test",
		},
		{
			name: "URL with special characters",
			url:  "http://example.com/stream?name=test user&value=10%",
		},
		{
			name: "URL with path segments",
			url:  "http://example.com/path/to/stream/file.m3u8",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded := EncodeURL(tt.url)
			decoded, err := DecodeURL(encoded)
			if err != nil {
				t.Errorf("DecodeURL failed: %v", err)
			}
			if decoded != tt.url {
				t.Errorf("Encode/Decode roundtrip failed: got %q, want %q", decoded, tt.url)
			}
		})
	}
}

func TestDecodeURLError(t *testing.T) {
	// Test invalid percent encoding
	_, err := DecodeURL("%ZZ")
	if err == nil {
		t.Error("Expected error for invalid percent encoding")
	}
}
