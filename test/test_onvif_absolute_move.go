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

	fmt.Printf("=== ONVIF AbsoluteMove 테스트 ===\n\n")

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

	// Test AbsoluteMove - Move to Pan=0.0, Tilt=0.0 (Home Position)
	fmt.Println("\n=== AbsoluteMove: Pan=0.0, Tilt=0.0 (Home Position) ===")
	absMoveReq := onvif_ptz.AbsoluteMove{
		ProfileToken: profileToken,
		Position: xsd_onvif.PTZVector{
			PanTilt: xsd_onvif.Vector2D{
				X: 0.0, // Pan 0.0
				Y: 0.0, // Tilt 0.0
			},
			Zoom: xsd_onvif.Vector1D{
				X: 0.0,
			},
		},
	}

	absMoveResp, err := dev.CallMethod(absMoveReq)
	if err != nil {
		fmt.Printf("❌ AbsoluteMove 실패: %v\n", err)
	} else {
		absBody, _ := io.ReadAll(absMoveResp.Body)
		absMoveResp.Body.Close()
		fmt.Printf("✅ AbsoluteMove 응답: %s\n", absMoveResp.Status)
		if len(absBody) < 500 {
			fmt.Printf("응답 본문: %s\n", string(absBody))
		}
	}

	// Check status after 2 seconds
	time.Sleep(2 * time.Second)
	fmt.Println("\n=== 2초 후 상태 조회 ===")
	getStatus(dev, profileToken)

	// Move to Pan=0.5 (right)
	fmt.Println("\n=== AbsoluteMove: Pan=0.5 (Right), Tilt=0.0 ===")
	absMoveReq2 := onvif_ptz.AbsoluteMove{
		ProfileToken: profileToken,
		Position: xsd_onvif.PTZVector{
			PanTilt: xsd_onvif.Vector2D{
				X: 0.5, // Pan 0.5 (right)
				Y: 0.0,
			},
			Zoom: xsd_onvif.Vector1D{
				X: 0.0,
			},
		},
	}

	absMoveResp2, err := dev.CallMethod(absMoveReq2)
	if err != nil {
		fmt.Printf("❌ AbsoluteMove 실패: %v\n", err)
	} else {
		absBody2, _ := io.ReadAll(absMoveResp2.Body)
		absMoveResp2.Body.Close()
		fmt.Printf("✅ AbsoluteMove 응답: %s\n", absMoveResp2.Status)
		if len(absBody2) < 500 {
			fmt.Printf("응답 본문: %s\n", string(absBody2))
		}
	}

	// Check status after 2 seconds
	time.Sleep(2 * time.Second)
	fmt.Println("\n=== 2초 후 상태 조회 ===")
	getStatus(dev, profileToken)

	// Move to Pan=-0.5 (left)
	fmt.Println("\n=== AbsoluteMove: Pan=-0.5 (Left), Tilt=0.0 ===")
	absMoveReq3 := onvif_ptz.AbsoluteMove{
		ProfileToken: profileToken,
		Position: xsd_onvif.PTZVector{
			PanTilt: xsd_onvif.Vector2D{
				X: -0.5, // Pan -0.5 (left)
				Y: 0.0,
			},
			Zoom: xsd_onvif.Vector1D{
				X: 0.0,
			},
		},
	}

	absMoveResp3, err := dev.CallMethod(absMoveReq3)
	if err != nil {
		fmt.Printf("❌ AbsoluteMove 실패: %v\n", err)
	} else {
		absBody3, _ := io.ReadAll(absMoveResp3.Body)
		absMoveResp3.Body.Close()
		fmt.Printf("✅ AbsoluteMove 응답: %s\n", absMoveResp3.Status)
		if len(absBody3) < 500 {
			fmt.Printf("응답 본문: %s\n", string(absBody3))
		}
	}

	// Check status after 2 seconds
	time.Sleep(2 * time.Second)
	fmt.Println("\n=== 2초 후 상태 조회 ===")
	getStatus(dev, profileToken)

	fmt.Println("\n=== 테스트 완료 ===")
	fmt.Println("카메라가 Home(0,0) -> Right(0.5,0) -> Left(-0.5,0) 순서로 움직였는지 확인하세요!")
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
