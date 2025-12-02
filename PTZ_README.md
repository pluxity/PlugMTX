# PTZ Control Implementation for MediaMTX

## Overview
PTZ (Pan-Tilt-Zoom) control has been successfully implemented for Hikvision cameras in MediaMTX.

## Supported Cameras
- CCTV-TEST1 (192.168.10.53)
- CCTV-TEST2 (192.168.10.54)
- CCTV-TEST3 (192.168.10.55)

## Features Implemented

### 1. Backend API (Go)
Located in: `internal/ptz/hikvision.go` and `internal/servers/webrtc/ptz_handler.go`

#### API Endpoints
All endpoints are available at: `http://localhost:8889/ptz/`

- **GET /ptz/cameras** - List all PTZ-enabled cameras
- **POST /ptz/:camera/move** - Move camera
  ```json
  {
    "pan": 0,     // -100 to 100 (left/right)
    "tilt": 0,    // -100 to 100 (down/up)
    "zoom": 0     // -100 to 100 (out/in)
  }
  ```
- **POST /ptz/:camera/stop** - Stop camera movement
- **GET /ptz/:camera/status** - Get current camera status
- **GET /ptz/:camera/presets** - List available presets
- **POST /ptz/:camera/preset/:presetId** - Go to specific preset

### 2. Dashboard UI
Located in: `internal/servers/webrtc/dashboard.html`

#### PTZ Controls
Each PTZ camera displays:
- **Directional Pad**: 8-way control (Up, Down, Left, Right, and center Home button)
- **Zoom Controls**: Zoom In / Zoom Out buttons
- **Touch Support**: Mobile-friendly touch controls

#### Control Behavior
- Press and hold buttons to move camera
- Release button to stop movement
- Home button (⌂) returns camera to preset position 34 (Back to origin)

## Testing

### Test PTZ API Directly

1. **List PTZ Cameras**
```bash
curl http://localhost:8889/ptz/cameras
```

2. **Move Camera Up**
```bash
curl -X POST http://localhost:8889/ptz/CCTV-TEST1/move \
  -H "Content-Type: application/json" \
  -d '{"pan":0,"tilt":40,"zoom":0}'
```

3. **Stop Camera**
```bash
curl -X POST http://localhost:8889/ptz/CCTV-TEST1/stop
```

4. **Get Camera Status**
```bash
curl http://localhost:8889/ptz/CCTV-TEST1/status
```

5. **Get Presets**
```bash
curl http://localhost:8889/ptz/CCTV-TEST1/presets
```

6. **Go to Preset**
```bash
curl -X POST http://localhost:8889/ptz/CCTV-TEST1/preset/34
```

### Test via Dashboard

1. Start MediaMTX:
```bash
./mediamtx.exe
```

2. Open browser:
```
http://localhost:8889/dashboard
```

3. Find CCTV-TEST1, CCTV-TEST2, or CCTV-TEST3
4. Use the PTZ controls below each camera stream

## Technical Details

### Hikvision ISAPI
The implementation uses Hikvision's ISAPI (Internet Server Application Programming Interface):

- **Continuous Movement**: `/ISAPI/PTZCtrl/channels/1/continuous`
- **Status**: `/ISAPI/PTZCtrl/channels/1/status`
- **Presets**: `/ISAPI/PTZCtrl/channels/1/presets`

### Authentication
- Uses HTTP Digest Authentication
- Credentials configured in `ptz_handler.go`

### Movement Parameters
- **Pan/Tilt**: -100 (left/down) to 100 (right/up)
- **Zoom**: -100 (zoom out) to 100 (zoom in)
- **Speed**: Default 40 (configurable in dashboard.html)

## Files Modified/Created

### New Files
1. `internal/ptz/hikvision.go` - PTZ control library
2. `internal/servers/webrtc/ptz_handler.go` - API handlers
3. `test_ptz.py` - Python test script
4. `test_ptz.sh` - Bash test script

### Modified Files
1. `internal/servers/webrtc/http_server.go` - Added PTZ routes
2. `internal/servers/webrtc/dashboard.html` - Added PTZ UI controls

## Available Presets (Hikvision Default)
- Preset 1: Custom position
- Preset 33: Auto-flip
- Preset 34: Back to origin
- Preset 35-38: Call patrol 1-4
- Preset 39: Day mode
- Preset 40: Night mode
- Preset 41-44: Call pattern 1-4

## Future Enhancements
- [ ] Add preset management UI
- [ ] Support for more camera brands (ONVIF)
- [ ] Configurable PTZ settings via config file
- [ ] Pattern/tour support
- [ ] Speed control slider
- [ ] Keyboard shortcuts for PTZ control
