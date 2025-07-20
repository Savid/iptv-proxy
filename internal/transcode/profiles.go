// Package transcode handles video and audio transcoding operations.
package transcode

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/savid/iptv-proxy/config"
	"github.com/savid/iptv-proxy/internal/types"
)

// StreamInfo contains information about a media stream.
type StreamInfo struct {
	VideoBitrate int // in kbps
	AudioBitrate int // in kbps
	Width        int
	Height       int
	Framerate    float64
}

// CreateProfile creates a transcoding profile from codec settings.
func CreateProfile(videoCodec, audioCodec, videoBitrate, audioBitrate string) types.TranscodingProfile {
	profile := types.TranscodingProfile{
		Name:         "custom",
		VideoCodec:   videoCodec,
		AudioCodec:   audioCodec,
		VideoBitrate: videoBitrate,
		AudioBitrate: audioBitrate,
		Container:    "mpegts",
		ExtraArgs:    []string{},
	}

	// Add common args
	extraArgs := []string{
		"-err_detect", "ignore_err",
		"-fflags", "+genpts+discardcorrupt+nobuffer",
		"-analyzeduration", "10M",
		"-probesize", "10M",
		"-max_delay", "5000000",
		"-reconnect", "1",
		"-reconnect_at_eof", "1",
		"-reconnect_streamed", "1",
		"-reconnect_delay_max", "5",
		"-f", "mpegts",
		"-mpegts_copyts", "1",
		"-avoid_negative_ts", "disabled",
		"-max_muxing_queue_size", "1024",
	}

	// Add video codec specific args
	switch videoCodec {
	case "mpeg2":
		extraArgs = append(extraArgs,
			"-c:v", "mpeg2video",
			"-b:v", videoBitrate,
			"-maxrate", "8000k",
			"-bufsize", "4000k",
			"-g", "15", // GOP size for MPEG-2
			"-bf", "2", // B-frames
			"-pix_fmt", "yuv420p",
		)
	case codecH264:
		extraArgs = append(extraArgs,
			"-c:v", "libx264",
			"-b:v", videoBitrate,
			"-preset", "medium",
			"-profile:v", "high",
			"-level", "4.1",
			"-pix_fmt", "yuv420p",
		)
	case codecH265:
		extraArgs = append(extraArgs,
			"-c:v", "libx265",
			"-b:v", videoBitrate,
			"-preset", "medium",
			"-pix_fmt", "yuv420p",
		)
	case codecCopy:
		extraArgs = append(extraArgs, "-c:v", "copy")
	}

	// Add audio codec specific args
	switch audioCodec {
	case "mp2":
		extraArgs = append(extraArgs,
			"-c:a", "mp2",
			"-b:a", audioBitrate,
			"-ar", "48000",
			"-ac", "2",
		)
	case codecMP3:
		extraArgs = append(extraArgs,
			"-c:a", "libmp3lame",
			"-b:a", audioBitrate,
			"-ar", "44100",
			"-ac", "2",
		)
	case codecAAC:
		extraArgs = append(extraArgs,
			"-c:a", "aac",
			"-b:a", audioBitrate,
			"-ar", "48000",
			"-ac", "2",
		)
	case codecCopy:
		extraArgs = append(extraArgs, "-c:a", "copy")
	}

	profile.ExtraArgs = extraArgs
	return profile
}

// getProfileVideoBitrate returns the video bitrate based on quality settings.
func getProfileVideoBitrate(cfg *config.Config, mapper *QualityMapper) string {
	if cfg.VideoQuality == "custom" {
		return cfg.CustomVideoBitrate
	}
	return mapper.GetVideoBitrate(cfg.VideoQuality, cfg.VideoCodec)
}

// getProfileAudioBitrate returns the audio bitrate based on quality settings.
func getProfileAudioBitrate(cfg *config.Config, mapper *QualityMapper) string {
	if cfg.AudioQuality == "custom" {
		return cfg.CustomAudioBitrate
	}
	return mapper.GetAudioBitrate(cfg.AudioQuality, cfg.AudioCodec)
}

// NewTranscodingProfile creates a transcoding profile from the new config structure.
func NewTranscodingProfile(cfg *config.Config, mapper *QualityMapper) *types.TranscodingProfile {
	profile := &types.TranscodingProfile{
		Name:      "stream",
		Container: "mpegts",
		ExtraArgs: []string{},
	}

	// Handle transcode mode
	if cfg.TranscodeMode == "copy" {
		// Copy mode - just copy streams
		profile.VideoCodec = codecCopy
		profile.AudioCodec = codecCopy
		profile.VideoBitrate = ""
		profile.AudioBitrate = ""
	} else {
		// Transcode mode - use specified codecs and quality settings
		profile.VideoCodec = cfg.VideoCodec
		profile.AudioCodec = cfg.AudioCodec
		profile.VideoBitrate = getProfileVideoBitrate(cfg, mapper)
		profile.AudioBitrate = getProfileAudioBitrate(cfg, mapper)
	}

	// Common FFmpeg arguments for streaming
	// NOTE: We don't add codec arguments here because the hardware selector
	// will add the appropriate codec arguments based on hardware capabilities
	extraArgs := []string{
		"-err_detect", "ignore_err",
		"-fflags", "+genpts+discardcorrupt+nobuffer",
		"-analyzeduration", "10M",
		"-probesize", "10M",
		"-max_delay", "5000000",
		"-reconnect", "1",
		"-reconnect_at_eof", "1",
		"-reconnect_streamed", "1",
		"-reconnect_delay_max", "5",
		"-f", "mpegts",
		"-mpegts_copyts", "1",
		"-avoid_negative_ts", "disabled",
		"-max_muxing_queue_size", "1024",
	}

	profile.ExtraArgs = extraArgs
	return profile
}

