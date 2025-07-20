package epg

import (
	"crypto/md5" //nolint:gosec // MD5 is used for ID generation, not security
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/savid/iptv-proxy/internal/m3u"
	"github.com/sirupsen/logrus"
)

// Filter filters EPG data to only include channels and programs that match the M3U playlist.
func Filter(epgData *TV, m3uChannels []m3u.Channel) (*TV, map[string]string) {
	channelMap := buildChannelMap(m3uChannels)

	matchedChannels, channelIDMap := matchChannels(epgData.Channels, channelMap)

	// Track which channel IDs have programmes
	channelsWithPrograms := make(map[string]bool)

	// Track original IDs for duplicated channels
	originalIDMap := make(map[string][]string)
	for channelID := range channelIDMap {
		// Check if this is a suffixed ID (contains "-" followed by a number)
		if idx := strings.LastIndex(channelID, "-"); idx > 0 {
			if suffix := channelID[idx+1:]; isNumericSuffix(suffix) {
				originalID := channelID[:idx]
				originalIDMap[originalID] = append(originalIDMap[originalID], channelID)
			}
		}
	}

	var filteredPrograms []Programme
	for _, program := range epgData.Programs {
		if _, exists := channelIDMap[program.Channel]; exists {
			filteredPrograms = append(filteredPrograms, program)
			channelsWithPrograms[program.Channel] = true
		}

		// Also duplicate programmes for suffixed channel IDs
		if suffixedIDs, exists := originalIDMap[program.Channel]; exists {
			for _, suffixedID := range suffixedIDs {
				duplicatedProgram := program
				duplicatedProgram.Channel = suffixedID
				filteredPrograms = append(filteredPrograms, duplicatedProgram)
				channelsWithPrograms[suffixedID] = true
			}
		}
	}

	// Generate fake programmes for matched channels without programmes
	fakeProgramsForMatched := generateFakeProgrammes(matchedChannels, channelsWithPrograms)
	filteredPrograms = append(filteredPrograms, fakeProgramsForMatched...)

	// Generate fake channels and programmes for unmatched M3U channels
	fakeChannels, fakePrograms := generateFakeEPGData(m3uChannels, matchedChannels)
	matchedChannels = append(matchedChannels, fakeChannels...)
	filteredPrograms = append(filteredPrograms, fakePrograms...)

	// Add fake channels to the channel ID map
	for _, fakeChannel := range fakeChannels {
		channelIDMap[fakeChannel.ID] = fakeChannel.DisplayName
	}

	return &TV{
		XMLName:  epgData.XMLName,
		Channels: matchedChannels,
		Programs: filteredPrograms,
	}, channelIDMap
}

func buildChannelMap(m3uChannels []m3u.Channel) map[string]bool {
	channelMap := make(map[string]bool)

	for _, channel := range m3uChannels {
		// Use Name (which becomes GuideName in lineup.json) instead of TVGName
		if channel.Name != "" {
			channelMap[channel.Name] = true
		}
	}

	return channelMap
}

func matchChannels(epgChannels []Channel, channelMap map[string]bool) ([]Channel, map[string]string) {
	var matchedChannels []Channel
	channelIDMap := make(map[string]string)
	duplicateCheck := make(map[string]bool)
	idUsageCount := make(map[string]int)

	for _, epgChannel := range epgChannels {
		if channelMap[epgChannel.DisplayName] {
			if duplicateCheck[epgChannel.DisplayName] {
				logrus.WithFields(logrus.Fields{
					"channel": epgChannel.DisplayName,
					"id":      epgChannel.ID,
				}).Warn("Duplicate EPG channel found")
				continue
			}

			// If channel has empty ID, generate one based on display name
			if epgChannel.ID == "" {
				epgChannel.ID = generateChannelID(epgChannel.DisplayName)
				logrus.WithFields(logrus.Fields{
					"channel": epgChannel.DisplayName,
					"id":      epgChannel.ID,
				}).Debug("Generated ID for EPG channel with empty ID")
			}

			// Check if this ID has been used before
			originalID := epgChannel.ID
			if count, exists := idUsageCount[originalID]; exists {
				// Append suffix for duplicate IDs
				epgChannel.ID = fmt.Sprintf("%s-%d", originalID, count+1)
				logrus.WithFields(logrus.Fields{
					"channel":    epgChannel.DisplayName,
					"originalID": originalID,
					"newID":      epgChannel.ID,
				}).Debug("Appended suffix to duplicate channel ID")
			}
			idUsageCount[originalID]++

			matchedChannels = append(matchedChannels, epgChannel)
			channelIDMap[epgChannel.ID] = epgChannel.DisplayName
			duplicateCheck[epgChannel.DisplayName] = true
		}
	}

	unmatchedCount := 0
	var unmatchedChannels []string
	for channelName := range channelMap {
		if !duplicateCheck[channelName] {
			unmatchedCount++
			unmatchedChannels = append(unmatchedChannels, channelName)
		}
	}

	if unmatchedCount > 0 {
		logrus.WithField("count", unmatchedCount).Warn("M3U channels have no EPG match")
		logrus.Debug("Unmatched M3U channels:")
		for _, channel := range unmatchedChannels {
			logrus.Debugf("  - %s", channel)
		}
	}

	logrus.WithField("matched", len(matchedChannels)).Info("Matched channels between M3U and EPG")

	return matchedChannels, channelIDMap
}

