package ptz

import (
	"encoding/xml"
	"fmt"
	"io"

	"github.com/use-go/onvif"
	"github.com/use-go/onvif/device"
	"github.com/use-go/onvif/media"
	onvif_ptz "github.com/use-go/onvif/ptz"
	"github.com/use-go/onvif/xsd"
	xsd_onvif "github.com/use-go/onvif/xsd/onvif"
)

// OnvifPTZ ONVIF 호환 카메라의 PTZ 제어 처리
type OnvifPTZ struct {
	Host             string
	Port             int
	Username         string
	Password         string
	device           *onvif.Device
	profileToken     xsd_onvif.ReferenceToken
	videoSourceToken xsd_onvif.ReferenceToken
}

// NewOnvifPTZ creates a new ONVIF PTZ controller
func NewOnvifPTZ(host string, port int, username, password string) *OnvifPTZ {
	if port == 0 {
		port = 80
	}

	return &OnvifPTZ{
		Host:     host,
		Port:     port,
		Username: username,
		Password: password,
	}
}

// Connect ONVIF 장치와 연결 수립 (Controller 인터페이스 구현)
func (o *OnvifPTZ) Connect() error {
	// Create ONVIF device
	// Xaddr should be in format "host:port" only, not full URL
	dev, err := onvif.NewDevice(onvif.DeviceParams{
		Xaddr:    fmt.Sprintf("%s:%d", o.Host, o.Port),
		Username: o.Username,
		Password: o.Password,
	})
	if err != nil {
		return fmt.Errorf("failed to create ONVIF device: %w", err)
	}

	o.device = dev

	// Get device information to verify connection
	getInfoReq := device.GetDeviceInformation{}
	_, err = dev.CallMethod(getInfoReq)
	if err != nil {
		return fmt.Errorf("failed to get device information: %w", err)
	}

	// Get media profiles
	getProfilesReq := media.GetProfiles{}
	profilesResp, err := dev.CallMethod(getProfilesReq)
	if err != nil {
		return fmt.Errorf("failed to get profiles: %w", err)
	}

	// Read response body
	body, err := io.ReadAll(profilesResp.Body)
	if err != nil {
		return fmt.Errorf("failed to read profiles response: %w", err)
	}
	profilesResp.Body.Close()

	// Parse profiles response
	// Use local names to ignore namespaces
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

	if err := xml.Unmarshal(body, &envelope); err != nil {
		return fmt.Errorf("failed to parse profiles: %w", err)
	}

	if len(envelope.Body.GetProfilesResponse.Profiles) == 0 {
		return fmt.Errorf("no media profiles found")
	}

	// Use the first profile
	profile := envelope.Body.GetProfilesResponse.Profiles[0]
	o.profileToken = xsd_onvif.ReferenceToken(profile.Token)
	o.videoSourceToken = xsd_onvif.ReferenceToken(profile.VideoSourceConfiguration.SourceToken)

	return nil
}

// ensureConnected checks if device is connected, connects if not
func (o *OnvifPTZ) ensureConnected() error {
	if o.device == nil {
		return o.Connect()
	}
	return nil
}

// Move 연속 PTZ 이동 수행 (ContinuousMove)
// pan: -100 ~ 100 (음수=좌, 양수=우, 0=정지)
// tilt: -100 ~ 100 (음수=아래, 양수=위, 0=정지)
// zoom: -100 ~ 100 (음수=줌 아웃, 양수=줌 인, 0=정지)
func (o *OnvifPTZ) Move(pan, tilt, zoom int) error {
	if err := o.ensureConnected(); err != nil {
		return err
	}

	// Convert -100~100 to -1.0~1.0 for velocity
	panVelocity := float64(pan) / 100.0
	tiltVelocity := float64(tilt) / 100.0
	zoomVelocity := float64(zoom) / 100.0

	// Timeout is REQUIRED for ContinuousMove
	timeout := xsd.Duration("PT60S")

	req := onvif_ptz.ContinuousMove{
		ProfileToken: o.profileToken,
		Velocity: xsd_onvif.PTZSpeed{
			PanTilt: xsd_onvif.Vector2D{
				X: panVelocity,
				Y: tiltVelocity,
			},
			Zoom: xsd_onvif.Vector1D{
				X: zoomVelocity,
			},
		},
		Timeout: timeout,
	}

	resp, err := o.device.CallMethod(req)
	if err != nil {
		return err
	}

	if resp != nil {
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("ContinuousMove failed with status %d: %s", resp.StatusCode, string(body))
		}
	}

	return nil
}

