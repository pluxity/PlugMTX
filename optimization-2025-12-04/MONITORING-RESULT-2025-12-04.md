# MediaMTX 서버 모니터링 결과
**Date**: 2025-12-04 16:25 KST
**Duration**: 3분 (180초)
**Server**: http://27.102.205.67:9999

---

## ⚠️ 중요: 최적화 미적용 확인

### Build ID 분석
- **서버 실행중인 바이너리**: `d3e67514e17069eda1ffba5c706b9f68dfa65789`
- **새로 빌드한 바이너리**: `CvPkZuY_yhWXQ4hKu3O6` (다름)

**결론**: 서버에서 실행중인 바이너리는 **최적화 적용 전 버전**입니다.

---

## 현재 서버 성능 (최적화 적용 전)

### CPU Profile
- **Total CPU**: 67.48s / 180s = **37.46%**
- **Duration**: 180.12s

**주요 CPU 소비 함수**:
| Function | Time | % | Category |
|----------|------|---|----------|
| syscall.Syscall6 | 24.87s | 36.86% | Network I/O |
| crypto/sha1.blockGeneric | 2.58s | 3.82% | SRTP HMAC |
| crypto/sha1.blockAVX2 | 2.09s | 3.10% | SRTP HMAC |
| runtime.findObject | 1.28s | 1.90% | GC |
| runtime.memmove | 1.25s | 1.85% | Memory copy |
| runtime.scanobject | 1.16s | 1.72% | GC |

**crypto/sha1 합계**: 4.67s (6.92%) ← **SHA1이 여전히 사용됨 (GCM 미적용)**

### Heap Profile
- **Total Memory**: 45.17 MB
- **InterleavedFrame**: 14.87 MB (32.92%)
- **runtime.malg**: 2.56 MB (5.67%)
- **ice.init.func1**: 2.58 MB (5.71%)

### Goroutine Count
- **Total**: 2,682 goroutines

---

## 비교 분석 (vs 이전 프로파일링)

### 이전 프로파일 (2025-12-04 15:48 - profile_max_tuning)
- CPU: 66.57s / 180s = 36.95%
- Memory: 28.31 MB
- Goroutines: 2,688
- Build ID: **d3e67514...** (동일)

### 현재 프로파일 (2025-12-04 16:22)
- CPU: 67.48s / 180s = 37.46%
- Memory: 45.17 MB
- Goroutines: 2,682
- Build ID: **d3e67514...** (동일)

### 변화량
- CPU: +0.91s (+1.4%) - 약간 증가
- Memory: +16.86 MB (+59.6%) - **대폭 증가**
- Goroutines: -6 (-0.2%) - 거의 동일
- **SHA1 사용량**: 4.22s → 4.67s (+10.7%) - 증가

**결론**: 동일한 바이너리가 실행중이지만, 시간대나 트래픽 패턴 차이로 메모리 사용량이 증가했습니다.

---

## 최적화 미적용 원인

### 1. Docker 이미지 미배포
새로 빌드된 Docker 이미지 `mediamtx_max_tuned_2025-12-04.tar`가 서버에 로드되지 않았거나, 기존 컨테이너가 여전히 실행중입니다.

### 2. 확인 방법
서버에서 현재 실행중인 컨테이너 확인:
```bash
docker ps
docker inspect <container_id> | grep -i image
```

---

## 배포 필요 작업

### 1. 기존 컨테이너 중지
```bash
docker stop <container_name>
docker rm <container_name>
```

### 2. 새 이미지 로드
```bash
# 로컬에서 서버로 전송 (이미 전송되었다면 생략)
scp mediamtx_max_tuned_2025-12-04.tar user@27.102.205.67:/path/

# 서버에서 이미지 로드
docker load -i mediamtx_max_tuned_2025-12-04.tar

# 이미지 확인
docker images | grep max-tuned
```

### 3. 새 컨테이너 실행
```bash
docker run -d \
  --name mediamtx-max-tuned \
  -p 8119:8119 \
  -p 8120:8120 \
  -p 8121:8121 \
  -p 8117:8117 \
  -p 8118:8118/udp \
  -p 9999:9999 \
  mediamtx:max-tuned-2025-12-04
```

### 4. 재프로파일링
```bash
# CPU 프로파일 (3분)
curl -o cpu_after_deployment.prof http://27.102.205.67:9999/debug/pprof/profile?seconds=180

# Heap 프로파일
curl -o heap_after_deployment.prof http://27.102.205.67:9999/debug/pprof/heap

# Build ID 확인 (새 바이너리인지 확인)
go tool pprof cpu_after_deployment.prof
# Build ID가 "CvPkZuY..." 로 시작하면 성공
```

---

## 예상 성능 (배포 후)

### GOGC=200 효과
- GC 빈도 감소
- runtime.findObject: 1.28s → **0.8-1.0s** (-20-37%)
- runtime.scanobject: 1.16s → **0.6-0.8s** (-31-48%)
- **총 GC 감소**: ~1s (-1.5% CPU)
- **메모리 증가**: +5-10 MB (GOGC 트레이드오프)

### SRTP GCM 효과
- crypto/sha1 제거
- crypto/sha1: 4.67s → **0-1s** (-79-100%)
- **총 SHA1 감소**: ~3.5-4.5s (-5-7% CPU)

### 예상 총 성과
- **CPU**: 37.46% → **28-30%** (-20-25%)
- **Memory**: 45.17 MB → **50-55 MB** (GOGC로 증가하지만 안정적)
- **Goroutines**: 2,682 (변화 없음)

---

## 검증 체크리스트

배포 후 확인사항:

- [ ] **Build ID 변경 확인**: `CvPkZuY...` 로 시작
- [ ] **SHA1 사용량 감소**: crypto/sha1 < 1s (현재 4.67s)
- [ ] **GC 오버헤드 감소**: findObject + scanobject < 2s (현재 2.44s)
- [ ] **CPU 사용률**: < 30% (현재 37.46%)
- [ ] **메모리 사용량**: 50-55 MB 범위 (GOGC 증가 반영)
- [ ] **연결 안정성**: 64개 스트림 정상 작동
- [ ] **WebRTC 재생**: 브라우저에서 정상 재생

---

## 위험도 평가

### GOGC=200
- **예상 효과**: GC -1-2% CPU
- **부작용**: 메모리 +5-10 MB
- **현재 메모리**: 45 MB → 50-55 MB (충분히 여유 있음)
- **위험도**: ⚠️ Very Low

### SRTP GCM
- **예상 효과**: SHA1 제거 -4-5% CPU
- **부작용**: 브라우저 호환성 (매우 낮은 확률)
- **Fallback**: 자동으로 SHA1으로 폴백
- **위험도**: ⚠️ Low

---

## Next Steps

1. ✅ **Docker 이미지 파일 준비 완료**: `mediamtx_max_tuned_2025-12-04.tar` (15 MB)
2. ⏳ **서버에 이미지 전송** (필요시)
3. ⏳ **기존 컨테이너 중지**
4. ⏳ **새 이미지 로드 및 실행**
5. ⏳ **3분 프로파일링 재실행**
6. ⏳ **최적화 효과 검증**

---

## Summary

**현재 상태**: 최적화 적용 전 바이너리가 실행중
**조치 필요**: 새 Docker 이미지 배포 필요
**예상 개선**: CPU -20-25% (37.46% → 28-30%)
**배포 파일**: `mediamtx_max_tuned_2025-12-04.tar` (준비 완료)
