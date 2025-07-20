// Package testchannels provides test pattern generation for IPTV testing.
package testchannels

import (
	"fmt"
	"log"
	"net/http"
	"time"
)

// Server handles HTTP requests for test channel streams.
type Server struct {
	generator *TestPatternGenerator
	port      int
	logger    *log.Logger
}

// NewServer creates a new test channel server.
func NewServer(port int, logger *log.Logger) *Server {
	return &Server{
		generator: NewTestPatternGenerator(),
		port:      port,
		logger:    logger,
	}
}

// StartTestChannelServer starts the HTTP server for test channels.
func StartTestChannelServer(port int, logger *log.Logger) error {
	server := NewServer(port, logger)

	mux := http.NewServeMux()
	mux.HandleFunc("/test/", server.handleTestStream)

	addr := fmt.Sprintf(":%d", port)
	logger.Printf("Starting test channel server on %s", addr)

	// Create HTTP server with timeouts for security
	httpServer := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	return httpServer.ListenAndServe()
}

// handleTestStream serves a test channel stream.
func (s *Server) handleTestStream(w http.ResponseWriter, r *http.Request) {
	// Extract channel index from URL
	var index int
	if _, err := fmt.Sscanf(r.URL.Path, "/test/%d", &index); err != nil {
		http.Error(w, "Invalid channel ID", http.StatusBadRequest)
		return
	}

	// Get the test profile
	profile, ok := GetTestProfileByIndex(index)
	if !ok {
		http.Error(w, "Channel not found", http.StatusNotFound)
		return
	}

	s.logger.Printf("Serving test channel %d: %s", index, profile.Name)

	// Generate the stream
	stream, err := s.generator.GenerateStream(profile)
	if err != nil {
		s.logger.Printf("Failed to generate test stream: %v", err)
		http.Error(w, "Failed to generate stream", http.StatusInternalServerError)
		return
	}
	defer func() {
		if err := stream.Close(); err != nil {
			s.logger.Printf("Failed to close test stream: %v", err)
		}
	}()

	// Set appropriate headers
	w.Header().Set("Content-Type", "video/mp2t")
	w.Header().Set("Cache-Control", "no-cache")

	// Copy the stream to the response
	buf := make([]byte, 32*1024) // 32KB buffer
	for {
		n, err := stream.Read(buf)
		if n > 0 {
			if _, writeErr := w.Write(buf[:n]); writeErr != nil {
				s.logger.Printf("Client disconnected: %v", writeErr)
				return
			}
		}
		if err != nil {
			if err.Error() != "EOF" {
				s.logger.Printf("Stream read error: %v", err)
			}
			return
		}
	}
}
