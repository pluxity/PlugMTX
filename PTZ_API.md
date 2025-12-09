# PTZ API 명세서

MediaMTX PTZ (Pan-Tilt-Zoom) 제어 API 문서입니다. ONVIF 표준 프로토콜을 사용하여 다양한 제조사의 PTZ 카메라를 HTTP API를 통해 제어할 수 있습니다.

## Base URL

```
http://localhost:9997/v3/ptz
```

## 인증

현재 구현에서는 MediaMTX의 기본 인증 설정을 따릅니다.

---

## 엔드포인트 목록

### 1. PTZ 카메라 목록 조회

PTZ 기능이 활성화된 모든 카메라의 목록을 반환합니다.

**Endpoint:** `GET /cameras`

**요청 예시:**
```bash
curl http://localhost:9997/v3/ptz/cameras
```

**응답 예시:**
```json
{
  "success": true,
  "data": [
    "CCTV-TEST-001",
    "CCTV-TEST-002",
    "CCTV-TEST-003"
  ]
}
```

---

### 2. PTZ 이동 제어 (Continuous)

카메라의 Pan, Tilt, Zoom을 연속적으로 제어합니다. 속도 기반으로 동작하며, Stop 명령이 있을 때까지 계속 이동합니다.

**Endpoint:** `POST /:camera/move`

**요청 파라미터:**

| 파라미터 | 타입 | 범위 | 설명 |
|---------|------|------|------|
| pan | int | -100 ~ 100 | 좌우 이동 속도 (음수: 왼쪽, 양수: 오른쪽, 0: 정지) |
| tilt | int | -100 ~ 100 | 상하 이동 속도 (음수: 아래, 양수: 위, 0: 정지) |
| zoom | int | -100 ~ 100 | 줌 속도 (음수: 줌 아웃, 양수: 줌 인, 0: 정지) |

**요청 예시:**
```bash
curl -X POST http://localhost:9997/v3/ptz/CCTV-TEST-001/move \
  -H "Content-Type: application/json" \
  -d '{
    "pan": 50,
    "tilt": 30,
    "zoom": 20
  }'
```

**응답 예시:**
```json
{
  "success": true,
  "message": "Continuous move command sent successfully"
}
```

---

### 3. PTZ 상대 이동 (Relative)

현재 위치에서 상대적인 거리만큼 이동합니다. 목표 위치에 도달하면 자동으로 정지합니다.

**Endpoint:** `POST /:camera/move/relative`

**요청 파라미터:**

| 파라미터 | 타입 | 범위 | 설명 |
|---------|------|------|------|
| pan | int | -100 ~ 100 | 좌우 상대 이동 (음수: 왼쪽, 양수: 오른쪽, 0: 없음) |
| tilt | int | -100 ~ 100 | 상하 상대 이동 (음수: 아래, 양수: 위, 0: 없음) |
| zoom | int | -100 ~ 100 | 줌 상대 이동 (음수: 줌 아웃, 양수: 줌 인, 0: 없음) |

**요청 예시:**
```bash
curl -X POST http://localhost:9997/v3/ptz/CCTV-TEST-001/move/relative \
  -H "Content-Type: application/json" \
  -d '{
    "pan": 50,
    "tilt": 0,
    "zoom": 0
  }'
```

**응답 예시:**
```json
{
  "success": true,
  "message": "Relative move command sent successfully"
}
```

---

### 4. PTZ 이동 정지

현재 진행 중인 모든 PTZ 이동을 즉시 정지합니다.

**Endpoint:** `POST /:camera/stop`

**요청 예시:**
```bash
curl -X POST http://localhost:9997/v3/ptz/CCTV-TEST-001/stop
```

**응답 예시:**
```json
{
  "success": true,
  "message": "PTZ stopped successfully"
}
```

---

### 5. PTZ 상태 조회

현재 카메라의 PTZ 위치 상태를 조회합니다.

**Endpoint:** `GET /:camera/status`

