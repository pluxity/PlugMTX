package ptz

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// PTZPreset Hikvision PTZ 프리셋 구조체
type PTZPreset struct {
	Enabled      bool   `xml:"enabled" json:"enabled"`
	ID           int    `xml:"id" json:"id"`
	PresetName   string `xml:"presetName" json:"name"`
	AbsoluteHigh struct {
		Elevation    int `xml:"elevation" json:"elevation"`
		Azimuth      int `xml:"azimuth" json:"azimuth"`
		AbsoluteZoom int `xml:"absoluteZoom" json:"zoom"`
	} `xml:"AbsoluteHigh" json:"position"`
}

// PTZPresetList Hikvision 카메라의 PTZ 프리셋 목록
type PTZPresetList struct {
	XMLName xml.Name    `xml:"PTZPresetList"`
	Presets []PTZPreset `xml:"PTZPreset" json:"presets"`
}

// ImageChannel 카메라 이미지 설정
type ImageChannel struct {
	XMLName            xml.Name `xml:"ImageChannel"`
	FocusConfiguration struct {
		FocusStyle   string `xml:"focusStyle" json:"focusStyle"`
		FocusLimited int    `xml:"focusLimited" json:"focusLimited"`
	} `xml:"FocusConfiguration" json:"focus"`
	Iris struct {
		IrisLevel         int `xml:"IrisLevel" json:"level"`
		MaxIrisLevelLimit int `xml:"maxIrisLevelLimit" json:"maxLimit"`
		MinIrisLevelLimit int `xml:"minIrisLevelLimit" json:"minLimit"`
	} `xml:"Iris" json:"iris"`
	Brightness int `xml:"brightnessLevel" json:"brightness"`
	Contrast   int `xml:"contrastLevel" json:"contrast"`
	Saturation int `xml:"saturationLevel" json:"saturation"`
	Sharpness  int `xml:"sharpnessLevel" json:"sharpness"`
}

// PTZStatus 현재 PTZ 위치 상태
type PTZStatus struct {
	XMLName      xml.Name `xml:"PTZStatus" json:"-"`
	AbsoluteHigh struct {
		Elevation    int `xml:"elevation" json:"elevation"`
		Azimuth      int `xml:"azimuth" json:"azimuth"`
		AbsoluteZoom int `xml:"absoluteZoom" json:"zoom"`
	} `xml:"AbsoluteHigh" json:"position"`
}

// HikvisionPTZ ISAPI를 통한 Hikvision 카메라 PTZ 제어
type HikvisionPTZ struct {
	Host     string
	Port     int
	Username string
	Password string
	client   *http.Client
}

