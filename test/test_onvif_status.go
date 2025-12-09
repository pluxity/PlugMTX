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

	fmt.Printf("=== ONVIF PTZ Status 테스트 ===\n\n")

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

	// Send ContinuousMove
	fmt.Println("\n=== ContinuousMove 전송 (pan=1.0, 5초) ===")
	timeout := xsd.Duration("PT5S")
	moveReq := onvif_ptz.ContinuousMove{
		ProfileToken: profileToken,
		Velocity: xsd_onvif.PTZSpeed{
			PanTilt: xsd_onvif.Vector2D{
				X: 1.0,  // 최대 속도로 우측
				Y: 0.0,
			},
			Zoom: xsd_onvif.Vector1D{
				X: 0.0,
			},
		},
		Timeout: timeout,
	}

	moveResp, err := dev.CallMethod(moveReq)
	if err != nil {
		fmt.Printf("❌ ContinuousMove 실패: %v\n", err)
	} else {
		moveBody, _ := io.ReadAll(moveResp.Body)
		moveResp.Body.Close()
		fmt.Printf("✅ ContinuousMove 응답: %s\n", moveResp.Status)
		if len(moveBody) < 500 {
			fmt.Printf("응답 본문: %s\n", string(moveBody))
		}
	}

	// Check status after 1 second
	time.Sleep(1 * time.Second)
	fmt.Println("\n=== 1초 후 상태 조회 ===")
	getStatus(dev, profileToken)

	// Check status after 3 seconds
	time.Sleep(2 * time.Second)
	fmt.Println("\n=== 3초 후 상태 조회 ===")
	getStatus(dev, profileToken)

	// Wait for timeout
	time.Sleep(3 * time.Second)
	fmt.Println("\n=== 6초 후 상태 조회 (타임아웃 후) ===")
	getStatus(dev, profileToken)

	fmt.Println("\n=== 테스트 완료 ===")
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