**요청 예시:**
```bash
curl http://localhost:9997/v3/ptz/CCTV-TEST-001/status
```

**응답 예시:**
```json
{
  "success": true,
  "data": {
    "position": {
      "elevation": 102,
      "azimuth": 1345,
      "zoom": 10
    }
  }
}
```

**응답 필드:**

| 필드 | 타입 | 설명 |
|-----|------|------|
| position.elevation | int | 수직 위치 (Tilt) |
| position.azimuth | int | 수평 위치 (Pan) |
| position.zoom | int | 현재 줌 레벨 |

---

## 포커스 제어

### 6. 포커스 조정

카메라의 포커스를 조정합니다.

**Endpoint:** `POST /:camera/focus`

**요청 파라미터:**

| 파라미터 | 타입 | 범위 | 설명 |
|---------|------|------|------|
| speed | int | -100 ~ 100 | 포커스 조정 속도 (음수: near, 양수: far, 0: 정지) |

**요청 예시:**
```bash
curl -X POST http://localhost:9997/v3/ptz/CCTV-TEST-001/focus \
  -H "Content-Type: application/json" \
  -d '{"speed": 50}'
```

**응답 예시:**
```json
{
  "success": true,
  "message": "Focus adjustment command sent successfully"
}
```

**프로토콜 지원**:
- ✅ Hikvision ISAPI: 완전 지원
- ✅ ONVIF: PTZ Zoom 채널 사용하여 지원

---

### 7. 포커스 상태 조회

현재 카메라의 포커스 설정을 조회합니다.

**Endpoint:** `GET /:camera/focus`

**요청 예시:**
```bash
curl http://localhost:9997/v3/ptz/CCTV-TEST-001/focus
```

**응답 예시:**
```json
{
  "success": true,
  "data": {
    "focusStyle": "SEMIAUTOMATIC",
    "focusLimited": 300
  }
}
```

**응답 필드:**

| 필드 | 타입 | 설명 |
|-----|------|------|
| focusStyle | string | 포커스 모드 (MANUAL, SEMIAUTOMATIC, AUTOMATIC) |
| focusLimited | int | 포커스 제한 값 |

---

## 조리개(Iris) 제어

### 8. 조리개 조정

카메라의 조리개(Iris)를 조정합니다.

**Endpoint:** `POST /:camera/iris`

**요청 파라미터:**

| 파라미터 | 타입 | 범위 | 설명 |
|---------|------|------|------|
| speed | int | -100 ~ 100 | 조리개 조정 속도 (음수: 닫기, 양수: 열기, 0: 정지) |

**요청 예시:**
```bash
curl -X POST http://localhost:9997/v3/ptz/CCTV-TEST-001/iris \
  -H "Content-Type: application/json" \
  -d '{"speed": 30}'
```

**응답 예시 (성공):**
```json
{
  "success": true,
  "message": "Iris adjustment command sent successfully"
}
```

**응답 예시 (ONVIF 미지원):**
```json
{
  "success": false,
  "error": "iris control not supported via ONVIF on this camera (use Hikvision ISAPI if available)"
}
```

**프로토콜 지원**:
- ✅ Hikvision ISAPI: 완전 지원
- ❌ ONVIF: 대부분 카메라에서 미지원
- 상세 정보: [docs/FOCUS_IRIS.md](docs/FOCUS_IRIS.md), [docs/ONVIF_IRIS_TEST_RESULT.md](docs/ONVIF_IRIS_TEST_RESULT.md)

---

### 9. 조리개 상태 조회

현재 카메라의 조리개 설정을 조회합니다.

**Endpoint:** `GET /:camera/iris`

**요청 예시:**
```bash
curl http://localhost:9997/v3/ptz/CCTV-TEST-001/iris
```

**응답 예시:**
```json
{
  "success": true,
  "data": {
    "level": 160,
    "maxLimit": 100,
    "minLimit": 0
  }
}
```

