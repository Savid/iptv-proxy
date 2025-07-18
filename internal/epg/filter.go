package epg

import (
	"github.com/savid/iptv-proxy/internal/m3u"
	"github.com/sirupsen/logrus"
)

// Filter filters EPG data to only include channels and programs that match the M3U playlist.
func Filter(epgData *TV, m3uChannels []m3u.Channel) (*TV, map[string]string) {
	channelMap := buildChannelMap(m3uChannels)

	matchedChannels, channelIDMap := matchChannels(epgData.Channels, channelMap)

	var filteredPrograms []Programme
	for _, program := range epgData.Programs {
		if _, exists := channelIDMap[program.Channel]; exists {
			filteredPrograms = append(filteredPrograms, program)
		}
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
		if channel.TVGName != "" {
			channelMap[channel.TVGName] = true
		}
	}

	return channelMap
}

func matchChannels(epgChannels []Channel, channelMap map[string]bool) ([]Channel, map[string]string) {
	var matchedChannels []Channel
	channelIDMap := make(map[string]string)
	duplicateCheck := make(map[string]bool)

	for _, epgChannel := range epgChannels {
		if channelMap[epgChannel.DisplayName] {
			if duplicateCheck[epgChannel.DisplayName] {
				logrus.WithFields(logrus.Fields{
					"channel": epgChannel.DisplayName,
					"id":      epgChannel.ID,
				}).Warn("Duplicate EPG channel found")
				continue
			}

			matchedChannels = append(matchedChannels, epgChannel)
			channelIDMap[epgChannel.ID] = epgChannel.DisplayName
			duplicateCheck[epgChannel.DisplayName] = true
		}
	}

	unmatchedCount := 0
	for tvgName := range channelMap {
		if !duplicateCheck[tvgName] {
			unmatchedCount++
		}
	}

	if unmatchedCount > 0 {
		logrus.WithField("count", unmatchedCount).Warn("M3U channels have no EPG match")
	}

	logrus.WithField("matched", len(matchedChannels)).Info("Matched channels between M3U and EPG")

	return matchedChannels, channelIDMap
}
