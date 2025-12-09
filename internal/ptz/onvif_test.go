package ptz

import (
	"fmt"
	"os"
	"testing"
	"time"
)

// 테스트용 카메라 설정
// 환경 변수로 실제 카메라 정보 제공 가능
func getTestCamera() (host string, port int, username, password string, skip bool) {
	host = os.Getenv("TEST_CAMERA_HOST")
	username = os.Getenv("TEST_CAMERA_USER")
	password = os.Getenv("TEST_CAMERA_PASS")
	portStr := os.Getenv("TEST_CAMERA_PORT")

	if host == "" || username == "" || password == "" {
		return "", 0, "", "", true
	}

	port = 80 // 기본 ONVIF 포트
	if portStr != "" {
		var err error
		_, err = fmt.Sscanf(portStr, "%d", &port)
		if err != nil {
			port = 80
		}
	}
	return host, port, username, password, false
}

func TestOnvifPTZ_Connect(t *testing.T) {
	host, port, username, password, skip := getTestCamera()
	if skip {
		t.Skip("Skipping test: No camera configured. Set TEST_CAMERA_HOST, TEST_CAMERA_USER, TEST_CAMERA_PASS")
	}

	ptz := NewOnvifPTZ(host, port, username, password)
	if ptz == nil {
		t.Fatal("NewOnvifPTZ returned nil")
	}

	err := ptz.Connect()
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	// 연결 후 device와 profileToken이 설정되었는지 확인
	if ptz.device == nil {
		t.Error("device is nil after Connect")
	}

	if ptz.profileToken == "" {
		t.Error("profileToken is empty after Connect")
	}

	t.Logf("Successfully connected to camera at %s:%d", host, port)
	t.Logf("Profile Token: %s", ptz.profileToken)
}

func TestOnvifPTZ_Move(t *testing.T) {
	host, port, username, password, skip := getTestCamera()
	if skip {
		t.Skip("Skipping test: No camera configured")
	}

	ptz := NewOnvifPTZ(host, port, username, password)
	err := ptz.Connect()
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	// 오른쪽으로 이동 테스트
	t.Log("Testing pan right (speed: 30)")
	err = ptz.Move(30, 0, 0)
	if err != nil {
		t.Errorf("Move(pan right) failed: %v", err)
	}

	time.Sleep(2 * time.Second)

	// 정지
	t.Log("Stopping movement")
	err = ptz.Stop()
	if err != nil {
		t.Errorf("Stop failed: %v", err)
	}

	time.Sleep(1 * time.Second)

	// 위로 이동 테스트
	t.Log("Testing tilt up (speed: 30)")
	err = ptz.Move(0, 30, 0)
	if err != nil {
		t.Errorf("Move(tilt up) failed: %v", err)
	}

	time.Sleep(2 * time.Second)

	err = ptz.Stop()
	if err != nil {
		t.Errorf("Stop failed: %v", err)
	}

	time.Sleep(1 * time.Second)

	// 줌 인 테스트
	t.Log("Testing zoom in (speed: 30)")
	err = ptz.Move(0, 0, 30)
	if err != nil {
		t.Errorf("Move(zoom in) failed: %v", err)
	}

	time.Sleep(2 * time.Second)

	err = ptz.Stop()
	if err != nil {
		t.Errorf("Stop failed: %v", err)
	}

	t.Log("Move tests completed successfully")
}

func TestOnvifPTZ_GetStatus(t *testing.T) {
	host, port, username, password, skip := getTestCamera()
	if skip {
		t.Skip("Skipping test: No camera configured")
	}

	ptz := NewOnvifPTZ(host, port, username, password)
	err := ptz.Connect()
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	status, err := ptz.GetStatus()
	if err != nil {
		t.Fatalf("GetStatus failed: %v", err)
	}

	if status == nil {
		t.Fatal("GetStatus returned nil status")
	}

	t.Logf("Current PTZ Status:")
	t.Logf("  Pan: %.2f", status.Pan)
	t.Logf("  Tilt: %.2f", status.Tilt)
	t.Logf("  Zoom: %.2f", status.Zoom)
}

func TestOnvifPTZ_Presets(t *testing.T) {
	host, port, username, password, skip := getTestCamera()
	if skip {
		t.Skip("Skipping test: No camera configured")
	}

	ptz := NewOnvifPTZ(host, port, username, password)
	err := ptz.Connect()
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	// 1. 기존 프리셋 목록 조회
	t.Log("Getting initial preset list")
	presets, err := ptz.GetPresets()
	if err != nil {
		t.Fatalf("GetPresets failed: %v", err)
	}

	t.Logf("Found %d existing presets", len(presets))
	for _, preset := range presets {
		t.Logf("  Preset %d: %s", preset.ID, preset.Name)
	}

	// 2. 테스트 프리셋 생성
	testPresetID := 99
	testPresetName := "GoTestPreset"

	t.Logf("Creating test preset %d with name '%s'", testPresetID, testPresetName)
	err = ptz.SetPreset(testPresetID, testPresetName)
	if err != nil {
		t.Errorf("SetPreset failed: %v", err)
	} else {
		t.Log("Preset created successfully")
	}

	time.Sleep(1 * time.Second)

	// 3. 프리셋이 생성되었는지 확인
	t.Log("Verifying preset was created")
	presets, err = ptz.GetPresets()
	if err != nil {
		t.Errorf("GetPresets failed: %v", err)
	} else {
		found := false
		for _, preset := range presets {
			if preset.ID == testPresetID {
				found = true
				t.Logf("Found test preset: ID=%d, Name=%s", preset.ID, preset.Name)
				break
			}
		}
		if !found {
			t.Errorf("Test preset %d was not found in preset list", testPresetID)
		}
	}

	// 4. 다른 위치로 이동
	t.Log("Moving camera to different position")
	err = ptz.Move(20, 20, 0)
	if err != nil {
		t.Errorf("Move failed: %v", err)
	}
	time.Sleep(2 * time.Second)
	ptz.Stop()
	time.Sleep(1 * time.Second)

	// 5. 프리셋으로 복귀
	t.Logf("Going to preset %d", testPresetID)
	err = ptz.GotoPreset(testPresetID)
	if err != nil {
		t.Errorf("GotoPreset failed: %v", err)
	} else {
		t.Log("Successfully moved to preset")
	}

	time.Sleep(3 * time.Second)

	// 6. 테스트 프리셋 삭제
	t.Logf("Deleting test preset %d", testPresetID)
	err = ptz.DeletePreset(testPresetID)
	if err != nil {
		t.Errorf("DeletePreset failed: %v", err)
	} else {
		t.Log("Preset deleted successfully")
	}

	time.Sleep(1 * time.Second)

	// 7. 프리셋이 삭제되었는지 확인
	t.Log("Verifying preset was deleted")
	presets, err = ptz.GetPresets()
	if err != nil {
		t.Errorf("GetPresets failed: %v", err)
	} else {
		for _, preset := range presets {
			if preset.ID == testPresetID {
				t.Errorf("Test preset %d still exists after deletion", testPresetID)
				break
			}
		}
	}

	t.Log("Preset tests completed successfully")
}

