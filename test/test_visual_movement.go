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
	"github.com/use-go/onvif/xsd"
	xsd_onvif "github.com/use-go/onvif/xsd/onvif"
)

func main() {
	host := "14.51.233.129"
	port := 10081
	username := "admin"
	password := "pluxity123!@#"

	fmt.Printf("=== 카메라 실제 움직임 확인 테스트 ===\n\n")
	fmt.Println("⚠️  카메라를 직접 육안으로 관찰하면서 테스트하세요!")
	fmt.Println("⚠️  카메라가 실제로 회전하는지 확인하세요!\n")

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
	fmt.Printf("프로필: %s\n\n", profileToken)

	// Get initial status
	fmt.Println("=== 초기 상태 ===")
	initialPan := getStatus(dev, profileToken)

	// Test 1: 우측으로 30초간 최대 속도 회전
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("테스트 1: 우측으로 30초간 최대 속도 회전 (Pan = 1.0)")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("🎥 지금 카메라를 보세요! 30초간 우측으로 회전합니다!")
	fmt.Println(strings.Repeat("=", 60) + "\n")

	timeout30s := xsd.Duration("PT30S")
	moveRight := onvif_ptz.ContinuousMove{
		ProfileToken: profileToken,
		Velocity: xsd_onvif.PTZSpeed{
			PanTilt: xsd_onvif.Vector2D{
				X: 1.0, // 최대 속도 우측
				Y: 0.0,
			},
			Zoom: xsd_onvif.Vector1D{
				X: 0.0,
			},
		},
		Timeout: timeout30s,
	}

	resp, err := dev.CallMethod(moveRight)
	if err != nil {
		fmt.Printf("❌ ContinuousMove 실패: %v\n", err)
		return
	}
	resp.Body.Close()
	fmt.Printf("✅ 명령 전송 완료 (응답: %s)\n\n", resp.Status)

	// Monitor status every 3 seconds
	for i := 1; i <= 10; i++ {
		time.Sleep(3 * time.Second)
		fmt.Printf("[%d초] ", i*3)
		currentPan := getStatus(dev, profileToken)

		change := currentPan - initialPan
		fmt.Printf("       변화량: %+.4f (초기값 대비)\n", change)
	}

	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("❓ 카메라가 실제로 회전했나요? (Y/N)")
	fmt.Println(strings.Repeat("=", 60))

	// Wait for movement to complete
	time.Sleep(1 * time.Second)
	fmt.Println("\n=== 최종 상태 ===")
	finalPan := getStatus(dev, profileToken)

	totalChange := finalPan - initialPan
	fmt.Printf("\n📊 총 변화량: %+.4f (%.1f도)\n", totalChange, totalChange*180)

	if totalChange > 0.5 {
		fmt.Println("✅ Pan 값이 크게 변했습니다. 카메라가 움직였을 것으로 예상됩니다.")
	} else if totalChange > 0.1 {
		fmt.Println("⚠️  Pan 값이 조금 변했습니다. 작은 움직임이었을 수 있습니다.")
	} else {
		fmt.Println("❌ Pan 값 변화가 거의 없습니다. 카메라가 움직이지 않은 것 같습니다.")
	}
}

func getStatus(dev *onvif.Device, profileToken xsd_onvif.ReferenceToken) float64 {
	statusReq := onvif_ptz.GetStatus{
		ProfileToken: profileToken,
	}

	statusResp, err := dev.CallMethod(statusReq)
	if err != nil {
		fmt.Printf("❌ GetStatus 실패: %v\n", err)
		return 0
	}

	body, _ := io.ReadAll(statusResp.Body)
	statusResp.Body.Close()

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
		return 0
	}

	status := envelope.Body.GetStatusResponse.PTZStatus
	fmt.Printf("Pan: %7.4f, Tilt: %7.4f | 상태: %s\n",
		status.Position.PanTilt.X,
		status.Position.PanTilt.Y,
		status.MoveStatus.PanTilt)

	return status.Position.PanTilt.X
}
