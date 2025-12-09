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

	fmt.Printf("=== ONVIF Focus/Iris 테스트 ===\n\n")

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

	// Test 1: ContinuousMove with Focus
	fmt.Println("=== 테스트 1: ContinuousMove with Focus ===")
	testFocusContinuous(dev, profileToken)

	time.Sleep(2 * time.Second)

	// Test 2: Stop
	fmt.Println("\n=== 테스트 2: Stop ===")
	stopReq := onvif_ptz.Stop{
		ProfileToken: profileToken,
		PanTilt:      xsd.Boolean(true),
		Zoom:         xsd.Boolean(true),
	}
	stopResp, _ := dev.CallMethod(stopReq)
	if stopResp != nil {
		stopResp.Body.Close()
		fmt.Printf("✅ Stop 성공 (코드: %d)\n", stopResp.StatusCode)
	}
}

func testFocusContinuous(dev *onvif.Device, profileToken xsd_onvif.ReferenceToken) {
	timeout := xsd.Duration("PT5S")

	// Focus speed test
	req := onvif_ptz.ContinuousMove{
		ProfileToken: profileToken,
		Velocity: xsd_onvif.PTZSpeed{
			PanTilt: xsd_onvif.Vector2D{
				X: 0.0,
				Y: 0.0,
			},
			Zoom: xsd_onvif.Vector1D{
				X: 0.0,
			},
		},
		Timeout: timeout,
	}

	resp, err := dev.CallMethod(req)
	if err != nil {
		fmt.Printf("❌ 실패: %v\n", err)
		return
	}

	if resp != nil {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("응답 코드: %d (%s)\n", resp.StatusCode, resp.Status)

		if resp.StatusCode != 200 {
			fmt.Printf("에러 응답: %s\n", string(body))
		} else {
			fmt.Printf("✅ ContinuousMove 성공\n")
		}
	}
}
