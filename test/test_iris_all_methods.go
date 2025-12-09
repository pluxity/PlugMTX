package main

import (
	"encoding/xml"
	"fmt"
	"io"
	"time"

	"github.com/use-go/onvif"
	onvif_imaging "github.com/use-go/onvif/Imaging"
	"github.com/use-go/onvif/device"
	"github.com/use-go/onvif/media"
	"github.com/use-go/onvif/xsd"
	xsd_onvif "github.com/use-go/onvif/xsd/onvif"
)

func main() {
	host := "14.51.233.129"
	port := 10081
	username := "admin"
	password := "pluxity123!@#"

	fmt.Printf("=== ONVIF Iris 완전 테스트 ===\n\n")

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
					Token                    string `xml:"token,attr"`
					Name                     string
					VideoSourceConfiguration struct {
						SourceToken string
					}
				}
			}
		}
	}

	xml.Unmarshal(body, &envelope)
	profile := envelope.Body.GetProfilesResponse.Profiles[0]
	videoSourceToken := xsd_onvif.ReferenceToken(profile.VideoSourceConfiguration.SourceToken)
	fmt.Printf("VideoSourceToken: %s\n\n", videoSourceToken)

	// Test 1: GetOptions - Iris 지원 확인
	fmt.Println("=== 테스트 1: GetOptions - Iris 지원 확인 ===")
	testGetOptions(dev, videoSourceToken)
	time.Sleep(1 * time.Second)

	// Test 2: GetImagingSettings - 현재 설정 확인
	fmt.Println("\n=== 테스트 2: GetImagingSettings - 현재 설정 확인 ===")
	currentSettings := testGetImagingSettings(dev, videoSourceToken)
	time.Sleep(1 * time.Second)

	// Test 3: SetImagingSettings - Iris만 변경 (최소한의 설정)
	fmt.Println("\n=== 테스트 3: SetImagingSettings - Iris만 변경 (최소 설정) ===")
	testSetIrisMinimal(dev, videoSourceToken)
	time.Sleep(1 * time.Second)

	// Test 4: SetImagingSettings - 전체 설정 보존하고 Iris만 변경
	fmt.Println("\n=== 테스트 4: SetImagingSettings - 전체 설정 보존 ===")
	if currentSettings != nil {
		testSetIrisWithFullSettings(dev, videoSourceToken, currentSettings)
		time.Sleep(1 * time.Second)
	}

	// Test 5: SetImagingSettings - Exposure 전체를 AUTO로 하고 Iris 제외
	fmt.Println("\n=== 테스트 5: SetImagingSettings - AUTO 모드로 전환 후 재시도 ===")
	testSetIrisWithAutoMode(dev, videoSourceToken)
	time.Sleep(1 * time.Second)

	// Test 6: Move - 연속 제어 (가능하다면)
	fmt.Println("\n=== 테스트 6: Move - 연속 제어 ===")
	testIrisMove(dev, videoSourceToken)
	time.Sleep(1 * time.Second)

	// Test 7: SetImagingSettings - Iris를 원래값으로 복원하면서 다른 설정들은 건드리지 않기
	fmt.Println("\n=== 테스트 7: SetImagingSettings - BacklightCompensation 제거 ===")
	testSetIrisWithoutBLC(dev, videoSourceToken, currentSettings)

	fmt.Println("\n=== 테스트 완료 ===")
}

func testGetOptions(dev *onvif.Device, videoSourceToken xsd_onvif.ReferenceToken) {
	req := onvif_imaging.GetOptions{
		VideoSourceToken: videoSourceToken,
	}

	resp, err := dev.CallMethod(req)
	if err != nil {
		fmt.Printf("❌ 실패: %v\n", err)
		return
	}

	if resp != nil {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("응답 코드: %d\n", resp.StatusCode)

		if resp.StatusCode == 200 {
			var envelope struct {
				Body struct {
					GetOptionsResponse struct {
						ImagingOptions struct {
							Exposure struct {
								Mode []string
								Iris struct {
									Min xsd.Float
									Max xsd.Float
								}
							}
						}
					}
				}
			}
			xml.Unmarshal(body, &envelope)

			fmt.Printf("✅ Iris 지원:\n")
			fmt.Printf("   Min: %f\n", float64(envelope.Body.GetOptionsResponse.ImagingOptions.Exposure.Iris.Min))
			fmt.Printf("   Max: %f\n", float64(envelope.Body.GetOptionsResponse.ImagingOptions.Exposure.Iris.Max))
			fmt.Printf("   Exposure Modes: %v\n", envelope.Body.GetOptionsResponse.ImagingOptions.Exposure.Mode)
		} else {
			fmt.Printf("에러 응답: %s\n", string(body))
		}
	}
}

func testGetImagingSettings(dev *onvif.Device, videoSourceToken xsd_onvif.ReferenceToken) *xsd_onvif.ImagingSettings20 {
	req := onvif_imaging.GetImagingSettings{
		VideoSourceToken: videoSourceToken,
	}

	resp, err := dev.CallMethod(req)
	if err != nil {
		fmt.Printf("❌ 실패: %v\n", err)
		return nil
	}

	if resp != nil {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("응답 코드: %d\n", resp.StatusCode)

		if resp.StatusCode == 200 {
			var envelope struct {
				Body struct {
					GetImagingSettingsResponse struct {
						ImagingSettings xsd_onvif.ImagingSettings20
					}
				}
			}
			xml.Unmarshal(body, &envelope)

			settings := envelope.Body.GetImagingSettingsResponse.ImagingSettings
			fmt.Printf("✅ 현재 설정:\n")
			fmt.Printf("   Exposure Mode: %s\n", settings.Exposure.Mode)
			fmt.Printf("   Iris: %f\n", float64(settings.Exposure.Iris))
			fmt.Printf("   BacklightCompensation Mode: %s\n", settings.BacklightCompensation.Mode)

			// 전체 XML 출력
			fmt.Printf("\n전체 설정 XML:\n%s\n", string(body))

			return &settings
		} else {
			fmt.Printf("에러 응답: %s\n", string(body))
		}
	}
	return nil
}

