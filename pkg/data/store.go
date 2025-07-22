package data

import (
	"sync"
	"time"

	"github.com/savid/iptv-proxy/pkg/m3u"
)

// Store provides thread-safe in-memory storage for M3U and EPG data.
type Store struct {
	mu                  sync.RWMutex
	m3uData             *M3UData
	epgData             *EPGData
	lastSync            time.Time
	testChannelsEnabled bool
}

// M3UData contains M3U playlist data and metadata.
type M3UData struct {
	Raw       []byte
	Channels  []m3u.Channel
	UpdatedAt time.Time
}

// EPGData contains EPG XML data in both raw and filtered formats.
type EPGData struct {
	Raw       []byte
	Filtered  []byte
	UpdatedAt time.Time
}

// NewStore creates a new empty data store.
func NewStore() *Store {
	return &Store{}
}

// SetM3U stores M3U data in the store.
func (s *Store) SetM3U(raw []byte, channels []m3u.Channel) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.m3uData = &M3UData{
		Raw:       raw,
		Channels:  channels,
		UpdatedAt: time.Now(),
	}
	s.lastSync = time.Now()
}

// SetEPG stores EPG data in the store.
func (s *Store) SetEPG(raw []byte, filtered []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.epgData = &EPGData{
		Raw:       raw,
		Filtered:  filtered,
		UpdatedAt: time.Now(),
	}
	s.lastSync = time.Now()
}

// GetM3U retrieves M3U data from the store. Returns false if no data is available.
func (s *Store) GetM3U() ([]byte, []m3u.Channel, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.m3uData == nil {
		return nil, nil, false
	}

	return s.m3uData.Raw, s.m3uData.Channels, true
}

// GetEPG retrieves filtered EPG data from the store. Returns false if no data is available.
func (s *Store) GetEPG() ([]byte, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.epgData == nil {
		return nil, false
	}

	return s.epgData.Filtered, true
}

// HasData returns true if the store contains both M3U and EPG data.
func (s *Store) HasData() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.m3uData != nil && s.epgData != nil
}

// LastSync returns the time of the last data synchronization.
func (s *Store) LastSync() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.lastSync
}

// SetTestChannelsEnabled enables or disables test channels.
func (s *Store) SetTestChannelsEnabled(enabled bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.testChannelsEnabled = enabled
}

// IsTestChannelsEnabled returns whether test channels are enabled.
func (s *Store) IsTestChannelsEnabled() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.testChannelsEnabled
}
