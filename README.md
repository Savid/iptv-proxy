# IPTV Proxy

A feature-rich IPTV proxy application written in Go that provides advanced streaming capabilities:
- **Stream Proxying**: Acts as a middleman for IPTV streams with transcoding support
- **M3U Playlist Rewriting**: Dynamically rewrites playlist URLs to route through the proxy
- **EPG Filtering**: Matches and filters Electronic Program Guide data to only include channels present in the M3U playlist
- **GPU Transcoding**: Hardware-accelerated video transcoding with automatic CPU fallback
- **Test Channels**: Built-in test pattern generator for validating client compatibility
- **HDHomeRun Emulation**: Compatible with Plex, Jellyfin, and other HDHomeRun clients

## Features

- High-performance stream proxying with concurrent connection handling
- GPU-accelerated transcoding (NVIDIA NVENC, Intel Quick Sync, AMD VCE/VCN)
- Automatic hardware detection and fallback to CPU encoding
- Built-in test channels with various resolutions and patterns
- Circular buffer with retry logic for reliable streaming
- HDHomeRun device emulation for seamless integration
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

### Basic Usage
```bash
./iptv-proxy \
  -m3u "http://example.com/playlist.m3u" \
  -epg "http://example.com/epg.xml" \
  -base "http://localhost:8080"
```

### With Test Channels and Transcoding
```bash
./iptv-proxy \
  -m3u "http://example.com/playlist.m3u" \
  -epg "http://example.com/epg.xml" \
  -base "http://localhost:8080" \
  -test-channels \
  -video-codec h264 \
  -audio-codec aac \
  -hardware-accel auto \
  -buffer-size 20
```

### Full Example with All Options
```bash
./iptv-proxy \
  -m3u "http://example.com/playlist.m3u" \
  -epg "http://example.com/epg.xml" \
  -base "http://192.168.1.100:8080" \
  -bind "0.0.0.0" \
  -port 8080 \
  -log-level info \
  -refresh-interval 30m \
  -test-channels \
  -video-codec h264 \
  -audio-codec aac \
  -video-bitrate 5000k \
  -audio-bitrate 192k \
  -hardware-accel nvidia \
  -buffer-size 20 \
  -buffer-duration 15s \
  -tuner-count 4
```

### Command Line Arguments

