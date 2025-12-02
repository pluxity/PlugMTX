#!/usr/bin/env python3
"""
Hikvision PTZ Test Script
Tests both ONVIF and ISAPI methods for PTZ control
"""

import requests
from requests.auth import HTTPDigestAuth
import xml.etree.ElementTree as ET

# Camera credentials
CAMERA_IP = "192.168.10.53"
USERNAME = "admin"
PASSWORD = "live0416"

# Test cameras
CAMERAS = [
    {"name": "CCTV-TEST1", "ip": "192.168.10.53"},
    {"name": "CCTV-TEST2", "ip": "192.168.10.54"},
    {"name": "CCTV-TEST3", "ip": "192.168.10.55"},
]

def test_hikvision_isapi_ptz(camera_ip, username, password):
    """Test Hikvision ISAPI PTZ control"""
    print(f"\n=== Testing Hikvision ISAPI PTZ for {camera_ip} ===")

    # Test 1: Get PTZ capabilities
    try:
        url = f"http://{camera_ip}/ISAPI/PTZCtrl/channels/1/capabilities"
        response = requests.get(url, auth=HTTPDigestAuth(username, password), timeout=5)
        print(f"✓ PTZ Capabilities: {response.status_code}")
        if response.status_code == 200:
            print(f"  Response length: {len(response.text)} bytes")
    except Exception as e:
        print(f"✗ PTZ Capabilities failed: {e}")

    # Test 2: Get PTZ status
    try:
        url = f"http://{camera_ip}/ISAPI/PTZCtrl/channels/1/status"
        response = requests.get(url, auth=HTTPDigestAuth(username, password), timeout=5)
        print(f"✓ PTZ Status: {response.status_code}")
        if response.status_code == 200:
            print(f"  Response: {response.text[:200]}")
    except Exception as e:
        print(f"✗ PTZ Status failed: {e}")

    # Test 3: Continuous move (UP)
    try:
        url = f"http://{camera_ip}/ISAPI/PTZCtrl/channels/1/continuous"

        # Pan/Tilt up command
        ptz_data = """<?xml version="1.0" encoding="UTF-8"?>
<PTZData>
    <pan>0</pan>
    <tilt>50</tilt>
</PTZData>"""

        response = requests.put(
            url,
            data=ptz_data,
            auth=HTTPDigestAuth(username, password),
            headers={'Content-Type': 'application/xml'},
            timeout=5
        )
        print(f"✓ PTZ Move UP: {response.status_code}")

        # Stop command
        ptz_stop = """<?xml version="1.0" encoding="UTF-8"?>
<PTZData>
    <pan>0</pan>
    <tilt>0</tilt>
</PTZData>"""

        import time
        time.sleep(1)  # Move for 1 second

        response = requests.put(
            url,
            data=ptz_stop,
            auth=HTTPDigestAuth(username, password),
            headers={'Content-Type': 'application/xml'},
            timeout=5
        )
        print(f"✓ PTZ STOP: {response.status_code}")

    except Exception as e:
        print(f"✗ PTZ Move failed: {e}")

def test_hikvision_isapi_preset(camera_ip, username, password):
    """Test Hikvision ISAPI Preset control"""
    print(f"\n=== Testing Hikvision ISAPI Presets for {camera_ip} ===")

    # Get presets
    try:
        url = f"http://{camera_ip}/ISAPI/PTZCtrl/channels/1/presets"
        response = requests.get(url, auth=HTTPDigestAuth(username, password), timeout=5)
        print(f"✓ Get Presets: {response.status_code}")
        if response.status_code == 200:
            print(f"  Available presets: {response.text[:500]}")
    except Exception as e:
        print(f"✗ Get Presets failed: {e}")

def test_onvif_ptz(camera_ip, username, password):
    """Test ONVIF PTZ control (basic check)"""
    print(f"\n=== Testing ONVIF for {camera_ip} ===")

    # Check if ONVIF service is available
    try:
        url = f"http://{camera_ip}/onvif/device_service"

        # GetCapabilities SOAP request
        soap_request = """<?xml version="1.0" encoding="UTF-8"?>
<s:Envelope xmlns:s="http://www.w3.org/2003/05/soap-envelope">
    <s:Body xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:xsd="http://www.w3.org/2001/XMLSchema">
        <GetCapabilities xmlns="http://www.onvif.org/ver10/device/wsdl">
        </GetCapabilities>
    </s:Body>
</s:Envelope>"""

        response = requests.post(
            url,
            data=soap_request,
            auth=HTTPDigestAuth(username, password),
            headers={'Content-Type': 'application/soap+xml'},
            timeout=5
        )
        print(f"✓ ONVIF Device Service: {response.status_code}")
        if response.status_code == 200:
            print(f"  ONVIF is available")
    except Exception as e:
        print(f"✗ ONVIF check failed: {e}")

def main():
    print("=" * 60)
    print("Hikvision PTZ Control Test")
    print("=" * 60)

    for camera in CAMERAS:
        print(f"\n{'=' * 60}")
        print(f"Testing: {camera['name']} ({camera['ip']})")
        print(f"{'=' * 60}")

        # Test ISAPI methods
        test_hikvision_isapi_ptz(camera['ip'], USERNAME, PASSWORD)
        test_hikvision_isapi_preset(camera['ip'], USERNAME, PASSWORD)

        # Test ONVIF
        test_onvif_ptz(camera['ip'], USERNAME, PASSWORD)

    print("\n" + "=" * 60)
    print("Test completed!")
    print("=" * 60)

if __name__ == "__main__":
    main()
