# MediaMTX Dashboard Guide

## Overview
MediaMTX now includes three separate dashboard interfaces for different use cases:
- **WebRTC Dashboard** - Low-latency live streaming monitoring
- **HLS Dashboard** - HTTP Live Streaming monitoring
- **PTZ Control** - Pan-Tilt-Zoom camera control interface

All dashboards dynamically load stream information from the MediaMTX API.

## Dashboard Pages

### 1. WebRTC Dashboard
**URL:** `http://localhost:8889/dashboard`

#### Features
- Real-time WebRTC stream monitoring
- Low latency (< 1 second)
- Grid layout (2, 3, 4, or 6 columns)
- Connection status indicators
- Fullscreen support for each stream
- Auto-refresh capability

#### Best For
- Real-time monitoring
- Interactive applications
- Low-latency requirements

### 2. HLS Dashboard
**URL:** `http://localhost:8889/dashboard-hls`

#### Features
- HLS stream monitoring via HLS.js
- Compatible with all modern browsers
- Grid layout (2, 3, 4, or 6 columns)
- Connection status indicators
- Fullscreen support
- Auto-recovery on errors

#### Best For
- Broader browser compatibility
- Mobile devices
- Longer viewing sessions
- Network-adaptive streaming

#### Requirements
- HLS server must be enabled in `mediamtx.yml`:
```yaml
hls: yes
hlsAddress: :8888
```

### 3. PTZ Control
**URL:** `http://localhost:8889/ptz`

#### Features
- Dedicated PTZ camera control interface
- Live video preview (WebRTC)
- 8-directional pan/tilt control
- Zoom in/out controls
- Adjustable speed slider (10-100)
- Preset management
- One camera at a time focus

#### Controls
- **Pan/Tilt Pad**: Click and hold directional buttons
- **Home Button (⌂)**: Return to preset position 34 (origin)
- **Zoom Buttons**: Zoom in/out
- **Speed Slider**: Adjust movement speed
- **Presets**: Click to move camera to saved position

#### PTZ-Enabled Cameras
Currently configured:
- CCTV-TEST1
- CCTV-TEST2
- CCTV-TEST3

## Dynamic Stream Loading

All dashboards load streams dynamically from the MediaMTX API:

**API Endpoint:** `http://localhost:9997/v3/paths/list`

### API Response Format
```json
{
  "itemCount": 3,
  "pageCount": 1,
  "items": [
    {
      "name": "CCTV-TEST1",
      "confName": "CCTV-TEST1",
      "source": {
        "type": "rtspSource",
        "id": ""
      },
      "ready": true,
      "readyTime": "2025-11-27T16:27:34.2548237+09:00",
      "tracks": ["H264"],
      "bytesReceived": 176480528,
      "bytesSent": 175772700,
      "readers": [...]
    }
  ]
}
```

### Benefits of Dynamic Loading
- ✅ No hardcoded stream lists
- ✅ Automatically displays all configured paths
- ✅ Reflects real-time configuration changes
- ✅ Shows only active/ready streams
- ✅ Easy to add new cameras (just configure in `mediamtx.yml`)

## Configuration

### Required Settings in `mediamtx.yml`

#### For WebRTC Dashboard
```yaml
webrtc: yes
webrtcAddress: :8889
```

#### For HLS Dashboard
```yaml
hls: yes
hlsAddress: :8888
```

#### For API Access
```yaml
api: yes
apiAddress: :9997
```

#### Example Path Configuration
```yaml
paths:
  my-camera:
    source: rtsp://username:password@192.168.1.100:554/stream
    sourceOnDemand: yes
    rtspTransport: tcp
```

## Navigation

All dashboards include a navigation bar with links to:
- WebRTC Dashboard
- HLS Dashboard
- PTZ Control

Click any link to switch between dashboards.

## Features Common to All Dashboards

### Grid Layout
- 2 columns (large preview)
- 3 columns (default, balanced)
- 4 columns (compact)
- 6 columns (overview)

### Stream Status Indicators
- 🟡 Yellow (Connecting) - Stream is initializing
- 🟢 Green (Connected) - Stream is playing
- 🔴 Red (Error) - Connection failed

### Refresh
Click "Refresh All" to:
- Reload stream list from API
- Reconnect all streams
- Update camera information

## Keyboard Shortcuts (Future Enhancement)
Currently not implemented, but planned:
- Arrow keys for PTZ control
- Space for play/pause
- F for fullscreen
- R for refresh

