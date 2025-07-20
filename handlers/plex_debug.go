// Package handlers contains HTTP request handlers.
package handlers

import (
	"fmt"
	"net/http"
	"strings"
)

// PlexDebugHandler provides debugging information for Plex client detection.
func PlexDebugHandler(w http.ResponseWriter, r *http.Request) {
	userAgent := r.Header.Get("User-Agent")

	// Detect client type
	clientType := "Unknown"
	switch {
	case strings.Contains(userAgent, "Plex Media Player"):
		clientType = "Plex Media Player (TV/Desktop)"
	case strings.Contains(userAgent, "PlexWeb"):
		clientType = "Plex Web"
	case strings.Contains(userAgent, "Android") && strings.Contains(userAgent, "Plex"):
		clientType = "Plex Android"
	case strings.Contains(userAgent, "iOS") && strings.Contains(userAgent, "Plex"):
		clientType = "Plex iOS"
	case strings.Contains(userAgent, "Safari") && r.Header.Get("X-Plex-Product") != "":
		clientType = "Plex Web (Safari)"
	case strings.Contains(userAgent, "Chrome") && r.Header.Get("X-Plex-Product") != "":
		clientType = "Plex Web (Chrome)"
	}

	// Determine recommended encoding
	recommendedProfile := "high"
	recommendedGenerator := "standard"
	if clientType == "Plex Media Player (TV/Desktop)" || strings.HasPrefix(clientType, "Plex Web") {
		recommendedProfile = "main"
		recommendedGenerator = "tv-compatible"
	}

	// Write debug info
	w.Header().Set("Content-Type", "text/plain")
	_, _ = fmt.Fprintf(w, "Plex Client Debug Information\n")
	_, _ = fmt.Fprintf(w, "=============================\n\n")
	_, _ = fmt.Fprintf(w, "User-Agent: %s\n", userAgent)
	_, _ = fmt.Fprintf(w, "Detected Client Type: %s\n", clientType)
	_, _ = fmt.Fprintf(w, "Recommended H.264 Profile: %s\n", recommendedProfile)
	_, _ = fmt.Fprintf(w, "Recommended Generator: %s\n\n", recommendedGenerator)

	_, _ = fmt.Fprintf(w, "All Headers:\n")
	_, _ = fmt.Fprintf(w, "------------\n")
	for name, values := range r.Header {
		for _, value := range values {
			_, _ = fmt.Fprintf(w, "%s: %s\n", name, value)
		}
	}

	_, _ = fmt.Fprintf(w, "\nPlex-Specific Headers:\n")
	_, _ = fmt.Fprintf(w, "---------------------\n")
	for name, values := range r.Header {
		if strings.HasPrefix(name, "X-Plex-") {
			for _, value := range values {
				_, _ = fmt.Fprintf(w, "%s: %s\n", name, value)
			}
		}
	}
}