func testSetIrisMinimal(dev *onvif.Device, videoSourceToken xsd_onvif.ReferenceToken) {
	// 최소한의 설정만 보내기
	newIris := float64(-15.0)

	req := onvif_imaging.SetImagingSettings{
		VideoSourceToken: videoSourceToken,
		ImagingSettings: xsd_onvif.ImagingSettings20{
			Exposure: xsd_onvif.Exposure20{
				Mode: xsd_onvif.ExposureMode("MANUAL"),
				Iris: newIris,
			},
		},
	}

	resp, err := dev.CallMethod(req)
	if err != nil {
		fmt.Printf("❌ 실패: %v\n", err)
		return
	}

	if resp != nil {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("응답 코드: %d\n", resp.StatusCode)

		if resp.StatusCode == 200 {
			fmt.Printf("✅ 성공: Iris 값을 %f로 설정\n", float64(newIris))
		} else {
			fmt.Printf("❌ 에러 응답: %s\n", string(body))
		}
	}
}

func testSetIrisWithFullSettings(dev *onvif.Device, videoSourceToken xsd_onvif.ReferenceToken, currentSettings *xsd_onvif.ImagingSettings20) {
	// 현재 설정을 모두 보존하고 Iris만 변경
	settings := *currentSettings
	newIris := float64(-18.0)

	settings.Exposure.Mode = xsd_onvif.ExposureMode("MANUAL")
	settings.Exposure.Iris = newIris

	req := onvif_imaging.SetImagingSettings{
		VideoSourceToken: videoSourceToken,
		ImagingSettings:  settings,
	}

	resp, err := dev.CallMethod(req)
	if err != nil {
		fmt.Printf("❌ 실패: %v\n", err)
		return
	}

	if resp != nil {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("응답 코드: %d\n", resp.StatusCode)

		if resp.StatusCode == 200 {
			fmt.Printf("✅ 성공: Iris 값을 %f로 설정\n", float64(newIris))
		} else {
			fmt.Printf("❌ 에러 응답: %s\n", string(body))
		}
	}
}

func testSetIrisWithAutoMode(dev *onvif.Device, videoSourceToken xsd_onvif.ReferenceToken) {
	// AUTO 모드로 변경 (Iris 제외)
	req := onvif_imaging.SetImagingSettings{
		VideoSourceToken: videoSourceToken,
		ImagingSettings: xsd_onvif.ImagingSettings20{
			Exposure: xsd_onvif.Exposure20{
				Mode: xsd_onvif.ExposureMode("AUTO"),
			},
		},
	}

	resp, err := dev.CallMethod(req)
	if err != nil {
		fmt.Printf("❌ 실패 (AUTO 모드): %v\n", err)
		return
	}

	if resp != nil {
		resp.Body.Close()
		if resp.StatusCode == 200 {
			fmt.Printf("✅ AUTO 모드로 변경 성공\n")

			// 잠시 대기 후 MANUAL + Iris로 다시 시도
			time.Sleep(1 * time.Second)

			fmt.Println("   이제 MANUAL 모드 + Iris 설정 시도...")
			testSetIrisMinimal(dev, videoSourceToken)
		}
	}
}

func testIrisMove(dev *onvif.Device, videoSourceToken xsd_onvif.ReferenceToken) {
	// ONVIF Imaging Service Move에는 Focus는 있지만 Iris는 없습니다
	// Focus만 테스트합니다
	req := onvif_imaging.Move{
		VideoSourceToken: videoSourceToken,
		Focus: xsd_onvif.FocusMove{
			Continuous: xsd_onvif.ContinuousFocus{
				Speed: 0.5,
			},
		},
	}

	resp, err := dev.CallMethod(req)
	if err != nil {
		fmt.Printf("❌ 실패: %v\n", err)
		return
	}

	if resp != nil {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("응답 코드: %d\n", resp.StatusCode)

		if resp.StatusCode == 200 {
			fmt.Printf("✅ Move 명령 성공 (Focus만, Iris는 ONVIF Move에서 지원하지 않음)\n")
		} else {
			fmt.Printf("❌ 에러 응답: %s\n", string(body))
		}
	}
}

func testSetIrisWithoutBLC(dev *onvif.Device, videoSourceToken xsd_onvif.ReferenceToken, currentSettings *xsd_onvif.ImagingSettings20) {
	if currentSettings == nil {
		fmt.Println("현재 설정이 없어서 건너뜀")
		return
	}

	// BacklightCompensation을 빈 값으로 설정
	settings := *currentSettings
	newIris := float64(-16.0)

	settings.Exposure.Mode = xsd_onvif.ExposureMode("MANUAL")
	settings.Exposure.Iris = newIris
	settings.BacklightCompensation = xsd_onvif.BacklightCompensation20{} // 빈 값

	req := onvif_imaging.SetImagingSettings{
		VideoSourceToken: videoSourceToken,
		ImagingSettings:  settings,
	}

	resp, err := dev.CallMethod(req)
	if err != nil {
		fmt.Printf("❌ 실패: %v\n", err)
		return
	}

	if resp != nil {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("응답 코드: %d\n", resp.StatusCode)

		if resp.StatusCode == 200 {
			fmt.Printf("✅ 성공: Iris 값을 %f로 설정 (BLC 제거)\n", newIris)
		} else {
			fmt.Printf("❌ 에러 응답: %s\n", string(body))
		}
	}
}
