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
	xsd_onvif "github.com/use-go/onvif/xsd/onvif"
)

func main() {
	host := "14.51.233.129"
	port := 10081
	username := "admin"
	password := "pluxity123!@#"

	fmt.Printf("=== ONVIF RelativeMove 테스트 ===\n\n")

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
	fmt.Println("=== 초기 상태 조회 ===")
	getStatus(dev, profileToken)

	// Test 1: RelativeMove - Small movement to the right
	fmt.Println("\n=== RelativeMove: Pan +0.1 (작은 우측 이동) ===")
	relMoveReq1 := onvif_ptz.RelativeMove{
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

	relMoveResp1, err := dev.CallMethod(relMoveReq1)
	if err != nil {
		fmt.Printf("❌ RelativeMove 실패: %v\n", err)
	} else {
		relBody1, _ := io.ReadAll(relMoveResp1.Body)
		relMoveResp1.Body.Close()
		fmt.Printf("✅ RelativeMove 응답: %s\n", relMoveResp1.Status)
		if len(relBody1) < 500 {
			fmt.Printf("응답 본문: %s\n", string(relBody1))
		}
	}

	// Check status after 2 seconds
	time.Sleep(2 * time.Second)
	fmt.Println("\n=== 2초 후 상태 조회 ===")
	getStatus(dev, profileToken)

	// Test 2: RelativeMove - Small movement to the left
	fmt.Println("\n=== RelativeMove: Pan -0.1 (작은 좌측 이동) ===")
	relMoveReq2 := onvif_ptz.RelativeMove{
		ProfileToken: profileToken,
		Translation: xsd_onvif.PTZVector{
			PanTilt: xsd_onvif.Vector2D{
				X: -0.1, // Pan -0.1 (left)
				Y: 0.0,
			},
			Zoom: xsd_onvif.Vector1D{
				X: 0.0,
			},
		},
	}

	relMoveResp2, err := dev.CallMethod(relMoveReq2)
	if err != nil {
		fmt.Printf("❌ RelativeMove 실패: %v\n", err)
	} else {
		relBody2, _ := io.ReadAll(relMoveResp2.Body)
		relMoveResp2.Body.Close()
		fmt.Printf("✅ RelativeMove 응답: %s\n", relMoveResp2.Status)
		if len(relBody2) < 500 {
			fmt.Printf("응답 본문: %s\n", string(relBody2))
		}
	}

	// Check status after 2 seconds
	time.Sleep(2 * time.Second)
	fmt.Println("\n=== 2초 후 상태 조회 ===")
	getStatus(dev, profileToken)

	// Test 3: RelativeMove - Medium movement up
	fmt.Println("\n=== RelativeMove: Tilt +0.2 (위로 이동) ===")
	relMoveReq3 := onvif_ptz.RelativeMove{
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

	relMoveResp3, err := dev.CallMethod(relMoveReq3)
	if err != nil {
		fmt.Printf("❌ RelativeMove 실패: %v\n", err)
	} else {
		relBody3, _ := io.ReadAll(relMoveResp3.Body)
		relMoveResp3.Body.Close()
		fmt.Printf("✅ RelativeMove 응답: %s\n", relMoveResp3.Status)
		if len(relBody3) < 500 {
			fmt.Printf("응답 본문: %s\n", string(relBody3))
		}
	}

	// Check status after 2 seconds
	time.Sleep(2 * time.Second)
	fmt.Println("\n=== 2초 후 상태 조회 ===")
	getStatus(dev, profileToken)

	fmt.Println("\n=== 테스트 완료 ===")
	fmt.Println("카메라가 우측 -> 좌측 -> 위 순서로 움직였는지 확인하세요!")
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
		fmt.Printf("응답 본문: %s\n", string(body))
		return
	}

	status := envelope.Body.GetStatusResponse.PTZStatus
	fmt.Printf("위치 - Pan: %.4f, Tilt: %.4f, Zoom: %.4f\n",
		status.Position.PanTilt.X,
		status.Position.PanTilt.Y,
		status.Position.Zoom.X)
	fmt.Printf("상태 - PanTilt: %s, Zoom: %s\n",
		status.MoveStatus.PanTilt,
		status.MoveStatus.Zoom)
}
