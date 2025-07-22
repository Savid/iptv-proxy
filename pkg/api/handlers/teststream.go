// Package handlers contains HTTP request handlers.
package handlers

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/savid/iptv-proxy/pkg/testchannels"
)

// TestChannelHandler handles requests for test channel streams.
func TestChannelHandler(w http.ResponseWriter, r *http.Request) {
	// Extract channel index from URL path
	// Expected format: /test/{index}
	var index int
	if _, err := fmt.Sscanf(r.URL.Path, "/test/%d", &index); err != nil {
		fmt.Printf("TestChannelHandler: Invalid path format: %s\n", r.URL.Path)
		http.Error(w, "Invalid test channel ID", http.StatusBadRequest)
		return
	}

	// Get the test profile
	profile, ok := testchannels.GetTestProfileByIndex(index)
	if !ok {
		fmt.Printf("TestChannelHandler: Channel index %d not found\n", index)
		http.Error(w, "Test channel not found", http.StatusNotFound)
		return
	}

	fmt.Printf("TestChannelHandler: Starting stream for channel %d (%s)\n", index, profile.Name)

	// Detect client type from User-Agent
	userAgent := r.Header.Get("User-Agent")
	isWebTV := strings.Contains(userAgent, "Plex Media Player") ||
		strings.Contains(userAgent, "PlexWeb") ||
		strings.Contains(userAgent, "Safari") || // Plex Web often uses Safari UA
		strings.Contains(userAgent, "Chrome") // Or Chrome UA

	// Create appropriate generator based on client type
	var stream io.ReadCloser
	var err error
	if isWebTV {
		// Use TV-compatible generator for Web/TV clients
		generator := testchannels.NewTVCompatibleGenerator()
		stream, err = generator.GenerateStream(profile)
		fmt.Printf("TestChannelHandler: Using TV-compatible generator for channel %d (User-Agent: %s)\n", index, userAgent)
	} else {
		// Use standard generator for Android/mobile clients
		generator := testchannels.NewTestPatternGenerator()
		stream, err = generator.GenerateStream(profile)
		fmt.Printf("TestChannelHandler: Using standard generator for channel %d (User-Agent: %s)\n", index, userAgent)
	}

	if err != nil {
		fmt.Printf("TestChannelHandler: Failed to generate stream for %s: %v\n", profile.Name, err)
		http.Error(w, fmt.Sprintf("Failed to generate test stream: %v", err), http.StatusInternalServerError)
		return
	}
	defer func() {
		if err := stream.Close(); err != nil {
			// Log error but don't fail the request
			fmt.Printf("Failed to close test stream: %v\n", err)
		}
	}()

	// Set headers that Plex expects
	// Use video/mp2t for MPEG-TS streams (Plex prefers this for live streams)
	w.Header().Set("Content-Type", "video/mp2t")
	// Let Go handle Transfer-Encoding automatically for proper chunking

	// Write headers
	w.WriteHeader(http.StatusOK)

	// Stream the content with proper error handling
	buf := make([]byte, 188*100) // Use MPEG-TS packet aligned buffer (188 bytes * 100)
	for {
		select {
		case <-r.Context().Done():
			// Client disconnected
			fmt.Printf("TestChannelHandler: Client disconnected for channel %d\n", index)
			return
		default:
			n, err := stream.Read(buf)
			if n > 0 {
				if _, writeErr := w.Write(buf[:n]); writeErr != nil {
					// Client disconnected
					fmt.Printf("TestChannelHandler: Write error for channel %d: %v\n", index, writeErr)
					return
				}
				// Flush after each write for live streaming
				if f, ok := w.(http.Flusher); ok {
					f.Flush()
				}
			}
			if err != nil {
				if err != io.EOF {
					fmt.Printf("TestChannelHandler: Read error for channel %d: %v\n", index, err)
				}
				// End of stream or error
				return
			}
		}
	}
}
