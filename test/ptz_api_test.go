package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"
)

const (
	baseURL    = "http://localhost:9997/v3/ptz"
	testCamera = "CCTV-TEST-001"
)

type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

type PTZStatus struct {
	Pan  float64 `json:"pan"`
	Tilt float64 `json:"tilt"`
	Zoom float64 `json:"zoom"`
}

type PTZPreset struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// shouldSkipCameraTests 카메라 테스트를 Skip 할지 결정
// 환경 변수 SKIP_CAMERA_TESTS=true 이면 Skip
func shouldSkipCameraTests() bool {
	return os.Getenv("SKIP_CAMERA_TESTS") == "true"
}

func TestPTZAPI_GetCameras(t *testing.T) {
	resp, err := http.Get(baseURL + "/cameras")
	if err != nil {
		t.Fatalf("Failed to get cameras: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", resp.StatusCode)
	}

	var apiResp struct {
		Success bool     `json:"success"`
		Data    []string `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if !apiResp.Success {
		t.Error("API returned success=false")
	}

	if len(apiResp.Data) == 0 {
		t.Error("No cameras returned")
	}

	t.Logf("Found %d cameras: %v", len(apiResp.Data), apiResp.Data)

	// Check if test camera exists
	found := false
	for _, cam := range apiResp.Data {
		if cam == testCamera {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Test camera %s not found in list", testCamera)
	}
}

func TestPTZAPI_Move(t *testing.T) {
	if shouldSkipCameraTests() {
		t.Skip("Skipping move test - requires ONVIF enabled camera (set SKIP_CAMERA_TESTS=false to run)")
	}

	moveData := map[string]int{
		"pan":  30,
		"tilt": 20,
		"zoom": 0,
	}

	jsonData, _ := json.Marshal(moveData)
	url := fmt.Sprintf("%s/%s/move", baseURL, testCamera)

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		t.Fatalf("Failed to send move command: %v", err)
	}
	defer resp.Body.Close()

	var apiResp APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if !apiResp.Success {
		t.Errorf("Move command failed: %s", apiResp.Message)
	} else {
		t.Log("Move command succeeded")
	}
}

func TestPTZAPI_Stop(t *testing.T) {
	if shouldSkipCameraTests() {
		t.Skip("Skipping stop test - requires ONVIF enabled camera (set SKIP_CAMERA_TESTS=false to run)")
	}

	url := fmt.Sprintf("%s/%s/stop", baseURL, testCamera)
	resp, err := http.Post(url, "application/json", nil)
	if err != nil {
		t.Fatalf("Failed to send stop command: %v", err)
	}
	defer resp.Body.Close()

	var apiResp APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if !apiResp.Success {
		t.Errorf("Stop command failed: %s", apiResp.Message)
	} else {
		t.Log("Stop command succeeded")
	}
}

func TestPTZAPI_GetStatus(t *testing.T) {
	if shouldSkipCameraTests() {
		t.Skip("Skipping status test - requires ONVIF enabled camera (set SKIP_CAMERA_TESTS=false to run)")
	}

	url := fmt.Sprintf("%s/%s/status", baseURL, testCamera)
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("Failed to get status: %v", err)
	}
	defer resp.Body.Close()

	var apiResp struct {
		Success bool      `json:"success"`
		Message string    `json:"message,omitempty"`
		Data    PTZStatus `json:"data,omitempty"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if !apiResp.Success {
		t.Errorf("Get status failed: %s", apiResp.Message)
	} else {
		t.Logf("PTZ Status: Pan=%.2f, Tilt=%.2f, Zoom=%.2f",
			apiResp.Data.Pan,
			apiResp.Data.Tilt,
			apiResp.Data.Zoom)
	}
}

func TestPTZAPI_GetPresets(t *testing.T) {
	if shouldSkipCameraTests() {
		t.Skip("Skipping presets test - requires ONVIF enabled camera (set SKIP_CAMERA_TESTS=false to run)")
	}

	url := fmt.Sprintf("%s/%s/presets", baseURL, testCamera)
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("Failed to get presets: %v", err)
	}
	defer resp.Body.Close()

	var apiResp struct {
		Success bool        `json:"success"`
		Message string      `json:"message,omitempty"`
		Data    []PTZPreset `json:"data,omitempty"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if !apiResp.Success {
		t.Errorf("Get presets failed: %s", apiResp.Message)
	} else {
		t.Logf("Found %d presets", len(apiResp.Data))
		for _, preset := range apiResp.Data {
			t.Logf("  Preset %d: %s", preset.ID, preset.Name)
		}
	}
}

func TestPTZAPI_SetPreset(t *testing.T) {
	if shouldSkipCameraTests() {
		t.Skip("Skipping set preset test - requires ONVIF enabled camera (set SKIP_CAMERA_TESTS=false to run)")
	}

	presetData := map[string]string{
		"name": "TestPresetAPI",
	}

	jsonData, _ := json.Marshal(presetData)
	url := fmt.Sprintf("%s/%s/presets/99", baseURL, testCamera)

	client := &http.Client{}
	req, _ := http.NewRequest(http.MethodPut, url, bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to set preset: %v", err)
	}
	defer resp.Body.Close()

	var apiResp APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if !apiResp.Success {
		t.Errorf("Set preset failed: %s", apiResp.Message)
	} else {
		t.Log("Set preset succeeded")
	}
}

func TestPTZAPI_GotoPreset(t *testing.T) {
	if shouldSkipCameraTests() {
		t.Skip("Skipping goto preset test - requires ONVIF enabled camera (set SKIP_CAMERA_TESTS=false to run)")
	}

	url := fmt.Sprintf("%s/%s/presets/1", baseURL, testCamera)
	resp, err := http.Post(url, "application/json", nil)
	if err != nil {
		t.Fatalf("Failed to goto preset: %v", err)
	}
	defer resp.Body.Close()

	var apiResp APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if !apiResp.Success {
		t.Errorf("Goto preset failed: %s", apiResp.Message)
	} else {
		t.Log("Goto preset succeeded")
	}
}

func TestPTZAPI_DeletePreset(t *testing.T) {
	if shouldSkipCameraTests() {
		t.Skip("Skipping delete preset test - requires ONVIF enabled camera (set SKIP_CAMERA_TESTS=false to run)")
	}

	url := fmt.Sprintf("%s/%s/presets/99", baseURL, testCamera)

	client := &http.Client{}
	req, _ := http.NewRequest(http.MethodDelete, url, nil)

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to delete preset: %v", err)
	}
	defer resp.Body.Close()

	var apiResp APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if !apiResp.Success {
		t.Errorf("Delete preset failed: %s", apiResp.Message)
	} else {
		t.Log("Delete preset succeeded")
	}
}

func TestPTZAPI_Focus(t *testing.T) {
	if shouldSkipCameraTests() {
		t.Skip("Skipping focus test - requires ONVIF enabled camera and Imaging service (set SKIP_CAMERA_TESTS=false to run)")
	}

	focusData := map[string]int{
		"speed": 50,
	}

	jsonData, _ := json.Marshal(focusData)
	url := fmt.Sprintf("%s/%s/focus", baseURL, testCamera)

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		t.Fatalf("Failed to send focus command: %v", err)
	}
	defer resp.Body.Close()

	var apiResp APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Focus should return "not implemented" error
	if apiResp.Success {
		t.Error("Focus should return not implemented error")
	} else {
		t.Logf("Focus correctly returned error: %s", apiResp.Message)
	}
}

func TestPTZAPI_Iris(t *testing.T) {
	if shouldSkipCameraTests() {
		t.Skip("Skipping iris test - requires ONVIF enabled camera and Imaging service (set SKIP_CAMERA_TESTS=false to run)")
	}

	irisData := map[string]int{
		"speed": 30,
	}

	jsonData, _ := json.Marshal(irisData)
	url := fmt.Sprintf("%s/%s/iris", baseURL, testCamera)

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		t.Fatalf("Failed to send iris command: %v", err)
	}
	defer resp.Body.Close()

	var apiResp APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Iris should return "not implemented" error
	if apiResp.Success {
		t.Error("Iris should return not implemented error")
	} else {
		t.Logf("Iris correctly returned error: %s", apiResp.Message)
	}
}

func TestPTZAPI_CompleteWorkflow(t *testing.T) {
	if shouldSkipCameraTests() {
		t.Skip("Skipping complete workflow test - requires ONVIF enabled camera (set SKIP_CAMERA_TESTS=false to run)")
	}

	// 1. Get cameras
	t.Log("Step 1: Getting camera list")
	resp, err := http.Get(baseURL + "/cameras")
	if err != nil {
		t.Fatalf("Failed to get cameras: %v", err)
	}
	resp.Body.Close()

	// 2. Move camera
	t.Log("Step 2: Moving camera")
	moveData := map[string]int{"pan": 20, "tilt": 15, "zoom": 0}
	jsonData, _ := json.Marshal(moveData)
	url := fmt.Sprintf("%s/%s/move", baseURL, testCamera)
	http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	time.Sleep(2 * time.Second)

	// 3. Stop camera
	t.Log("Step 3: Stopping camera")
	url = fmt.Sprintf("%s/%s/stop", baseURL, testCamera)
	http.Post(url, "application/json", nil)
	time.Sleep(1 * time.Second)

	// 4. Get status
	t.Log("Step 4: Getting PTZ status")
	url = fmt.Sprintf("%s/%s/status", baseURL, testCamera)
	resp, _ = http.Get(url)
	var statusResp struct {
		Success bool      `json:"success"`
		Data    PTZStatus `json:"data"`
	}
	json.NewDecoder(resp.Body).Decode(&statusResp)
	resp.Body.Close()

	if statusResp.Success {
		t.Logf("Current position: Pan=%.2f, Tilt=%.2f, Zoom=%.2f",
			statusResp.Data.Pan,
			statusResp.Data.Tilt,
			statusResp.Data.Zoom)
	}

	t.Log("Workflow test completed")
}

func TestPTZAPI_ErrorHandling(t *testing.T) {
	// Test invalid camera
	t.Run("InvalidCamera", func(t *testing.T) {
		url := fmt.Sprintf("%s/INVALID-CAMERA/status", baseURL)
		resp, err := http.Get(url)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			var apiResp APIResponse
			json.NewDecoder(resp.Body).Decode(&apiResp)
			if apiResp.Success {
				t.Error("Expected failure for invalid camera")
			} else {
				t.Logf("Correctly returned error: %s", apiResp.Message)
			}
		}
	})

	// Test invalid preset ID
	t.Run("InvalidPresetID", func(t *testing.T) {
		t.Skip("Skipping - requires ONVIF enabled camera")

		url := fmt.Sprintf("%s/%s/presets/99999", baseURL, testCamera)
		resp, err := http.Post(url, "application/json", nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		var apiResp APIResponse
		json.NewDecoder(resp.Body).Decode(&apiResp)
		if !apiResp.Success {
			t.Logf("Correctly returned error for invalid preset: %s", apiResp.Message)
		}
	})

	// Test malformed JSON
	t.Run("MalformedJSON", func(t *testing.T) {
		url := fmt.Sprintf("%s/%s/move", baseURL, testCamera)
		resp, err := http.Post(url, "application/json", bytes.NewBufferString("{invalid json"))
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Logf("Correctly rejected malformed JSON with status %d", resp.StatusCode)
		}
	})
}
