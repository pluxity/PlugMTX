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

	fmt.Printf("=== ONVIF RelativeMove 최종 테스트 ===\n\n")

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

	// Space URIs from configuration
	relativeSpace := "http://www.onvif.org/ver10/tptz/PanTiltSpaces/TranslationGenericSpace"
	zoomSpace := "http://www.onvif.org/ver10/tptz/ZoomSpaces/TranslationGenericSpace"

	// Get initial status
	fmt.Println("=== 초기 상태 ===")
	initial := getStatus(dev, profileToken)

	// Test RelativeMove with proper Space
	fmt.Println("\n=== RelativeMove (Pan +0.2 with Space) ===")
	relReq := onvif_ptz.RelativeMove{
		ProfileToken: profileToken,
		Translation: xsd_onvif.PTZVector{
			PanTilt: xsd_onvif.Vector2D{
				X:     0.2,
				Y:     0.0,
				Space: xsd.AnyURI(relativeSpace),
			},
			Zoom: xsd_onvif.Vector1D{
				X:     0.0,
				Space: xsd.AnyURI(zoomSpace),
			},
		},
	}

	relResp, err := dev.CallMethod(relReq)
	if err != nil {
		fmt.Printf("❌ 실패: %v\n", err)
		return
	}

	relBody, _ := io.ReadAll(relResp.Body)
	relResp.Body.Close()

	fmt.Printf("응답 코드: %d (%s)\n", relResp.StatusCode, relResp.Status)

	if relResp.StatusCode != 200 {
		fmt.Printf("에러 응답: %s\n", string(relBody[:min(500, len(relBody))]))
		return
	}

	fmt.Println("✅ RelativeMove 성공!")

	time.Sleep(3 * time.Second)
	fmt.Println("\n=== 3초 후 상태 ===")
	after := getStatus(dev, profileToken)

	fmt.Printf("\n변화량: Pan %+.4f, Tilt %+.4f\n",
		after.Pan-initial.Pan,
		after.Tilt-initial.Tilt)

	if after.Pan != initial.Pan {
		fmt.Println("\n🎉 RelativeMove가 작동합니다!")
		fmt.Println("카메라가 실제로 움직였는지 확인하세요!")
	} else {
		fmt.Println("\n⚠️  Status는 변하지 않았습니다.")
	}
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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
