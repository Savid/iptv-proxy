package epg

import (
	"testing"

	"github.com/savid/iptv-proxy/internal/m3u"
)

func TestFilter(t *testing.T) {
	// Create test EPG data
	epgData := &TV{
		Channels: []Channel{
			{ID: "foxsports502.au", DisplayName: "FOX SPORTS 502"},
			{ID: "foxsports503.au", DisplayName: "FOX SPORTS 503"},
			{ID: "espn.us", DisplayName: "US: ESPN"},
			{ID: "notmatched", DisplayName: "Not Matched Channel"},
		},
		Programs: []Programme{
			{Channel: "foxsports502.au", Title: "Program 1"},
			{Channel: "foxsports503.au", Title: "Program 2"},
			{Channel: "espn.us", Title: "Program 3"},
			{Channel: "notmatched", Title: "Program 4"},
		},
	}

	// Create test M3U channels
	m3uChannels := []m3u.Channel{
		{TVGName: "FOX SPORTS 502"},
		{TVGName: "FOX SPORTS 503"},
		{TVGName: "US: ESPN"},
	}

	// Run filter
	filtered, channelMap := Filter(epgData, m3uChannels)

	// Test filtered channel count
	if len(filtered.Channels) != 3 {
		t.Errorf("Expected 3 filtered channels, got %d", len(filtered.Channels))
	}

	// Test filtered program count
	if len(filtered.Programs) != 3 {
		t.Errorf("Expected 3 filtered programs, got %d", len(filtered.Programs))
	}

	// Test channel map
	if len(channelMap) != 3 {
		t.Errorf("Expected 3 channel mappings, got %d", len(channelMap))
	}

	// Verify correct channels are included
	expectedChannels := map[string]bool{
		"foxsports502.au": true,
		"foxsports503.au": true,
		"espn.us":         true,
	}

	for _, ch := range filtered.Channels {
		if !expectedChannels[ch.ID] {
			t.Errorf("Unexpected channel in filtered results: %s", ch.ID)
		}
		delete(expectedChannels, ch.ID)
	}

	if len(expectedChannels) > 0 {
		t.Errorf("Missing expected channels in filtered results")
	}

	// Verify channel map content
	if channelMap["foxsports502.au"] != "FOX SPORTS 502" {
		t.Errorf("Expected channel map entry 'foxsports502.au' -> 'FOX SPORTS 502', got '%s'", channelMap["foxsports502.au"])
	}
}

func TestFilterDuplicates(t *testing.T) {
	// Create test EPG data with duplicates
	epgData := &TV{
		Channels: []Channel{
			{ID: "ch1", DisplayName: "Channel 1"},
			{ID: "ch2", DisplayName: "Channel 1"}, // Duplicate display name
			{ID: "ch3", DisplayName: "Channel 2"},
		},
		Programs: []Programme{
			{Channel: "ch1", Title: "Program 1"},
			{Channel: "ch2", Title: "Program 2"},
			{Channel: "ch3", Title: "Program 3"},
		},
	}

	// Create test M3U channels
	m3uChannels := []m3u.Channel{
		{TVGName: "Channel 1"},
		{TVGName: "Channel 2"},
	}

	// Run filter
	filtered, _ := Filter(epgData, m3uChannels)

	// Should only include first occurrence of duplicate
	if len(filtered.Channels) != 2 {
		t.Errorf("Expected 2 filtered channels (with duplicate removed), got %d", len(filtered.Channels))
	}

	// Check that ch1 is included and ch2 is excluded
	foundCh1 := false
	foundCh2 := false
	for _, ch := range filtered.Channels {
		if ch.ID == "ch1" {
			foundCh1 = true
		}
		if ch.ID == "ch2" {
			foundCh2 = true
		}
	}

	if !foundCh1 {
		t.Error("Expected ch1 to be included")
	}
	if foundCh2 {
		t.Error("Expected ch2 (duplicate) to be excluded")
	}
}

func TestFilterNoMatches(t *testing.T) {
	// Create test EPG data
	epgData := &TV{
		Channels: []Channel{
			{ID: "ch1", DisplayName: "Channel 1"},
			{ID: "ch2", DisplayName: "Channel 2"},
		},
		Programs: []Programme{
			{Channel: "ch1", Title: "Program 1"},
			{Channel: "ch2", Title: "Program 2"},
		},
	}

	// Create test M3U channels with no matches
	m3uChannels := []m3u.Channel{
		{TVGName: "Different Channel 1"},
		{TVGName: "Different Channel 2"},
	}

	// Run filter
	filtered, channelMap := Filter(epgData, m3uChannels)

	// Should have no matches
	if len(filtered.Channels) != 0 {
		t.Errorf("Expected 0 filtered channels, got %d", len(filtered.Channels))
	}
	if len(filtered.Programs) != 0 {
		t.Errorf("Expected 0 filtered programs, got %d", len(filtered.Programs))
	}
	if len(channelMap) != 0 {
		t.Errorf("Expected 0 channel mappings, got %d", len(channelMap))
	}
}