**응답 필드:**

| 필드 | 타입 | 설명 |
|-----|------|------|
| level | int | 현재 조리개 레벨 |
| maxLimit | int | 최대 조리개 값 |
| minLimit | int | 최소 조리개 값 |

---

## 프리셋 관리

### 10. 프리셋 목록 조회

저장된 모든 PTZ 프리셋을 조회합니다.

**Endpoint:** `GET /:camera/presets`

**요청 예시:**
```bash
curl http://localhost:9997/v3/ptz/CCTV-TEST-001/presets
```

**응답 예시:**
```json
{
  "success": true,
  "data": [
    {
      "enabled": true,
      "id": 1,
      "name": "Main Entrance",
      "position": {
        "elevation": 1200,
        "azimuth": 3600,
        "zoom": 100
      }
    },
    {
      "enabled": true,
      "id": 2,
      "name": "Parking Lot",
      "position": {
        "elevation": 800,
        "azimuth": 1800,
        "zoom": 50
      }
    }
  ]
}
```

**응답 필드:**

| 필드 | 타입 | 설명 |
|-----|------|------|
| enabled | boolean | 프리셋 활성화 여부 |
| id | int | 프리셋 ID (1-300) |
| name | string | 프리셋 이름 |
| position.elevation | int | 수직 위치 (Tilt) |
| position.azimuth | int | 수평 위치 (Pan) |
| position.zoom | int | 줌 레벨 |

---

### 11. 프리셋으로 이동

저장된 프리셋 위치로 카메라를 이동시킵니다.

**Endpoint:** `POST /:camera/presets/:presetId`

**URL 파라미터:**

| 파라미터 | 타입 | 설명 |
|---------|------|------|
| presetId | int | 이동할 프리셋 ID (1-300) |

**요청 예시:**
```bash
curl -X POST http://localhost:9997/v3/ptz/CCTV-TEST-001/presets/1
```

**응답 예시:**
```json
{
  "success": true,
  "message": "Moving to preset 1"
}
```

---

### 12. 프리셋 생성/수정

현재 PTZ 위치를 프리셋으로 저장합니다.

**Endpoint:** `PUT /:camera/presets/:presetId`

**URL 파라미터:**

| 파라미터 | 타입 | 설명 |
|---------|------|------|
| presetId | int | 저장할 프리셋 ID (1-300) |

**요청 파라미터:**

| 파라미터 | 타입 | 필수 | 설명 |
|---------|------|------|------|
| name | string | 선택 | 프리셋 이름 (미입력시 "Preset{ID}" 형식으로 자동 생성) |

**요청 예시:**
```bash
curl -X PUT http://localhost:9997/v3/ptz/CCTV-TEST-001/presets/1 \
  -H "Content-Type: application/json" \
  -d '{"name": "Main Entrance"}'
```

**응답 예시:**
```json
{
  "success": true,
  "message": "Preset 1 saved as 'Main Entrance'",
  "data": {
    "enabled": true,
    "id": 1,
    "name": "Main Entrance",
    "position": {
      "elevation": 1200,
      "azimuth": 3600,
      "zoom": 100
    }
  }
}
```

---

### 13. 프리셋 삭제

저장된 프리셋을 삭제합니다.

**Endpoint:** `DELETE /:camera/presets/:presetId`

**URL 파라미터:**

| 파라미터 | 타입 | 설명 |
|---------|------|------|
| presetId | int | 삭제할 프리셋 ID (1-300) |

**요청 예시:**
```bash
curl -X DELETE http://localhost:9997/v3/ptz/CCTV-TEST-001/presets/1
```

**응답 예시:**
```json
{
  "success": true,
  "message": "Preset 1 deleted"
}
```

---

## 에러 응답

모든 에러는 다음 형식으로 반환됩니다:

```json
{
  "success": false,
  "message": "에러 설명 메시지"
}
```

### 일반적인 에러 코드

