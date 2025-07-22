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

	"github.com/savid/iptv-proxy/pkg/hardware"
	"github.com/savid/iptv-proxy/pkg/types"
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

// commandSection represents different sections of FFmpeg command.
type commandSection struct {
	global  []string
	input   []string
	filters []string
	video   []string
	audio   []string
	output  []string
}

// buildCommand constructs the FFmpeg command arguments.
func (t *FFmpegTranscoder) buildCommand() []string {
	sections := &commandSection{
		global: []string{
			"-hide_banner",
			"-loglevel", "warning",
			"-stats",
		},
		input:   []string{},
		filters: []string{},
		video:   []string{},
		audio:   []string{},
		output:  []string{},
	}

	// Get hardware-specific arguments
	hardwareArgs := t.selector.GetFFmpegArgs(t.hardware, t.profile)

	// Parse and categorize hardware arguments
	t.categorizeHardwareArgs(hardwareArgs, sections)

	// Add input options
	sections.input = append(sections.input,
		"-fflags", "+genpts+discardcorrupt+nobuffer",
		"-err_detect", "ignore_err",
		"-i", t.inputURL,
	)

	// Add profile's extra arguments, categorizing them appropriately
	t.categorizeProfileArgs(t.profile.ExtraArgs, sections)

	// Build final command by assembling sections in order
	args := []string{}
	args = append(args, sections.global...)
	args = append(args, sections.input...)
	args = append(args, sections.filters...)
	args = append(args, sections.video...)
	args = append(args, sections.audio...)
	args = append(args, sections.output...)

	// Ensure output format is set
	if !t.hasArg(sections.output, "-f") {
		args = append(args, "-f", t.profile.Container)
	}

	// Output to stdout
	args = append(args, "pipe:1")

	return args
}

// categorizeHardwareArgs sorts hardware arguments into appropriate sections.
func (t *FFmpegTranscoder) categorizeHardwareArgs(args []string, sections *commandSection) {
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-gpu":
			// NVIDIA GPU selection - goes in video section with encoder
			if i+1 < len(args) {
				sections.video = append(sections.video, args[i], args[i+1])
				i++
			}
		case "-vaapi_device", "-init_hw_device", "-filter_hw_device":
			// VA-API device setup - goes in global section
			if i+1 < len(args) {
				sections.global = append(sections.global, args[i], args[i+1])
				i++
			}
		case "-c:v":
			// Video codec - goes in video section
			if i+1 < len(args) {
				sections.video = append(sections.video, args[i], args[i+1])
				i++
			}
		case "-c:a":
			// Audio codec - goes in audio section
			if i+1 < len(args) {
				sections.audio = append(sections.audio, args[i], args[i+1])
				i++
			}
		case "-b:v":
			// Video bitrate - goes in video section
			if i+1 < len(args) {
				sections.video = append(sections.video, args[i], args[i+1])
				i++
			}
		case "-b:a":
			// Audio bitrate - goes in audio section
			if i+1 < len(args) {
				sections.audio = append(sections.audio, args[i], args[i+1])
				i++
			}
		default:
			// Other hardware-specific options go to video section
			sections.video = append(sections.video, args[i])
		}
	}
}

// categorizeProfileArgs sorts profile arguments into appropriate sections.
func (t *FFmpegTranscoder) categorizeProfileArgs(args []string, sections *commandSection) {
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-f":
			// Format - goes in output section
			if i+1 < len(args) {
				sections.output = append(sections.output, args[i], args[i+1])
				i++
			}
		case "-c:v":
			// Skip if already set by hardware args
			if !t.hasArg(sections.video, "-c:v") && i+1 < len(args) {
				sections.video = append(sections.video, args[i], args[i+1])
				i++
			} else if i+1 < len(args) {
				i++ // Skip the value too
			}
		case "-c:a":
			// Skip if already set by hardware args
			if !t.hasArg(sections.audio, "-c:a") && i+1 < len(args) {
				sections.audio = append(sections.audio, args[i], args[i+1])
				i++
			} else if i+1 < len(args) {
				i++ // Skip the value too
			}
		case "-b:v":
			// Skip if already set by hardware args
			if !t.hasArg(sections.video, "-b:v") && i+1 < len(args) {
				sections.video = append(sections.video, args[i], args[i+1])
				i++
			} else if i+1 < len(args) {
				i++ // Skip the value too
			}
		case "-b:a":
			// Skip if already set by hardware args
			if !t.hasArg(sections.audio, "-b:a") && i+1 < len(args) {
				sections.audio = append(sections.audio, args[i], args[i+1])
				i++
			} else if i+1 < len(args) {
				i++ // Skip the value too
			}
		default:
			// Other args go to output section
			sections.output = append(sections.output, args[i])
		}
	}
}

// hasArg checks if an argument exists in a slice.
func (t *FFmpegTranscoder) hasArg(args []string, arg string) bool {
	for _, a := range args {
		if a == arg {
			return true
		}
	}
	return false
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
