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
	// ErrInternalAddress is returned when trying to proxy to internal addresses.
	ErrInternalAddress = errors.New("cannot proxy to internal addresses")
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

	req.Header.Set("User-Agent", r.Header.Get("User-Agent"))
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", "IPTV-Proxy/1.0")
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch stream: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	copyHeaders(w.Header(), resp.Header)
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

	host := strings.ToLower(u.Hostname())
	if host == "localhost" || host == "127.0.0.1" || host == "0.0.0.0" ||
		strings.HasPrefix(host, "192.168.") || strings.HasPrefix(host, "10.") ||
		strings.HasPrefix(host, "172.16.") || strings.HasPrefix(host, "172.17.") ||
		strings.HasPrefix(host, "172.18.") || strings.HasPrefix(host, "172.19.") ||
		strings.HasPrefix(host, "172.20.") || strings.HasPrefix(host, "172.21.") ||
		strings.HasPrefix(host, "172.22.") || strings.HasPrefix(host, "172.23.") ||
		strings.HasPrefix(host, "172.24.") || strings.HasPrefix(host, "172.25.") ||
		strings.HasPrefix(host, "172.26.") || strings.HasPrefix(host, "172.27.") ||
		strings.HasPrefix(host, "172.28.") || strings.HasPrefix(host, "172.29.") ||
		strings.HasPrefix(host, "172.30.") || strings.HasPrefix(host, "172.31.") {
		return ErrInternalAddress
	}

	return nil
}
