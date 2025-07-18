package data

import (
	"testing"
	"time"

	"github.com/savid/iptv-proxy/internal/m3u"
)

func TestStoreOperations(t *testing.T) {
	store := NewStore()

	// Test initial state
	if store.HasData() {
		t.Error("New store should not have data")
	}

	// Test getting M3U when empty
	_, _, ok := store.GetM3U()
	if ok {
		t.Error("GetM3U should return false when no data")
	}

	// Test getting EPG when empty
	_, ok = store.GetEPG()
	if ok {
		t.Error("GetEPG should return false when no data")
	}

	// Set M3U data
	m3uData := []byte("#EXTM3U\n#EXTINF:-1,Test Channel\nhttp://example.com/stream")
	channels := []m3u.Channel{
		{Name: "Test Channel", URL: "http://example.com/stream"},
	}
	store.SetM3U(m3uData, channels)

	// Test getting M3U after setting
	gotData, gotChannels, ok := store.GetM3U()
	if !ok {
		t.Error("GetM3U should return true after setting data")
	}
	if string(gotData) != string(m3uData) {
		t.Errorf("Expected M3U data %q, got %q", m3uData, gotData)
	}
	if len(gotChannels) != 1 || gotChannels[0].Name != "Test Channel" {
		t.Error("Channels not stored correctly")
	}

	// Store still doesn't have all data
	if store.HasData() {
		t.Error("Store should not report having data without EPG")
	}

	// Set EPG data
	epgRaw := []byte(`<?xml version="1.0"?><tv></tv>`)
	epgFiltered := []byte(`<?xml version="1.0"?><tv></tv>`)
	store.SetEPG(epgRaw, epgFiltered)

	// Now store has all data
	if !store.HasData() {
		t.Error("Store should report having data after setting both M3U and EPG")
	}

	// Test getting EPG
	gotEPG, ok := store.GetEPG()
	if !ok {
		t.Error("GetEPG should return true after setting data")
	}
	if string(gotEPG) != string(epgFiltered) {
		t.Errorf("Expected EPG data %q, got %q", epgFiltered, gotEPG)
	}

	// Test LastSync
	lastSync := store.LastSync()
	if time.Since(lastSync) > time.Second {
		t.Error("LastSync should be recent")
	}
}

func TestStoreConcurrency(_ *testing.T) {
	store := NewStore()
	done := make(chan bool)

	// Concurrent writes
	go func() {
		for i := 0; i < 100; i++ {
			store.SetM3U([]byte("test"), []m3u.Channel{{Name: "test"}})
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			store.SetEPG([]byte("test"), []byte("test"))
		}
		done <- true
	}()

	// Concurrent reads
	go func() {
		for i := 0; i < 100; i++ {
			store.GetM3U()
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			store.GetEPG()
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			store.HasData()
		}
		done <- true
	}()

	// Wait for all goroutines
	for i := 0; i < 5; i++ {
		<-done
	}
}
