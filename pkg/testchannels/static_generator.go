package testchannels

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"time"
)

// StaticTestGenerator generates a static test file and loops it.
type StaticTestGenerator struct {
	cache      map[string][]byte
	cacheMutex sync.RWMutex
}

// NewStaticTestGenerator creates a new static test generator.
func NewStaticTestGenerator() *StaticTestGenerator {
	return &StaticTestGenerator{
		cache: make(map[string][]byte),
	}
}

// GenerateStaticStream creates a looping test stream from a pre-generated file.
func (g *StaticTestGenerator) GenerateStaticStream(profile TestChannelProfile) (io.ReadCloser, error) {
	// Check cache first
	g.cacheMutex.RLock()
	data, exists := g.cache[profile.Name]
	g.cacheMutex.RUnlock()

	if !exists {
		// Generate a 10-second clip
		var err error
		data, err = g.generateClip(profile)
		if err != nil {
			return nil, fmt.Errorf("failed to generate clip: %w", err)
		}

		// Cache it
		g.cacheMutex.Lock()
		g.cache[profile.Name] = data
		g.cacheMutex.Unlock()
	}

	// Return a looping reader
	return &loopingReader{
		data:   data,
		reader: bytes.NewReader(data),
	}, nil
}

func (g *StaticTestGenerator) generateClip(profile TestChannelProfile) ([]byte, error) {
	// Generate a 10-second test clip
	args := []string{
		"-f", "lavfi",
		"-i", fmt.Sprintf("%s=duration=10:size=%s:rate=%d",
			profile.TestPattern, profile.Resolution, profile.Framerate),
		"-f", "lavfi",
		"-i", fmt.Sprintf("sine=frequency=1000:sample_rate=%d:duration=10",
			profile.AudioRate),
		"-c:v", "libx264",
		"-preset", "ultrafast",
		"-profile:v", "baseline",
		"-level", "3.0",
		"-b:v", profile.Bitrate,
		"-pix_fmt", "yuv420p",
		"-g", fmt.Sprintf("%d", profile.Framerate),
		"-c:a", "aac",
		"-b:a", profile.AudioBitrate,
		"-ac", fmt.Sprintf("%d", profile.AudioChannels),
		"-ar", fmt.Sprintf("%d", profile.AudioRate),
		"-f", "mpegts",
		"-mpegts_copyts", "0",
		"-muxdelay", "0",
		"pipe:1",
	}

	// #nosec G204 - ffmpeg path is hardcoded and args are built from validated profile data
	cmd := exec.Command("ffmpeg", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		fmt.Printf("FFmpeg error: %s\n", stderr.String())
		return nil, fmt.Errorf("ffmpeg failed: %w", err)
	}

	return stdout.Bytes(), nil
}

// loopingReader reads data in a loop.
type loopingReader struct {
	data   []byte
	reader *bytes.Reader
	closed bool
	mu     sync.Mutex
}

func (r *loopingReader) Read(p []byte) (n int, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed {
		return 0, io.EOF
	}

	n, err = r.reader.Read(p)
	if err == io.EOF {
		// Reset to beginning
		if _, seekErr := r.reader.Seek(0, 0); seekErr != nil {
			return 0, seekErr
		}
		// Add a small delay to simulate real-time streaming
		time.Sleep(10 * time.Millisecond)
		return r.Read(p)
	}
	return n, err
}

func (r *loopingReader) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.closed = true
	return nil
}
