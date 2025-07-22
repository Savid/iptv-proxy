// Package types contains shared type definitions for the IPTV transcoding system.
package types

import "time"

// BufferConfig defines the configuration for the advanced buffering system.
type BufferConfig struct {
	Size          int           // Size of the buffer in bytes.
	PrefetchRatio float64       // Ratio of buffer to prefetch (0.0-1.0).
	MinThreshold  int           // Minimum bytes before allowing reads.
	MaxRetries    int           // Maximum number of retry attempts.
	RetryDelay    time.Duration // Initial delay between retries.
}

// BufferStats tracks the current state and performance of a buffer.
type BufferStats struct {
	BytesBuffered int64   // Total bytes currently in the buffer.
	BytesConsumed int64   // Total bytes read from the buffer.
	BufferLevel   float64 // Current buffer fill level (0.0-1.0).
	Underruns     int     // Number of buffer underrun events.
	Retries       int     // Number of retry attempts made.
}
