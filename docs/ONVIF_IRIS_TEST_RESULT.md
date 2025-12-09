# ONVIF Iris 제어 테스트 결과

## 테스트 일자
2025-12-09

## 테스트 대상 카메라
- **제조사**: Hikvision
- **모델**: 14.51.233.129:10081
- **프로토콜**: ONVIF

## 테스트 목적
GetOptions에서 Iris 지원이 확인되었으나, 실제 ONVIF를 통한 Iris 제어가 가능한지 검증

## 테스트 방법

### 1. GetOptions - Iris 지원 확인
**결과**: ✅ 성공
```
Min: -22.0
Max: 0.0
Exposure Modes: [MANUAL, AUTO]
```

### 2. GetImagingSettings - 현재 설정 조회
**결과**: ✅ 성공
```xml
<tt:Exposure>
    <tt:Mode>AUTO</tt:Mode>
    <tt:MinExposureTime>33</tt:MinExposureTime>
    <tt:MaxExposureTime>33333</tt:MaxExposureTime>
    <tt:MinIris>-22</tt:MinIris>
    <tt:MaxIris>0</tt:MaxIris>
</tt:Exposure>
```

### 3. SetImagingSettings - Iris만 변경 (최소 설정)
**시도**: MANUAL 모드 + Iris 값만 전송
**결과**: ❌ 실패
**에러**: `Invalid BLC` (500 Internal Server Error)

```xml
<env:Fault>
    <env:Code>
        <env:Value>env:Sender</env:Value>
        <env:Subcode>
            <env:Value>ter:InvalidArgVal</env:Value>
            <env:Subcode>
                <env:Value>ter:InvalidParameter</env:Value>
            </env:Subcode>
        </env:Subcode>
    </env:Code>
    <env:Reason>
        <env:Text xml:lang="en">the parameter value is illegal</env:Text>
    </env:Reason>
    <env:Detail>
        <env:Text>Invalid BLC</env:Text>
    </env:Detail>
</env:Fault>
```

### 4. SetImagingSettings - 전체 설정 보존하고 Iris만 변경
**시도**: GetImagingSettings로 현재 설정을 받아온 후, Iris만 수정하여 전송
**결과**: ❌ 실패
**에러**: `Invalid BLC` (500 Internal Server Error)

### 5. SetImagingSettings - AUTO 모드 전환 후 재시도
**시도**: 먼저 AUTO 모드로 변경 후, MANUAL + Iris 설정 재시도
**결과**: ❌ 실패 (AUTO 모드 설정은 응답 없음)

### 6. Move - 연속 제어
**시도**: Imaging Service의 Move 명령 사용
**결과**: ❌ 실패
**에러**: `Not support Absolute` (500 Internal Server Error)

```xml
<env:Fault>
    <env:Code>
        <env:Value>env:Sender</env:Value>
        <env:Subcode>
            <env:Value>ter:InvalidArgVal</env:Value>
            <env:Subcode>
                <env:Value>ter:SettingsInvalid</env:Value>
            </env:Subcode>
        </env:Subcode>
    </env:Code>
    <env:Reason>
        <env:Text xml:lang="en">The requested settings are incorrect.</env:Text>
    </env:Reason>
    <env:Detail>
        <env:Text>Not support Absolute</env:Text>
    </env:Detail>
</env:Fault>
```

### 7. SetImagingSettings - BacklightCompensation 제거
**시도**: BacklightCompensation을 빈 구조체로 설정하여 전송
**결과**: ❌ 실패
**에러**: `Invalid BLC` (500 Internal Server Error)

## 결론

### ❌ ONVIF를 통한 Iris 제어는 불가능

1. **GetOptions는 참조 정보만 제공**
   - GetOptions에서 Iris Min/Max 값이 표시되는 것은 카메라가 지원하는 **물리적 범위**를 나타냄
   - ONVIF 프로토콜을 통한 **제어 가능 여부**를 의미하지 않음

2. **모든 제어 시도 실패**
   - SetImagingSettings: 모든 변형 시도에서 "Invalid BLC" 에러 발생
   - Move: "Not support Absolute" 에러 발생

3. **Hikvision의 ONVIF 구현 한계**
   - Hikvision 카메라는 ONVIF 표준을 부분적으로만 구현
   - Iris 제어는 Hikvision 자체 ISAPI 프로토콜로만 가능
   - ONVIF는 조회(GetOptions, GetImagingSettings)만 지원

## 대안

Iris 제어가 필요한 경우 **Hikvision ISAPI** 프로토콜을 사용해야 합니다.

### ISAPI Iris 제어 예시
```http
PUT /ISAPI/System/Video/inputs/channels/1/focus
Content-Type: application/xml

<?xml version="1.0" encoding="UTF-8"?>
<FocusConfiguration>
    <autoIrisEnabled>false</autoIrisEnabled>
    <irisValue>50</irisValue>
</FocusConfiguration>
```

## 최종 상태

| 기능 | ONVIF 지원 | 비고 |
|------|-----------|------|
| Focus | ✅ 지원 | PTZ ContinuousMove의 Zoom 채널 사용 |
| Iris | ❌ 미지원 | Hikvision ISAPI로만 가능 |

## 구현 코드

현재 `internal/ptz/onvif.go`의 Iris() 함수는 명확한 에러 메시지를 반환합니다:

```go
func (o *OnvifPTZ) Iris(speed int) error {
    if err := o.ensureConnected(); err != nil {
        return err
    }

    return fmt.Errorf("iris control not supported via ONVIF on this camera (use Hikvision ISAPI if available)")
}
```