// RelativeMove 상대적 PTZ 이동 수행 (일정 거리 이동 후 자동 정지)
// pan: -100 ~ 100 (음수=좌, 양수=우)
// tilt: -100 ~ 100 (음수=아래, 양수=위)
// zoom: -100 ~ 100 (음수=줌 아웃, 양수=줌 인)
func (o *OnvifPTZ) RelativeMove(pan, tilt, zoom int) error {
	if err := o.ensureConnected(); err != nil {
		return err
	}

	// Convert -100~100 to -1.0~1.0 for translation
	panTranslation := float64(pan) / 100.0
	tiltTranslation := float64(tilt) / 100.0
	zoomTranslation := float64(zoom) / 100.0

	// Space URIs are REQUIRED for RelativeMove to work
	panTiltSpace := xsd.AnyURI("http://www.onvif.org/ver10/tptz/PanTiltSpaces/TranslationGenericSpace")
	zoomSpace := xsd.AnyURI("http://www.onvif.org/ver10/tptz/ZoomSpaces/TranslationGenericSpace")

	req := onvif_ptz.RelativeMove{
		ProfileToken: o.profileToken,
		Translation: xsd_onvif.PTZVector{
			PanTilt: xsd_onvif.Vector2D{
				X:     panTranslation,
				Y:     tiltTranslation,
				Space: panTiltSpace,
			},
			Zoom: xsd_onvif.Vector1D{
				X:     zoomTranslation,
				Space: zoomSpace,
			},
		},
	}

	resp, err := o.device.CallMethod(req)
	if err != nil {
		return err
	}

	if resp != nil {
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("RelativeMove failed with status %d: %s", resp.StatusCode, string(body))
		}
	}

	return nil
}

// Stop 모든 PTZ 움직임 정지
func (o *OnvifPTZ) Stop() error {
	if err := o.ensureConnected(); err != nil {
		return err
	}

	req := onvif_ptz.Stop{
		ProfileToken: o.profileToken,
		PanTilt:      xsd.Boolean(true),
		Zoom:         xsd.Boolean(true),
	}

	resp, err := o.device.CallMethod(req)
	if err != nil {
		return err
	}

	if resp != nil {
		defer resp.Body.Close()
	}

	return nil
}

// Focus 연속 포커스 조정 수행
// speed: -100 ~ 100 (음수=근거리 포커스, 양수=원거리 포커스, 0=정지)
func (o *OnvifPTZ) Focus(speed int) error {
	if err := o.ensureConnected(); err != nil {
		return err
	}

	// speed가 0이면 Stop
	if speed == 0 {
		return o.Stop()
	}

	// Convert -100~100 to -1.0~1.0
	focusSpeed := float64(speed) / 100.0

	// Timeout is REQUIRED for ContinuousMove
	timeout := xsd.Duration("PT60S")

	// Try PTZ ContinuousMove with Focus
	// Some cameras support Focus in PTZ service instead of Imaging service
	req := onvif_ptz.ContinuousMove{
		ProfileToken: o.profileToken,
		Velocity: xsd_onvif.PTZSpeed{
			PanTilt: xsd_onvif.Vector2D{
				X: 0,
				Y: 0,
			},
			Zoom: xsd_onvif.Vector1D{
				X: focusSpeed, // Use Zoom channel for Focus control
			},
		},
		Timeout: timeout,
	}

	resp, err := o.device.CallMethod(req)
	if err != nil {
		return fmt.Errorf("ptz continuous move focus failed: %w", err)
	}

	if resp != nil {
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("ptz continuous move focus failed with status %d: %s", resp.StatusCode, string(body))
		}
	}

	return nil
}

// Iris 연속 조리개 조정 수행
// speed: -100 ~ 100 (음수=조리개 닫힘, 양수=조리개 열림, 0=정지)
// Note: ONVIF Iris control is not supported by most cameras
func (o *OnvifPTZ) Iris(speed int) error {
	if err := o.ensureConnected(); err != nil {
		return err
	}

	// ONVIF Iris control via SetImagingSettings is not reliably supported
	// Most cameras reject SetImagingSettings with Iris parameter
	return fmt.Errorf("iris control not supported via ONVIF on this camera (use Hikvision ISAPI if available)")
}

// GetStatus 현재 PTZ 상태 조회 및 파싱된 상태 반환 (Controller 인터페이스 구현)
func (o *OnvifPTZ) GetStatus() (*Status, error) {
	if err := o.ensureConnected(); err != nil {
		return nil, err
	}

	req := onvif_ptz.GetStatus{
		ProfileToken: o.profileToken,
	}

	resp, err := o.device.CallMethod(req)
	if err != nil {
		return nil, err
	}

	// 응답 본문 읽기
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read status response: %w", err)
	}
	resp.Body.Close()

	// 상태 응답 파싱
	var envelope struct {
		XMLName xml.Name `xml:"Envelope"`
		Body    struct {
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
				} `xml:"PTZStatus"`
			} `xml:"GetStatusResponse"`
		} `xml:"Body"`
	}

	if err := xml.Unmarshal(body, &envelope); err != nil {
		return nil, fmt.Errorf("failed to parse status: %w", err)
	}

	pos := envelope.Body.GetStatusResponse.PTZStatus.Position

	// 정규화된 값을 각도로 변환 (근사치)
	// 인터페이스의 Status 타입으로 반환
	return &Status{
		Pan:  pos.PanTilt.X * 1800, // 정규화된 값을 각도로 변환
		Tilt: pos.PanTilt.Y * 900,  // 정규화된 값을 각도로 변환
		Zoom: pos.Zoom.X * 100,     // 정규화된 값을 백분율로 변환
	}, nil
}