func TestOnvifPTZ_Focus(t *testing.T) {
	host, port, username, password, skip := getTestCamera()
	if skip {
		t.Skip("Skipping test: No camera configured")
	}

	ptz := NewOnvifPTZ(host, port, username, password)
	err := ptz.Connect()
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	// Focus는 현재 미구현 - 에러 확인
	err = ptz.Focus(50)
	if err == nil {
		t.Error("Expected Focus to return error (not implemented), but got nil")
	} else {
		t.Logf("Focus correctly returned error: %v", err)
	}
}

func TestOnvifPTZ_Iris(t *testing.T) {
	host, port, username, password, skip := getTestCamera()
	if skip {
		t.Skip("Skipping test: No camera configured")
	}

	ptz := NewOnvifPTZ(host, port, username, password)
	err := ptz.Connect()
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	// Iris는 현재 미구현 - 에러 확인
	err = ptz.Iris(50)
	if err == nil {
		t.Error("Expected Iris to return error (not implemented), but got nil")
	} else {
		t.Logf("Iris correctly returned error: %v", err)
	}
}

func TestOnvifPTZ_GetImageSettings(t *testing.T) {
	host, port, username, password, skip := getTestCamera()
	if skip {
		t.Skip("Skipping test: No camera configured")
	}

	ptz := NewOnvifPTZ(host, port, username, password)
	err := ptz.Connect()
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	imageSettings, err := ptz.GetImageSettings()
	if err != nil {
		t.Errorf("GetImageSettings failed: %v", err)
		return
	}

	if imageSettings == nil {
		t.Fatal("GetImageSettings returned nil")
	}

	t.Logf("Image Settings:")
	t.Logf("  Brightness: %d", imageSettings.Brightness)
	t.Logf("  Contrast: %d", imageSettings.Contrast)
	t.Logf("  Saturation: %d", imageSettings.Saturation)
	t.Logf("  Sharpness: %d", imageSettings.Sharpness)
}

func TestOnvifPTZ_EnsureConnected(t *testing.T) {
	host, port, username, password, skip := getTestCamera()
	if skip {
		t.Skip("Skipping test: No camera configured")
	}

	ptz := NewOnvifPTZ(host, port, username, password)

	// 명시적으로 Connect 호출하지 않고 ensureConnected 테스트
	err := ptz.ensureConnected()
	if err != nil {
		t.Fatalf("ensureConnected failed: %v", err)
	}

	if ptz.device == nil {
		t.Error("device is nil after ensureConnected")
	}

	// 두 번째 호출은 이미 연결되어 있어야 함
	err = ptz.ensureConnected()
	if err != nil {
		t.Errorf("ensureConnected (second call) failed: %v", err)
	}
}

func TestOnvifPTZ_MultipleOperations(t *testing.T) {
	host, port, username, password, skip := getTestCamera()
	if skip {
		t.Skip("Skipping test: No camera configured")
	}

	ptz := NewOnvifPTZ(host, port, username, password)
	err := ptz.Connect()
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	// 복합 동작 테스트: Pan + Tilt + Zoom 동시에
	t.Log("Testing combined movement (pan + tilt + zoom)")
	err = ptz.Move(20, 15, 10)
	if err != nil {
		t.Errorf("Combined Move failed: %v", err)
	}

	time.Sleep(2 * time.Second)

	// 상태 조회
	status, err := ptz.GetStatus()
	if err != nil {
		t.Errorf("GetStatus failed: %v", err)
	} else {
		t.Logf("Position during movement: Pan=%.2f, Tilt=%.2f, Zoom=%.2f",
			status.Pan, status.Tilt, status.Zoom)
	}

	// 정지
	err = ptz.Stop()
	if err != nil {
		t.Errorf("Stop failed: %v", err)
	}

	time.Sleep(1 * time.Second)

	// 정지 후 상태 조회
	status, err = ptz.GetStatus()
	if err != nil {
		t.Errorf("GetStatus after stop failed: %v", err)
	} else {
		t.Logf("Position after stop: Pan=%.2f, Tilt=%.2f, Zoom=%.2f",
			status.Pan, status.Tilt, status.Zoom)
	}

	t.Log("Multiple operations test completed successfully")
}
