package main

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

func main() {
	baseURL := "http://localhost:9997/v3/ptz/CCTV-TEST-002"

	fmt.Println("=== ONVIF Focus 제어 테스트 ===\n")

	// Test 1: Focus Near (근거리)
	fmt.Println("테스트 1: Focus Near (근거리, speed: -50)")
	testFocus(baseURL, -50)
	time.Sleep(3 * time.Second)

	// Test 2: Stop
	fmt.Println("\n테스트 2: Stop (speed: 0)")
	testFocus(baseURL, 0)
	time.Sleep(1 * time.Second)

	// Test 3: Focus Far (원거리)
	fmt.Println("\n테스트 3: Focus Far (원거리, speed: 50)")
	testFocus(baseURL, 50)
	time.Sleep(3 * time.Second)

	// Test 4: Stop
	fmt.Println("\n테스트 4: Stop (speed: 0)")
	testFocus(baseURL, 0)

	fmt.Println("\n=== 테스트 완료 ===")
}

func testFocus(baseURL string, speed int) {
	url := baseURL + "/focus"
	body := fmt.Sprintf(`{"speed": %d}`, speed)

	req, err := http.NewRequest("POST", url, strings.NewReader(body))
	if err != nil {
		fmt.Printf("❌ 요청 생성 실패: %v\n", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("❌ 요청 실패: %v\n", err)
		return
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == 200 {
		fmt.Printf("✅ 성공 (코드: %d)\n", resp.StatusCode)
		fmt.Printf("   응답: %s\n", string(respBody))
	} else {
		fmt.Printf("❌ 실패 (코드: %d)\n", resp.StatusCode)
		fmt.Printf("   응답: %s\n", string(respBody))
	}
}
