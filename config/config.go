// Package config provides configuration management for the IPTV proxy server.
package config

import (
	"errors"
	"flag"
	"fmt"
	"net/url"
	"time"
)

var (
	// ErrM3UURLRequired is returned when M3U URL is not provided.
	ErrM3UURLRequired = errors.New("m3u URL is required")
	// ErrEPGURLRequired is returned when EPG URL is not provided.
	ErrEPGURLRequired = errors.New("epg URL is required")
	// ErrBaseURLRequired is returned when base URL is not provided.
	ErrBaseURLRequired = errors.New("base URL is required")
	// ErrInvalidPort is returned when port number is invalid.
	ErrInvalidPort = errors.New("invalid port number")
	// ErrRefreshIntervalPositive is returned when refresh interval is not positive.
	ErrRefreshIntervalPositive = errors.New("refresh interval must be positive")
	// ErrInvalidLogLevel is returned when log level is invalid.
	ErrInvalidLogLevel = errors.New("invalid log level")
)

// Config holds the application configuration.
type Config struct {
	M3UURL          string
	EPGURL          string
	BaseURL         string
	Port            int
	LogLevel        string
	RefreshInterval time.Duration
}

// New creates a new configuration instance by parsing command-line flags.
func New() (*Config, error) {
	cfg := &Config{}

	flag.StringVar(&cfg.M3UURL, "m3u", "", "URL of the M3U playlist (required)")
	flag.StringVar(&cfg.EPGURL, "epg", "", "URL of the EPG XML file (required)")
	flag.StringVar(&cfg.BaseURL, "base", "", "Base URL for rewritten stream URLs (e.g., http://localhost:8080) (required)")
	flag.IntVar(&cfg.Port, "port", 8080, "Port to listen on")
	flag.StringVar(&cfg.LogLevel, "log-level", "info", "Log level (debug, info, warn, error)")
	flag.DurationVar(&cfg.RefreshInterval, "refresh-interval", 30*time.Minute, "Interval between data refreshes")

	flag.Parse()

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate checks if the configuration is valid.
func (c *Config) Validate() error {
	if c.M3UURL == "" {
		return ErrM3UURLRequired
	}

	if c.EPGURL == "" {
		return ErrEPGURLRequired
	}

	if c.BaseURL == "" {
		return ErrBaseURLRequired
	}

	if _, err := url.Parse(c.M3UURL); err != nil {
		return fmt.Errorf("invalid M3U URL: %w", err)
	}

	if _, err := url.Parse(c.EPGURL); err != nil {
		return fmt.Errorf("invalid EPG URL: %w", err)
	}

	if _, err := url.Parse(c.BaseURL); err != nil {
		return fmt.Errorf("invalid base URL: %w", err)
	}

	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("%w: %d", ErrInvalidPort, c.Port)
	}

	if c.RefreshInterval <= 0 {
		return ErrRefreshIntervalPositive
	}

	validLogLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !validLogLevels[c.LogLevel] {
		return fmt.Errorf("%w: %s (must be debug, info, warn, or error)", ErrInvalidLogLevel, c.LogLevel)
	}

	return nil
}
