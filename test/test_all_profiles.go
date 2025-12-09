package main

import (
	"encoding/xml"
	"fmt"
	"io"
	"time"

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

	fmt.Printf("=== ONVIF 모든 프로필 PTZ 테스트 ===\n\n")

	// Create ONVIF device
	dev, err := onvif.NewDevice(onvif.DeviceParams{
		Xaddr:    fmt.Sprintf("%s:%d", host, port),
		Username: username,
		Password: password,
	})
	if err != nil {
		fmt.Printf("❌ ONVIF 장치 생성 실패: %v\n", err)
		return
	}

	// Get device information
	getInfoReq := device.GetDeviceInformation{}
	_, err = dev.CallMethod(getInfoReq)
	if err != nil {
		fmt.Printf("❌ 장치 정보 조회 실패: %v\n", err)
		return
	}

	// Get media profiles
	fmt.Println("=== 미디어 프로필 조회 ===")
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
					PTZConfiguration struct {
						Token string `xml:"token,attr"`
						Name  string
					} `xml:"PTZConfiguration"`
				}
			}
		}
	}

	xml.Unmarshal(body, &envelope)
	profiles := envelope.Body.GetProfilesResponse.Profiles

	fmt.Printf("\n총 %d개의 프로필 발견:\n", len(profiles))
	for i, profile := range profiles {
		fmt.Printf("  [%d] Token: %s, Name: %s, PTZ Config: %s\n",
			i+1, profile.Token, profile.Name, profile.PTZConfiguration.Token)
	}
	fmt.Println()

	// Test each profile
	for i, profile := range profiles {
		fmt.Printf("\n\n=====================================\n")
		fmt.Printf("프로필 [%d/%d] 테스트: %s\n", i+1, len(profiles), profile.Name)
		fmt.Printf("=====================================\n\n")

		profileToken := xsd_onvif.ReferenceToken(profile.Token)
		testProfile(dev, profileToken, profile.Name)

		if i < len(profiles)-1 {
			fmt.Println("\n다음 프로필 테스트 전 3초 대기...")
			time.Sleep(3 * time.Second)
		}
	}

	fmt.Println("\n\n=== 모든 프로필 테스트 완료 ===")
}

