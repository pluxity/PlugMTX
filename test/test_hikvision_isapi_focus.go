package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"time"
)

func main() {
	host := "14.51.233.129"
	port := 10081
	username := "admin"
	password := "pluxity123!@#"

	fmt.Printf("=== Hikvision ISAPI Focus/Iris 테스트 ===\n\n")

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Test 1: Get Imaging Capabilities
	fmt.Println("--- Test 1: Get Imaging Capabilities ---")
	capURL := fmt.Sprintf("http://%s:%d/ISAPI/Image/channels/1/capabilities", host, port)
	req, _ := http.NewRequest("GET", capURL, nil)
	req.SetBasicAuth(username, password)

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("❌ 요청 실패: %v\n", err)
	} else {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		fmt.Printf("✅ 응답 상태: %s\n", resp.Status)
		fmt.Printf("응답 내용:\n%s\n\n", string(body))
	}

	// Test 2: Get Current Focus Settings
	fmt.Println("--- Test 2: Get Current Focus Settings ---")
	focusURL := fmt.Sprintf("http://%s:%d/ISAPI/Image/channels/1/focus", host, port)
	req, _ = http.NewRequest("GET", focusURL, nil)
	req.SetBasicAuth(username, password)

	resp, err = client.Do(req)
	if err != nil {
		fmt.Printf("❌ 요청 실패: %v\n", err)
	} else {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		fmt.Printf("✅ 응답 상태: %s\n", resp.Status)
		fmt.Printf("응답 내용:\n%s\n\n", string(body))
	}

	// Test 3: Continuous Focus Near
	fmt.Println("--- Test 3: Continuous Focus Near (근거리) ---")
	focusCmdURL := fmt.Sprintf("http://%s:%d/ISAPI/System/Video/inputs/channels/1/focus", host, port)

	// PUT 방식으로 포커스 제어 시도
	focusXML := `<?xml version="1.0" encoding="UTF-8"?>
<FocusData>
    <focus>near</focus>
</FocusData>`

	req, _ = http.NewRequest("PUT", focusCmdURL, bytes.NewBuffer([]byte(focusXML)))
	req.SetBasicAuth(username, password)
	req.Header.Set("Content-Type", "application/xml")

	resp, err = client.Do(req)
	if err != nil {
		fmt.Printf("❌ 요청 실패: %v\n", err)
	} else {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		fmt.Printf("✅ 응답 상태: %s\n", resp.Status)
		fmt.Printf("응답 내용:\n%s\n\n", string(body))
	}

	// Test 4: Stop Focus
	time.Sleep(1 * time.Second)
	fmt.Println("--- Test 4: Stop Focus ---")
	focusStopXML := `<?xml version="1.0" encoding="UTF-8"?>
<FocusData>
    <focus>stop</focus>
</FocusData>`

	req, _ = http.NewRequest("PUT", focusCmdURL, bytes.NewBuffer([]byte(focusStopXML)))
	req.SetBasicAuth(username, password)
	req.Header.Set("Content-Type", "application/xml")

	resp, err = client.Do(req)
	if err != nil {
		fmt.Printf("❌ 요청 실패: %v\n", err)
	} else {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		fmt.Printf("✅ 응답 상태: %s\n", resp.Status)
		fmt.Printf("응답 내용:\n%s\n\n", string(body))
	}

	// Test 5: Get Iris Settings
	fmt.Println("--- Test 5: Get Current Iris Settings ---")
	irisURL := fmt.Sprintf("http://%s:%d/ISAPI/Image/channels/1/iris", host, port)
	req, _ = http.NewRequest("GET", irisURL, nil)
	req.SetBasicAuth(username, password)

	resp, err = client.Do(req)
	if err != nil {
		fmt.Printf("❌ 요청 실패: %v\n", err)
	} else {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		fmt.Printf("✅ 응답 상태: %s\n", resp.Status)
		fmt.Printf("응답 내용:\n%s\n\n", string(body))
	}

	// Test 6: Iris Open
	fmt.Println("--- Test 6: Iris Open (조리개 열기) ---")
	irisCmdURL := fmt.Sprintf("http://%s:%d/ISAPI/System/Video/inputs/channels/1/iris", host, port)

	irisOpenXML := `<?xml version="1.0" encoding="UTF-8"?>
<IrisData>
    <iris>open</iris>
</IrisData>`

	req, _ = http.NewRequest("PUT", irisCmdURL, bytes.NewBuffer([]byte(irisOpenXML)))
	req.SetBasicAuth(username, password)
	req.Header.Set("Content-Type", "application/xml")

	resp, err = client.Do(req)
	if err != nil {
		fmt.Printf("❌ 요청 실패: %v\n", err)
	} else {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		fmt.Printf("✅ 응답 상태: %s\n", resp.Status)
		fmt.Printf("응답 내용:\n%s\n\n", string(body))
	}

	// Test 7: Stop Iris
	time.Sleep(1 * time.Second)
	fmt.Println("--- Test 7: Stop Iris ---")
	irisStopXML := `<?xml version="1.0" encoding="UTF-8"?>
<IrisData>
    <iris>stop</iris>
</IrisData>`

	req, _ = http.NewRequest("PUT", irisCmdURL, bytes.NewBuffer([]byte(irisStopXML)))
	req.SetBasicAuth(username, password)
	req.Header.Set("Content-Type", "application/xml")

	resp, err = client.Do(req)
	if err != nil {
		fmt.Printf("❌ 요청 실패: %v\n", err)
	} else {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		fmt.Printf("✅ 응답 상태: %s\n", resp.Status)
		fmt.Printf("응답 내용:\n%s\n\n", string(body))
	}

	fmt.Println("=== 테스트 완료 ===")
}
