package main

import (
	"fmt"
	"io"
	"time"

	"github.com/use-go/onvif"
	onvif_ptz "github.com/use-go/onvif/ptz"
	xsd_onvif "github.com/use-go/onvif/xsd/onvif"
)

func main() {
	host := "14.51.233.129"
	port := 10081
	username := "admin"
	password := "pluxity123!@#"

	fmt.Printf("=== ONVIF ContinuousMove 테스트 ===\n\n")

	// ONVIF 장치 연결
	dev, err := onvif.NewDevice(onvif.DeviceParams{
		Xaddr:    fmt.Sprintf("%s:%d", host, port),
		Username: username,
		Password: password,
	})
	if err != nil {
		fmt.Printf("❌ 연결 실패: %v\n", err)
		return
	}

	fmt.Println("✅ ONVIF 장치 연결 성공")

	profileToken := xsd_onvif.ReferenceToken("Profile_1")

	// Test 1: Pan 우측으로 이동 (0.5 velocity)
	fmt.Println("\n--- Test 1: Pan 우측으로 이동 (velocity 0.5) ---")
	req1 := onvif_ptz.ContinuousMove{
		ProfileToken: profileToken,
		Velocity: xsd_onvif.PTZSpeed{
			PanTilt: xsd_onvif.Vector2D{
				X: 0.5, // 우측
				Y: 0.0,
			},
			Zoom: xsd_onvif.Vector1D{
				X: 0.0,
			},
		},
	}

	resp1, err := dev.CallMethod(req1)
	if err != nil {
		fmt.Printf("❌ ContinuousMove 실패: %v\n", err)
	} else {
		body, _ := io.ReadAll(resp1.Body)
		resp1.Body.Close()
		fmt.Printf("✅ 응답 상태: %s\n", resp1.Status)
		fmt.Printf("응답 내용:\n%s\n", string(body))
	}

	// 2초 대기 후 정지
	fmt.Println("\n2초 대기...")
	time.Sleep(2 * time.Second)

	// Stop
	fmt.Println("\n--- Stop 명령 전송 ---")
	stopReq := onvif_ptz.Stop{
		ProfileToken: profileToken,
		PanTilt:      true,
		Zoom:         true,
	}

	stopResp, err := dev.CallMethod(stopReq)
	if err != nil {
		fmt.Printf("❌ Stop 실패: %v\n", err)
	} else {
		stopBody, _ := io.ReadAll(stopResp.Body)
		stopResp.Body.Close()
		fmt.Printf("✅ Stop 응답: %s\n", stopResp.Status)
		fmt.Printf("응답 내용:\n%s\n", string(stopBody))
	}

	fmt.Println("\n=== 테스트 완료 ===")
}
