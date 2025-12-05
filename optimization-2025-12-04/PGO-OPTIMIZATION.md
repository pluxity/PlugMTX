# MediaMTX PGO (Profile-Guided Optimization) 배포 가이드
**Date**: 2025-12-04 17:03 KST
**Version**: pgo-optimized-2025-12-04
**Status**: Ready for deployment

---

## PGO (Profile-Guided Optimization)란?

### 개념
Profile-Guided Optimization (PGO)은 실제 프로파일 데이터를 기반으로 컴파일러가 최적화를 수행하는 기법입니다.

### 작동 방식
1. **프로파일 수집**: 실제 프로덕션 환경에서 CPU 프로파일 수집
2. **핫 패스 식별**: 가장 자주 실행되는 코드 경로 파악
3. **최적화**: 컴파일러가 핫 패스를 최적화 (인라이닝, 분기 예측 등)
4. **재빌드**: 최적화된 바이너리 생성

### Go에서의 PGO
- **지원 버전**: Go 1.20+ (현재 사용: Go 1.25.3)
- **사용 방법**: `default.pgo` 파일을 소스 디렉토리에 배치
- **자동 인식**: `go build` 시 자동으로 PGO 적용

---

## 적용된 최적화

### 1. Profile-Guided Optimization (PGO)

**프로파일 소스**: `cpu_ultra_deployed.prof`
- Duration: 180.11s
- Total samples: 46.82s (26.00% CPU)
- Build ID: 0e975f52...

**PGO가 최적화하는 영역**:
1. **Function Inlining**: 자주 호출되는 함수 인라이닝
   - syscall.Syscall6 호출 경로
   - RTP 패킷 처리 경로
   - WebRTC 전송 경로

2. **Branch Prediction**: 분기 예측 최적화
   - 에러 핸들링 경로
   - 조건문 최적화

3. **Register Allocation**: 레지스터 할당 최적화
   - 핫 루프 내 변수들
   - 자주 사용되는 포인터

4. **Code Layout**: 코드 배치 최적화
   - 캐시 효율성 향상
   - 명령어 캐시 미스 감소

**예상 효과**:
- **syscall 경로**: 5-10% 개선 (인라이닝 및 레지스터 최적화)
- **전체 CPU**: 2-5% 추가 개선
- **캐시 효율**: 10-20% 개선

---

## 이전 최적화와의 차이

### Ultra vs PGO

| 항목 | Ultra | PGO | 차이점 |
|------|-------|-----|--------|
| **GOGC** | 300 | 300 | 동일 |
| **SHA1** | 제거 (GCM only) | 제거 (GCM only) | 동일 |
| **컴파일 최적화** | 기본 | **PGO 적용** | ✨ 신규 |
| **인라이닝** | 기본 레벨 | **공격적** | ✨ 개선 |
| **분기 예측** | 기본 | **프로파일 기반** | ✨ 개선 |
| **코드 배치** | 기본 | **최적화됨** | ✨ 개선 |

---

## 예상 성능

### CPU 예상치

**현재 (Ultra)**:
- Total CPU: 46.82s (26.00%)
- syscall.Syscall6: 20.93s (44.70%)
- 기타: ~26s

**PGO 예상**:
- Total CPU: **44-45s (24-25%)**
- syscall.Syscall6: **19-20s (42-44%)** (-5-10% 경로 최적화)
- 기타: ~25s (-3-5% 인라이닝)

**예상 개선**: -2-3% CPU (-1-2s)

### 메모리 예상치

**현재 (Ultra)**: 32.87 MB

**PGO 예상**: **32-33 MB** (거의 동일)

**이유**: PGO는 주로 CPU 최적화에 집중, 메모리는 큰 변화 없음

---

## PGO 최적화 세부 사항

### 1. 핫 패스 (Hot Path) 최적화

**최적화 대상 함수** (프로파일 기반):

#### syscall.Syscall6 (44.70%)
```go
// Before PGO: 일반 함수 호출
func sendPacket(data []byte) {
    syscall.Syscall6(...)
}

// After PGO: 호출 경로 최적화
// - 레지스터에 인자 유지
// - 불필요한 스택 프레임 제거
// - 분기 예측 개선
```