// ApplyHardware modifies a profile to use hardware acceleration.
func ApplyHardware(profile types.TranscodingProfile, hw types.HardwareInfo) types.TranscodingProfile {
	// Don't modify copy profiles
	if profile.VideoCodec == "copy" {
		return profile
	}

	// Hardware codec selection is handled by the Selector.GetFFmpegArgs
	profile.HardwareAccel = hw.Type
	return profile
}

// CalculateAdaptiveBitrate determines optimal bitrates based on source stream info.
func CalculateAdaptiveBitrate(source StreamInfo) (videoBitrate, audioBitrate string) {
	// Video bitrate calculation based on resolution and framerate
	baseRate := 0
	pixels := source.Width * source.Height

	// Resolution-based base rates
	switch {
	case pixels >= 3840*2160: // 4K
		baseRate = 15000
	case pixels >= 2560*1440: // 1440p
		baseRate = 10000
	case pixels >= 1920*1080: // 1080p
		baseRate = 5000
	case pixels >= 1280*720: // 720p
		baseRate = 2500
	case pixels >= 854*480: // 480p
		baseRate = 1500
	default: // 360p and below
		baseRate = 800
	}

	// Adjust for framerate
	if source.Framerate > 30 {
		baseRate = int(float64(baseRate) * (source.Framerate / 30.0))
	}

	// If source bitrate is known and lower than calculated, use source
	if source.VideoBitrate > 0 && source.VideoBitrate < baseRate {
		baseRate = source.VideoBitrate
	}

	// Audio bitrate based on source or defaults
	audioRate := 128 // Default 128kbps
	if source.AudioBitrate > 0 {
		audioRate = source.AudioBitrate
		// Cap at reasonable values
		if audioRate > 320 {
			audioRate = 320
		}
	}

	return fmt.Sprintf("%dk", baseRate), fmt.Sprintf("%dk", audioRate)
}

// ProbeStream analyzes a stream to get its properties.
func ProbeStream(url string) (StreamInfo, error) {
	cmd := exec.Command("ffprobe",
		"-v", "quiet",
		"-print_format", "json",
		"-show_streams",
		"-show_format",
		url,
	)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return StreamInfo{}, fmt.Errorf("ffprobe failed: %w, stderr: %s", err, stderr.String())
	}

	// Parse JSON output
	var probeData struct {
		Streams []struct {
			CodecType    string `json:"codec_type"`
			Width        int    `json:"width"`
			Height       int    `json:"height"`
			BitRate      string `json:"bit_rate"`
			AvgFrameRate string `json:"avg_frame_rate"`
		} `json:"streams"`
		Format struct {
			BitRate string `json:"bit_rate"`
		} `json:"format"`
	}

	if err := json.Unmarshal(stdout.Bytes(), &probeData); err != nil {
		return StreamInfo{}, fmt.Errorf("failed to parse ffprobe output: %w", err)
	}

	info := StreamInfo{}

	// Find video and audio streams
	for _, stream := range probeData.Streams {
		switch stream.CodecType {
		case "video":
			info.Width = stream.Width
			info.Height = stream.Height

			// Parse bitrate
			if stream.BitRate != "" {
				if br, err := strconv.Atoi(stream.BitRate); err == nil {
					info.VideoBitrate = br / 1000 // Convert to kbps
				}
			}

			// Parse framerate
			if stream.AvgFrameRate != "" {
				parts := strings.Split(stream.AvgFrameRate, "/")
				if len(parts) == 2 {
					num, _ := strconv.ParseFloat(parts[0], 64)
					den, _ := strconv.ParseFloat(parts[1], 64)
					if den > 0 {
						info.Framerate = num / den
					}
				}
			}

		case "audio":
			// Parse audio bitrate
			if stream.BitRate != "" {
				if br, err := strconv.Atoi(stream.BitRate); err == nil {
					info.AudioBitrate = br / 1000 // Convert to kbps
				}
			}
		}
	}

	// If individual stream bitrates not found, estimate from total
	if info.VideoBitrate == 0 && probeData.Format.BitRate != "" {
		if totalBitrate, err := strconv.Atoi(probeData.Format.BitRate); err == nil {
			// Estimate 90% for video, 10% for audio
			info.VideoBitrate = (totalBitrate * 9 / 10) / 1000
			if info.AudioBitrate == 0 {
				info.AudioBitrate = (totalBitrate * 1 / 10) / 1000
			}
		}
	}

	// Set defaults if still missing
	if info.Framerate == 0 {
		info.Framerate = 30
	}
	if info.AudioBitrate == 0 {
		info.AudioBitrate = 128
	}

	return info, nil
}
