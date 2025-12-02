# MediaMTX Quick Guide

## 개요

RTSP 카메라 스트림을 WebRTC/HLS로 변환하는 MediaMTX 서버입니다.
- WebRTC/HLS 대시보드로 실시간 스트리밍 확인
- Hikvision 카메라 PTZ 제어 기능

## 배포 방법

### 1. 로컬에서 빌드 및 배포

```powershell
# PowerShell에서 실행
.\deploy.ps1
```

자동으로:
1. Docker 이미지 빌드
2. TAR 파일로 저장
3. 서버(192.168.10.181)로 전송
4. 서버에서 이미지 로드 및 컨테이너 실행

### 2. 배포 완료 확인

```bash
# 서버에서
cd /home/pluxity/docker/aiot
docker compose -f mediamtx.yml ps
docker compose -f mediamtx.yml logs -f
```

## 포트 구성

| 서비스 | 포트 | 용도 |
|--------|------|------|
| WebRTC | 8117 | HTTP 시그널링 + 대시보드 |
| WebRTC UDP | 8118 | ICE/미디어 전송 (UDP) |
| API | 8119 | REST API |
| HLS | 8120 | HLS 스트리밍 |
| RTSP | 8121 | RTSP 프록시 |

## 접속 URL

서버 IP: `192.168.10.181`

```
WebRTC 대시보드:  http://192.168.10.181:8117/dashboard
HLS 대시보드:     http://192.168.10.181:8117/dashboard-hls
PTZ 제어:         http://192.168.10.181:8117/ptz
API:              http://192.168.10.181:8119/v3/paths/list
```

## 주요 파일 구조

```
/home/pluxity/docker/aiot/
├── mediamtx.yml              # Docker Compose 설정
├── mediamtx-conf.yml         # MediaMTX 서버 설정
├── .env                      # 환경 변수 (포트 등)
└── log/
    └── mediamtx/
        └── mediamtx.log      # 애플리케이션 로그
```

## 설정 변경

### 포트 변경

`.env` 파일 수정:

```env
MEDIA_MTX_WEBRTC_PORT=8117
MEDIA_MTX_WEBRTC_UDP_PORT=8118
MEDIA_MTX_API_PORT=8119
MEDIA_MTX_HLS_PORT=8120
MEDIA_MTX_RTSP_PORT=8121
```

재시작:
```bash
docker compose -f mediamtx.yml restart
```

### 카메라 추가/변경

로컬에서 `mediamtx.yml` 파일의 `paths:` 섹션 수정 후 재배포:

```yaml
paths:
  CAMERA-NAME:
    source: rtsp://username:password@ip:port/path
    sourceOnDemand: true
    rtspTransport: tcp
```

```powershell
.\deploy.ps1
```

### PTZ 카메라 추가

`internal/servers/webrtc/ptz_handler.go` 파일의 `ptzCameras` 맵에 추가:

```go
var ptzCameras = map[string]PTZConfig{
    "CAMERA-NAME": {
        Host: "192.168.x.x",
        Username: "admin",
        Password: "password",
    },
}
```

재배포 필요.

## 로그 확인

### 애플리케이션 로그
```bash
tail -f /home/pluxity/docker/aiot/log/mediamtx/mediamtx.log
```

### Docker 컨테이너 로그
```bash
docker logs -f aiot-mediamtx
# 또는
docker compose -f mediamtx.yml logs -f
```

## 문제 해결

### 컨테이너 재시작
```bash
cd /home/pluxity/docker/aiot
docker compose -f mediamtx.yml restart
```

### 컨테이너 상태 확인
```bash
docker compose -f mediamtx.yml ps
docker compose -f mediamtx.yml logs --tail=100
```

### WebRTC 연결 안됨
1. UDP 포트 8118 방화벽 확인
2. `mediamtx-conf.yml`의 `webrtcAdditionalHosts` 확인
3. 브라우저 콘솔에서 에러 확인

### API 응답 없음
```bash
# API 테스트
curl http://192.168.10.181:8119/v3/paths/list
```

### 포트 충돌
```bash
# 포트 사용 확인
sudo lsof -i :8117
sudo netstat -tulpn | grep 8117
```

## 개발 환경

### 로컬 빌드만
```bash
docker build -t mediamtx:local .
```

### 로컬 실행 (개발용)
```bash
go run .
```

브라우저: `http://localhost:8117/dashboard`

## 리소스 제한

기본 설정 (docker-compose.prod.yml):
- CPU: 2 코어 (예약 0.5)
- 메모리: 2GB (예약 512MB)

변경 시 `docker-compose.prod.yml`의 `deploy.resources` 섹션 수정.

## 참고

- MediaMTX 공식 문서: https://mediamtx.org/
- 설정 파일 레퍼런스: https://mediamtx.org/docs/references/configuration-file
- 지원하는 프로토콜: RTSP, RTMP, HLS, WebRTC, SRT
