package main

import (
	"encoding/xml"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/use-go/onvif"
	"github.com/use-go/onvif/device"
	"github.com/use-go/onvif/media"
	onvif_ptz "github.com/use-go/onvif/ptz"
	xsd_onvif "github.com/use-go/onvif/xsd/onvif"
)

func main() {
	host := "14.51.233.129"
	port := 10081
	username := "admin"
	password := "pluxity123!@#"

	fmt.Printf("=== ONVIF 명령 지원 테스트 ===\n\n")

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
	profileToken := xsd_onvif.ReferenceToken(envelope.Body.GetProfilesResponse.Profiles[0].Token)
	fmt.Printf("프로필 토큰: %s\n\n", profileToken)

	// Get initial status
	fmt.Println("=== 초기 상태 ===")
	initialStatus := getStatus(dev, profileToken)

	// Test 1: RelativeMove - Small Pan
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("테스트 1: RelativeMove (Pan +0.1)")
	fmt.Println(strings.Repeat("=", 60))

	relReq1 := onvif_ptz.RelativeMove{
		ProfileToken: profileToken,
		Translation: xsd_onvif.PTZVector{
			PanTilt: xsd_onvif.Vector2D{
				X: 0.1, // Pan +0.1 (right)
				Y: 0.0,
			},
			Zoom: xsd_onvif.Vector1D{
				X: 0.0,
			},
		},
	}

	relResp1, err := dev.CallMethod(relReq1)
	if err != nil {
		fmt.Printf("❌ 실패: %v\n", err)
	} else {
		relBody1, _ := io.ReadAll(relResp1.Body)
		relResp1.Body.Close()
		fmt.Printf("✅ 응답 코드: %d (%s)\n", relResp1.StatusCode, relResp1.Status)

		if relResp1.StatusCode != 200 {
			fmt.Printf("에러 응답: %s\n", string(relBody1))
		}
	}

	time.Sleep(2 * time.Second)
	fmt.Println("2초 후 상태:")
	status1 := getStatus(dev, profileToken)
	fmt.Printf("변화량: Pan %+.4f\n", status1.Pan-initialStatus.Pan)

	// Test 2: RelativeMove - Larger Pan
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("테스트 2: RelativeMove (Pan +0.3)")
	fmt.Println(strings.Repeat("=", 60))

	relReq2 := onvif_ptz.RelativeMove{
		ProfileToken: profileToken,
		Translation: xsd_onvif.PTZVector{
			PanTilt: xsd_onvif.Vector2D{
				X: 0.3, // Pan +0.3 (right)
				Y: 0.0,
			},
			Zoom: xsd_onvif.Vector1D{
				X: 0.0,
			},
		},
	}

	relResp2, err := dev.CallMethod(relReq2)
	if err != nil {
		fmt.Printf("❌ 실패: %v\n", err)
	} else {
		relBody2, _ := io.ReadAll(relResp2.Body)
		relResp2.Body.Close()
		fmt.Printf("✅ 응답 코드: %d (%s)\n", relResp2.StatusCode, relResp2.Status)

		if relResp2.StatusCode != 200 {
			fmt.Printf("에러 응답: %s\n", string(relBody2))
		}
	}

	time.Sleep(3 * time.Second)
	fmt.Println("3초 후 상태:")
	status2 := getStatus(dev, profileToken)
	fmt.Printf("변화량: Pan %+.4f (초기 대비 %+.4f)\n", status2.Pan-status1.Pan, status2.Pan-initialStatus.Pan)

	// Test 3: RelativeMove - Negative Pan (go back)
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("테스트 3: RelativeMove (Pan -0.4, 원위치)")
	fmt.Println(strings.Repeat("=", 60))

	relReq3 := onvif_ptz.RelativeMove{
		ProfileToken: profileToken,
		Translation: xsd_onvif.PTZVector{
			PanTilt: xsd_onvif.Vector2D{
				X: -0.4, // Pan -0.4 (left)
				Y: 0.0,
			},
			Zoom: xsd_onvif.Vector1D{
				X: 0.0,
			},
		},
	}

	relResp3, err := dev.CallMethod(relReq3)
	if err != nil {
		fmt.Printf("❌ 실패: %v\n", err)
	} else {
		relBody3, _ := io.ReadAll(relResp3.Body)
		relResp3.Body.Close()
		fmt.Printf("✅ 응답 코드: %d (%s)\n", relResp3.StatusCode, relResp3.Status)

		if relResp3.StatusCode != 200 {
			fmt.Printf("에러 응답: %s\n", string(relBody3))
		}
	}

	time.Sleep(3 * time.Second)
	fmt.Println("3초 후 상태:")
	status3 := getStatus(dev, profileToken)
	fmt.Printf("변화량: Pan %+.4f (초기 대비 %+.4f)\n", status3.Pan-status2.Pan, status3.Pan-initialStatus.Pan)

	// Test 4: RelativeMove - Tilt
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("테스트 4: RelativeMove (Tilt +0.2)")
	fmt.Println(strings.Repeat("=", 60))

	relReq4 := onvif_ptz.RelativeMove{
		ProfileToken: profileToken,
		Translation: xsd_onvif.PTZVector{
			PanTilt: xsd_onvif.Vector2D{
				X: 0.0,
				Y: 0.2, // Tilt +0.2 (up)
			},
			Zoom: xsd_onvif.Vector1D{
				X: 0.0,
			},
		},
	}

	relResp4, err := dev.CallMethod(relReq4)
	if err != nil {
		fmt.Printf("❌ 실패: %v\n", err)
	} else {
		relBody4, _ := io.ReadAll(relResp4.Body)
		relResp4.Body.Close()
		fmt.Printf("✅ 응답 코드: %d (%s)\n", relResp4.StatusCode, relResp4.Status)

		if relResp4.StatusCode != 200 {
			fmt.Printf("에러 응답: %s\n", string(relBody4))
		}
	}

	time.Sleep(3 * time.Second)
	fmt.Println("3초 후 상태:")
	status4 := getStatus(dev, profileToken)
	fmt.Printf("변화량: Tilt %+.4f\n", status4.Tilt-status3.Tilt)

	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("테스트 완료!")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("\n카메라가 실제로 움직였나요?")
	fmt.Println("움직였다면 RelativeMove를 지원합니다!")
	fmt.Println("안 움직였다면 다른 방법을 시도해야 합니다.")
}

