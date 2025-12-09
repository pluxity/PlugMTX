package main

import (
	"encoding/xml"
	"fmt"
	"io"

	"github.com/use-go/onvif"
	onvif_imaging "github.com/use-go/onvif/Imaging"
	onvif_media "github.com/use-go/onvif/media"
	xsd_onvif "github.com/use-go/onvif/xsd/onvif"
)

func main() {
	host := "14.51.233.129"
	port := 10081
	username := "admin"
	password := "pluxity123!@#"

	fmt.Printf("=== ONVIF Imaging Service 테스트 ===\n\n")

	// ONVIF 장치 연결
	dev, err := onvif.NewDevice(onvif.DeviceParams{
		Xaddr:    fmt.Sprintf("%s:%d", host, port),
		Username: username,
		Password: password,
	})
	if err != nil {
		fmt.Printf("❌ 연결 실패: %v\n", err)
		return
	}

	fmt.Println("✅ ONVIF 장치 연결 성공")

	// Get media profiles
	getProfilesReq := onvif_media.GetProfiles{}
	profilesResp, err := dev.CallMethod(getProfilesReq)
	if err != nil {
		fmt.Printf("❌ GetProfiles 실패: %v\n", err)
		return
	}

	body, _ := io.ReadAll(profilesResp.Body)
	profilesResp.Body.Close()

	// 먼저 전체 응답을 출력해서 구조 확인
	fmt.Println("--- Profiles Response ---")
	fmt.Printf("%s\n\n", string(body))

	var envelope struct {
		Body struct {
			GetProfilesResponse struct {
				Profiles []struct {
					Token                     string `xml:"token,attr"`
					Name                      string
					VideoSourceConfiguration struct {
						SourceToken string
					}
				}
			}
		}
	}

	if err := xml.Unmarshal(body, &envelope); err != nil {
		fmt.Printf("❌ 프로파일 파싱 실패: %v\n", err)
		return
	}

	if len(envelope.Body.GetProfilesResponse.Profiles) == 0 {
		fmt.Printf("❌ 프로파일을 찾을 수 없습니다\n")
		return
	}

	profile := envelope.Body.GetProfilesResponse.Profiles[0]
	profileToken := xsd_onvif.ReferenceToken(profile.Token)
	videoSourceToken := xsd_onvif.ReferenceToken(profile.VideoSourceConfiguration.SourceToken)

	fmt.Printf("✅ Profile Token: %s\n", profileToken)
	fmt.Printf("✅ VideoSource Token: %s\n\n", videoSourceToken)

	// Test 1: GetImagingSettings
	fmt.Println("--- Test 1: GetImagingSettings ---")
	getSettingsReq := onvif_imaging.GetImagingSettings{
		VideoSourceToken: videoSourceToken,
	}

	settingsResp, err := dev.CallMethod(getSettingsReq)
	if err != nil {
		fmt.Printf("❌ GetImagingSettings 실패: %v\n", err)
	} else {
		body, _ := io.ReadAll(settingsResp.Body)
		settingsResp.Body.Close()
		fmt.Printf("✅ 응답 상태: %s\n", settingsResp.Status)
		fmt.Printf("응답 내용:\n%s\n\n", string(body))
	}

	// Test 2: GetOptions
	fmt.Println("--- Test 2: GetOptions ---")
	getOptionsReq := onvif_imaging.GetOptions{
		VideoSourceToken: videoSourceToken,
	}

	optionsResp, err := dev.CallMethod(getOptionsReq)
	if err != nil {
		fmt.Printf("❌ GetOptions 실패: %v\n", err)
	} else {
		body, _ := io.ReadAll(optionsResp.Body)
		optionsResp.Body.Close()
		fmt.Printf("✅ 응답 상태: %s\n", settingsResp.Status)
		fmt.Printf("응답 내용:\n%s\n\n", string(body))
	}

	// Test 3: Move - Focus Far (원거리 포커스)
	fmt.Println("--- Test 3: Move Focus Far ---")
	moveFocusReq := onvif_imaging.Move{
		VideoSourceToken: videoSourceToken,
		Focus: xsd_onvif.FocusMove{
			Continuous: xsd_onvif.ContinuousFocus{
				Speed: 0.5, // 양수 = 원거리 포커스
			},
		},
	}

	moveFocusResp, err := dev.CallMethod(moveFocusReq)
	if err != nil {
		fmt.Printf("❌ Move Focus 실패: %v\n", err)
	} else {
		body, _ := io.ReadAll(moveFocusResp.Body)
		moveFocusResp.Body.Close()
		fmt.Printf("✅ 응답 상태: %s\n", moveFocusResp.Status)
		fmt.Printf("응답 내용:\n%s\n\n", string(body))
	}

	// Test 4: Stop
	fmt.Println("--- Test 4: Stop ---")
	stopReq := onvif_imaging.Stop{
		VideoSourceToken: videoSourceToken,
	}

	stopResp, err := dev.CallMethod(stopReq)
	if err != nil {
		fmt.Printf("❌ Stop 실패: %v\n", err)
	} else {
		body, _ := io.ReadAll(stopResp.Body)
		stopResp.Body.Close()
		fmt.Printf("✅ 응답 상태: %s\n", stopResp.Status)
		fmt.Printf("응답 내용:\n%s\n\n", string(body))
	}

	fmt.Println("=== 테스트 완료 ===")
}
