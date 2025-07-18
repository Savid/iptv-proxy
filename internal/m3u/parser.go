// Package m3u provides parsing and rewriting functionality for M3U playlist files.
package m3u

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var (
	// ErrIncompleteChannel is returned when an #EXTINF line has no corresponding URL.
	ErrIncompleteChannel = errors.New("found #EXTINF without URL at end of file")
	// ErrOrphanedChannel is returned when a new #EXTINF is found before the previous one has a URL.
	ErrOrphanedChannel = errors.New("found #EXTINF without URL for previous channel")
)

// Channel represents a single channel entry in an M3U playlist.
type Channel struct {
	Name     string
	URL      string
	TVGName  string
	TVGLogo  string
	Group    string
	Original string
}

// Parse extracts channel information from M3U playlist data.
func Parse(data []byte) ([]Channel, error) {
	var channels []Channel
	reader := bytes.NewReader(data)
	scanner := bufio.NewScanner(reader)

	var currentChannel *Channel

	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)

		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "#EXTM3U") {
			continue
		}

		if strings.HasPrefix(line, "#EXTINF:") {
			if currentChannel != nil {
				return nil, ErrOrphanedChannel
			}

			currentChannel = &Channel{
				Original: line,
			}

			currentChannel.TVGName = extractAttribute(line, "tvg-name")
			currentChannel.TVGLogo = extractAttribute(line, "tvg-logo")
			currentChannel.Group = extractAttribute(line, "group-title")

			parts := strings.SplitN(line, ",", 2)
			if len(parts) == 2 {
				currentChannel.Name = strings.TrimSpace(parts[1])
			}
		} else if !strings.HasPrefix(line, "#") && currentChannel != nil {
			currentChannel.URL = line
			channels = append(channels, *currentChannel)
			currentChannel = nil
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error scanning M3U data: %w", err)
	}

	if currentChannel != nil {
		return nil, ErrIncompleteChannel
	}

	return channels, nil
}

func extractAttribute(line, attr string) string {
	pattern := fmt.Sprintf(`%s="([^"]*)"`, regexp.QuoteMeta(attr))
	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(line)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}