type PTZStatus struct {
	Pan  float64
	Tilt float64
	Zoom float64
}

func getStatus(dev *onvif.Device, profileToken xsd_onvif.ReferenceToken) PTZStatus {
	statusReq := onvif_ptz.GetStatus{
		ProfileToken: profileToken,
	}

	statusResp, err := dev.CallMethod(statusReq)
	if err != nil {
		fmt.Printf("❌ GetStatus 실패: %v\n", err)
		return PTZStatus{}
	}

	body, _ := io.ReadAll(statusResp.Body)
	statusResp.Body.Close()

	var envelope struct {
		Body struct {
			GetStatusResponse struct {
				PTZStatus struct {
					Position struct {
						PanTilt struct {
							X float64 `xml:"x,attr"`
							Y float64 `xml:"y,attr"`
						} `xml:"PanTilt"`
						Zoom struct {
							X float64 `xml:"x,attr"`
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

	xml.Unmarshal(body, &envelope)
	status := envelope.Body.GetStatusResponse.PTZStatus

	fmt.Printf("  Pan: %7.4f, Tilt: %7.4f, Zoom: %7.4f | 상태: %s\n",
		status.Position.PanTilt.X,
		status.Position.PanTilt.Y,
		status.Position.Zoom.X,
		status.MoveStatus.PanTilt)

	return PTZStatus{
		Pan:  status.Position.PanTilt.X,
		Tilt: status.Position.PanTilt.Y,
		Zoom: status.Position.Zoom.X,
	}
}
