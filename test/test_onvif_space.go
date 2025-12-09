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

	fmt.Printf("=== ONVIF Space 확인 및 RelativeMove 테스트 ===\n\n")

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

	var profilesEnvelope struct {
		Body struct {
			GetProfilesResponse struct {
				Profiles []struct {
					Token            string `xml:"token,attr"`
					Name             string
					PTZConfiguration struct {
						Token                   string `xml:"token,attr"`
						Name                    string
						DefaultRelativeTranslationSpace struct {
							URI   string `xml:"URI"`
							XRange struct {
								Min float64
								Max float64
							} `xml:"XRange"`
							YRange struct {
								Min float64
								Max float64
							} `xml:"YRange"`
						} `xml:"DefaultRelativeTranslationSpace"`
						DefaultAbsolutePantTiltPositionSpace struct {
							URI string `xml:"URI"`
						} `xml:"DefaultAbsolutePantTiltPositionSpace"`
					} `xml:"PTZConfiguration"`
				}
			}
		}
	}

	xml.Unmarshal(body, &profilesEnvelope)
	profile := profilesEnvelope.Body.GetProfilesResponse.Profiles[0]
	profileToken := xsd_onvif.ReferenceToken(profile.Token)

	fmt.Printf("프로필: %s\n", profileToken)
	fmt.Printf("PTZ 설정 토큰: %s\n\n", profile.PTZConfiguration.Token)

	// Print supported spaces
	fmt.Println("=== 지원하는 좌표계 (Space) ===")
	relativeSpace := profile.PTZConfiguration.DefaultRelativeTranslationSpace.URI
	absoluteSpace := profile.PTZConfiguration.DefaultAbsolutePantTiltPositionSpace.URI

	fmt.Printf("Relative Translation Space: %s\n", relativeSpace)
	if relativeSpace != "" {
		fmt.Printf("  X Range: %.2f ~ %.2f\n",
			profile.PTZConfiguration.DefaultRelativeTranslationSpace.XRange.Min,
			profile.PTZConfiguration.DefaultRelativeTranslationSpace.XRange.Max)
		fmt.Printf("  Y Range: %.2f ~ %.2f\n",
			profile.PTZConfiguration.DefaultRelativeTranslationSpace.YRange.Min,
			profile.PTZConfiguration.DefaultRelativeTranslationSpace.YRange.Max)
	}
	fmt.Printf("Absolute PanTilt Space: %s\n\n", absoluteSpace)

	if relativeSpace == "" {
		fmt.Println("❌ RelativeSpace가 정의되지 않았습니다!")
		fmt.Println("이 카메라는 RelativeMove를 지원하지 않을 수 있습니다.")
		return
	}

	// Get initial status
	fmt.Println("=== 초기 상태 ===")
	initialStatus := getStatus(dev, profileToken)

	// Test RelativeMove WITH proper Space
	fmt.Println("\n=== RelativeMove with Space (Pan +0.1) ===")

	relReq := onvif_ptz.RelativeMove{
		ProfileToken: profileToken,
		Translation: xsd_onvif.PTZVector{
			PanTilt: xsd_onvif.Vector2D{
				X:     0.1,
				Y:     0.0,
				Space: xsd.AnyURI(relativeSpace), // Explicitly set the space
			},
			Zoom: xsd_onvif.Vector1D{
				X:     0.0,
				Space: xsd.AnyURI(relativeSpace),
			},
		},
	}

	relResp, err := dev.CallMethod(relReq)
	if err != nil {
		fmt.Printf("❌ 실패: %v\n", err)
	} else {
		relBody, _ := io.ReadAll(relResp.Body)
		relResp.Body.Close()
		fmt.Printf("✅ 응답 코드: %d (%s)\n", relResp.StatusCode, relResp.Status)

		if relResp.StatusCode != 200 {
			fmt.Printf("에러 응답 (처음 500자): %s\n", string(relBody[:min(500, len(relBody))]))
		} else {
			fmt.Printf("성공! 응답 길이: %d bytes\n", len(relBody))
		}
	}

	time.Sleep(3 * time.Second)
	fmt.Println("\n3초 후 상태:")
	status := getStatus(dev, profileToken)
	fmt.Printf("변화량: Pan %+.4f, Tilt %+.4f\n",
		status.Pan-initialStatus.Pan,
		status.Tilt-initialStatus.Tilt)

	if status.Pan != initialStatus.Pan || status.Tilt != initialStatus.Tilt {
		fmt.Println("\n✅ RelativeMove가 작동합니다!")
	} else {
		fmt.Println("\n❌ 여전히 작동하지 않습니다.")
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
