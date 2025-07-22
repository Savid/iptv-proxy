// Package data provides in-memory data storage and fetching for IPTV proxy.
package data

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/savid/iptv-proxy/config"
	"github.com/savid/iptv-proxy/pkg/epg"
	"github.com/savid/iptv-proxy/pkg/m3u"
	"github.com/sirupsen/logrus"
)

var (
	// ErrUnexpectedStatus is returned when the HTTP response has an unexpected status code.
	ErrUnexpectedStatus = errors.New("unexpected status code")
)

// Fetcher handles fetching M3U and EPG data from remote sources.
type Fetcher struct {
	config *config.Config
	client *http.Client
	logger *logrus.Logger
}

// FetchResult contains the results of fetching both M3U and EPG data.
type FetchResult struct {
	M3U struct {
		Raw      []byte
		Channels []m3u.Channel
	}
	EPG struct {
		Raw      []byte
		Filtered []byte
	}
	Error error
}

// NewFetcher creates a new fetcher instance.
func NewFetcher(cfg *config.Config, logger *logrus.Logger) *Fetcher {
	return &Fetcher{
		config: cfg,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
		logger: logger,
	}
}

// FetchAll fetches both M3U and EPG data, respecting their dependencies.
func (f *Fetcher) FetchAll() (*FetchResult, error) {
	result := &FetchResult{}

	// Fetch M3U first
	m3uRaw, channels, err := f.fetchM3U()
	if err != nil {
		result.Error = fmt.Errorf("failed to fetch M3U: %w", err)
		return result, result.Error
	}

	result.M3U.Raw = m3uRaw
	result.M3U.Channels = channels

	// Fetch and filter EPG based on M3U channels
	epgRaw, epgFiltered, err := f.fetchAndFilterEPG(channels)
	if err != nil {
		result.Error = fmt.Errorf("failed to fetch EPG: %w", err)
		return result, result.Error
	}

	result.EPG.Raw = epgRaw
	result.EPG.Filtered = epgFiltered

	return result, nil
}

func (f *Fetcher) fetchM3U() ([]byte, []m3u.Channel, error) {
	f.logger.WithField("url", f.config.M3UURL).Info("Fetching M3U data")

	// Set specific timeout for M3U fetch
	client := &http.Client{Timeout: 30 * time.Second}

	resp, err := client.Get(f.config.M3UURL)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to fetch M3U: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("%w: %d", ErrUnexpectedStatus, resp.StatusCode)
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read M3U body: %w", err)
	}

	// Parse M3U
	channels, err := m3u.Parse(body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse M3U: %w", err)
	}

	// Rewrite M3U URLs
	rewrittenM3U := m3u.Rewrite(channels, f.config.BaseURL)

	f.logger.WithField("channels", len(channels)).Info("Successfully fetched and processed M3U")
	return rewrittenM3U, channels, nil
}

func (f *Fetcher) fetchAndFilterEPG(channels []m3u.Channel) (raw, filtered []byte, err error) {
	f.logger.WithField("url", f.config.EPGURL).Info("Fetching EPG data")

	resp, err := f.client.Get(f.config.EPGURL)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to fetch EPG: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("%w: %d", ErrUnexpectedStatus, resp.StatusCode)
	}

	// Read the raw EPG data
	raw, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read EPG body: %w", err)
	}

	// Parse EPG from raw data
	tv, err := epg.ParseStream(bytes.NewReader(raw))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse EPG: %w", err)
	}

	// Filter EPG based on M3U channels
	filteredTV, channelMap := epg.Filter(tv, channels)

	// Encode filtered EPG to XML
	var buf bytes.Buffer
	buf.WriteString(xml.Header)
	encoder := xml.NewEncoder(&buf)
	encoder.Indent("", "  ")
	if err := encoder.Encode(filteredTV); err != nil {
		return nil, nil, fmt.Errorf("failed to encode filtered EPG: %w", err)
	}

	f.logger.WithFields(logrus.Fields{
		"original_channels": len(tv.Channels),
		"filtered_channels": len(filteredTV.Channels),
		"matched_channels":  len(channelMap),
	}).Info("Successfully fetched and filtered EPG")

	return raw, buf.Bytes(), nil
}
