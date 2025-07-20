package testchannels

import (
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
)

// PlexTestHandler is optimized specifically for Plex compatibility.
type PlexTestHandler struct {
	generator *TestPatternGenerator
}

// NewPlexTestHandler creates a new Plex-optimized test handler.
func NewPlexTestHandler() *PlexTestHandler {
	return &PlexTestHandler{
		generator: NewTestPatternGenerator(),
	}
}

func (h *PlexTestHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Log request for debugging
	userAgent := r.Header.Get("User-Agent")
	fmt.Printf("[PlexTest] Request from %s: %s (UA: %s)\n", r.RemoteAddr, r.URL.Path, userAgent)

	// Extract channel index
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 3 {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

	index, err := strconv.Atoi(parts[2])
	if err != nil {
		http.Error(w, "Invalid channel ID", http.StatusBadRequest)
		return
	}

	// Get test profile
	profile, ok := GetTestProfileByIndex(index)
	if !ok {
		http.Error(w, "Channel not found", http.StatusNotFound)
		return
	}

	fmt.Printf("[PlexTest] Serving channel %d: %s\n", index, profile.Name)

	// Use a simpler FFmpeg command specifically for Plex
	args := h.buildPlexOptimizedArgs(profile)

	// #nosec G204 - ffmpeg path is hardcoded and args are built from validated profile data
	cmd := exec.Command("ffmpeg", args...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Printf("[PlexTest] Failed to create stdout pipe: %v\n", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		fmt.Printf("[PlexTest] Failed to create stderr pipe: %v\n", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	// Start FFmpeg
	if err := cmd.Start(); err != nil {
		fmt.Printf("[PlexTest] Failed to start FFmpeg: %v\n", err)
		http.Error(w, "Failed to start stream", http.StatusInternalServerError)
		return
	}

	// Log FFmpeg output
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := stderr.Read(buf)
			if n > 0 {
				fmt.Printf("[PlexTest FFmpeg] %s", string(buf[:n]))
			}
			if err != nil {
				break
			}
		}
	}()

	// Set headers - minimal set for Plex
	w.Header().Set("Content-Type", "video/mp2t")
	w.Header().Set("Cache-Control", "no-cache")

	// Important: Let Go handle chunked encoding automatically
	// Do NOT set Transfer-Encoding manually

	// Start streaming
	w.WriteHeader(http.StatusOK)

	// Create a done channel for cleanup
	done := make(chan struct{})
	defer close(done)

	// Clean up on exit
	defer func() {
		stdout.Close()
		if cmd.Process != nil {
			cmd.Process.Kill()
			cmd.Wait()
		}
	}()

	// Monitor client disconnect
	go func() {
		select {
		case <-r.Context().Done():
			fmt.Printf("[PlexTest] Client disconnected\n")
			if cmd.Process != nil {
				cmd.Process.Kill()
			}
		case <-done:
			return
		}
	}()

	// Stream with small buffer for lower latency
	buf := make([]byte, 1316) // 7 MPEG-TS packets
	for {
		n, err := stdout.Read(buf)
		if n > 0 {
			if _, writeErr := w.Write(buf[:n]); writeErr != nil {
				fmt.Printf("[PlexTest] Write error: %v\n", writeErr)
				return
			}

			// Flush immediately for live streaming
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}
		if err != nil {
			if err != io.EOF {
				fmt.Printf("[PlexTest] Read error: %v\n", err)
			}
			return
		}
	}
}

func (h *PlexTestHandler) buildPlexOptimizedArgs(profile TestChannelProfile) []string {
	// Use very simple, Plex-compatible settings
	return []string{
		"-re",
		"-f", "lavfi",
		"-i", fmt.Sprintf("testsrc2=size=%s:rate=30", profile.Resolution),
		"-f", "lavfi",
		"-i", "sine=frequency=440:sample_rate=48000",
		"-c:v", "libx264",
		"-preset", "ultrafast",
		"-profile:v", "baseline",
		"-level", "3.0",
		"-b:v", "2M",
		"-maxrate", "2M",
		"-bufsize", "4M",
		"-pix_fmt", "yuv420p",
		"-g", "60", // 2 second GOP
		"-keyint_min", "30",
		"-c:a", "aac",
		"-b:a", "128k",
		"-ac", "2",
		"-ar", "48000",
		"-f", "mpegts",
		"-mpegts_copyts", "0",
		"-mpegts_flags", "+resend_headers",
		"-muxdelay", "0",
		"-pes_payload_size", "2930",
		"pipe:1",
	}
}
