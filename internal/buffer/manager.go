// Package buffer provides advanced buffering capabilities for media streams.
package buffer

import (
	"context"
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	"github.com/savid/iptv-proxy/internal/types"
)

// BufferManager manages a circular buffer with prefetch and retry capabilities.
type BufferManager struct {
	buffer       *CircularBuffer
	config       types.BufferConfig
	retryManager *RetryManager
	logger       *log.Logger

	// Prefetch control
	prefetchActive bool
	prefetchMu     sync.Mutex

	// Statistics
	underruns int
	mu        sync.RWMutex
}

// NewBufferManager creates a new buffer manager with the specified configuration.
func NewBufferManager(config types.BufferConfig, logger *log.Logger) *BufferManager {
	return &BufferManager{
		buffer: NewCircularBuffer(config.Size),
		config: config,
		retryManager: NewRetryManager(
			config.MaxRetries,
			config.RetryDelay,
			1.5, // exponential backoff factor
		),
		logger: logger,
	}
}

// Start begins buffering data from the reader.
func (m *BufferManager) Start(ctx context.Context, reader io.Reader) error {
	// Start prefetch goroutine
	go m.prefetchLoop(ctx, reader)
	return nil
}

// prefetchLoop continuously reads from the source and fills the buffer.
func (m *BufferManager) prefetchLoop(ctx context.Context, reader io.Reader) {
	m.prefetchMu.Lock()
	m.prefetchActive = true
	m.prefetchMu.Unlock()

	defer func() {
		m.prefetchMu.Lock()
		m.prefetchActive = false
		m.prefetchMu.Unlock()
		m.buffer.Close()
	}()

	buf := make([]byte, 32*1024) // 32KB read buffer

	for {
		select {
		case <-ctx.Done():
			return
		default:
			// Check if we need to throttle based on buffer level
			stats := m.buffer.Stats()
			if stats.BufferLevel > m.config.PrefetchRatio {
				// Buffer is sufficiently full, wait a bit
				time.Sleep(100 * time.Millisecond)
				continue
			}

			// Read with retry
			n, err := m.retryManager.RetryRead(reader, buf)
			if err != nil {
				if err == io.EOF {
					m.logger.Printf("Source stream ended")
					return
				}
				m.logger.Printf("Read error after retries: %v", err)
				return
			}

			// Write to buffer
			written := 0
			for written < n {
				nw, err := m.buffer.Write(buf[written:n])
				if err != nil {
					m.logger.Printf("Buffer write error: %v", err)
					return
				}
				written += nw
			}
		}
	}
}

// Read reads data from the buffer, blocking if necessary.
func (m *BufferManager) Read(p []byte) (int, error) {
	// Wait for minimum threshold before allowing reads
	if err := m.WaitForData(m.config.MinThreshold); err != nil {
		return 0, err
	}

	n, err := m.buffer.Read(p)
	if err != nil {
		return 0, err
	}

	// Check for underrun
	if n == 0 && m.isPrefetchActive() {
		m.mu.Lock()
		m.underruns++
		m.mu.Unlock()
		m.logger.Printf("Buffer underrun detected (total: %d)", m.underruns)
	}

	return n, nil
}

// WaitForData blocks until at least minBytes are available in the buffer.
func (m *BufferManager) WaitForData(minBytes int) error {
	timeout := time.After(30 * time.Second)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return fmt.Errorf("timeout waiting for data")
		case <-ticker.C:
			if m.buffer.Available() >= minBytes {
				return nil
			}
			if !m.isPrefetchActive() && m.buffer.Available() == 0 {
				return io.EOF
			}
		}
	}
}

// Stats returns current buffer statistics.
func (m *BufferManager) Stats() types.BufferStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := m.buffer.Stats()
	stats.Underruns = m.underruns
	stats.Retries = m.retryManager.GetRetryCount()
	return stats
}

// Close stops the buffer manager and releases resources.
func (m *BufferManager) Close() error {
	m.buffer.Close()
	return nil
}

// isPrefetchActive checks if the prefetch loop is still running.
func (m *BufferManager) isPrefetchActive() bool {
	m.prefetchMu.Lock()
	defer m.prefetchMu.Unlock()
	return m.prefetchActive
}
