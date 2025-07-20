// Package transcode handles video and audio transcoding operations.
package transcode

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os/exec"
	"sync"

	"github.com/savid/iptv-proxy/internal/hardware"
	"github.com/savid/iptv-proxy/internal/types"
)

var (
	// ErrTranscoderClosed is returned when operations are attempted on a closed transcoder.
	ErrTranscoderClosed = errors.New("transcoder already closed")
	// ErrStdinNotAvailable is returned when stdin pipe is not available.
	ErrStdinNotAvailable = errors.New("stdin not available")
	// ErrStdoutNotAvailable is returned when stdout pipe is not available.
	ErrStdoutNotAvailable = errors.New("stdout not available")
)

// CloseError wraps multiple errors that occurred during close.
type CloseError struct {
	Errors []error
}

func (e CloseError) Error() string {
	return fmt.Sprintf("close errors: %v", e.Errors)
}

// FFmpegTranscoder handles transcoding using FFmpeg.
type FFmpegTranscoder struct {
	profile      types.TranscodingProfile
	hardware     types.HardwareInfo
	bufferConfig types.BufferConfig
	selector     *hardware.Selector
	inputURL     string
	cmd          *exec.Cmd
	stdin        io.WriteCloser
	stdout       io.ReadCloser
	stderr       io.ReadCloser
	logger       *log.Logger
	mu           sync.Mutex
	closed       bool
}

// NewFFmpegTranscoder creates a new FFmpeg-based transcoder.
func NewFFmpegTranscoder(
	profile types.TranscodingProfile,
	hardware types.HardwareInfo,
	bufferConfig types.BufferConfig,
	selector *hardware.Selector,
	inputURL string,
	logger *log.Logger,
) *FFmpegTranscoder {
	return &FFmpegTranscoder{
		profile:      profile,
		hardware:     hardware,
		bufferConfig: bufferConfig,
		selector:     selector,
		inputURL:     inputURL,
		logger:       logger,
	}
}

// Start begins the transcoding process.
func (t *FFmpegTranscoder) Start(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed {
		return ErrTranscoderClosed
	}

	args := t.buildCommand()
	t.logger.Printf("Starting FFmpeg with args: %v", args)

	t.cmd = exec.CommandContext(ctx, "ffmpeg", args...) // #nosec G204 - args are internally constructed

	// Set up pipes
	var err error
	if t.inputURL == "-" {
		t.stdin, err = t.cmd.StdinPipe()
		if err != nil {
			return fmt.Errorf("failed to create stdin pipe: %w", err)
		}
	}

	t.stdout, err = t.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	t.stderr, err = t.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start FFmpeg
	if err := t.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start FFmpeg: %w", err)
	}

	// Log stderr in background
	go t.logStderr()

	return nil
}

// buildCommand constructs the FFmpeg command arguments.
func (t *FFmpegTranscoder) buildCommand() []string {
	args := []string{
		"-hide_banner",
		"-loglevel", "warning",
		"-stats",
	}

	// Add hardware-specific arguments at the beginning
	hardwareArgs := t.selector.GetFFmpegArgs(t.hardware, t.profile)

	// Extract codec and device arguments from hardware args
	var codecArgs []string
	var deviceArgs []string
	for i := 0; i < len(hardwareArgs); i++ {
		switch hardwareArgs[i] {
		case "-c:v", "-c:a":
			if i+1 < len(hardwareArgs) {
				codecArgs = append(codecArgs, hardwareArgs[i], hardwareArgs[i+1])
				i++
			}
		case "-vaapi_device", "-init_hw_device", "-filter_hw_device":
			if i+1 < len(hardwareArgs) {
				deviceArgs = append(deviceArgs, hardwareArgs[i], hardwareArgs[i+1])
				i++
			}
		default:
			codecArgs = append(codecArgs, hardwareArgs[i])
		}
	}

	// Add device args first
	args = append(args, deviceArgs...)

	// Input options
	args = append(args,
		"-fflags", "+genpts+discardcorrupt+nobuffer",
		"-err_detect", "ignore_err",
		"-i", t.inputURL,
	)

	// Add codec arguments if any from hardware
	if len(codecArgs) > 0 {
		args = append(args, codecArgs...)
	}

	// Add profile's extra arguments (includes codec settings)
	args = append(args, t.profile.ExtraArgs...)

	// Output format (if not already specified in ExtraArgs)
	hasFormat := false
	for _, arg := range t.profile.ExtraArgs {
		if arg == "-f" {
			hasFormat = true
			break
		}
	}
	if !hasFormat {
		args = append(args, "-f", t.profile.Container)
	}

	// Output to stdout
	args = append(args, "pipe:1")

	return args
}

// Write writes data to the transcoder input (for pipe input).
func (t *FFmpegTranscoder) Write(p []byte) (n int, err error) {
	if t.stdin == nil {
		return 0, ErrStdinNotAvailable
	}
	return t.stdin.Write(p)
}

// Read reads transcoded data from the output.
func (t *FFmpegTranscoder) Read(p []byte) (n int, err error) {
	if t.stdout == nil {
		return 0, ErrStdoutNotAvailable
	}
	return t.stdout.Read(p)
}

// Close stops the transcoding process and cleans up resources.
func (t *FFmpegTranscoder) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed {
		return nil
	}
	t.closed = true

	var errs []error

	// Close stdin to signal FFmpeg to finish
	if t.stdin != nil {
		if err := t.stdin.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close stdin: %w", err))
		}
	}

	// Wait for FFmpeg to finish
	if t.cmd != nil && t.cmd.Process != nil {
		if err := t.cmd.Wait(); err != nil {
			// Exit status 255 is often normal termination for streaming
			var exitErr *exec.ExitError
			if !errors.As(err, &exitErr) || exitErr.ExitCode() != 255 {
				errs = append(errs, fmt.Errorf("FFmpeg process error: %w", err))
			}
		}
	}

	// Close pipes
	if t.stdout != nil {
		if err := t.stdout.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close stdout: %w", err))
		}
	}
	if t.stderr != nil {
		if err := t.stderr.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close stderr: %w", err))
		}
	}

	if len(errs) > 0 {
		return CloseError{Errors: errs}
	}
	return nil
}

// logStderr logs FFmpeg stderr output.
func (t *FFmpegTranscoder) logStderr() {
	buf := make([]byte, 1024)
	for {
		n, err := t.stderr.Read(buf)
		if n > 0 {
			t.logger.Printf("FFmpeg: %s", string(buf[:n]))
		}
		if err != nil {
			break
		}
	}
}
