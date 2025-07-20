// Package testchannels provides test pattern generation for IPTV testing.
package testchannels

// TestChannelProfile defines the configuration for a test channel.
type TestChannelProfile struct {
	Name          string
	Resolution    string
	Framerate     int
	Bitrate       string
	AudioCodec    string
	AudioRate     int
	AudioChannels int
	AudioBitrate  string
	TestPattern   string // video pattern type: testsrc, testsrc2, smptebars, smptehdbars
}

// TestProfiles contains predefined test channel profiles.
//
//nolint:gochecknoglobals // Test profiles are immutable configuration data
var TestProfiles = []TestChannelProfile{
	// Video focused tests
	{
		Name:          "4K HDR 60fps",
		Resolution:    "3840x2160",
		Framerate:     60,
		Bitrate:       "25M",
		AudioCodec:    "aac",
		AudioChannels: 2,
		AudioRate:     48000,
		AudioBitrate:  "192k",
		TestPattern:   "testsrc2",
	},
	{
		Name:          "4K 30fps",
		Resolution:    "3840x2160",
		Framerate:     30,
		Bitrate:       "15M",
		AudioCodec:    "aac",
		AudioChannels: 2,
		AudioRate:     48000,
		AudioBitrate:  "192k",
		TestPattern:   "testsrc2",
	},
	{
		Name:          "1080p 60fps",
		Resolution:    "1920x1080",
		Framerate:     60,
		Bitrate:       "8M",
		AudioCodec:    "aac",
		AudioChannels: 2,
		AudioRate:     48000,
		AudioBitrate:  "192k",
		TestPattern:   "testsrc",
	},
	{
		Name:          "1080p 30fps",
		Resolution:    "1920x1080",
		Framerate:     30,
		Bitrate:       "5M",
		AudioCodec:    "aac",
		AudioChannels: 2,
		AudioRate:     48000,
		AudioBitrate:  "128k",
		TestPattern:   "testsrc",
	},
	{
		Name:          "720p 60fps",
		Resolution:    "1280x720",
		Framerate:     60,
		Bitrate:       "4M",
		AudioCodec:    "aac",
		AudioChannels: 2,
		AudioRate:     48000,
		AudioBitrate:  "128k",
		TestPattern:   "smptebars",
	},
	{
		Name:          "720p 30fps",
		Resolution:    "1280x720",
		Framerate:     30,
		Bitrate:       "2.5M",
		AudioCodec:    "aac",
		AudioChannels: 2,
		AudioRate:     48000,
		AudioBitrate:  "128k",
		TestPattern:   "smptebars",
	},
	// Audio focused tests
	{
		Name:          "Audio 5.1 Surround",
		Resolution:    "1920x1080",
		Framerate:     30,
		Bitrate:       "5M",
		AudioCodec:    "aac",
		AudioChannels: 6,
		AudioRate:     48000,
		AudioBitrate:  "448k",
		TestPattern:   "smptehdbars",
	},
	{
		Name:          "Audio 7.1 Surround",
		Resolution:    "1920x1080",
		Framerate:     30,
		Bitrate:       "5M",
		AudioCodec:    "aac",
		AudioChannels: 8,
		AudioRate:     48000,
		AudioBitrate:  "640k",
		TestPattern:   "smptehdbars",
	},
	{
		Name:          "Audio High Bitrate Stereo",
		Resolution:    "1920x1080",
		Framerate:     30,
		Bitrate:       "5M",
		AudioCodec:    "aac",
		AudioChannels: 2,
		AudioRate:     96000,
		AudioBitrate:  "320k",
		TestPattern:   "testsrc",
	},
	{
		Name:          "Audio Low Bitrate",
		Resolution:    "1280x720",
		Framerate:     30,
		Bitrate:       "2M",
		AudioCodec:    "aac",
		AudioChannels: 2,
		AudioRate:     44100,
		AudioBitrate:  "96k",
		TestPattern:   "testsrc",
	},
}

// GetTestProfile returns a test profile by name.
func GetTestProfile(name string) (TestChannelProfile, bool) {
	for _, profile := range TestProfiles {
		if profile.Name == name {
			return profile, true
		}
	}
	return TestChannelProfile{}, false
}

// GetTestProfileByIndex returns a test profile by index.
func GetTestProfileByIndex(index int) (TestChannelProfile, bool) {
	if index < 0 || index >= len(TestProfiles) {
		return TestChannelProfile{}, false
	}
	return TestProfiles[index], true
}