// NewHikvisionPTZ 새로운 Hikvision PTZ 컨트롤러 생성
func NewHikvisionPTZ(host string, port int, username, password string) *HikvisionPTZ {
	return &HikvisionPTZ{
		Host:     host,
		Port:     port,
		Username: username,
		Password: password,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Connect PTZ 카메라와 연결 수립 (Controller 인터페이스 구현)
// Hikvision ISAPI는 연결 유지가 필요 없으므로 기본 검증만 수행
func (h *HikvisionPTZ) Connect() error {
	// 기본 연결 테스트: PTZ 상태 조회
	_, err := h.GetStatus()
	return err
}

// getHostPort 호스트:포트 조합 반환
func (h *HikvisionPTZ) getHostPort() string {
	if h.Port != 0 {
		return fmt.Sprintf("%s:%d", h.Host, h.Port)
	}
	return h.Host
}

// Move 순간 PTZ 이동 수행 (Momentary - 상대적 이동)
// pan: -100 ~ 100 (음수=좌, 양수=우, 0=정지)
// tilt: -100 ~ 100 (음수=아래, 양수=위, 0=정지)
// zoom: -100 ~ 100 (음수=줌 아웃, 양수=줌 인, 0=정지)
// momentary 방식은 지정한 거리만큼 상대적으로 이동 후 자동 정지
func (h *HikvisionPTZ) Move(pan, tilt, zoom int) error {
	xmlData := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<PTZData>
    <pan>%d</pan>
    <tilt>%d</tilt>
    <zoom>%d</zoom>
</PTZData>`, pan, tilt, zoom)

	url := fmt.Sprintf("http://%s/ISAPI/PTZCtrl/channels/1/momentary", h.getHostPort())
	return h.sendRequest("PUT", url, xmlData)
}

// Focus 연속 포커스 조정 수행
// speed: -100 ~ 100 (음수=근거리 포커스, 양수=원거리 포커스, 0=정지)
func (h *HikvisionPTZ) Focus(speed int) error {
	xmlData := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<PTZData>
    <pan>0</pan>
    <tilt>0</tilt>
    <zoom>0</zoom>
    <Momentary>
        <focus>%d</focus>
    </Momentary>
</PTZData>`, speed)

	url := fmt.Sprintf("http://%s/ISAPI/PTZCtrl/channels/1/continuous", h.getHostPort())
	return h.sendRequest("PUT", url, xmlData)
}

// Iris 연속 조리개 조정 수행
// speed: -100 ~ 100 (음수=조리개 닫힘, 양수=조리개 열림, 0=정지)
func (h *HikvisionPTZ) Iris(speed int) error {
	xmlData := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<PTZData>
    <pan>0</pan>
    <tilt>0</tilt>
    <zoom>0</zoom>
    <Momentary>
        <iris>%d</iris>
    </Momentary>
</PTZData>`, speed)

	url := fmt.Sprintf("http://%s/ISAPI/PTZCtrl/channels/1/continuous", h.getHostPort())
	return h.sendRequest("PUT", url, xmlData)
}

// Stop 모든 PTZ 움직임 정지
func (h *HikvisionPTZ) Stop() error {
	return h.Move(0, 0, 0)
}

// GetStatus 현재 PTZ 상태 조회 및 파싱된 상태 반환 (Controller 인터페이스 구현)
func (h *HikvisionPTZ) GetStatus() (*Status, error) {
	url := fmt.Sprintf("http://%s/ISAPI/PTZCtrl/channels/1/status", h.getHostPort())
	xmlData, err := h.sendGetRequest(url)
	if err != nil {
		return nil, err
	}

	var ptzStatus PTZStatus
	if err := xml.Unmarshal([]byte(xmlData), &ptzStatus); err != nil {
		return nil, fmt.Errorf("failed to parse PTZ status XML: %w", err)
	}

	// PTZStatus를 인터페이스의 Status로 변환
	return &Status{
		Pan:  float64(ptzStatus.AbsoluteHigh.Azimuth),
		Tilt: float64(ptzStatus.AbsoluteHigh.Elevation),
		Zoom: float64(ptzStatus.AbsoluteHigh.AbsoluteZoom),
	}, nil
}

// GetPresets 사용 가능한 프리셋 목록 조회 및 파싱된 프리셋 목록 반환 (Controller 인터페이스 구현)
func (h *HikvisionPTZ) GetPresets() ([]Preset, error) {
	url := fmt.Sprintf("http://%s/ISAPI/PTZCtrl/channels/1/presets", h.getHostPort())
	xmlData, err := h.sendGetRequest(url)
	if err != nil {
		return nil, err
	}

	var presetList PTZPresetList
	if err := xml.Unmarshal([]byte(xmlData), &presetList); err != nil {
		return nil, fmt.Errorf("failed to parse XML: %w", err)
	}

	// PTZPreset을 인터페이스의 Preset으로 변환
	presets := make([]Preset, 0)
	for _, p := range presetList.Presets {
		if p.Enabled {
			presets = append(presets, Preset{
				ID:   p.ID,
				Name: p.PresetName,
			})
		}
	}

	return presets, nil
}

// GetImageSettings 카메라 이미지 설정 조회 (포커스 및 조리개 설정 포함) (Controller 인터페이스 구현)
func (h *HikvisionPTZ) GetImageSettings() (*ImageSettings, error) {
	url := fmt.Sprintf("http://%s/ISAPI/Image/channels/1", h.getHostPort())
	xmlData, err := h.sendGetRequest(url)
	if err != nil {
		return nil, err
	}

	var imageChannel ImageChannel
	if err := xml.Unmarshal([]byte(xmlData), &imageChannel); err != nil {
		return nil, fmt.Errorf("failed to parse image settings XML: %w", err)
	}

	// ImageChannel을 인터페이스의 ImageSettings로 변환
	return &ImageSettings{
		Brightness: imageChannel.Brightness,
		Contrast:   imageChannel.Contrast,
		Saturation: imageChannel.Saturation,
		Sharpness:  imageChannel.Sharpness,
	}, nil
}

// GotoPreset 특정 프리셋 위치로 이동
func (h *HikvisionPTZ) GotoPreset(presetID int) error {
	xmlData := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<PTZData>
    <AbsoluteHigh>
        <presetID>%d</presetID>
    </AbsoluteHigh>
</PTZData>`, presetID)

	url := fmt.Sprintf("http://%s/ISAPI/PTZCtrl/channels/1/presets/%d/goto", h.getHostPort(), presetID)
	return h.sendRequest("PUT", url, xmlData)
}

