#!/bin/bash
# Hikvision PTZ Control Test Script

CAMERA_IP="192.168.10.53"
USERNAME="admin"
PASSWORD="live0416"

echo "=========================================="
echo "Hikvision PTZ Control Test"
echo "Camera: $CAMERA_IP"
echo "=========================================="

# Test 1: Get PTZ Capabilities
echo ""
echo "Test 1: Get PTZ Capabilities"
echo "------------------------------------------"
curl -s --digest -u "$USERNAME:$PASSWORD" \
  "http://$CAMERA_IP/ISAPI/PTZCtrl/channels/1/capabilities" \
  -w "\nHTTP Status: %{http_code}\n"

# Test 2: Get PTZ Status
echo ""
echo "Test 2: Get PTZ Status"
echo "------------------------------------------"
curl -s --digest -u "$USERNAME:$PASSWORD" \
  "http://$CAMERA_IP/ISAPI/PTZCtrl/channels/1/status" \
  -w "\nHTTP Status: %{http_code}\n"

# Test 3: PTZ Move UP
echo ""
echo "Test 3: PTZ Move UP (for 2 seconds)"
echo "------------------------------------------"
curl -s --digest -u "$USERNAME:$PASSWORD" \
  -X PUT \
  -H "Content-Type: application/xml" \
  -d '<?xml version="1.0" encoding="UTF-8"?>
<PTZData>
    <pan>0</pan>
    <tilt>50</tilt>
</PTZData>' \
  "http://$CAMERA_IP/ISAPI/PTZCtrl/channels/1/continuous" \
  -w "\nHTTP Status: %{http_code}\n"

sleep 2

# Test 4: PTZ STOP
echo ""
echo "Test 4: PTZ STOP"
echo "------------------------------------------"
curl -s --digest -u "$USERNAME:$PASSWORD" \
  -X PUT \
  -H "Content-Type: application/xml" \
  -d '<?xml version="1.0" encoding="UTF-8"?>
<PTZData>
    <pan>0</pan>
    <tilt>0</tilt>
</PTZData>' \
  "http://$CAMERA_IP/ISAPI/PTZCtrl/channels/1/continuous" \
  -w "\nHTTP Status: %{http_code}\n"

# Test 5: PTZ Move RIGHT
echo ""
echo "Test 5: PTZ Move RIGHT (for 2 seconds)"
echo "------------------------------------------"
curl -s --digest -u "$USERNAME:$PASSWORD" \
  -X PUT \
  -H "Content-Type: application/xml" \
  -d '<?xml version="1.0" encoding="UTF-8"?>
<PTZData>
    <pan>50</pan>
    <tilt>0</tilt>
</PTZData>' \
  "http://$CAMERA_IP/ISAPI/PTZCtrl/channels/1/continuous" \
  -w "\nHTTP Status: %{http_code}\n"

sleep 2

curl -s --digest -u "$USERNAME:$PASSWORD" \
  -X PUT \
  -H "Content-Type: application/xml" \
  -d '<?xml version="1.0" encoding="UTF-8"?>
<PTZData>
    <pan>0</pan>
    <tilt>0</tilt>
</PTZData>' \
  "http://$CAMERA_IP/ISAPI/PTZCtrl/channels/1/continuous" > /dev/null

# Test 6: Zoom IN
echo ""
echo "Test 6: Zoom IN (for 2 seconds)"
echo "------------------------------------------"
curl -s --digest -u "$USERNAME:$PASSWORD" \
  -X PUT \
  -H "Content-Type: application/xml" \
  -d '<?xml version="1.0" encoding="UTF-8"?>
<PTZData>
    <zoom>50</zoom>
</PTZData>' \
  "http://$CAMERA_IP/ISAPI/PTZCtrl/channels/1/continuous" \
  -w "\nHTTP Status: %{http_code}\n"

sleep 2

curl -s --digest -u "$USERNAME:$PASSWORD" \
  -X PUT \
  -H "Content-Type: application/xml" \
  -d '<?xml version="1.0" encoding="UTF-8"?>
<PTZData>
    <zoom>0</zoom>
</PTZData>' \
  "http://$CAMERA_IP/ISAPI/PTZCtrl/channels/1/continuous" > /dev/null

# Test 7: Get Presets
echo ""
echo "Test 7: Get Presets"
echo "------------------------------------------"
curl -s --digest -u "$USERNAME:$PASSWORD" \
  "http://$CAMERA_IP/ISAPI/PTZCtrl/channels/1/presets" \
  -w "\nHTTP Status: %{http_code}\n"

echo ""
echo "=========================================="
echo "Test completed!"
echo "=========================================="
