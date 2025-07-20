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
- **Multi-GPU support** - Select specific GPUs with device IDs (e.g., nvidia:0, nvidia:1)
- **Quality presets** - Simple low/medium/high presets instead of manual bitrates
- **Transcode modes** - Easy switching between copy and transcode modes
- **Codec validation** - Automatic compatibility checking between codecs and hardware
- Automatic hardware detection with detailed device listing at startup
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

### Stream Copy Mode (No Transcoding)
```bash
./iptv-proxy \
  -m3u "http://example.com/playlist.m3u" \
  -epg "http://example.com/epg.xml" \
  -base "http://localhost:8080" \
  -transcode-mode copy
```

### Transcoding with Quality Presets
```bash
./iptv-proxy \
  -m3u "http://example.com/playlist.m3u" \
  -epg "http://example.com/epg.xml" \
  -base "http://localhost:8080" \
  -video-quality high \
  -audio-quality medium \
  -hardware-device auto
```

### Multi-GPU Selection
```bash
./iptv-proxy \
  -m3u "http://example.com/playlist.m3u" \
  -epg "http://example.com/epg.xml" \
  -base "http://localhost:8080" \
  -hardware-device nvidia:1 \
  -video-codec h265
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
  -transcode-mode transcode \
  -hardware-device nvidia:0 \
  -video-codec h264 \
  -audio-codec aac \
  -video-quality custom \
  -audio-quality custom \
  -custom-video-bitrate 5000k \
  -custom-audio-bitrate 192k \
  -buffer-size 20 \
  -buffer-duration 15s \
  -buffer-prefetch-ratio 0.8 \
  -tuner-count 4
```

### Command Line Arguments

