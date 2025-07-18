// Package handlers provides HTTP handlers for the IPTV proxy server.
package handlers

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"

	"github.com/savid/iptv-proxy/config"
	"github.com/savid/iptv-proxy/internal/data"
)

// DeviceXML represents the UPnP device description.
type DeviceXML struct {
	XMLName     xml.Name `xml:"root"`
	Xmlns       string   `xml:"xmlns,attr"`
	URLBase     string   `xml:"URLBase"`
	SpecVersion SpecVersion
	Device      Device
}

// SpecVersion represents the UPnP spec version.
type SpecVersion struct {
	Major int `xml:"major"`
	Minor int `xml:"minor"`
}

// Device represents the UPnP device information.
type Device struct {
	DeviceType   string `xml:"deviceType"`
	FriendlyName string `xml:"friendlyName"`
	Manufacturer string `xml:"manufacturer"`
	ModelName    string `xml:"modelName"`
	ModelNumber  string `xml:"modelNumber"`
	SerialNumber string `xml:"serialNumber"`
	UDN          string `xml:"UDN"`
}

// DiscoveryJSON represents the device discovery response.
type DiscoveryJSON struct {
	FriendlyName    string `json:"FriendlyName"`
	Manufacturer    string `json:"Manufacturer"`
	ManufacturerURL string `json:"ManufacturerURL"`
	ModelNumber     string `json:"ModelNumber"`
	FirmwareName    string `json:"FirmwareName"`
	TunerCount      int    `json:"TunerCount"`
	FirmwareVersion string `json:"FirmwareVersion"`
	DeviceID        string `json:"DeviceID"`
	DeviceAuth      string `json:"DeviceAuth"`
	BaseURL         string `json:"BaseURL"`
	LineupURL       string `json:"LineupURL"`
}

// LineupItem represents a channel in the lineup.
type LineupItem struct {
	GuideNumber string `json:"GuideNumber"`
	GuideName   string `json:"GuideName"`
	URL         string `json:"URL"`
}

// LineupStatus represents the lineup scanning status.
type LineupStatus struct {
	ScanInProgress int      `json:"ScanInProgress"`
	ScanPossible   int      `json:"ScanPossible"`
	Source         string   `json:"Source"`
	SourceList     []string `json:"SourceList"`
}

// RootXMLHandler serves the UPnP device description at /.
func RootXMLHandler(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		device := DeviceXML{
			Xmlns:   "urn:schemas-upnp-org:device-1-0",
			URLBase: cfg.BaseURL,
			SpecVersion: SpecVersion{
				Major: 1,
				Minor: 0,
			},
			Device: Device{
				DeviceType:   "urn:schemas-upnp-org:device:MediaServer:1",
				FriendlyName: "IPTV-Proxy",
				Manufacturer: "Silicondust",
				ModelName:    "HDTC-2US",
				ModelNumber:  "HDTC-2US",
				SerialNumber: "",
				UDN:          "uuid:2025-01-IPTV-PROXY01",
			},
		}

		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusOK)

		// Write XML header
		if _, err := w.Write([]byte(xml.Header)); err != nil {
			http.Error(w, "Failed to write XML header", http.StatusInternalServerError)
			return
		}

		encoder := xml.NewEncoder(w)
		encoder.Indent("", "  ")
		if err := encoder.Encode(device); err != nil {
			http.Error(w, "Failed to encode XML", http.StatusInternalServerError)
			return
		}
	}
}

// DiscoveryHandler serves device discovery JSON at /discovery.json.
func DiscoveryHandler(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		discovery := DiscoveryJSON{
			FriendlyName:    "IPTV-Proxy",
			Manufacturer:    "Golang",
			ManufacturerURL: "https://github.com/Savid/iptv-proxy",
			ModelNumber:     "1.0",
			FirmwareName:    "bin_1.0",
			TunerCount:      cfg.TunerCount,
			FirmwareVersion: "1.0",
			DeviceID:        "2025-01-IPTV-PROXY01",
			DeviceAuth:      "iptv-proxy",
			BaseURL:         cfg.BaseURL,
			LineupURL:       fmt.Sprintf("%s/lineup.json", cfg.BaseURL),
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		if err := json.NewEncoder(w).Encode(discovery); err != nil {
			http.Error(w, "Failed to encode JSON", http.StatusInternalServerError)
			return
		}
	}
}

// LineupHandler serves channel lineup at /lineup.json.
func LineupHandler(cfg *config.Config, store *data.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		_, channels, ok := store.GetM3U()
		if !ok {
			http.Error(w, "No M3U data available", http.StatusServiceUnavailable)
			return
		}

		lineup := make([]LineupItem, 0, len(channels))
		for i, channel := range channels {
			// Generate proxy URL for the stream
			proxyURL := fmt.Sprintf("%s/stream/%s", cfg.BaseURL, url.QueryEscape(channel.URL))

			lineup = append(lineup, LineupItem{
				GuideNumber: fmt.Sprintf("%d", i+1),
				GuideName:   channel.Name,
				URL:         proxyURL,
			})
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		if err := json.NewEncoder(w).Encode(lineup); err != nil {
			http.Error(w, "Failed to encode JSON", http.StatusInternalServerError)
			return
		}
	}
}

// LineupStatusHandler serves the lineup scanning status at /lineup_status.json.
func LineupStatusHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		status := LineupStatus{
			ScanInProgress: 0,
			ScanPossible:   0,
			Source:         "Cable",
			SourceList:     []string{"Cable"},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		if err := json.NewEncoder(w).Encode(status); err != nil {
			http.Error(w, "Failed to encode JSON", http.StatusInternalServerError)
			return
		}
	}
}
