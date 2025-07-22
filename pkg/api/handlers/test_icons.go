// Package handlers contains HTTP request handlers.
package handlers

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

// TestIconHandler serves icon images for test channels.
func TestIconHandler(w http.ResponseWriter, r *http.Request) {
	// Extract type and index from URL
	// Expected format: /test-icon/{type}/{index}
	// type: "channel" or "program"
	path := strings.TrimPrefix(r.URL.Path, "/test-icon/")
	parts := strings.Split(path, "/")

	if len(parts) != 2 {
		http.Error(w, "Invalid icon path", http.StatusBadRequest)
		return
	}

	iconType := parts[0]
	index, err := strconv.Atoi(parts[1])
	if err != nil {
		http.Error(w, "Invalid channel index", http.StatusBadRequest)
		return
	}

	// Generate appropriate icon based on type and index
	var iconData []byte
	var contentType string

	switch iconType {
	case "channel":
		iconData = generateChannelIcon(index)
		contentType = "image/svg+xml"
	case "program":
		iconData = generateProgramIcon(index)
		contentType = "image/svg+xml"
	default:
		http.Error(w, "Invalid icon type", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "public, max-age=86400") // Cache for 24 hours
	_, _ = w.Write(iconData)
}

// generateChannelIcon creates an SVG icon for a test channel (100x100).
func generateChannelIcon(index int) []byte {
	colors := []string{"#8B0000", "#B22222", "#006400", "#228B22", "#00008B", "#4169E1", "#4B0082", "#8B008B", "#FF8C00", "#DAA520"}
	resolutions := []string{"4K", "4K", "1080p", "1080p", "720p", "720p", "5.1", "7.1", "HQ", "LQ"}

	color := colors[index%len(colors)]
	resolution := resolutions[index%len(resolutions)]

	svg := fmt.Sprintf(`<svg width="100" height="100" xmlns="http://www.w3.org/2000/svg">
  <rect width="100" height="100" fill="%s" rx="10"/>
  <text x="50" y="40" font-family="Arial, sans-serif" font-size="16" font-weight="bold" fill="white" text-anchor="middle">TEST</text>
  <text x="50" y="65" font-family="Arial, sans-serif" font-size="14" fill="white" text-anchor="middle">%s</text>
</svg>`, color, resolution)

	return []byte(svg)
}

// generateProgramIcon creates an SVG icon for a test program (300x200).
func generateProgramIcon(index int) []byte {
	colors := []string{"#8B0000", "#B22222", "#006400", "#228B22", "#00008B", "#4169E1", "#4B0082", "#8B008B", "#FF8C00", "#DAA520"}
	names := []string{"4K HDR 60fps", "4K 30fps", "1080p 60fps", "1080p 30fps", "720p 60fps", "720p 30fps", "Audio 5.1", "Audio 7.1", "Audio HQ", "Audio LQ"}
	patterns := []string{"Bars", "Bars", "Grid", "Grid", "SMPTE", "SMPTE", "Sine", "Sine", "Sweep", "Noise"}

	color := colors[index%len(colors)]
	name := names[index%len(names)]
	pattern := patterns[index%len(patterns)]

	svg := fmt.Sprintf(`<svg width="300" height="200" xmlns="http://www.w3.org/2000/svg">
  <rect width="300" height="200" fill="%s" rx="10"/>
  <text x="150" y="80" font-family="Arial, sans-serif" font-size="24" font-weight="bold" fill="white" text-anchor="middle">Test Pattern</text>
  <text x="150" y="110" font-family="Arial, sans-serif" font-size="18" fill="white" text-anchor="middle">%s</text>
  <text x="150" y="140" font-family="Arial, sans-serif" font-size="14" fill="white" opacity="0.8" text-anchor="middle">%s</text>
</svg>`, color, name, pattern)

	return []byte(svg)
}

// TestIconDataURL generates a data URL for embedding in M3U/EPG.
func TestIconDataURL(iconType string, index int) string {
	var iconData []byte

	switch iconType {
	case "channel":
		iconData = generateChannelIcon(index)
	case "program":
		iconData = generateProgramIcon(index)
	default:
		return ""
	}

	encoded := base64.StdEncoding.EncodeToString(iconData)
	return fmt.Sprintf("data:image/svg+xml;base64,%s", encoded)
}