| HTTP 상태 코드 | 설명 |
|---------------|------|
| 400 Bad Request | 잘못된 요청 파라미터 |
| 404 Not Found | PTZ가 설정되지 않은 카메라 또는 존재하지 않는 프리셋 |
| 500 Internal Server Error | 카메라 통신 실패 또는 서버 내부 오류 |

### 에러 예시

**카메라를 찾을 수 없는 경우:**
```json
{
  "success": false,
  "message": "PTZ not configured for camera: CCTV-INVALID"
}
```

**잘못된 프리셋 ID:**
```json
{
  "success": false,
  "message": "Invalid preset ID"
}
```

**카메라 통신 실패:**
```json
{
  "success": false,
  "message": "Failed to get presets: digest request failed with status 401"
}
```

---

## 설정

### mediamtx.yml 설정 예시

PTZ 기능을 사용하려면 `mediamtx.yml` 파일에서 해당 카메라에 PTZ 설정을 추가해야 합니다:

```yaml
paths:
  CCTV-TEST-001:
    source: rtsp://admin:password@192.168.1.100:554/Streaming/Channels/101
    ptz: true          # PTZ 기능 활성화
    ptzPort: 80        # ONVIF 서비스 포트 (일반적으로 80)
```

### 설정 파라미터

| 파라미터 | 타입 | 필수 | 설명 |
|---------|------|------|------|
| source | string | 필수 | RTSP URL (username:password 포함) |
| ptz | boolean | 필수 | PTZ 기능 활성화 여부 (true로 설정) |
| ptzPort | int | 선택 | ONVIF 서비스 포트 (기본값: 80) |

---

## 지원 카메라

ONVIF 표준 프로토콜을 지원하는 모든 PTZ 카메라와 호환됩니다. (Hikvision, Dahua, Axis, Sony 등)

### 구현된 기능

- ✅ Pan/Tilt/Zoom 제어 (ContinuousMove, RelativeMove)
- ✅ 포커스 조정 (Hikvision ISAPI, ONVIF PTZ Zoom 채널)
- ⚠️ 조리개 조정 (Hikvision ISAPI만 지원, ONVIF 미지원)
- ✅ 프리셋 CRUD (생성, 조회, 이동, 삭제)
- ✅ Digest 인증 (Hikvision), WS-Security 인증 (ONVIF)
- ✅ PTZ 상태 조회

---

## 참고사항

1. **연속 이동 제어**: `/move` 엔드포인트는 ONVIF ContinuousMove 명령을 사용합니다. 이동을 멈추려면 반드시 `/stop`을 호출하거나 속도를 0으로 설정해야 합니다.

2. **프리셋 ID 범위**: ONVIF 카메라는 일반적으로 여러 프리셋을 지원합니다. 카메라마다 지원하는 프리셋 개수가 다를 수 있습니다.

3. **동시 제어**: 하나의 카메라에 대해 동시에 여러 제어 명령을 보낼 수 있습니다 (예: pan + tilt + zoom을 동시에).

4. **인증**: 카메라의 RTSP URL에 포함된 username/password가 ONVIF 제어에도 사용됩니다. WS-Security 표준 인증을 사용합니다.

5. **포트 설정**: `ptzPort`를 지정하지 않으면 기본 HTTP 포트(80)가 사용됩니다. ONVIF 서비스 엔드포인트는 `http://[host]:[port]/onvif/device_service`입니다.

6. **Focus/Iris 제어**:
   - **Focus**: Hikvision ISAPI와 ONVIF 모두 지원. ONVIF는 PTZ ContinuousMove의 Zoom 채널을 사용합니다.
   - **Iris**: Hikvision ISAPI만 지원. ONVIF는 Imaging 서비스 한계로 미지원.
   - 상세 정보는 [docs/FOCUS_IRIS.md](docs/FOCUS_IRIS.md) 참고.

---

## 라이센스

MediaMTX PTZ API는 MediaMTX 프로젝트의 일부입니다.
