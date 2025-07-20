// Package testchannels provides test pattern generation for IPTV testing.
package testchannels

import (
	"fmt"
	"io"
	"os/exec"
)

// TestPatternGenerator creates test video streams using FFmpeg.
type TestPatternGenerator struct {
	ffmpegPath string
}

// NewTestPatternGenerator creates a new test pattern generator.
func NewTestPatternGenerator() *TestPatternGenerator {
	return &TestPatternGenerator{
		ffmpegPath: "ffmpeg",
	}
}

// GenerateStream creates a test stream based on the provided profile.
func (g *TestPatternGenerator) GenerateStream(profile TestChannelProfile) (io.ReadCloser, error) {
	args := g.buildFFmpegArgs(profile)

	// #nosec G204 - ffmpeg path is hardcoded and args are built from validated profile data
	cmd := exec.Command(g.ffmpegPath, args...)

	// Capture stderr for debugging
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		_ = stdout.Close()
		return nil, fmt.Errorf("failed to start ffmpeg: %w", err)
	}

	// Log stderr in background for debugging
	go func() {
		buf := make([]byte, 4096)
		const eofError = "EOF"
		for {
			n, err := stderr.Read(buf)
			if n > 0 {
				// Log FFmpeg output more clearly
				lines := string(buf[:n])
				fmt.Printf("[FFmpeg %s] %s", profile.Name, lines)
			}
			if err != nil {
				if err.Error() != eofError {
					fmt.Printf("[FFmpeg %s] stderr read error: %v\n", profile.Name, err)
				}
				break
			}
		}
	}()

	// Return a wrapper that will clean up the process when closed
	return &streamCloser{
		ReadCloser: stdout,
		cmd:        cmd,
	}, nil
}

// buildFFmpegArgs constructs FFmpeg arguments for test pattern generation.
func (g *TestPatternGenerator) buildFFmpegArgs(profile TestChannelProfile) []string {
	// Match the Plex transcoding profile settings exactly
	args := []string{
		"-re", // Real-time encoding
		"-f", "lavfi",
		"-i", fmt.Sprintf("testsrc2=size=%s:rate=%d", profile.Resolution, profile.Framerate),
		"-f", "lavfi",
		"-i", fmt.Sprintf("aevalsrc=sin(1000*2*PI*t):c=stereo:s=%d", profile.AudioRate),
		// Video encoding - H.264 that works with all Plex clients
		"-c:v", "libx264",
		"-profile:v", "high", // High profile for quality
		"-level", "4.1", // Widely supported level
		"-preset", "veryfast", // Faster encoding for test patterns
		"-b:v", profile.Bitrate,
		"-maxrate", profile.Bitrate,
		"-bufsize", fmt.Sprintf("%dk", parseKbps(profile.Bitrate)*2),
		"-pix_fmt", "yuv420p",
		"-g", "30", // Smaller GOP for better seeking
		"-keyint_min", "15",
		"-sc_threshold", "0", // Disable scene detection for consistent GOPs
		// Audio encoding - MP3 for maximum compatibility (works on all Plex clients)
		"-c:a", "libmp3lame",
		"-b:a", profile.AudioBitrate,
		"-ar", "44100", // Standard MP3 sample rate
		"-ac", "2", // Force stereo for compatibility
		// MPEG-TS output with HLS-friendly settings
		"-f", "mpegts",
		"-mpegts_copyts", "0",
		"-mpegts_flags", "+resend_headers+pat_pmt_at_frames",
		"-muxrate", "10M",
		"-pcr_period", "20",
		"-max_delay", "700000",
		"-muxdelay", "0.1",
		"-avoid_negative_ts", "make_zero",
		"-fflags", "+genpts+nobuffer",
		"-flush_packets", "1",
		// Force consistent timestamps for HLS compatibility
		"-vsync", "cfr",
		"-async", "1",
		"-start_at_zero",
		"pipe:1",
	}

	return args
}

// streamCloser wraps a ReadCloser and ensures the FFmpeg process is terminated.
type streamCloser struct {
	io.ReadCloser
	cmd *exec.Cmd
}

// Close terminates the FFmpeg process and closes the pipe.
func (s *streamCloser) Close() error {
	// Close the pipe first
	err := s.ReadCloser.Close()

	// Terminate the process
	if s.cmd.Process != nil {
		_ = s.cmd.Process.Kill()
		_ = s.cmd.Wait()
	}

	return err
}

// parseKbps extracts numeric kbps value from bitrate string.
func parseKbps(bitrate string) int {
	var kbps int
	_, _ = fmt.Sscanf(bitrate, "%dk", &kbps)
	if kbps == 0 {
		kbps = 2000 // Default 2Mbps
	}
	return kbps
}
