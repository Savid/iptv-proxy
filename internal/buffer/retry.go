// Package buffer provides advanced buffering capabilities for media streams.
package buffer

import (
	"fmt"
	"io"
	"sync/atomic"
	"time"
)

// RetryManager handles retry logic with exponential backoff.
type RetryManager struct {
	maxRetries int
	delay      time.Duration
	backoff    float64
	retryCount int64
}

// NewRetryManager creates a new retry manager with the specified configuration.
func NewRetryManager(maxRetries int, delay time.Duration, backoff float64) *RetryManager {
	return &RetryManager{
		maxRetries: maxRetries,
		delay:      delay,
		backoff:    backoff,
	}
}

// RetryRead attempts to read from the reader with retry logic.
func (r *RetryManager) RetryRead(reader io.Reader, buf []byte) (int, error) {
	var lastErr error
	currentDelay := r.delay

	for attempt := 0; attempt <= r.maxRetries; attempt++ {
		n, err := reader.Read(buf)
		if err == nil || err == io.EOF {
			return n, err
		}

		// Track the error
		lastErr = err
		atomic.AddInt64(&r.retryCount, 1)

		// Check if we should retry
		if attempt < r.maxRetries {
			time.Sleep(currentDelay)
			currentDelay = time.Duration(float64(currentDelay) * r.backoff)
		}
	}

	return 0, fmt.Errorf("read failed after %d retries: %w", r.maxRetries, lastErr)
}

// GetRetryCount returns the total number of retries performed.
func (r *RetryManager) GetRetryCount() int {
	return int(atomic.LoadInt64(&r.retryCount))
}

// Reset resets the retry counter.
func (r *RetryManager) Reset() {
	atomic.StoreInt64(&r.retryCount, 0)
}
