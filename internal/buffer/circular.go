// Package buffer provides advanced buffering capabilities for media streams.
package buffer

import (
	"errors"
	"io"
	"sync"

	"github.com/savid/iptv-proxy/internal/types"
)

// CircularBuffer implements a thread-safe circular buffer for streaming data.
type CircularBuffer struct {
	data         []byte
	size         int
	writePos     int
	readPos      int
	bytesWritten int64
	bytesRead    int64
	mu           sync.RWMutex
	cond         *sync.Cond
	closed       bool
}

// ErrBufferClosed is returned when operations are attempted on a closed buffer.
var ErrBufferClosed = errors.New("buffer is closed")

// NewCircularBuffer creates a new circular buffer with the specified size.
func NewCircularBuffer(size int) *CircularBuffer {
	b := &CircularBuffer{
		data: make([]byte, size),
		size: size,
	}
	b.cond = sync.NewCond(&b.mu)
	return b
}

// Write writes data to the buffer, blocking if necessary when the buffer is full.
func (b *CircularBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return 0, ErrBufferClosed
	}

	written := 0
	for written < len(p) {
		// Wait if buffer is full
		for b.Free() == 0 && !b.closed {
			b.cond.Wait()
		}

		if b.closed {
			return written, ErrBufferClosed
		}

		// Calculate how much we can write
		free := b.Free()
		toWrite := len(p) - written
		if toWrite > free {
			toWrite = free
		}

		// Write data in chunks to handle wrap-around
		for toWrite > 0 {
			// Calculate contiguous space until wrap
			contiguous := b.size - b.writePos
			if contiguous > toWrite {
				contiguous = toWrite
			}

			// Copy data
			copy(b.data[b.writePos:b.writePos+contiguous], p[written:written+contiguous])

			// Update positions
			b.writePos = (b.writePos + contiguous) % b.size
			written += contiguous
			toWrite -= contiguous
			b.bytesWritten += int64(contiguous)
		}

		// Signal readers that data is available
		b.cond.Broadcast()
	}

	return written, nil
}

// Read reads data from the buffer, blocking if necessary when the buffer is empty.
func (b *CircularBuffer) Read(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Wait for data to be available
	for b.Available() == 0 && !b.closed {
		b.cond.Wait()
	}

	if b.Available() == 0 && b.closed {
		return 0, io.EOF
	}

	// Calculate how much we can read
	available := b.Available()
	toRead := len(p)
	if toRead > available {
		toRead = available
	}

	read := 0
	// Read data in chunks to handle wrap-around
	for toRead > 0 {
		// Calculate contiguous data until wrap
		contiguous := b.size - b.readPos
		if contiguous > toRead {
			contiguous = toRead
		}

		// Copy data
		copy(p[read:read+contiguous], b.data[b.readPos:b.readPos+contiguous])

		// Update positions
		b.readPos = (b.readPos + contiguous) % b.size
		read += contiguous
		toRead -= contiguous
		b.bytesRead += int64(contiguous)
	}

	// Signal writers that space is available
	b.cond.Broadcast()

	return read, nil
}

// Available returns the number of bytes available for reading.
func (b *CircularBuffer) Available() int {
	if b.writePos >= b.readPos {
		return b.writePos - b.readPos
	}
	return b.size - b.readPos + b.writePos
}

// Free returns the number of bytes available for writing.
func (b *CircularBuffer) Free() int {
	return b.size - b.Available() - 1 // Reserve 1 byte to distinguish full from empty
}

// Stats returns current buffer statistics.
func (b *CircularBuffer) Stats() types.BufferStats {
	b.mu.RLock()
	defer b.mu.RUnlock()

	available := b.Available()
	return types.BufferStats{
		BytesBuffered: int64(available),
		BytesConsumed: b.bytesRead,
		BufferLevel:   float64(available) / float64(b.size),
		Underruns:     0, // Tracked by BufferManager
		Retries:       0, // Tracked by RetryManager
	}
}

// Close closes the buffer and wakes up any waiting readers/writers.
func (b *CircularBuffer) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.closed = true
	b.cond.Broadcast()
}