// GetPresets 사용 가능한 프리셋 목록 조회 및 파싱된 프리셋 목록 반환 (Controller 인터페이스 구현)
func (o *OnvifPTZ) GetPresets() ([]Preset, error) {
	if err := o.ensureConnected(); err != nil {
		return nil, err
	}

	req := onvif_ptz.GetPresets{
		ProfileToken: o.profileToken,
	}

	resp, err := o.device.CallMethod(req)
	if err != nil {
		return nil, err
	}

	// 응답 본문 읽기
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read presets response: %w", err)
	}
	resp.Body.Close()

	// 프리셋 응답 파싱
	var envelope struct {
		XMLName xml.Name `xml:"Envelope"`
		Body    struct {
			GetPresetsResponse struct {
				Preset []struct {
					Token       string `xml:"token,attr"`
					Name        string `xml:"Name"`
					PTZPosition struct {
						PanTilt *struct {
							X float64 `xml:"x,attr"`
							Y float64 `xml:"y,attr"`
						} `xml:"PanTilt"`
						Zoom *struct {
							X float64 `xml:"x,attr"`
						} `xml:"Zoom"`
					} `xml:"PTZPosition"`
				} `xml:"Preset"`
			} `xml:"GetPresetsResponse"`
		} `xml:"Body"`
	}

	if err := xml.Unmarshal(body, &envelope); err != nil {
		return nil, fmt.Errorf("failed to parse presets: %w", err)
	}

	// 인터페이스의 Preset 타입으로 변환
	var presets []Preset

	for _, preset := range envelope.Body.GetPresetsResponse.Preset {
		if preset.Token == "" {
			continue
		}

		p := Preset{
			Name: preset.Name,
		}

		// 토큰을 정수 ID로 파싱 시도
		fmt.Sscanf(preset.Token, "%d", &p.ID)
		if p.ID == 0 {
			// 토큰이 숫자가 아닌 경우 간단한 증분 사용
			p.ID = len(presets) + 1
		}

		presets = append(presets, p)
	}

	return presets, nil
}

// GotoPreset 특정 프리셋 위치로 이동
func (o *OnvifPTZ) GotoPreset(presetID int) error {
	if err := o.ensureConnected(); err != nil {
		return err
	}

	// Convert preset ID to token
	presetToken := xsd_onvif.ReferenceToken(fmt.Sprintf("%d", presetID))

	req := onvif_ptz.GotoPreset{
		ProfileToken: o.profileToken,
		PresetToken:  presetToken,
	}

	_, err := o.device.CallMethod(req)
	return err
}

// SetPreset 현재 PTZ 위치를 프리셋으로 저장
func (o *OnvifPTZ) SetPreset(presetID int, presetName string) error {
	if err := o.ensureConnected(); err != nil {
		return err
	}

	if presetName == "" {
		presetName = fmt.Sprintf("Preset%d", presetID)
	}

	// For ONVIF, use token as string
	presetToken := xsd_onvif.ReferenceToken(fmt.Sprintf("%d", presetID))

	req := onvif_ptz.SetPreset{
		ProfileToken: o.profileToken,
		PresetToken:  presetToken,
		PresetName:   xsd.String(presetName),
	}

	_, err := o.device.CallMethod(req)
	return err
}

// DeletePreset ID로 프리셋 삭제
func (o *OnvifPTZ) DeletePreset(presetID int) error {
	if err := o.ensureConnected(); err != nil {
		return err
	}

	presetToken := xsd_onvif.ReferenceToken(fmt.Sprintf("%d", presetID))

	req := onvif_ptz.RemovePreset{
		ProfileToken: o.profileToken,
		PresetToken:  presetToken,
	}

	_, err := o.device.CallMethod(req)
	return err
}

// GetImageSettings 카메라 이미지 설정 조회 (포커스 및 조리개 설정 포함) (Controller 인터페이스 구현)
func (o *OnvifPTZ) GetImageSettings() (*ImageSettings, error) {
	if err := o.ensureConnected(); err != nil {
		return nil, err
	}

	// ONVIF Imaging 서비스를 여기서 사용
	// 현재는 플레이스홀더 데이터 반환
	// 인터페이스의 ImageSettings 타입으로 반환
	return &ImageSettings{
		Brightness: 50,
		Contrast:   50,
		Saturation: 50,
		Sharpness:  50,
	}, nil
}
