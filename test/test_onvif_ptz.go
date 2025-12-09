package main

import (
	"encoding/xml"
	"fmt"
	"io"

	"github.com/use-go/onvif"
	"github.com/use-go/onvif/device"
	"github.com/use-go/onvif/media"
	onvif_ptz "github.com/use-go/onvif/ptz"
	"github.com/use-go/onvif/xsd"
	xsd_onvif "github.com/use-go/onvif/xsd/onvif"
)

func main() {
	host := "14.51.233.129"
	port := 10081
	username := "admin"
	password := "pluxity123!@#"

	fmt.Printf("=== ONVIF PTZ 테스트 ===\n\n")

	// Create ONVIF device
	fmt.Printf("1. ONVIF 장치 생성: %s:%d\n", host, port)
	dev, err := onvif.NewDevice(onvif.DeviceParams{
		Xaddr:    fmt.Sprintf("%s:%d", host, port),
		Username: username,
		Password: password,
	})
	if err != nil {
		fmt.Printf("❌ ONVIF 장치 생성 실패: %v\n", err)
		return
	}
	fmt.Printf("✅ ONVIF 장치 생성 성공\n\n")

	// Get device information
	fmt.Println("2. 장치 정보 조회")
	getInfoReq := device.GetDeviceInformation{}
	_, err = dev.CallMethod(getInfoReq)
	if err != nil {
		fmt.Printf("❌ 장치 정보 조회 실패: %v\n", err)
		return
	}
	fmt.Printf("✅ 장치 정보 조회 성공\n\n")

	// Get media profiles
	fmt.Println("3. 미디어 프로필 조회")
	getProfilesReq := media.GetProfiles{}
	profilesResp, err := dev.CallMethod(getProfilesReq)
	if err != nil {
		fmt.Printf("❌ 프로필 조회 실패: %v\n", err)
		return
	}

	body, _ := io.ReadAll(profilesResp.Body)
	profilesResp.Body.Close()

	var envelope struct {
		Body struct {
			GetProfilesResponse struct {
				Profiles []struct {
					Token string `xml:"token,attr"`
					Name  string
				}
			}
		}
	}

	xml.Unmarshal(body, &envelope)
	if len(envelope.Body.GetProfilesResponse.Profiles) == 0 {
		fmt.Printf("❌ 프로필이 없습니다\n")
		return
	}

	profileToken := xsd_onvif.ReferenceToken(envelope.Body.GetProfilesResponse.Profiles[0].Token)
	fmt.Printf("✅ 프로필 발견: %s\n\n", profileToken)

	// Test 1: ContinuousMove WITHOUT Timeout
	fmt.Println("4. ContinuousMove WITHOUT Timeout (pan=0.5)")
	req1 := onvif_ptz.ContinuousMove{
		ProfileToken: profileToken,
		Velocity: xsd_onvif.PTZSpeed{
			PanTilt: xsd_onvif.Vector2D{
				X: 0.5,
				Y: 0.0,
			},
			Zoom: xsd_onvif.Vector1D{
				X: 0.0,
			},
		},
	}

	resp1, err := dev.CallMethod(req1)
	if err != nil {
		fmt.Printf("❌ ContinuousMove (no timeout) 실패: %v\n", err)
	} else {
		body1, _ := io.ReadAll(resp1.Body)
		resp1.Body.Close()
		fmt.Printf("✅ 응답 상태: %s\n", resp1.Status)
		fmt.Printf("응답 본문 길이: %d bytes\n", len(body1))
		if len(body1) < 500 {
			fmt.Printf("응답: %s\n", string(body1))
		}
	}
	fmt.Println()

	// Test 2: ContinuousMove WITH Timeout (PT1S)
	fmt.Println("5. ContinuousMove WITH Timeout PT1S (pan=0.5)")
	timeout1s := xsd.Duration("PT1S")
	req2 := onvif_ptz.ContinuousMove{
		ProfileToken: profileToken,
		Velocity: xsd_onvif.PTZSpeed{
			PanTilt: xsd_onvif.Vector2D{
				X: 0.5,
				Y: 0.0,
			},
			Zoom: xsd_onvif.Vector1D{
				X: 0.0,
			},
		},
		Timeout: timeout1s,
	}

	resp2, err := dev.CallMethod(req2)
	if err != nil {
		fmt.Printf("❌ ContinuousMove (PT1S) 실패: %v\n", err)
	} else {
		body2, _ := io.ReadAll(resp2.Body)
		resp2.Body.Close()
		fmt.Printf("✅ 응답 상태: %s\n", resp2.Status)
		fmt.Printf("응답 본문 길이: %d bytes\n", len(body2))
		if len(body2) < 500 {
			fmt.Printf("응답: %s\n", string(body2))
		}
	}
	fmt.Println()

	// Test 3: ContinuousMove WITH Timeout (PT5S)
	fmt.Println("6. ContinuousMove WITH Timeout PT5S (pan=0.5)")
	timeout5s := xsd.Duration("PT5S")
	req3 := onvif_ptz.ContinuousMove{
		ProfileToken: profileToken,
		Velocity: xsd_onvif.PTZSpeed{
			PanTilt: xsd_onvif.Vector2D{
				X: 0.5,
				Y: 0.0,
			},
			Zoom: xsd_onvif.Vector1D{
				X: 0.0,
			},
		},
		Timeout: timeout5s,
	}

	resp3, err := dev.CallMethod(req3)
	if err != nil {
		fmt.Printf("❌ ContinuousMove (PT5S) 실패: %v\n", err)
	} else {
		body3, _ := io.ReadAll(resp3.Body)
		resp3.Body.Close()
		fmt.Printf("✅ 응답 상태: %s\n", resp3.Status)
		fmt.Printf("응답 본문 길이: %d bytes\n", len(body3))
		if len(body3) < 500 {
			fmt.Printf("응답: %s\n", string(body3))
		}
	}
	fmt.Println()

	// Test 4: Stop
	fmt.Println("7. Stop PTZ")
	stopReq := onvif_ptz.Stop{
		ProfileToken: profileToken,
		PanTilt:      xsd.Boolean(true),
		Zoom:         xsd.Boolean(true),
	}

	stopResp, err := dev.CallMethod(stopReq)
	if err != nil {
		fmt.Printf("❌ Stop 실패: %v\n", err)
	} else {
		stopBody, _ := io.ReadAll(stopResp.Body)
		stopResp.Body.Close()
		fmt.Printf("✅ 응답 상태: %s\n", stopResp.Status)
		fmt.Printf("응답 본문 길이: %d bytes\n", len(stopBody))
	}

	fmt.Println("\n=== 테스트 완료 ===")
	fmt.Println("카메라가 움직였는지 확인해주세요!")
}
