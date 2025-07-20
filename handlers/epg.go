package handlers

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/savid/iptv-proxy/config"
	"github.com/savid/iptv-proxy/internal/data"
	"github.com/savid/iptv-proxy/internal/testchannels"
	"github.com/sirupsen/logrus"
)

// EPGHandler handles HTTP requests for EPG (Electronic Program Guide) data.
type EPGHandler struct {
	store  *data.Store
	config *config.Config
	logger *logrus.Logger
}

// NewEPGHandler creates a new EPG handler instance.
func NewEPGHandler(store *data.Store, cfg *config.Config, logger *logrus.Logger) *EPGHandler {
	return &EPGHandler{
		store:  store,
		config: cfg,
		logger: logger,
	}
}

func (h *EPGHandler) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	data, ok := h.store.GetEPG()
	if !ok {
		h.logger.Error("EPG data not available")
		http.Error(w, "EPG data not available", http.StatusServiceUnavailable)
		return
	}

	// If test channels are enabled, append their EPG data
	if h.config.EnableTestChannels {
		modifiedData := h.appendTestChannelEPG(data)
		w.Header().Set("Content-Type", "application/xml; charset=utf-8")
		_, _ = w.Write(modifiedData)
	} else {
		w.Header().Set("Content-Type", "application/xml; charset=utf-8")
		_, _ = w.Write(data)
	}
}

// appendTestChannelEPG adds EPG data for test channels
func (h *EPGHandler) appendTestChannelEPG(originalEPG []byte) []byte {
	// Parse the original EPG to find the closing </tv> tag
	epgStr := string(originalEPG)
	closingTag := "</tv>"

	if !strings.Contains(epgStr, closingTag) {
		// If no closing tag, return original
		return originalEPG
	}

	// Generate EPG entries for test channels
	var testEPG bytes.Buffer
	now := time.Now().UTC()

	for i, profile := range testchannels.TestProfiles {
		channelID := fmt.Sprintf("test-%d", i)

		// Add channel definition
		channelIconURL := fmt.Sprintf("%s/test-icon/channel/%d", h.config.BaseURL, i)
		testEPG.WriteString(fmt.Sprintf(`  <channel id="%s">
    <display-name>Test: %s</display-name>
    <icon src="%s" />
  </channel>
`, channelID, profile.Name, channelIconURL))

		// Add programmes for the next 24 hours (continuous show)
		for hour := 0; hour < 24; hour++ {
			startTime := now.Add(time.Duration(hour) * time.Hour)
			endTime := startTime.Add(time.Hour)

			programIconURL := fmt.Sprintf("%s/test-icon/program/%d", h.config.BaseURL, i)
			testEPG.WriteString(fmt.Sprintf(`  <programme start="%s" stop="%s" channel="%s">
    <title lang="en">Test Pattern: %s</title>
    <desc lang="en">Continuous test pattern stream at %s resolution, %dfps, %s video bitrate. Audio: %d channels at %dHz, %s bitrate. This is a synthetic test stream for validating transcoding and playback capabilities.</desc>
    <category lang="en">Test</category>
    <icon src="%s" />
  </programme>
`,
				startTime.Format("20060102150405 +0000"),
				endTime.Format("20060102150405 +0000"),
				channelID,
				profile.Name,
				profile.Resolution,
				profile.Framerate,
				profile.Bitrate,
				profile.AudioChannels,
				profile.AudioRate,
				profile.AudioBitrate,
				programIconURL))
		}
	}

	// Insert test EPG before closing tag
	result := strings.Replace(epgStr, closingTag, testEPG.String()+closingTag, 1)
	return []byte(result)
}