## Troubleshooting

### No Streams Showing
1. Check API is running: `curl http://localhost:9997/v3/paths/list`
2. Verify streams are configured in `mediamtx.yml`
3. Check browser console for errors
4. Ensure MediaMTX is running

### WebRTC Not Connecting
1. Verify `webrtc: yes` in config
2. Check firewall settings
3. Ensure source streams are accessible
4. Try HLS dashboard as alternative

### HLS Not Playing
1. Verify `hls: yes` in config
2. Check HLS server is running on port 8888
3. Try accessing stream directly: `http://localhost:8888/{stream}/index.m3u8`

### PTZ Not Working
1. Verify camera is in PTZ cameras list
2. Check camera IP and credentials in `ptz_handler.go`
3. Test PTZ API directly: `curl http://localhost:8889/ptz/cameras`
4. Ensure camera supports ISAPI protocol

## Adding New Cameras

### Regular Cameras
Simply add to `mediamtx.yml`:
```yaml
paths:
  new-camera:
    source: rtsp://user:pass@ip:port/path
    sourceOnDemand: yes
```

Restart MediaMTX and the camera will appear in all dashboards automatically.

### PTZ Cameras
1. Add to `mediamtx.yml` as above
2. Edit `internal/servers/webrtc/ptz_handler.go`:
```go
var ptzCameras = map[string]PTZConfig{
    "new-camera": {
        Host:     "192.168.1.100",
        Username: "admin",
        Password: "password",
    },
}
```
3. Rebuild: `go build -o mediamtx.exe`
4. Restart MediaMTX

## Performance Tips

### WebRTC Dashboard
- Use for < 10 simultaneous streams
- Best performance with hardware acceleration enabled
- Chrome/Edge recommended for best WebRTC support

### HLS Dashboard
- Better for > 10 simultaneous streams
- More CPU efficient
- Better for mobile devices
- Slight delay (2-5 seconds) is normal

### PTZ Control
- One camera at a time for best performance
- Adjustable speed for network conditions
- Lower speed = smoother movement on slow networks

## Browser Compatibility

### WebRTC Dashboard
- ✅ Chrome/Edge (Recommended)
- ✅ Firefox
- ✅ Safari
- ⚠️ Mobile browsers (limited)

### HLS Dashboard
- ✅ All modern browsers
- ✅ Mobile browsers (iOS Safari, Android Chrome)
- ✅ Smart TVs with browser

### PTZ Control
- ✅ Chrome/Edge (Recommended)
- ✅ Firefox
- ✅ Safari
- ⚠️ Requires WebRTC for video preview

## API Endpoints Used

### MediaMTX API (Port 9997)
- `GET /v3/paths/list` - List all configured paths/streams

### PTZ API (Port 8889)
- `GET /ptz/cameras` - List PTZ-enabled cameras
- `POST /ptz/:camera/move` - Move camera
- `POST /ptz/:camera/stop` - Stop movement
- `GET /ptz/:camera/status` - Get camera status
- `GET /ptz/:camera/presets` - List presets
- `POST /ptz/:camera/preset/:id` - Go to preset

## Files

### Dashboard Files
- `internal/servers/webrtc/dashboard_webrtc.html` - WebRTC dashboard
- `internal/servers/webrtc/dashboard_hls.html` - HLS dashboard
- `internal/servers/webrtc/ptz.html` - PTZ control page
- `internal/servers/webrtc/dashboard.html` - Legacy (redirects to WebRTC)

### Backend Files
- `internal/servers/webrtc/http_server.go` - Routing
- `internal/servers/webrtc/ptz_handler.go` - PTZ API handlers
- `internal/ptz/hikvision.go` - Hikvision PTZ library

## Quick Start

1. **Start MediaMTX**
```bash
./mediamtx.exe
```

2. **Access Dashboards**
- WebRTC: http://localhost:8889/dashboard
- HLS: http://localhost:8889/dashboard-hls
- PTZ: http://localhost:8889/ptz

3. **Verify Streams**
```bash
curl http://localhost:9997/v3/paths/list
```

4. **Control PTZ Camera**
- Open PTZ page
- Select camera from dropdown
- Use directional pad to control
- Click presets to move to saved positions

## Support

For issues or questions:
1. Check MediaMTX logs
2. Verify configuration in `mediamtx.yml`
3. Test API endpoints directly
4. Check browser console for JavaScript errors
