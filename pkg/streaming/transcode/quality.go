// Package transcode provides video transcoding functionality.
package transcode

const (
	// Additional video codec constants not defined in analyzer.go.
	codecVP9   = "vp9"
	codecMPEG2 = "mpeg2"

	// Additional audio codec constants not defined in analyzer.go.
	codecMP2  = "mp2"
	codecOpus = "opus"

	// Common bitrate constants.
	bitrate128k = "128k"
	bitrate192k = "192k"
	bitrate224k = "224k"
	bitrate256k = "256k"
	bitrate320k = "320k"
	bitrate96k  = "96k"
)

// QualityMapper maps quality presets to bitrate values for different codecs.
type QualityMapper struct{}

// NewQualityMapper creates a new quality preset mapper instance.
func NewQualityMapper() *QualityMapper {
	return &QualityMapper{}
}

// GetVideoBitrate returns the appropriate video bitrate for the given preset and codec.
func (q *QualityMapper) GetVideoBitrate(preset string, codec string) string {
	switch preset {
	case "low":
		switch codec {
		case codecH264, codecH265:
			return "2M"
		case codecMPEG2:
			return "4M"
		case codecVP9:
			return "1.5M"
		default:
			return "2M"
		}

	case "medium":
		switch codec {
		case codecH264, codecH265:
			return "4M"
		case codecMPEG2:
			return "6M"
		case codecVP9:
			return "3M"
		default:
			return "4M"
		}

	case "high":
		switch codec {
		case codecH264, codecH265:
			return "8M"
		case codecMPEG2:
			return "10M"
		case codecVP9:
			return "6M"
		default:
			return "8M"
		}

	default:
		// For custom or unknown presets, return medium defaults
		return q.GetVideoBitrate("medium", codec)
	}
}

// GetAudioBitrate returns the appropriate audio bitrate for the given preset and codec.
func (q *QualityMapper) GetAudioBitrate(preset string, codec string) string {
	switch preset {
	case "low":
		switch codec {
		case codecAAC, codecMP3:
			return bitrate128k
		case codecMP2:
			return bitrate192k
		case codecOpus:
			return bitrate96k
		default:
			return bitrate128k
		}

	case "medium":
		switch codec {
		case codecAAC, codecMP3:
			return bitrate192k
		case codecMP2:
			return bitrate224k
		case codecOpus:
			return bitrate128k
		default:
			return bitrate192k
		}

	case "high":
		switch codec {
		case codecAAC, codecMP3:
			return bitrate256k
		case codecMP2:
			return bitrate320k
		case codecOpus:
			return bitrate192k
		default:
			return bitrate256k
		}

	default:
		// For custom or unknown presets, return medium defaults
		return q.GetAudioBitrate("medium", codec)
	}
}