// SetPreset 현재 PTZ 위치를 프리셋으로 저장
func (h *HikvisionPTZ) SetPreset(presetID int, presetName string) error {
	xmlData := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<PTZPreset>
    <id>%d</id>
    <presetName>%s</presetName>
</PTZPreset>`, presetID, presetName)

	url := fmt.Sprintf("http://%s/ISAPI/PTZCtrl/channels/1/presets/%d", h.getHostPort(), presetID)
	return h.sendRequest("PUT", url, xmlData)
}

// DeletePreset ID로 프리셋 삭제
func (h *HikvisionPTZ) DeletePreset(presetID int) error {
	url := fmt.Sprintf("http://%s/ISAPI/PTZCtrl/channels/1/presets/%d", h.getHostPort(), presetID)

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(h.Username, h.Password)

	resp, err := h.client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		// Digest 인증으로 재시도
		return h.sendDigestDeleteRequest(url, resp)
	}

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

// sendRequest Digest 인증을 사용한 HTTP 요청 전송
func (h *HikvisionPTZ) sendRequest(method, urlStr, body string) error {
	req, err := http.NewRequest(method, urlStr, strings.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/xml")
	req.SetBasicAuth(h.Username, h.Password)

	resp, err := h.client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		// Digest 인증으로 재시도
		return h.sendDigestRequest(method, urlStr, body, resp)
	}

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

// sendGetRequest GET 요청 전송 및 응답 반환
func (h *HikvisionPTZ) sendGetRequest(urlStr string) (string, error) {
	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(h.Username, h.Password)

	resp, err := h.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		// Digest 인증으로 재시도
		return h.sendDigestGetRequest(urlStr, resp)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return string(bodyBytes), nil
}

// sendDigestGetRequest Digest 인증을 사용한 GET 요청 전송 및 응답 반환
func (h *HikvisionPTZ) sendDigestGetRequest(urlStr string, authResp *http.Response) (string, error) {
	// WWW-Authenticate 헤더 파싱
	authHeader := authResp.Header.Get("WWW-Authenticate")
	if authHeader == "" {
		return "", fmt.Errorf("no WWW-Authenticate header")
	}

	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create digest request: %w", err)
	}

	// Digest 챌린지 파싱
	digestParams := parseDigestAuth(authHeader)

	// Digest 응답 계산
	uri := req.URL.Path
	if req.URL.RawQuery != "" {
		uri += "?" + req.URL.RawQuery
	}

	ha1 := md5Hash(h.Username + ":" + digestParams["realm"] + ":" + h.Password)
	ha2 := md5Hash("GET:" + uri)

	var response string
	var authHeaderValue string

	if qop, ok := digestParams["qop"]; ok && qop == "auth" {
		// qop 있음
		cnonce := "0a4f113b"
		nc := "00000001"
		response = md5Hash(ha1 + ":" + digestParams["nonce"] + ":" + nc + ":" + cnonce + ":" + qop + ":" + ha2)

		authHeaderValue = fmt.Sprintf(`Digest username="%s", realm="%s", nonce="%s", uri="%s", qop=%s, nc=%s, cnonce="%s", response="%s"`,
			h.Username, digestParams["realm"], digestParams["nonce"], uri, qop, nc, cnonce, response)
	} else {
		// qop 없음
		response = md5Hash(ha1 + ":" + digestParams["nonce"] + ":" + ha2)

		authHeaderValue = fmt.Sprintf(`Digest username="%s", realm="%s", nonce="%s", uri="%s", response="%s"`,
			h.Username, digestParams["realm"], digestParams["nonce"], uri, response)
	}

	req.Header.Set("Authorization", authHeaderValue)

	resp, err := h.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("digest request failed: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("digest request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return string(bodyBytes), nil
}

// sendDigestDeleteRequest Digest 인증을 사용한 DELETE 요청 전송
func (h *HikvisionPTZ) sendDigestDeleteRequest(urlStr string, authResp *http.Response) error {
	// WWW-Authenticate 헤더 파싱
	authHeader := authResp.Header.Get("WWW-Authenticate")
	if authHeader == "" {
		return fmt.Errorf("no WWW-Authenticate header")
	}

	req, err := http.NewRequest("DELETE", urlStr, nil)
	if err != nil {
		return fmt.Errorf("failed to create digest request: %w", err)
	}

	// Digest 챌린지 파싱
	digestParams := parseDigestAuth(authHeader)

	// Digest 응답 계산
	uri := req.URL.Path
	if req.URL.RawQuery != "" {
		uri += "?" + req.URL.RawQuery
	}

	ha1 := md5Hash(h.Username + ":" + digestParams["realm"] + ":" + h.Password)
	ha2 := md5Hash("DELETE:" + uri)

	var response string
	var authHeaderValue string

	if qop, ok := digestParams["qop"]; ok && qop == "auth" {
		// qop 있음
		cnonce := "0a4f113b"
		nc := "00000001"
		response = md5Hash(ha1 + ":" + digestParams["nonce"] + ":" + nc + ":" + cnonce + ":" + qop + ":" + ha2)

		authHeaderValue = fmt.Sprintf(`Digest username="%s", realm="%s", nonce="%s", uri="%s", qop=%s, nc=%s, cnonce="%s", response="%s"`,
			h.Username, digestParams["realm"], digestParams["nonce"], uri, qop, nc, cnonce, response)
	} else {
		// qop 없음
		response = md5Hash(ha1 + ":" + digestParams["nonce"] + ":" + ha2)

		authHeaderValue = fmt.Sprintf(`Digest username="%s", realm="%s", nonce="%s", uri="%s", response="%s"`,
			h.Username, digestParams["realm"], digestParams["nonce"], uri, response)
	}

	req.Header.Set("Authorization", authHeaderValue)

	resp, err := h.client.Do(req)
	if err != nil {
		return fmt.Errorf("digest request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("digest request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

// sendDigestRequest Digest 인증을 사용한 요청 전송
func (h *HikvisionPTZ) sendDigestRequest(method, urlStr, body string, authResp *http.Response) error {
	// WWW-Authenticate 헤더 파싱
	authHeader := authResp.Header.Get("WWW-Authenticate")
	if authHeader == "" {
		return fmt.Errorf("no WWW-Authenticate header")
	}

	req, err := http.NewRequest(method, urlStr, strings.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create digest request: %w", err)
	}

	req.Header.Set("Content-Type", "application/xml")

	// Digest 챌린지 파싱
	digestParams := parseDigestAuth(authHeader)

	// Digest 응답 계산
	uri := req.URL.Path
	if req.URL.RawQuery != "" {
		uri += "?" + req.URL.RawQuery
	}

	ha1 := md5Hash(h.Username + ":" + digestParams["realm"] + ":" + h.Password)
	ha2 := md5Hash(method + ":" + uri)

	var response string
	var authHeaderValue string

	if qop, ok := digestParams["qop"]; ok && qop == "auth" {
		// qop 있음
		cnonce := "0a4f113b"
		nc := "00000001"
		response = md5Hash(ha1 + ":" + digestParams["nonce"] + ":" + nc + ":" + cnonce + ":" + qop + ":" + ha2)

		authHeaderValue = fmt.Sprintf(`Digest username="%s", realm="%s", nonce="%s", uri="%s", qop=%s, nc=%s, cnonce="%s", response="%s"`,
			h.Username, digestParams["realm"], digestParams["nonce"], uri, qop, nc, cnonce, response)
	} else {
		// qop 없음
		response = md5Hash(ha1 + ":" + digestParams["nonce"] + ":" + ha2)

		authHeaderValue = fmt.Sprintf(`Digest username="%s", realm="%s", nonce="%s", uri="%s", response="%s"`,
			h.Username, digestParams["realm"], digestParams["nonce"], uri, response)
	}

	req.Header.Set("Authorization", authHeaderValue)

	resp, err := h.client.Do(req)
	if err != nil {
		return fmt.Errorf("digest request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("digest request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

// parseDigestAuth WWW-Authenticate 헤더 파싱
func parseDigestAuth(authHeader string) map[string]string {
	params := make(map[string]string)

	// "Digest " 접두사 제거
	authHeader = strings.TrimPrefix(authHeader, "Digest ")

	// 쉼표로 분할
	parts := strings.Split(authHeader, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		keyValue := strings.SplitN(part, "=", 2)
		if len(keyValue) == 2 {
			key := strings.TrimSpace(keyValue[0])
			value := strings.Trim(strings.TrimSpace(keyValue[1]), `"`)
			params[key] = value
		}
	}

	return params
}

// md5Hash MD5 해시 계산
func md5Hash(text string) string {
	hash := md5.Sum([]byte(text))
	return hex.EncodeToString(hash[:])
}

// ExtractHostFromRTSP RTSP URL에서 호스트 및 인증 정보 추출
// rtsp://username:password@host:port/path -> host, username, password
func ExtractHostFromRTSP(rtspURL string) (host, username, password string, err error) {
	u, err := url.Parse(rtspURL)
	if err != nil {
		return "", "", "", err
	}

	host = u.Host
	if u.User != nil {
		username = u.User.Username()
		password, _ = u.User.Password()
	}

	return host, username, password, nil
}