func testProfile(dev *onvif.Device, profileToken xsd_onvif.ReferenceToken, profileName string) {
	// Get initial status
	fmt.Println("=== 초기 상태 조회 ===")
	getStatus(dev, profileToken)

	// Test 1: ContinuousMove without timeout
	fmt.Println("\n=== 테스트 1: ContinuousMove (Timeout 없음, Pan=0.5) ===")
	contReq1 := onvif_ptz.ContinuousMove{
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
	resp1, err1 := dev.CallMethod(contReq1)
	if err1 != nil {
		fmt.Printf("❌ 실패: %v\n", err1)
	} else {
		body1, _ := io.ReadAll(resp1.Body)
		resp1.Body.Close()
		fmt.Printf("✅ 응답: %s (코드: %d)\n", resp1.Status, resp1.StatusCode)
		if resp1.StatusCode != 200 && len(body1) < 1000 {
			fmt.Printf("에러 본문: %s\n", string(body1))
		}
	}

	time.Sleep(1 * time.Second)
	getStatus(dev, profileToken)

	// Stop
	stopReq := onvif_ptz.Stop{
		ProfileToken: profileToken,
		PanTilt:      xsd.Boolean(true),
		Zoom:         xsd.Boolean(true),
	}
	dev.CallMethod(stopReq)

	// Test 2: ContinuousMove with timeout
	fmt.Println("\n=== 테스트 2: ContinuousMove (Timeout PT2S, Pan=0.5) ===")
	timeout := xsd.Duration("PT2S")
	contReq2 := onvif_ptz.ContinuousMove{
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
		Timeout: timeout,
	}
	resp2, err2 := dev.CallMethod(contReq2)
	if err2 != nil {
		fmt.Printf("❌ 실패: %v\n", err2)
	} else {
		body2, _ := io.ReadAll(resp2.Body)
		resp2.Body.Close()
		fmt.Printf("✅ 응답: %s (코드: %d)\n", resp2.Status, resp2.StatusCode)
		if resp2.StatusCode != 200 && len(body2) < 1000 {
			fmt.Printf("에러 본문: %s\n", string(body2))
		}
	}

	time.Sleep(1 * time.Second)
	getStatus(dev, profileToken)

	time.Sleep(2 * time.Second)
	fmt.Println("\n=== Timeout 후 상태 ===")
	getStatus(dev, profileToken)

	// Test 3: RelativeMove
	fmt.Println("\n=== 테스트 3: RelativeMove (Pan +0.1) ===")
	relReq := onvif_ptz.RelativeMove{
		ProfileToken: profileToken,
		Translation: xsd_onvif.PTZVector{
			PanTilt: xsd_onvif.Vector2D{
				X: 0.1,
				Y: 0.0,
			},
			Zoom: xsd_onvif.Vector1D{
				X: 0.0,
			},
		},
	}
	resp3, err3 := dev.CallMethod(relReq)
	if err3 != nil {
		fmt.Printf("❌ 실패: %v\n", err3)
	} else {
		body3, _ := io.ReadAll(resp3.Body)
		resp3.Body.Close()
		fmt.Printf("✅ 응답: %s (코드: %d)\n", resp3.Status, resp3.StatusCode)
		if resp3.StatusCode != 200 && len(body3) < 1000 {
			fmt.Printf("에러 본문: %s\n", string(body3))
		}
	}

	time.Sleep(2 * time.Second)
	getStatus(dev, profileToken)

	// Test 4: AbsoluteMove
	fmt.Println("\n=== 테스트 4: AbsoluteMove (Pan=0.0, Tilt=0.0) ===")
	absReq := onvif_ptz.AbsoluteMove{
		ProfileToken: profileToken,
		Position: xsd_onvif.PTZVector{
			PanTilt: xsd_onvif.Vector2D{
				X: 0.0,
				Y: 0.0,
			},
			Zoom: xsd_onvif.Vector1D{
				X: 0.0,
			},
		},
	}
	resp4, err4 := dev.CallMethod(absReq)
	if err4 != nil {
		fmt.Printf("❌ 실패: %v\n", err4)
	} else {
		body4, _ := io.ReadAll(resp4.Body)
		resp4.Body.Close()
		fmt.Printf("✅ 응답: %s (코드: %d)\n", resp4.Status, resp4.StatusCode)
		if resp4.StatusCode != 200 && len(body4) < 1000 {
			fmt.Printf("에러 본문: %s\n", string(body4))
		}
	}

	time.Sleep(2 * time.Second)
	getStatus(dev, profileToken)
}

func getStatus(dev *onvif.Device, profileToken xsd_onvif.ReferenceToken) {
	statusReq := onvif_ptz.GetStatus{
		ProfileToken: profileToken,
	}

	statusResp, err := dev.CallMethod(statusReq)
	if err != nil {
		fmt.Printf("❌ GetStatus 실패: %v\n", err)
		return
	}

	body, _ := io.ReadAll(statusResp.Body)
	statusResp.Body.Close()

	// Parse status
	var envelope struct {
		Body struct {
			GetStatusResponse struct {
				PTZStatus struct {
					Position struct {
						PanTilt struct {
							X     float64 `xml:"x,attr"`
							Y     float64 `xml:"y,attr"`
							Space string  `xml:"space,attr"`
						} `xml:"PanTilt"`
						Zoom struct {
							X     float64 `xml:"x,attr"`
							Space string  `xml:"space,attr"`
						} `xml:"Zoom"`
					} `xml:"Position"`
					MoveStatus struct {
						PanTilt string `xml:"PanTilt"`
						Zoom    string `xml:"Zoom"`
					} `xml:"MoveStatus"`
				} `xml:"PTZStatus"`
			} `xml:"GetStatusResponse"`
		} `xml:"Body"`
	}

	if err := xml.Unmarshal(body, &envelope); err != nil {
		fmt.Printf("❌ 상태 파싱 실패: %v\n", err)
		return
	}

	status := envelope.Body.GetStatusResponse.PTZStatus
	fmt.Printf("📍 위치 - Pan: %.4f, Tilt: %.4f, Zoom: %.4f | 상태 - PanTilt: %s, Zoom: %s\n",
		status.Position.PanTilt.X,
		status.Position.PanTilt.Y,
		status.Position.Zoom.X,
		status.MoveStatus.PanTilt,
		status.MoveStatus.Zoom)
}
