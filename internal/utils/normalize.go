// Package utils provides utility functions for IPTV proxy operations.
package utils

import (
	"net/url"
	"regexp"
	"strings"
)

var countryCodeRegex = regexp.MustCompile(`^[A-Z]{2,3}:\s*`)

// NormalizeChannelName standardizes channel names by removing common prefixes and normalizing case.
func NormalizeChannelName(name string) string {
	normalized := name
	normalized = countryCodeRegex.ReplaceAllString(normalized, "")
	normalized = strings.ToLower(normalized)
	normalized = strings.TrimSpace(normalized)

	replacements := []struct {
		old string
		new string
	}{
		{" ", ""},
		{"-", ""},
		{"_", ""},
		{".", ""},
		{"&", "and"},
		{"+", "plus"},
	}

	for _, r := range replacements {
		normalized = strings.ReplaceAll(normalized, r.old, r.new)
	}

	return normalized
}

// ExtractChannelName extracts the channel name from a tvg-name attribute, handling country prefixes.
func ExtractChannelName(tvgName string) string {
	if tvgName == "" {
		return ""
	}

	name := tvgName

	if idx := strings.Index(name, " ("); idx != -1 {
		name = name[:idx]
	}

	if idx := strings.Index(name, " ["); idx != -1 {
		name = name[:idx]
	}

	return strings.TrimSpace(name)
}

// EncodeURL encodes a URL for use in query parameters.
func EncodeURL(rawURL string) string {
	return url.QueryEscape(rawURL)
}

// DecodeURL decodes a URL from query parameter encoding.
func DecodeURL(encoded string) (string, error) {
	return url.QueryUnescape(encoded)
}
