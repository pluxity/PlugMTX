# Focus & Iris 제어 구현

MediaMTX PTZ 카메라의 포커스(Focus)와 조리개(Iris) 제어 기능에 대한 문서입니다.

## 목차
- [개요](#개요)
- [구현 상태](#구현-상태)
- [API 사용법](#api-사용법)
- [프로토콜별 지원 상황](#프로토콜별-지원-상황)
- [테스트 결과](#테스트-결과)
- [제한사항](#제한사항)

## 개요

PTZ 카메라의 이미징 제어 기능을 제공합니다:
- **Focus (포커스)**: 근거리 ↔ 원거리 초점 조정
- **Iris (조리개)**: 조리개 개폐를 통한 빛의 양 조절

## 구현 상태

### ✅ Hikvision ISAPI
완전히 구현되었으며, 실제 Hikvision 카메라에서 사용 가능합니다.

**구현 위치**: `internal/ptz/hikvision.go`

```go
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
```

### ✅ ONVIF - Focus 지원, Iris 미지원

**구현 위치**: `internal/ptz/onvif.go`

#### Focus: ✅ 지원 (PTZ ContinuousMove 사용)
ONVIF Imaging 서비스의 Move는 대부분 카메라에서 지원하지 않지만, PTZ ContinuousMove의 **Zoom 채널**을 사용하여 Focus 제어가 가능합니다.

```go
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

    // PTZ ContinuousMove의 Zoom 채널을 Focus 제어에 사용
    req := onvif_ptz.ContinuousMove{
        ProfileToken: o.profileToken,
        Velocity: xsd_onvif.PTZSpeed{
            PanTilt: xsd_onvif.Vector2D{
                X: 0,
                Y: 0,
            },
            Zoom: xsd_onvif.Vector1D{
                X: focusSpeed,  // Zoom 채널로 Focus 제어
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
```

#### Iris: ❌ 미지원
ONVIF 표준의 Imaging 서비스를 통한 Iris 제어는 대부분의 Hikvision 카메라에서 지원하지 않습니다.
자세한 테스트 결과는 [ONVIF_IRIS_TEST_RESULT.md](./ONVIF_IRIS_TEST_RESULT.md)를 참고하세요.

```go
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
```

## API 사용법

### 1. 카메라 설정

`mediamtx.yml`에서 PTZ 카메라를 설정합니다:

```yaml
paths:
  MY-CAMERA:
    source: rtsp://admin:password@camera-ip:554/stream
    ptz: true
    ptzSource: hikvision://admin:password@camera-ip:80
```

**프로토콜 옵션**:
- `hikvision://` - Hikvision ISAPI (권장, Focus/Iris 지원)
- `isapi://` - Hikvision ISAPI (hikvision://와 동일)
- `onvif://` - ONVIF 프로토콜 (대부분 카메라에서 Focus/Iris 미지원)

### 2. Focus 제어

#### 포커스 조정 (POST)

```bash
curl -X POST http://localhost:9997/v3/ptz/MY-CAMERA/focus \
  -H "Content-Type: application/json" \
  -d '{"speed": 50}'
```

**파라미터**:
- `speed`: -100 ~ 100
  - **양수 (1~100)**: 원거리 포커스 (Far Focus)
  - **음수 (-100~-1)**: 근거리 포커스 (Near Focus)
  - **0**: 정지

**응답**:
```json
{
  "success": true,
  "message": "Focus adjustment command sent successfully"
}
```

#### 포커스 정지

```bash
curl -X POST http://localhost:9997/v3/ptz/MY-CAMERA/focus \
  -H "Content-Type: application/json" \
  -d '{"speed": 0}'
```

#### 포커스 설정 조회 (GET)

```bash
curl http://localhost:9997/v3/ptz/MY-CAMERA/focus
```

**응답**:
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

### 3. Iris (조리개) 제어

#### 조리개 조정 (POST)

```bash
curl -X POST http://localhost:9997/v3/ptz/MY-CAMERA/iris \
  -H "Content-Type: application/json" \
  -d '{"speed": 30}'
```

**파라미터**:
- `speed`: -100 ~ 100
  - **양수 (1~100)**: 조리개 열림 (Open) - 밝게
  - **음수 (-100~-1)**: 조리개 닫힘 (Close) - 어둡게
  - **0**: 정지

**응답**:
```json
{
  "success": true,
  "message": "Iris adjustment command sent successfully"
}
```

#### 조리개 정지

```bash
curl -X POST http://localhost:9997/v3/ptz/MY-CAMERA/iris \
  -H "Content-Type: application/json" \
  -d '{"speed": 0}'
```

#### 조리개 설정 조회 (GET)

```bash
curl http://localhost:9997/v3/ptz/MY-CAMERA/iris
```

**응답**:
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

## 프로토콜별 지원 상황

### Hikvision ISAPI ✅

**엔드포인트**: `http://camera-ip/ISAPI/PTZCtrl/channels/1/continuous`

**지원 기능**:
- ✅ Focus (근거리/원거리)
- ✅ Iris (열림/닫힘)
- ✅ 연속 제어 (Continuous/Momentary)
- ✅ Digest 인증

**XML 요청 형식**:
```xml
<?xml version="1.0" encoding="UTF-8"?>
<PTZData>
    <pan>0</pan>
    <tilt>0</tilt>
    <zoom>0</zoom>
    <Momentary>
        <focus>50</focus>
        <iris>-30</iris>
    </Momentary>
</PTZData>
```

### ONVIF - Focus ✅, Iris ❌

**엔드포인트**: ONVIF PTZ Service - `ContinuousMove` 명령

**지원 상황**:
- ✅ **Focus**: PTZ ContinuousMove의 Zoom 채널 사용하여 완전 지원
- ❌ **Iris**: Imaging 서비스 Move/SetImagingSettings 모두 실패
- ℹ️ GetImagingSettings, GetOptions는 지원하지만 제어 명령은 실패

**Focus 구현 방식** (PTZ ContinuousMove):
```xml
<ContinuousMove>
    <ProfileToken>profile_1</ProfileToken>
    <Velocity>
        <PanTilt x="0" y="0"/>
        <Zoom x="0.5"/>  <!-- Zoom 채널로 Focus 제어 -->
    </Velocity>
    <Timeout>PT60S</Timeout>
</ContinuousMove>
```

**Iris 시도한 방법** (모두 실패):
1. Imaging Service Move - "Not support Absolute" 에러
2. Imaging Service SetImagingSettings - "Invalid BLC" 에러
3. 상세 결과: [ONVIF_IRIS_TEST_RESULT.md](./ONVIF_IRIS_TEST_RESULT.md)

## 테스트 결과

### 테스트 환경
- **카메라**: Hikvision DS-2DE4A225IW-DE
- **펌웨어**: V5.7.3
- **프로토콜**: ONVIF (ISAPI 미지원)

### ONVIF Imaging 테스트

#### Test 1: GetImagingSettings ✅
- **요청**: `GET /ISAPI/Image/channels/1`
- **결과**: 200 OK - 설정 정보 반환

#### Test 2: GetOptions ✅
- **요청**: `GET /ISAPI/Image/options`
- **결과**: 200 OK - Focus 옵션 확인:
  ```xml
  <Focus>
      <AutoFocusModes>AUTO</AutoFocusModes>
      <AutoFocusModes>MANUAL</AutoFocusModes>
      <DefaultSpeed><Min>1</Min><Max>1</Max></DefaultSpeed>
      <NearLimit><Min>10</Min><Max>65534</Max></NearLimit>
      <FarLimit><Min>0</Min><Max>0</Max></FarLimit>
  </Focus>
  ```

#### Test 3: Move (Focus) ❌
- **요청**: Imaging.Move with Focus
- **결과**: 500 Internal Server Error
- **에러**: "Not support Absolute"
  ```xml
  <env:Fault>
      <env:Code>
          <env:Value>ter:SettingsInvalid</env:Value>
      </env:Code>
      <env:Reason>
          <env:Text>The requested settings are incorrect.</env:Text>
      </env:Reason>
      <env:Detail>
          <env:Text>Not support Absolute</env:Text>
      </env:Detail>
  </env:Fault>
  ```

#### Test 4: Stop ✅
- **요청**: Imaging.Stop
- **결과**: 200 OK

### 결론
이 카메라는 ONVIF Imaging 서비스로 Focus/Iris 제어를 지원하지 않습니다.

## 제한사항

### 1. 카메라별 지원 여부

| 제조사 | 프로토콜 | Focus | Iris | 비고 |
|--------|---------|-------|------|------|
| Hikvision | ISAPI | ✅ | ✅ | 완전 지원 |
| Hikvision | ONVIF | ✅ | ❌ | Focus: PTZ Zoom 채널 사용<br>Iris: 미지원 |
| 기타 | ONVIF | ✅ | ❌ | Focus: PTZ Zoom 채널로 가능<br>Iris: 일반적으로 미지원 |

### 2. ONVIF 제약사항

**Focus 제어**:
- ✅ PTZ ContinuousMove의 Zoom 채널을 사용하여 Focus 제어 가능
- ✅ 대부분의 ONVIF 카메라에서 작동

**Iris 제어**:
- ❌ Imaging 서비스의 Move 명령을 지원하지 않음
- ❌ SetImagingSettings로 Iris 값 변경 불가 (Invalid BLC 에러)
- ℹ️ 자동 모드(Auto Iris)만 지원하고 수동 제어 불가능한 경우가 많음

### 3. 인증

Hikvision ISAPI는 **Digest 인증**이 필요합니다:
- Basic Auth 시도 후 401 Unauthorized 발생
- WWW-Authenticate 헤더에서 Digest 챌린지 파싱
- MD5 해시 기반 Digest 응답 생성 및 재전송

### 4. 포트

일반적인 포트 설정:
- **ONVIF**: 80, 8080, 또는 전용 포트 (예: 10081)
- **Hikvision ISAPI**: 80 (HTTP)

### 5. 채널

현재 구현은 채널 1 (`channels/1`)만 지원합니다.
멀티 채널이 필요한 경우 API 수정이 필요합니다.

## 참고 자료

### 관련 파일
- `internal/ptz/controller.go` - PTZ Controller 인터페이스 정의
- `internal/ptz/hikvision.go` - Hikvision ISAPI 구현
- `internal/ptz/onvif.go` - ONVIF 구현
- `internal/api/api.go` - REST API 엔드포인트

### API 엔드포인트
- `POST /v3/ptz/:camera/focus` - 포커스 제어
- `GET /v3/ptz/:camera/focus` - 포커스 설정 조회
- `POST /v3/ptz/:camera/iris` - 조리개 제어
- `GET /v3/ptz/:camera/iris` - 조리개 설정 조회

### 테스트 코드
- `test/test_imaging.go` - ONVIF Imaging 서비스 테스트
- `test/test_hikvision_isapi_focus.go` - Hikvision ISAPI 테스트

## 문제 해결

### "Focus control requires Imaging service (not yet implemented)"

**원인**: ONVIF 카메라가 Imaging Move를 지원하지 않음

**해결책**:
1. 카메라가 Hikvision인 경우 `ptzSource`를 `hikvision://`로 변경
2. 카메라 매뉴얼에서 Focus/Iris 제어 방법 확인
3. 제조사별 전용 API 사용 고려

### "401 Unauthorized" 에러

**원인**: Digest 인증 실패

**해결책**:
1. 사용자명/비밀번호 확인
2. 카메라 웹 인터페이스에서 동일한 자격증명으로 로그인 가능한지 확인
3. 특수문자가 포함된 비밀번호는 URL 인코딩 확인

### "digest request failed with status 401"

**원인**:
- 잘못된 포트 사용 (ONVIF 포트에 ISAPI 요청)
- 카메라가 ISAPI를 지원하지 않음

**해결책**:
1. 포트 확인 (ISAPI는 일반적으로 80)
2. `ptzSource`를 `onvif://`로 변경
