package ptz

// Controller PTZ 카메라 제어를 위한 인터페이스 정의
// ONVIF와 Hikvision 구현체 모두 이 인터페이스를 만족해야 함
type Controller interface {
	// Connect PTZ 카메라와 연결 수립
	Connect() error

	// Move 팬, 틸트, 줌 제어
	// pan: -100 ~ 100 (좌측에서 우측)
	// tilt: -100 ~ 100 (아래에서 위)
	// zoom: -100 ~ 100 (줌 아웃에서 줌 인)
	Move(pan, tilt, zoom int) error

	// Stop 모든 PTZ 움직임 정지
	Stop() error

	// GetStatus 현재 PTZ 위치 및 상태 반환
	GetStatus() (*Status, error)

	// GetPresets 사용 가능한 프리셋 목록 반환
	GetPresets() ([]Preset, error)

	// GotoPreset 저장된 프리셋 위치로 카메라 이동
	GotoPreset(presetID int) error

	// SetPreset 현재 위치를 프리셋으로 저장
	SetPreset(presetID int, name string) error

	// DeletePreset 저장된 프리셋 삭제
	DeletePreset(presetID int) error

	// Focus 카메라 포커스 제어
	// speed: -100 ~ 100 (원거리에서 근거리)
	Focus(speed int) error

	// Iris 카메라 조리개 제어
	// speed: -100 ~ 100 (닫힘에서 열림)
	Iris(speed int) error

	// GetImageSettings 현재 이미지 설정 반환
	GetImageSettings() (*ImageSettings, error)
}

// Status 현재 PTZ 카메라 상태
type Status struct {
	Pan  float64 `json:"pan"`  // 팬 위치
	Tilt float64 `json:"tilt"` // 틸트 위치
	Zoom float64 `json:"zoom"` // 줌 위치
}

// Preset 저장된 카메라 위치
type Preset struct {
	ID   int    `json:"id"`   // 프리셋 ID
	Name string `json:"name"` // 프리셋 이름
}

// ImageSettings 카메라 이미지 설정
type ImageSettings struct {
	Brightness int `json:"brightness"` // 밝기
	Contrast   int `json:"contrast"`   // 대비
	Saturation int `json:"saturation"` // 채도
	Sharpness  int `json:"sharpness"`  // 선명도
}

// ControllerConfig PTZ 컨트롤러 생성을 위한 설정
type ControllerConfig struct {
	Protocol string // 프로토콜: "onvif", "isapi", "hikvision"
	Host     string // 호스트 주소
	Port     int    // 포트 번호
	Username string // 사용자명
	Password string // 비밀번호
}

// NewController 프로토콜에 따라 새로운 PTZ 컨트롤러 생성
func NewController(config ControllerConfig) (Controller, error) {
	switch config.Protocol {
	case "onvif", "ptz":
		return NewOnvifPTZ(config.Host, config.Port, config.Username, config.Password), nil
	case "isapi", "hikvision":
		return NewHikvisionPTZ(config.Host, config.Port, config.Username, config.Password), nil
	default:
		// 기본값은 ONVIF (하위 호환성)
		return NewOnvifPTZ(config.Host, config.Port, config.Username, config.Password), nil
	}
}
