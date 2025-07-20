// Package testchannels provides test pattern generation for IPTV testing.
package testchannels

import (
	"fmt"
	"io"
	"os/exec"
)

// TVCompatibleGenerator creates test video streams optimized for Web/TV Plex clients.
type TVCompatibleGenerator struct {
	ffmpegPath string
}

// NewTVCompatibleGenerator creates a new TV-compatible test pattern generator.
func NewTVCompatibleGenerator() *TVCompatibleGenerator {
	return &TVCompatibleGenerator{
		ffmpegPath: "ffmpeg",
	}
}

// GenerateStream creates a TV-compatible test stream.
func (g *TVCompatibleGenerator) GenerateStream(profile TestChannelProfile) (io.ReadCloser, error) {
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
		for {
			n, err := stderr.Read(buf)
			if n > 0 {
				lines := string(buf[:n])
				fmt.Printf("[FFmpeg TV %s] %s", profile.Name, lines)
			}
			if err != nil {
				if err.Error() != "EOF" {
					fmt.Printf("[FFmpeg TV %s] stderr read error: %v\n", profile.Name, err)
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

// buildFFmpegArgs constructs FFmpeg arguments for TV-compatible test pattern generation.
func (g *TVCompatibleGenerator) buildFFmpegArgs(profile TestChannelProfile) []string {
	// TV/Web Plex needs more compatible settings
	args := []string{
		"-re", // Real-time encoding
		"-f", "lavfi",
		"-i", fmt.Sprintf("testsrc2=size=%s:rate=%d", profile.Resolution, profile.Framerate),
		"-f", "lavfi",
		"-i", fmt.Sprintf("sine=frequency=1000:sample_rate=%d", profile.AudioRate),
		// Video encoding - TV/Web compatible
		"-c:v", "libx264",
		"-profile:v", "main", // Main profile is more compatible than High
		"-level", "4.0", // Level 4.0 is widely supported
		"-preset", "veryfast", // Balance between speed and quality
		"-bf", "2", // 2 B-frames is more compatible than 3
		"-g", "30", // Smaller GOP for better seeking
		"-keyint_min", "15", // More frequent keyframes
		"-refs", "3", // Fewer reference frames
		"-x264opts", "cabac=1:ref=3:bframes=2:b-adapt=1:no-mbtree:weightp=0",
		"-b:v", profile.Bitrate,
		"-maxrate", profile.Bitrate,
		"-bufsize", fmt.Sprintf("%dk", parseKbps(profile.Bitrate)*2), // 2x bitrate buffer
		"-pix_fmt", "yuv420p",
		// Audio encoding - MP3 for universal web browser support
		"-c:a", "libmp3lame",
		"-b:a", profile.AudioBitrate,
		"-ar", "44100", // Standard MP3 sample rate
		"-ac", "2", // Force stereo for web compatibility
		// MPEG-TS output with TV-friendly settings
		"-f", "mpegts",
		"-mpegts_copyts", "0",
		"-pat_period", "0.1",
		"-sdt_period", "1.0",
		"-pcr_period", "20",
		"-muxrate", "10M",
		"-pes_payload_size", "2930",
		"-fflags", "+genpts+igndts+nobuffer",
		"-flags", "+cgop+global_header",
		"-avoid_negative_ts", "make_zero",
		"-max_muxing_queue_size", "1024",
		"pipe:1",
	}

	return args
}
