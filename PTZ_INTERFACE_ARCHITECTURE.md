# PTZ 인터페이스 아키텍처 문서

## 개요

MediaMTX의 PTZ (Pan-Tilt-Zoom) 카메라 제어를 위한 인터페이스 기반 아키텍처 구현 문서입니다.
이 아키텍처는 ONVIF와 Hikvision ISAPI 프로토콜을 동적으로 전환할 수 있도록 설계되었습니다.

## 목차

1. [아키텍처 개요](#아키텍처-개요)
2. [주요 컴포넌트](#주요-컴포넌트)
3. [지원 프로토콜](#지원-프로토콜)
4. [설정 방법](#설정-방법)
5. [API 사용법](#api-사용법)
6. [코드 구조](#코드-구조)
7. [구현 세부사항](#구현-세부사항)

## 아키텍처 개요

### 디자인 패턴

- **인터페이스 패턴**: 공통 PTZ Controller 인터페이스 정의
- **팩토리 패턴**: 프로토콜에 따라 적절한 구현체 동적 생성
- **전략 패턴**: 런타임에 PTZ 제어 전략 선택

### 장점

1. **확장성**: 새로운 PTZ 프로토콜 추가 용이
2. **유지보수성**: 각 프로토콜 구현이 독립적
3. **동적 전환**: 카메라별로 다른 프로토콜 사용 가능
4. **하위 호환성**: 기존 `ptz://` 스킴 계속 지원

## 주요 컴포넌트

### 1. PTZ Controller 인터페이스 (`internal/ptz/controller.go`)

모든 PTZ 구현체가 준수해야 하는 공통 인터페이스입니다.

```go
type Controller interface {
    Connect() error
    Move(pan, tilt, zoom int) error
    Stop() error
    GetStatus() (*Status, error)
    GetPresets() ([]Preset, error)
    GotoPreset(presetID int) error
    SetPreset(presetID int, name string) error
    DeletePreset(presetID int) error
    Focus(speed int) error
    Iris(speed int) error
    GetImageSettings() (*ImageSettings, error)
}
```

### 2. ONVIF 구현체 (`internal/ptz/onvif.go`)

ONVIF (Open Network Video Interface Forum) 표준 프로토콜 구현입니다.

**특징:**
- SOAP/XML 기반 통신
- WS-Security 인증
- 범용 IP 카메라 지원
- 표준화된 프로토콜

**지원 기능:**
- ✅ PTZ 이동 (Move)
- ✅ 정지 (Stop)
- ✅ 상태 조회 (GetStatus)
- ✅ 프리셋 관리 (GetPresets, GotoPreset, SetPreset, DeletePreset)
- ⚠️ 포커스 제어 (Imaging 서비스 필요 - 미구현)
- ⚠️ 조리개 제어 (Imaging 서비스 필요 - 미구현)

### 3. Hikvision ISAPI 구현체 (`internal/ptz/hikvision.go`)

Hikvision 카메라 전용 ISAPI (Internet Server Application Programming Interface) 프로토콜 구현입니다.

**특징:**
- HTTP/XML 기반 통신
- Digest 인증 지원
- Hikvision 카메라 특화 기능

**지원 기능:**
- ✅ PTZ 이동 (Move)
- ✅ 정지 (Stop)
- ✅ 상태 조회 (GetStatus)
- ✅ 프리셋 관리 (GetPresets, GotoPreset, SetPreset, DeletePreset)
- ✅ 포커스 제어 (Focus)
- ✅ 조리개 제어 (Iris)
- ✅ 이미지 설정 조회 (GetImageSettings)

### 4. 팩토리 (`internal/ptz/controller.go`)

프로토콜에 따라 적절한 PTZ 컨트롤러 인스턴스를 생성합니다.

```go
func NewController(config ControllerConfig) (Controller, error)
```

### 5. API 레이어 (`internal/api/api.go`)

REST API를 통해 PTZ 제어 기능을 제공합니다.

## 지원 프로토콜

### 1. ONVIF 프로토콜

**URL 스킴:**
- `ptz://user:password@host:port` (기본값, 하위 호환성)
- `onvif://user:password@host:port` (명시적)

**사용 예시:**
```yaml
ptz: ptz://admin:password123@192.168.1.100:80
ptz: onvif://admin:password123@192.168.1.100:80
```

**기본 포트:** 80

### 2. Hikvision ISAPI 프로토콜

**URL 스킴:**
- `isapi://user:password@host:port`
- `hikvision://user:password@host:port`

**사용 예시:**
```yaml
ptz: isapi://admin:pluxity123!@#@192.168.1.101:80
ptz: hikvision://admin:pluxity123!@#@192.168.1.101:80
```

**기본 포트:** 80

## 설정 방법

### mediamtx.yml 설정

```yaml
paths:
  # ONVIF 카메라 예시
  CCTV-ONVIF-001:
    source: rtsp://admin:live0416@192.168.1.100:554/Streaming/Channels/101
    ptz: onvif://admin:password123@192.168.1.100:80

  # Hikvision ISAPI 카메라 예시
  CCTV-HIKVISION-001:
    source: rtsp://admin:live0416@192.168.1.101:554/Streaming/Channels/101
    ptz: isapi://admin:pluxity123!@#@192.168.1.101:80

  # 기본 프로토콜 (ONVIF) 사용
  CCTV-DEFAULT-001:
    source: rtsp://admin:live0416@192.168.1.102:554/Streaming/Channels/101
    ptz: ptz://admin:password@192.168.1.102:80
```

### 특수 문자가 포함된 비밀번호

비밀번호에 `@`, `!`, `#` 등의 특수 문자가 포함되어도 URL 인코딩 없이 **평문 그대로** 사용합니다.

**예시:**
```yaml
ptz: isapi://admin:pluxity123!@#@192.168.1.101:80
```

서버 내부에서 파싱 시 마지막 `@`를 기준으로 분리하므로 비밀번호에 `@`가 포함되어도 정상 동작합니다.

## API 사용법

### 1. PTZ 카메라 목록 조회

**요청:**
```bash
GET /v3/ptz/cameras
```

**응답:**
```json
{
  "success": true,
  "data": ["CCTV-ONVIF-001", "CCTV-HIKVISION-001"]
}
```

### 2. PTZ 이동

**요청:**
```bash
POST /v3/ptz/{camera}/move
Content-Type: application/json

{
  "pan": 50,    // -100 ~ 100 (좌 → 우)
  "tilt": 30,   // -100 ~ 100 (하 → 상)
  "zoom": 0     // -100 ~ 100 (줌 아웃 → 줌 인)
}
```

**응답:**
```json
{
  "success": true,
  "message": "PTZ move command sent successfully"
}
```

### 3. PTZ 정지

**요청:**
```bash
POST /v3/ptz/{camera}/stop
```

**응답:**
```json
{
  "success": true,
  "message": "PTZ stopped successfully"
}
```

### 4. PTZ 상태 조회

**요청:**
```bash
GET /v3/ptz/{camera}/status
```

**응답:**
```json
{
  "success": true,
  "data": {
    "pan": 865.6992,
    "tilt": 900,
    "zoom": 0
  }
}
```

### 5. 프리셋 목록 조회

**요청:**
```bash
GET /v3/ptz/{camera}/presets
```

**응답:**
```json
{
  "success": true,
  "data": [
    {"id": 1, "name": "Entrance"},
    {"id": 2, "name": "Parking Lot"},
    {"id": 3, "name": "Main Gate"}
  ]
}
```

### 6. 프리셋 이동

**요청:**
```bash
POST /v3/ptz/{camera}/presets/{presetId}/goto
```

**응답:**
```json
{
  "success": true,
  "message": "Moved to preset successfully"
}
```

### 7. 프리셋 설정

**요청:**
```bash
POST /v3/ptz/{camera}/presets/{presetId}
Content-Type: application/json

{
  "name": "New Position"
}
```

**응답:**
```json
{
  "success": true,
  "message": "Preset set successfully",
  "data": {
    "id": 10,
    "name": "New Position"
  }
}
```

### 8. 프리셋 삭제

**요청:**
```bash
DELETE /v3/ptz/{camera}/presets/{presetId}
```

**응답:**
```json
{
  "success": true,
  "message": "Preset deleted successfully"
}
```

### 9. 포커스 조정

**요청:**
```bash
POST /v3/ptz/{camera}/focus
Content-Type: application/json

{
  "speed": 50   // -100 ~ 100 (원거리 → 근거리)
}
```

**응답:**
```json
{
  "success": true,
  "message": "Focus adjusted successfully"
}
```

### 10. 조리개 조정

**요청:**
```bash
POST /v3/ptz/{camera}/iris
Content-Type: application/json

{
  "speed": 30   // -100 ~ 100 (닫힘 → 열림)
}
```

**응답:**
```json
{
  "success": true,
  "message": "Iris adjusted successfully"
}
```

### 11. 포커스 설정 조회

**요청:**
```bash
GET /v3/ptz/{camera}/focus
```

**응답:**
```json
{
  "success": true,
  "data": {
    "brightness": 50,
    "contrast": 50,
    "saturation": 50,
    "sharpness": 50
  }
}
```

### 12. 조리개 설정 조회

**요청:**
```bash
GET /v3/ptz/{camera}/iris
```

**응답:**
```json
{
  "success": true,
  "data": {
    "brightness": 50,
    "contrast": 50,
    "saturation": 50,
    "sharpness": 50
  }
}
```

## 코드 구조

```
internal/
├── ptz/
│   ├── controller.go       # 인터페이스 정의 및 팩토리
│   ├── onvif.go           # ONVIF 구현체
│   └── hikvision.go       # Hikvision ISAPI 구현체
└── api/
    └── api.go             # REST API 핸들러
```

## 구현 세부사항

### URL 파싱 로직

`parsePTZURL` 함수는 PTZ URL을 파싱하여 프로토콜, 호스트, 포트, 인증 정보를 추출합니다.

```go
func parsePTZURL(ptzURL string) (protocol, host string, port int, username, password string, err error)
```

**파싱 로직:**

1. **프로토콜 추출**: `://` 앞부분
   - `ptz://` → `"ptz"` (ONVIF 기본값)
   - `onvif://` → `"onvif"`
   - `isapi://` → `"isapi"`
   - `hikvision://` → `"hikvision"`

2. **사용자 정보와 호스트 분리**: 마지막 `@` 기준
   - `admin:pass!@#@host:port`
   - → userinfo: `admin:pass!@#`
   - → hostPort: `host:port`

3. **사용자명과 비밀번호 분리**: 첫 번째 `:` 기준
   - `admin:pass!@#`
   - → username: `admin`
   - → password: `pass!@#`

4. **호스트와 포트 분리**: URL 파서 사용
   - `host:port` → host: `host`, port: `port`

### PTZ 컨트롤러 생성 흐름

```go
// 1. 설정에서 PTZ URL 파싱
protocol, host, port, username, password := parsePTZURL(config.PTZ)

// 2. 팩토리 함수로 컨트롤러 생성
controller := ptz.NewController(ptz.ControllerConfig{
    Protocol: protocol,
    Host:     host,
    Port:     port,
    Username: username,
    Password: password,
})

// 3. 카메라 연결
controller.Connect()

// 4. PTZ 명령 실행
controller.Move(pan, tilt, zoom)
```

### 프로토콜별 구현 차이

| 기능 | ONVIF | Hikvision ISAPI |
|------|-------|-----------------|
| PTZ 이동 | ContinuousMove | PUT /PTZCtrl/channels/1/continuous |
| 정지 | Stop | Move(0, 0, 0) |
| 상태 조회 | GetStatus | GET /PTZCtrl/channels/1/status |
| 프리셋 조회 | GetPresets | GET /PTZCtrl/channels/1/presets |
| 프리셋 이동 | GotoPreset | PUT /PTZCtrl/channels/1/presets/{id}/goto |
| 프리셋 설정 | SetPreset | PUT /PTZCtrl/channels/1/presets/{id} |
| 프리셋 삭제 | RemovePreset | DELETE /PTZCtrl/channels/1/presets/{id} |
| 포커스 | Imaging 서비스 (미구현) | PUT /PTZCtrl/channels/1/continuous |
| 조리개 | Imaging 서비스 (미구현) | PUT /PTZCtrl/channels/1/continuous |

### 인증 방식

**ONVIF:**
- WS-Security (Username Token)
- SOAP 헤더에 인증 정보 포함

**Hikvision ISAPI:**
- HTTP Digest Authentication
- Basic Auth 실패 시 Digest로 재시도

## 테스트

### 단위 테스트

ONVIF 구현체에 대한 테스트는 `internal/ptz/onvif_test.go`에 구현되어 있습니다.

**실행:**
```bash
go test ./internal/ptz/onvif_test.go -v
```

**테스트 항목:**
- Connect: 카메라 연결
- Move: PTZ 이동
- Stop: 정지
- GetStatus: 상태 조회
- GetPresets: 프리셋 목록
- GotoPreset: 프리셋 이동
- SetPreset: 프리셋 설정
- DeletePreset: 프리셋 삭제

### 통합 테스트

실제 카메라로 테스트:

```bash
# 서버 시작
./mediamtx

# PTZ 카메라 목록 확인
curl http://localhost:9997/v3/ptz/cameras

# 상태 조회
curl http://localhost:9997/v3/ptz/CCTV-TEST-001/status

# PTZ 이동
curl -X POST http://localhost:9997/v3/ptz/CCTV-TEST-001/move \
  -H "Content-Type: application/json" \
  -d '{"pan": 50, "tilt": 30, "zoom": 0}'
```

## 트러블슈팅

### 카메라 연결 실패

**문제:** `Failed to create PTZ controller: failed to connect to PTZ camera`

**원인:**
1. 잘못된 IP 주소 또는 포트
2. 네트워크 연결 문제
3. 잘못된 인증 정보
4. 잘못된 프로토콜 선택

**해결:**
1. 카메라 IP와 포트 확인
2. 네트워크 연결 테스트 (`ping`)
3. 웹 브라우저로 카메라 접속 테스트
4. 사용자명/비밀번호 확인
5. 카메라 모델에 맞는 프로토콜 선택

### 특수 문자 비밀번호 문제

**문제:** 비밀번호에 `@` 문자가 포함되어 파싱 오류

**해결:** 서버가 마지막 `@`를 기준으로 파싱하므로 평문 그대로 입력

```yaml
# 올바른 예시
ptz: isapi://admin:password!@#@192.168.1.100:80

# 잘못된 예시 (URL 인코딩 불필요)
ptz: isapi://admin:password!%40%23@192.168.1.100:80
```

### ONVIF 포커스/조리개 제어 불가

**문제:** `focus control requires Imaging service (not yet implemented)`

**원인:** ONVIF Imaging 서비스 미구현

**해결:**
1. Hikvision ISAPI 프로토콜 사용 (`isapi://`)
2. 또는 ONVIF Imaging 서비스 구현 추가

### 프리셋 삭제 실패

**문제:** 특정 프리셋 삭제 시 오류

**원인:** 카메라 펌웨어에서 보호된 프리셋 (예: "Start auto scan")

**해결:** 다른 프리셋 번호 사용 또는 카메라 설정 확인

## 확장 가이드

### 새로운 프로토콜 추가

1. **인터페이스 구현**

`internal/ptz/newprotocol.go` 생성:

```go
package ptz

type NewProtocolPTZ struct {
    Host     string
    Port     int
    Username string
    Password string
}

func NewNewProtocolPTZ(host string, port int, username, password string) *NewProtocolPTZ {
    return &NewProtocolPTZ{
        Host:     host,
        Port:     port,
        Username: username,
        Password: password,
    }
}

// Controller 인터페이스의 모든 메서드 구현
func (n *NewProtocolPTZ) Connect() error { /* ... */ }
func (n *NewProtocolPTZ) Move(pan, tilt, zoom int) error { /* ... */ }
// ... 나머지 메서드
```

2. **팩토리에 추가**

`internal/ptz/controller.go`의 `NewController` 함수 수정:

```go
func NewController(config ControllerConfig) (Controller, error) {
    switch config.Protocol {
    case "onvif", "ptz":
        return NewOnvifPTZ(config.Host, config.Port, config.Username, config.Password), nil
    case "isapi", "hikvision":
        return NewHikvisionPTZ(config.Host, config.Port, config.Username, config.Password), nil
    case "newprotocol":
        return NewNewProtocolPTZ(config.Host, config.Port, config.Username, config.Password), nil
    default:
        return NewOnvifPTZ(config.Host, config.Port, config.Username, config.Password), nil
    }
}
```

3. **URL 파서에 추가**

`internal/api/api.go`의 `parsePTZURL` 함수 수정:

```go
func parsePTZURL(ptzURL string) (protocol string, ...) {
    // ...
    if strings.HasPrefix(ptzURL, "newprotocol://") {
        protocol = "newprotocol"
        restURL = strings.TrimPrefix(ptzURL, "newprotocol://")
    }
    // ...
}
```

4. **테스트 작성**

`internal/ptz/newprotocol_test.go` 생성하여 테스트 작성

## 라이선스 및 크레딧

- **MediaMTX**: https://github.com/bluenviron/mediamtx
- **ONVIF Go Library**: https://github.com/use-go/onvif
- **개발자**: Claude & PluxMTX Team
- **문서 작성일**: 2025-12-09