**예상 효과**: 5-10% 빠른 syscall 호출

#### RTP 처리 경로 (cumulative 42-47%)
```go
// github.com/pion/webrtc/v4.(*TrackLocalStaticRTP).writeRTP
// github.com/pion/srtp/v3.(*SessionSRTP).writeRTP
// github.com/pion/webrtc/v4/internal/mux.(*Endpoint).Write

// PGO 최적화:
// - 작은 함수 인라이닝
// - 루프 언롤링
// - 분기 예측 최적화
```

**예상 효과**: 3-5% 빠른 RTP 처리

### 2. 인라이닝 최적화

**Before PGO**:
```go
// 작은 함수도 호출 오버헤드 발생
func (m *Mutex) Lock() {
    // ... 호출 스택 프레임 생성
}

func writePacket(p *Packet) {
    m.Lock()  // 함수 호출 오버헤드
    // ...
}
```

**After PGO**:
```go
// 핫 패스에서 자주 호출되는 함수 인라이닝
func writePacket(p *Packet) {
    // Lock() 코드가 인라인됨
    // 호출 오버헤드 제거
    // 레지스터 최적화 가능
}
```

### 3. 분기 예측 최적화

**Before PGO**:
```go
if err != nil {  // 50/50 예측
    return err  // 실제로는 거의 발생 안 함
}
// 성공 경로
```

**After PGO** (프로파일에서 에러가 거의 없다고 판단):
```go
if err != nil {  // likely(false) 힌트
    return err  // 예측 실패 경로로 배치
}
// 성공 경로 - 파이프라인 최적화
```

---

## 배포 방법

### 1. 기존 컨테이너 중지
```bash
docker stop mediamtx-ultra-optimized
docker rm mediamtx-ultra-optimized
```

### 2. 새 이미지 로드
```bash
# 서버로 전송 (필요시)
scp mediamtx_pgo_optimized_2025-12-04.tar user@27.102.205.67:/path/

# 서버에서 이미지 로드
docker load -i mediamtx_pgo_optimized_2025-12-04.tar

# 이미지 확인
docker images | grep pgo-optimized
```

### 3. 컨테이너 실행
```bash
docker run -d \
  --name mediamtx-pgo-optimized \
  -p 8119:8119 \
  -p 8120:8120 \
  -p 8121:8121 \
  -p 8117:8117 \
  -p 8118:8118/udp \
  -p 9999:9999 \
  mediamtx:pgo-optimized-2025-12-04
```

---

## 프로파일링 및 검증

### 3분 프로파일링
```bash
# CPU 프로파일
curl -o cpu_pgo_deployed.prof http://27.102.205.67:9999/debug/pprof/profile?seconds=180

# Heap 프로파일
curl -o heap_pgo_deployed.prof http://27.102.205.67:9999/debug/pprof/heap

# Goroutine 프로파일
curl -o goroutine_pgo_deployed.prof http://27.102.205.67:9999/debug/pprof/goroutine
```

### 검증 체크리스트

#### 성능 검증
- [ ] **CPU < 25%** (목표: 24-25%)
- [ ] **syscall < 20s** (목표: 19-20s)
- [ ] **Memory 32-33 MB** (거의 동일)
- [ ] **Build ID 변경 확인** (새 바이너리)

#### 기능 검증
- [ ] **64개 스트림 정상 연결**
- [ ] **WebRTC 재생 정상**
- [ ] **비디오 품질 저하 없음**
- [ ] **패킷 손실 < 0.1%**

---

## 예상 결과

### Best Case (예상 확률: 60%)
- **CPU**: 24-25% (-4-8% 개선)
- **syscall**: 19-20s (-5-10% 최적화)
- **Memory**: 32-33 MB (거의 동일)

**평가**: PGO가 예상대로 작동

### Normal Case (예상 확률: 30%)
- **CPU**: 25-26% (-0-4% 개선)
- **syscall**: 20-21s (-0-5% 최적화)
- **Memory**: 32-33 MB

**평가**: PGO 효과 약간 미흡하지만 여전히 개선

### Worst Case (예상 확률: 10%)
- **CPU**: 26% (변화 없음)
- **syscall**: 20.93s (변화 없음)
- **Memory**: 32-33 MB