// generateFakeEPGData creates fake EPG entries for channels that don't have EPG data.
func generateFakeEPGData(m3uChannels []m3u.Channel, matchedChannels []Channel) ([]Channel, []Programme) {
	// Create a map of already matched channels for quick lookup
	matchedMap := make(map[string]bool)
	for _, ch := range matchedChannels {
		matchedMap[ch.DisplayName] = true
	}

	// Pre-allocate slices with estimated capacity
	fakeChannels := make([]Channel, 0, len(m3uChannels))
	fakePrograms := make([]Programme, 0, len(m3uChannels))

	// Get current time and format for EPG
	now := time.Now().UTC()
	startTime := now.Format("20060102150405 +0000")
	endTime := now.Add(24 * time.Hour).Format("20060102150405 +0000")

	for _, m3uChannel := range m3uChannels {
		// Skip if channel already has EPG data (using Name which is GuideName)
		if matchedMap[m3uChannel.Name] {
			continue
		}

		// Skip channels without Name
		if m3uChannel.Name == "" {
			continue
		}

		// Generate a sensible channel ID by converting to lowercase and replacing spaces
		channelID := generateChannelID(m3uChannel.Name)

		// Create fake channel with DisplayName matching the M3U Name (GuideName)
		fakeChannel := Channel{
			ID:          channelID,
			DisplayName: m3uChannel.Name,
			Icon: Icon{
				Src: m3uChannel.TVGLogo,
			},
		}
		fakeChannels = append(fakeChannels, fakeChannel)

		// Create fake 24-hour programme
		fakeProgram := Programme{
			Channel:     channelID,
			Start:       startTime,
			Stop:        endTime,
			Title:       m3uChannel.Name,
			Description: "No programme information available",
		}
		fakePrograms = append(fakePrograms, fakeProgram)
	}

	if len(fakeChannels) > 0 {
		logrus.WithField("count", len(fakeChannels)).Info("Generated fake EPG data for channels without EPG")
	}

	return fakeChannels, fakePrograms
}

// generateChannelID creates a valid channel ID from a display name.
func generateChannelID(displayName string) string {
	// Use MD5 hash to create a consistent, unique ID
	// This avoids issues with special characters and ensures uniqueness
	hash := md5.Sum([]byte(displayName)) //nolint:gosec // MD5 is fine for ID generation
	return fmt.Sprintf("%x", hash)
}

// generateFakeProgrammes creates fake programme entries for channels that don't have any programmes.
func generateFakeProgrammes(channels []Channel, channelsWithPrograms map[string]bool) []Programme {
	// Pre-allocate with estimated capacity
	fakePrograms := make([]Programme, 0, len(channels))

	// Get current time and format for EPG
	now := time.Now().UTC()
	startTime := now.Format("20060102150405 +0000")
	endTime := now.Add(24 * time.Hour).Format("20060102150405 +0000")

	for _, channel := range channels {
		// Skip if channel already has programmes
		if channelsWithPrograms[channel.ID] {
			continue
		}

		// Create fake 24-hour programme
		fakeProgram := Programme{
			Channel:     channel.ID,
			Start:       startTime,
			Stop:        endTime,
			Title:       channel.DisplayName,
			Description: "No programme information available",
		}
		fakePrograms = append(fakePrograms, fakeProgram)

		logrus.WithFields(logrus.Fields{
			"channel": channel.DisplayName,
			"id":      channel.ID,
		}).Debug("Generated fake programme for channel without programmes")
	}

	return fakePrograms
}

// isNumericSuffix checks if a string contains only digits.
func isNumericSuffix(s string) bool {
	if s == "" {
		return false
	}
	_, err := strconv.Atoi(s)
	return err == nil
}
