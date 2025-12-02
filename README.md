# MediaMTX with PTZ Support

**프로덕션 배포용 MediaMTX with Dynamic Dashboards & PTZ Control**

## 🎯 주요 기능

### 대시보드
- ✅ **WebRTC Dashboard** - 실시간 저지연 스트리밍 모니터링
- ✅ **HLS Dashboard** - 브라우저 호환 HTTP 스트리밍
- ✅ **PTZ Control** - 전용 카메라 제어 인터페이스

### PTZ 지원
- ✅ Hikvision ISAPI 통합
- ✅ 8방향 Pan/Tilt 제어
- ✅ Zoom In/Out
- ✅ 속도 조절 (10-100)
- ✅ 프리셋 관리

### 동적 로딩
- ✅ API 기반 스트림 목록 자동 로드
- ✅ 하드코딩 없음
- ✅ 실시간 설정 반영

## 🚀 빠른 배포

### 1. 환경 설정
```powershell
# 환경 변수 파일 생성
Copy-Item .env.example .env
```

### 2. 카메라 설정
`mediamtx.yml` 파일에 카메라 스트림 추가:
```yaml
paths:
  camera1:
    source: rtsp://user:pass@192.168.1.100:554/stream
    sourceOnDemand: yes
    rtspTransport: tcp
```

### 3. 배포 실행
```powershell
.\deploy.ps1
```

## 🌐 접속 URL

| 서비스 | URL |
|--------|-----|
| WebRTC 대시보드 | http://SERVER_IP:8889/dashboard |
| HLS 대시보드 | http://SERVER_IP:8889/dashboard-hls |
| PTZ 제어 | http://SERVER_IP:8889/ptz |
| API | http://SERVER_IP:9997/v3/paths/list |

## 📚 상세 문서

- **[PRODUCTION_DEPLOYMENT.md](PRODUCTION_DEPLOYMENT.md)** - 프로덕션 배포 완전 가이드
- **[DASHBOARD_README.md](DASHBOARD_README.md)** - 대시보드 기능 상세
- **[PTZ_README.md](PTZ_README.md)** - PTZ 기능 상세
- **[QUICK_START.md](QUICK_START.md)** - 5분 빠른 시작

## 📝 라이센스

MIT License

---

**상태**: ✅ Production Ready | **버전**: 1.0.0