**평가**: PGO 효과 없음, Ultra와 동일

---

## PGO의 장점과 한계

### 장점 ✅

1. **프로파일 기반 최적화**
   - 실제 워크로드에 맞춤 최적화
   - 추측이 아닌 데이터 기반

2. **자동 최적화**
   - 수동 코드 수정 불필요
   - 컴파일러가 알아서 최적화

3. **안전성**
   - 코드 변경 없음
   - 동작은 동일, 성능만 개선

4. **누적 효과**
   - 다른 최적화와 시너지
   - 점진적 개선

### 한계 ⚠️

1. **개선 폭 제한**
   - 2-5% 수준 (대규모 개선 아님)
   - 이미 최적화된 코드에선 효과 미미

2. **프로파일 의존성**
   - 프로파일 품질에 따라 효과 다름
   - 워크로드가 변하면 재빌드 필요

3. **빌드 시간 증가**
   - PGO 빌드가 더 오래 걸림
   - 개발 사이클에 영향

4. **측정 어려움**
   - 효과가 미묘해서 노이즈와 구분 어려움
   - 여러 번 측정 필요

---

## 누적 최적화 히스토리

| 단계 | CPU | Memory | 주요 최적화 |
|------|-----|--------|-----------|
| **Baseline** | 82.01s (45.56%) | 252.81 MB | - |
| **Phase 1** | 48.42s (26.89%) | 38.61 MB | NACK, mDNS, GOGC=200, GCM+fallback |
| **Ultra** | 46.82s (26.00%) | 32.87 MB | GOGC=300, GCM only |
| **PGO** (예상) | **44-45s (24-25%)** | **32-33 MB** | Profile-Guided Optimization |

**누적 개선 (Baseline → PGO 예상)**:
- CPU: **-45-47%** (-37-38s)
- Memory: **-87%** (-220 MB)

---

## 롤백 계획

### 조건: PGO 효과 없음 또는 성능 저하

**증상**: CPU 여전히 26% 또는 증가

**해결**:
```bash
docker stop mediamtx-pgo-optimized
docker run -d --name mediamtx-ultra-optimized \
  -p 8119:8119 -p 8120:8120 -p 8121:8121 \
  -p 8117:8117 -p 8118:8118/udp -p 9999:9999 \
  mediamtx:ultra-optimized-2025-12-04
```

---

## 향후 최적화 방향

### 1. Iterative PGO
프로파일 → 빌드 → 프로파일 → 재빌드 반복으로 추가 개선

### 2. Multi-Stage PGO
여러 워크로드 프로파일 병합하여 더 나은 최적화

### 3. Packet Batching (근본적 개선)
현재 최대 병목인 syscall 근본 해결 (CPU -10-15%)

---

## 배포 패키지 정보

- **파일명**: `mediamtx_pgo_optimized_2025-12-04.tar`
- **크기**: 15 MB
- **Docker 이미지**: `mediamtx:pgo-optimized-2025-12-04`
- **Base**: Alpine 3.19
- **바이너리**: `mediamtx_pgo_optimized`
- **PGO 프로파일**: `cpu_ultra_deployed.prof` (46.82s, 26.00%)

---

## Summary

### 주요 변경사항
- ✅ **PGO 적용** (실제 프로파일 기반 최적화)
- ✅ **핫 패스 최적화** (syscall, RTP 처리)
- ✅ **인라이닝 개선** (자주 호출되는 함수)
- ✅ **분기 예측** (프로파일 기반)

### 예상 효과
- **CPU**: 26.00% → **24-25%** (-4-8%)
- **Memory**: 32.87 MB → **32-33 MB** (거의 동일)

### 위험도
- **전체**: ⚠️ Very Low (코드 변경 없음)
- **효과**: 2-5% 개선 예상 (보수적)

### Next Steps
1. ✅ 서버에 이미지 로드
2. ✅ 컨테이너 실행
3. ⏳ 3분 프로파일링
4. ⏳ 성능 검증
5. ⏳ Ultra와 비교

---

**배포 준비 완료!** 🚀
**목표 CPU**: 24-25%
**위험도**: Very Low
**기대**: 추가 2-5% 개선
