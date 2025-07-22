package m3u

import (
	"bytes"
	"fmt"
	"net/url"
	"strings"

	"github.com/savid/iptv-proxy/pkg/testchannels"
	"github.com/savid/iptv-proxy/pkg/utils"
)

// Rewrite takes a list of channels and rewrites their URLs to proxy through the given base URL.
func Rewrite(channels []Channel, baseURL string) []byte {
	var buf bytes.Buffer

	buf.WriteString("#EXTM3U\n")

	baseURL = strings.TrimRight(baseURL, "/")

	for _, channel := range channels {
		buf.WriteString(channel.Original)
		buf.WriteString("\n")

		rewrittenURL := rewriteURL(channel.URL, baseURL)
		buf.WriteString(rewrittenURL)
		buf.WriteString("\n")
	}

	return buf.Bytes()
}

func rewriteURL(originalURL, baseURL string) string {
	if originalURL == "" {
		return ""
	}

	_, err := url.Parse(originalURL)
	if err != nil {
		return originalURL
	}

	encodedURL := utils.EncodeURL(originalURL)
	return fmt.Sprintf("%s/stream/%s", baseURL, encodedURL)
}

// AppendTestChannels adds test channels to the M3U content.
func AppendTestChannels(m3uContent string, baseURL string) string {
	var buf bytes.Buffer

	// Write the original content without the final newline
	content := strings.TrimRight(m3uContent, "\n")
	buf.WriteString(content)
	buf.WriteString("\n")

	// Add test channels
	baseURL = strings.TrimRight(baseURL, "/")

	for i, profile := range testchannels.TestProfiles {
		// Create #EXTINF line with tvg-id to link with EPG
		iconURL := fmt.Sprintf("%s/test-icon/channel/%d", baseURL, i)
		extinf := fmt.Sprintf("#EXTINF:-1 tvg-id=\"test-%d\" tvg-name=\"Test: %s\" tvg-logo=\"%s\" group-title=\"Test Channels\",Test: %s",
			i, profile.Name, iconURL, profile.Name)
		buf.WriteString(extinf)
		buf.WriteString("\n")

		// Create URL for test channel
		testURL := fmt.Sprintf("%s/test/%d", baseURL, i)
		buf.WriteString(testURL)
		buf.WriteString("\n")
	}

	return buf.String()
}
