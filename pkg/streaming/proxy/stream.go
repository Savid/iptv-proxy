// Package proxy provides HTTP stream proxying functionality for IPTV streams.
package proxy

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var (
	// ErrUnsupportedScheme is returned when the URL scheme is not http or https.
	ErrUnsupportedScheme = errors.New("unsupported URL scheme")
	// ErrMissingHost is returned when the URL has no host.
	ErrMissingHost = errors.New("missing host in URL")
)

// getHopHeaders returns HTTP headers that should not be forwarded when proxying.
func getHopHeaders() []string {
	return []string{
		"Connection",
		"Keep-Alive",
		"Proxy-Authenticate",
		"Proxy-Authorization",
		"Te",
		"Trailers",
		"Transfer-Encoding",
		"Upgrade",
	}
}

// Stream handles proxying of HTTP streams from a target URL to the client.
// It validates the target URL, copies headers, and streams the response body.
func Stream(w http.ResponseWriter, r *http.Request, targetURL string) error {
	httpClient := &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
		},
		Timeout: 120 * time.Second,
	}

	if err := validateURL(targetURL); err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(r.Context(), "GET", targetURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	copyHeaders(req.Header, r.Header)

	// Only set User-Agent if client didn't provide one
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", "IPTV-Proxy/1.0")
	}

	// Only set Accept if client didn't provide one
	if req.Header.Get("Accept") == "" {
		req.Header.Set("Accept", "*/*")
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch stream: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	copyHeaders(w.Header(), resp.Header)

	// Only set default content type if upstream didn't provide one
	if w.Header().Get("Content-Type") == "" {
		// Default to video/mp2t for MPEG-TS streams which are common in IPTV
		w.Header().Set("Content-Type", "video/mp2t")
	}

	// Remove content-length if present to allow streaming
	// This is important for chunked transfer encoding
	w.Header().Del("Content-Length")

	w.WriteHeader(resp.StatusCode)

	ctx := r.Context()
	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _ = io.Copy(w, resp.Body)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
		return nil
	}
}

func copyHeaders(dst, src http.Header) {
	hopHeaders := getHopHeaders()
	for k, vv := range src {
		skip := false
		for _, h := range hopHeaders {
			if strings.EqualFold(k, h) {
				skip = true
				break
			}
		}
		if !skip {
			for _, v := range vv {
				dst.Add(k, v)
			}
		}
	}
}

func validateURL(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("%w: %s", ErrUnsupportedScheme, u.Scheme)
	}

	if u.Host == "" {
		return ErrMissingHost
	}

	return nil
}
