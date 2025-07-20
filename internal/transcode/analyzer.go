// Package transcode handles video and audio transcoding operations.
package transcode

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
)

// StreamCodecs contains codec information for a stream.
type StreamCodecs struct {
	VideoCodec    string
	AudioCodec    string
	VideoProfile  string
	VideoLevel    string
	AudioChannels int
}

// AnalyzeStream probes a stream to get its codec information.
func AnalyzeStream(url string) (StreamCodecs, error) {
	cmd := exec.Command("ffprobe",
		"-v", "quiet",
		"-print_format", "json",
		"-show_streams",
		"-analyzeduration", "1000000", // 1 second
		"-probesize", "1000000", // 1MB
		url,
	)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return StreamCodecs{}, fmt.Errorf("ffprobe failed: %w, stderr: %s", err, stderr.String())
	}

	// Parse JSON output
	var probeData struct {
		Streams []struct {
			CodecType string `json:"codec_type"`
			CodecName string `json:"codec_name"`
			Profile   string `json:"profile"`
			Level     int    `json:"level"`
			Channels  int    `json:"channels"`
		} `json:"streams"`
	}

	if err := json.Unmarshal(stdout.Bytes(), &probeData); err != nil {
		return StreamCodecs{}, fmt.Errorf("failed to parse ffprobe output: %w", err)
	}

	codecs := StreamCodecs{}

	// Find video and audio streams
	for _, stream := range probeData.Streams {
		switch stream.CodecType {
		case "video":
			codecs.VideoCodec = stream.CodecName
			codecs.VideoProfile = stream.Profile
			if stream.Level > 0 {
				codecs.VideoLevel = fmt.Sprintf("%.1f", float64(stream.Level)/10.0)
			}
		case "audio":
			codecs.AudioCodec = stream.CodecName
			codecs.AudioChannels = stream.Channels
		}
	}

	return codecs, nil
}

// GetOptimalCodecs returns the best video and audio codecs based on source.
func GetOptimalCodecs(codecs StreamCodecs, preferredVideoCodec, preferredAudioCodec string) (string, string) {
	// If preferred codecs are specified and not "auto", use them
	videoCodec := preferredVideoCodec
	audioCodec := preferredAudioCodec

	// Auto-detect video codec if needed
	if videoCodec == "auto" || videoCodec == "" {
		// Check if video needs transcoding
		if codecs.VideoCodec == "h264" || codecs.VideoCodec == "h265" {
			videoCodec = "copy"
		} else {
			// Default to h264 for compatibility
			videoCodec = "h264"
		}
	}

	// Auto-detect audio codec if needed
	if audioCodec == "auto" || audioCodec == "" {
		switch codecs.AudioCodec {
		case "aac", "mp3", "mp2":
			// These are fine, just copy
			audioCodec = "copy"
		case "ac3", "eac3", "dts", "truehd":
			// These need conversion for better compatibility
			audioCodec = "aac"
		default:
			// Unknown codec, transcode to aac for safety
			audioCodec = "aac"
		}
	}

	return videoCodec, audioCodec
}
