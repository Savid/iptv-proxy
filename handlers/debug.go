package handlers

import (
	"fmt"
	"net/http"
	"strings"
)

// DebugHandler logs all incoming request details for debugging
func DebugHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("\n=== DEBUG Request ===\n")
	fmt.Printf("Method: %s\n", r.Method)
	fmt.Printf("URL: %s\n", r.URL.String())
	fmt.Printf("Proto: %s\n", r.Proto)
	fmt.Printf("Host: %s\n", r.Host)
	fmt.Printf("RemoteAddr: %s\n", r.RemoteAddr)

	fmt.Printf("\nHeaders:\n")
	for name, values := range r.Header {
		for _, value := range values {
			fmt.Printf("  %s: %s\n", name, value)
		}
	}

	// Check if it's a Plex request
	userAgent := r.Header.Get("User-Agent")
	if strings.Contains(strings.ToLower(userAgent), "plex") {
		fmt.Printf("\n>>> This is a PLEX request! <<<\n")
	}

	fmt.Printf("=== END DEBUG ===\n\n")

	// Return a simple response
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Debug endpoint - check server logs for request details\n")
}
