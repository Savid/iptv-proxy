package epg

import (
	"os"
	"strings"
	"testing"
)

func TestParseStream(t *testing.T) {
	data, err := os.ReadFile("testdata/small_epg.xml")
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}

	reader := strings.NewReader(string(data))
	tv, err := ParseStream(reader)
	if err != nil {
		t.Fatalf("ParseStream failed: %v", err)
	}

	// Test channel count
	if len(tv.Channels) != 1 {
		t.Errorf("Expected 1 channel, got %d", len(tv.Channels))
	}

	// Test channel details
	if len(tv.Channels) > 0 {
		ch := tv.Channels[0]
		if ch.ID != "foxsports502.au" {
			t.Errorf("Expected channel ID 'foxsports502.au', got '%s'", ch.ID)
		}
		if ch.DisplayName != "FOX SPORTS 502" {
			t.Errorf("Expected display name 'FOX SPORTS 502', got '%s'", ch.DisplayName)
		}
		if ch.Icon.Src != "https://logo.iptveditor.com/foxsports502.png" {
			t.Errorf("Expected icon src 'https://logo.iptveditor.com/foxsports502.png', got '%s'", ch.Icon.Src)
		}
	}

	// Test programme count
	if len(tv.Programs) != 2 {
		t.Errorf("Expected 2 programs, got %d", len(tv.Programs))
	}

	// Test first programme
	if len(tv.Programs) > 0 {
		p := tv.Programs[0]
		if p.Channel != "foxsports502.au" {
			t.Errorf("Expected programme channel 'foxsports502.au', got '%s'", p.Channel)
		}
		if p.Title != "Tim Tszyu & Manny Pacquiao" {
			t.Errorf("Expected programme title 'Tim Tszyu & Manny Pacquiao', got '%s'", p.Title)
		}
		if p.Start != "20250716230000 +0000" {
			t.Errorf("Expected programme start '20250716230000 +0000', got '%s'", p.Start)
		}
		if p.Stop != "20250717003000 +0000" {
			t.Errorf("Expected programme stop '20250717003000 +0000', got '%s'", p.Stop)
		}
	}

	// Test second programme with description
	if len(tv.Programs) > 1 {
		p := tv.Programs[1]
		if p.Title != "NRL 360" {
			t.Errorf("Expected programme title 'NRL 360', got '%s'", p.Title)
		}
		if !strings.Contains(p.Description, "Braith Anasta") {
			t.Errorf("Expected programme description to contain 'Braith Anasta', got '%s'", p.Description)
		}
	}
}

func TestParseStreamInvalid(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "invalid XML",
			input:   "<tv><channel>unclosed",
			wantErr: true,
		},
		{
			name:    "empty XML",
			input:   "",
			wantErr: true,
		},
		{
			name:    "valid empty TV",
			input:   `<?xml version="1.0" encoding="utf-8"?><tv></tv>`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			_, err := ParseStream(reader)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseStream() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
