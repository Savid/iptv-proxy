package m3u

import (
	"bytes"
	"fmt"
	"net/url"
	"strings"

	"github.com/savid/iptv-proxy/internal/utils"
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
