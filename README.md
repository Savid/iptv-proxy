# IPTV Proxy

A lightweight IPTV proxy application written in Go that serves three primary functions:
- **Stream Proxying**: Acts as a middleman for IPTV streams
- **M3U Playlist Rewriting**: Dynamically rewrites playlist URLs to route through the proxy
- **EPG Filtering**: Matches and filters Electronic Program Guide data to only include channels present in the M3U playlist

## Features

- High-performance stream proxying with concurrent connection handling
- Automatic M3U playlist URL rewriting
- Intelligent EPG filtering with channel name normalization
- In-memory caching with configurable TTL
- Graceful shutdown handling
- Health check endpoint

## Installation

```bash
go build -o iptv-proxy
```

## Usage

```bash
./iptv-proxy \
  -m3u "http://example.com/playlist.m3u" \
  -epg "http://example.com/epg.xml" \
  -base "http://localhost:8080" \
  -bind "0.0.0.0" \
  -port 8080 \
  -refresh-interval 30m
```

### Command Line Arguments

- `-m3u` (required): URL of the M3U playlist
- `-epg` (required): URL of the EPG XML file
- `-base` (required): Base URL for rewritten stream URLs (e.g., http://localhost:8080)
- `-bind`: IP address to bind the server to (default: 0.0.0.0)
- `-port`: Port to listen on (default: 8080)
- `-log-level`: Log level - debug, info, warn, error (default: info)
- `-refresh-interval`: Interval between data refreshes (default: 30m)

## Endpoints

- `/iptv.m3u` - Serves the rewritten M3U playlist
- `/epg.xml` - Serves the filtered EPG data
- `/stream/{encoded_url}` - Proxies individual streams
- `/health` - Health check endpoint

## Architecture

The application uses Go's standard library to create a lightweight HTTP server that:
1. Fetches M3U playlists and EPG data from provided URLs
2. Processes them in-memory with efficient streaming techniques
3. Serves the modified content through dedicated endpoints

## Channel Matching

The EPG filter performs direct matching between:
- M3U playlist `tvg-name` attribute
- EPG XML `display-name` element

Channels are matched exactly as specified without any normalization or fuzzy matching.

## Performance

- Concurrent stream handling without blocking
- Connection pooling for upstream requests
- TTL-based caching to reduce load on sources
- Streaming XML parsing for large EPG files