- `-m3u` (required): URL of the M3U playlist
- `-epg` (required): URL of the EPG XML file
- `-base` (required): Base URL for rewritten stream URLs (e.g., http://localhost:8080)
- `-bind`: IP address to bind the server to (default: 0.0.0.0)
- `-port`: Port to listen on (default: 8080)
- `-log-level`: Log level - debug, info, warn, error (default: info)
- `-refresh-interval`: Interval between data refreshes (default: 30m)
- `-test-channels`: Enable test channels (default: false)
- `-video-codec`: Video codec - copy, h264, h265, mpeg2 (default: mpeg2)
- `-audio-codec`: Audio codec - copy, aac, mp3, mp2 (default: mp2)
- `-video-bitrate`: Video bitrate (e.g., 6000k, 8M) (default: 6000k)
- `-audio-bitrate`: Audio bitrate (e.g., 192k, 224k) (default: 224k)
- `-hardware-accel`: Hardware acceleration - auto, nvidia, intel, amd, none (default: auto)
- `-buffer-size`: Stream buffer size in MB (default: 10)
- `-buffer-duration`: Buffer duration (default: 10s)
- `-tuner-count`: Number of tuners to advertise for HDHomeRun (default: 4)

## Endpoints

### Core Endpoints
- `/iptv.m3u` - Serves the rewritten M3U playlist
- `/epg.xml` - Serves the filtered EPG data
- `/stream/{encoded_url}` - Proxies individual streams
- `/health` - Health check endpoint

### HDHomeRun Endpoints
- `/` - HDHomeRun device XML description
- `/discover.json` - Device discovery (Plex compatible)
- `/discovery.json` - Device discovery 
- `/lineup.json` - Channel lineup
- `/lineup_status.json` - Lineup scanning status

### Test Channel Endpoints (when enabled)
- `/test/{channel_id}` - Test pattern streams
- `/test-icon/channel/{id}` - Channel icons (100x100 SVG)
- `/test-icon/program/{id}` - Program icons (300x200 SVG)

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
- Hardware-accelerated transcoding for reduced CPU usage
- Circular buffering with automatic retry on failures

## GPU Transcoding

The proxy supports hardware-accelerated transcoding using:
- **NVIDIA**: NVENC (H.264/H.265)
- **Intel**: Quick Sync Video (H.264/H.265)
- **AMD**: VCE/VCN (H.264/H.265)

Hardware is automatically detected at startup. If no compatible GPU is found, the system falls back to CPU encoding.

### Transcoding Options

The proxy now uses individual codec settings instead of profiles:

#### Video Codecs
- `copy`: Pass through without re-encoding
- `h264`: H.264/AVC (supports GPU acceleration)
- `h265`: H.265/HEVC (supports GPU acceleration)
- `mpeg2`: MPEG-2 (CPU only, default)

#### Audio Codecs
- `copy`: Pass through without re-encoding
- `aac`: Advanced Audio Coding
- `mp3`: MPEG Layer 3
- `mp2`: MPEG Layer 2 (default)

#### Examples

**Default (MPEG-2/MP2 - equivalent to old plex-gump profile):**
```bash
./iptv-proxy -m3u URL -epg URL -base URL
```

**H.264/AAC for better compatibility:**
```bash
./iptv-proxy -m3u URL -epg URL -base URL \
  -video-codec h264 -audio-codec aac
```

**Copy both streams (no transcoding):**
```bash
./iptv-proxy -m3u URL -epg URL -base URL \
  -video-codec copy -audio-codec copy
```

**GPU-accelerated H.264:**
```bash
./iptv-proxy -m3u URL -epg URL -base URL \
  -video-codec h264 -hardware-accel nvidia
```

Note: GPU acceleration only works with h264 and h265 codecs.

## Test Channels

When enabled with `-test-channels`, the proxy generates 10 test channels:

| Channel | Resolution | FPS | Video | Audio |
|---------|------------|-----|-------|-------|
| Test 0  | 4K (3840x2160) | 60 | 20Mbps | 5.1 @ 384kbps |
| Test 1  | 4K (3840x2160) | 30 | 15Mbps | Stereo @ 192kbps |
| Test 2  | 1080p | 60 | 8Mbps | 5.1 @ 256kbps |
| Test 3  | 1080p | 30 | 5Mbps | Stereo @ 128kbps |
| Test 4  | 720p | 60 | 4Mbps | Stereo @ 128kbps |
| Test 5  | 720p | 30 | 2.5Mbps | Stereo @ 96kbps |
| Test 6  | 720p | 30 | 2Mbps | 5.1 @ 256kbps |
| Test 7  | 720p | 30 | 2Mbps | 7.1 @ 448kbps |
| Test 8  | 720p | 30 | 2Mbps | Stereo HQ @ 320kbps |
| Test 9  | 720p | 30 | 1Mbps | Mono @ 64kbps |

Test channels include:
- Dynamic SVG icons for each channel
- EPG data with 24-hour programming
- Various test patterns (SMPTE bars, grid, ball)

## Integration with Plex/Jellyfin

The proxy emulates an HDHomeRun device, making it compatible with:
- Plex Media Server
- Jellyfin
- Emby
- Any HDHomeRun-compatible software

### Setup with Plex
1. Start the proxy with your configuration
2. In Plex, go to Settings > Live TV & DVR
3. Click "Set Up Plex DVR"
4. Enter your proxy URL: `http://YOUR_IP:8080`
5. Plex will automatically detect channels

### Setup with Jellyfin
1. Start the proxy
2. In Jellyfin, go to Dashboard > Live TV
3. Add a new tuner device
4. Select HDHomeRun and enter: `http://YOUR_IP:8080`

## Troubleshooting

### Test Channels Not Working in Plex
- Use your machine's IP instead of localhost
- Check firewall settings for port 8080
- Enable verbose logging in Plex
- Run the included `plex_test_debug.sh` script

### GPU Transcoding Not Working
- Verify GPU drivers are installed
- Check `nvidia-smi` (NVIDIA) or `/dev/dri` (Intel/AMD)
- Use `-hardware-accel none` to force CPU encoding
- Ensure you're using h264 or h265 codecs (GPU doesn't support mpeg2)

### High CPU Usage
- Enable GPU transcoding with `-hardware-accel auto` and `-video-codec h264`
- Use `-video-codec copy -audio-codec copy` to disable transcoding
- Reduce concurrent streams or buffer size