#### Required Arguments
- `-m3u`: URL of the M3U playlist
- `-epg`: URL of the EPG XML file
- `-base`: Base URL for rewritten stream URLs (e.g., http://localhost:8080)

#### Server Configuration
- `-bind`: IP address to bind the server to (default: 0.0.0.0)
- `-port`: Port to listen on (default: 8080)
- `-log-level`: Log level - debug, info, warn, error (default: info)
- `-refresh-interval`: Interval between data refreshes (default: 30m)
- `-tuner-count`: Number of tuners to advertise for HDHomeRun (default: 2)

#### Transcoding Configuration
- `-transcode-mode`: Transcoding mode - copy or transcode (default: transcode)
- `-hardware-device`: Hardware device - auto, none, or device ID (e.g., nvidia:0, intel:0) (default: auto)
- `-video-codec`: Video codec when transcoding - h264, h265, vp9, mpeg2 (default: h264)
- `-audio-codec`: Audio codec when transcoding - aac, mp3, mp2, opus (default: aac)
- `-video-quality`: Video quality - low, medium, high, or custom (default: medium)
- `-audio-quality`: Audio quality - low, medium, high, or custom (default: medium)
- `-custom-video-bitrate`: Custom video bitrate when quality is 'custom' (e.g., 8M, 10000k)
- `-custom-audio-bitrate`: Custom audio bitrate when quality is 'custom' (e.g., 320k)

#### Buffer Configuration
- `-buffer-size`: Stream buffer size in MB (default: 10)
- `-buffer-duration`: Buffer duration (default: 10s)
- `-buffer-prefetch-ratio`: Buffer prefetch ratio 0.0-1.0 (default: 0.8)

#### Test Channels
- `-test-channels`: Enable test channels (default: false)
- `-test-port`: Port for test channel server (default: 8889)

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
- **Intel**: Quick Sync Video (H.264/H.265/VP9)
- **AMD**: VCE/VCN (H.264/H.265)

### Hardware Detection
At startup, the proxy automatically detects all available GPUs and displays them:
```
Available hardware devices:
  Device type=nvidia id=0 name=NVIDIA GeForce RTX 3080 capabilities=[h264 h265]
  Device type=nvidia id=1 name=NVIDIA GeForce GTX 1660 capabilities=[h264 h265]
  Device type=intel id=0 name=Intel GPU (iHD driver) capabilities=[h264 h265 vp9]
```

### Multi-GPU Support
You can select a specific GPU using the device ID:
- `-hardware-device auto`: Automatically select the best available GPU (default)
- `-hardware-device none`: Force CPU encoding
- `-hardware-device nvidia:0`: Use the first NVIDIA GPU
- `-hardware-device nvidia:1`: Use the second NVIDIA GPU
- `-hardware-device intel:0`: Use the first Intel GPU

### Transcoding Options

#### Transcode Modes
- `copy`: Pass through streams without re-encoding
- `transcode`: Re-encode streams with specified codecs and quality

#### Video Codecs (when transcoding)
- `h264`: H.264/AVC (supports GPU acceleration) - default
- `h265`: H.265/HEVC (supports GPU acceleration)
- `vp9`: VP9 (Intel GPU only)
- `mpeg2`: MPEG-2 (CPU only)

#### Audio Codecs (when transcoding)
- `aac`: Advanced Audio Coding - default
- `mp3`: MPEG Layer 3
- `mp2`: MPEG Layer 2
- `opus`: Opus codec

#### Quality Presets
Instead of specifying bitrates directly, you can use quality presets:

**Video Quality Presets:**
- `low`: Lower quality, smaller files (2M for H.264/H.265, 4M for MPEG-2)
- `medium`: Balanced quality and size (4M for H.264/H.265, 6M for MPEG-2) - default
- `high`: Higher quality, larger files (8M for H.264/H.265, 10M for MPEG-2)
- `custom`: Use `-custom-video-bitrate` to specify exact bitrate

**Audio Quality Presets:**
- `low`: Lower quality (128k for AAC/MP3, 192k for MP2)
- `medium`: Balanced quality (192k for AAC/MP3, 224k for MP2) - default
- `high`: Higher quality (256k for AAC/MP3, 320k for MP2)
- `custom`: Use `-custom-audio-bitrate` to specify exact bitrate

#### Examples

**Stream Copy (no transcoding):**
```bash
./iptv-proxy -m3u URL -epg URL -base URL \
  -transcode-mode copy
```

**Default Quality Transcoding (H.264/AAC with medium quality):**
```bash
./iptv-proxy -m3u URL -epg URL -base URL
```

**High Quality with GPU:**
```bash
./iptv-proxy -m3u URL -epg URL -base URL \
  -video-quality high \
  -audio-quality high \
  -hardware-device auto
```

**Custom Bitrate with Specific GPU:**
```bash
./iptv-proxy -m3u URL -epg URL -base URL \
  -video-quality custom \
  -custom-video-bitrate 10M \
  -hardware-device nvidia:0
```

**CPU-only MPEG-2 Encoding:**
```bash
./iptv-proxy -m3u URL -epg URL -base URL \
  -video-codec mpeg2 \
  -hardware-device none
```

Note: GPU acceleration only works with h264, h265, and vp9 (Intel only) codecs.

### Codec Compatibility

The proxy automatically validates codec compatibility with selected hardware:

| Hardware | Supported Codecs |
|----------|------------------|
| CPU | All codecs (h264, h265, vp9, mpeg2) |
| NVIDIA | h264, h265 |
| Intel | h264, h265, vp9 (if supported) |
| AMD | h264, h265 |

If you select an incompatible codec/hardware combination, the proxy will report an error at startup.

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
- Check if GPUs are detected at startup (should list available devices)
- Verify GPU drivers are installed
- Check `nvidia-smi` (NVIDIA) or `/dev/dri` (Intel/AMD)
- Use `-hardware-device none` to force CPU encoding
- Ensure you're using h264, h265, or vp9 codecs (GPU doesn't support mpeg2)
- Try selecting a specific GPU with `-hardware-device nvidia:0` or `-hardware-device intel:0`

### High CPU Usage
- Enable GPU transcoding with `-hardware-device auto`
- Use `-transcode-mode copy` to disable transcoding entirely
- Lower quality presets with `-video-quality low -audio-quality low`
- Reduce concurrent streams or buffer size