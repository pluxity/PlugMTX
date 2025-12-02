# MediaMTX WebRTC 스트리밍 시스템 기술 문서

## 목차
1. [시스템 개요](#시스템-개요)
2. [전체 아키텍처](#전체-아키텍처)
3. [MediaMTX의 RTSP to WebRTC 변환 프로세스](#mediamtx의-rtsp-to-webrtc-변환-프로세스)
4. [WHIP/WHEP 프로토콜](#whipwhep-프로토콜)
5. [Go 언어와 Pion 라이브러리](#go-언어와-pion-라이브러리)
6. [WebRTC 연결 시퀀스](#webrtc-연결-시퀀스)
7. [PTZ 제어 구조](#ptz-제어-구조)
8. [프로토콜 및 기술 스택](#프로토콜-및-기술-스택)

---

## 시스템 개요

### 주요 구성 요소
- **CCTV 카메라**: RTSP 프로토콜로 영상 스트림 송출
- **MediaMTX 서버**: Go 언어로 개발된 실시간 미디어 서버
  - RTSP 스트림을 수신하여 WebRTC로 변환
  - Pion WebRTC 라이브러리 사용
  - WHIP/WHEP 프로토콜 지원
- **웹 클라이언트**: 브라우저에서 WebRTC로 실시간 스트리밍 시청

### 핵심 기능
1. **프로토콜 변환**: RTSP → WebRTC (Remuxing, Transcoding 없음)
2. **PTZ 제어**: Hikvision 카메라의 Pan-Tilt-Zoom 제어
3. **멀티 프로토콜 지원**: WebRTC, HLS, RTSP 동시 지원
4. **동적 설정**: mediamtx.yml에서 포트 및 경로 설정

---

## 전체 아키텍처

### 아키텍처 다이어그램

```
                                           +-----------------+
                                           |   Web Browser   |
                                           |  (WebRTC Client)|
                                           +-----------------+
                                                   ^      |
                                                   |      | 3. WebRTC (SRTP)
                                                   |      |    Media Stream
                                 2. Signaling      |      |
                               (HTTP/WebSocket)    |      |
                                 Offer/Answer      |      v
+----------------+         +-----------------------------------+         +----------------+
|                |         |                                   |         |                |
|  CCTV Camera   |--RTSP-->|             MediaMTX              |<-ICE/-->|  STUN / TURN   |
| (RTSP Source)  | (RTP)   | (RTSP Client <--+--> WebRTC Server)|<--STUN->|     Server     |
|                |         |                                   |         |                |
+----------------+         +-----------------------------------+         +----------------+
      |                                      ^
      | 1. RTSP Media Stream                 | 4. ICE Candidate Exchange
      | (H.264/H.265, AAC etc.)              | (via Signaling Server)
      +--------------------------------------+
```

### 데이터 흐름 설명
1. **CCTV → MediaMTX**: RTSP 프로토콜을 통해 암호화되지 않은 RTP 미디어 스트림을 전송
2. **Web Browser ↔ MediaMTX**: HTTP/WebSocket을 통해 WebRTC 연결 설정을 위한 시그널링(Offer/Answer, ICE Candidates)을 수행
3. **MediaMTX → Web Browser**: ICE를 통해 설정된 최적의 경로로, 암호화된 SRTP 미디어 스트림을 전송

---

## MediaMTX의 RTSP to WebRTC 변환 프로세스

MediaMTX를 '**프로토콜 브리지(Protocol Bridge)**' 또는 '**미디어 게이트웨이(Media Gateway)**'로 개념화할 수 있습니다. 한쪽에서는 RTSP 클라이언트 역할을, 다른 쪽에서는 WebRTC 서버 역할을 동시에 수행합니다.

### 1. RTSP 스트림 수신 (Ingestion)

**역할**: MediaMTX는 CCTV 카메라에 대해 **RTSP 클라이언트**처럼 동작합니다.

**프로세스**:
1. 설정된 경로(`paths`)를 통해 MediaMTX가 CCTV 카메라의 RTSP 서버에 연결을 요청합니다 (`DESCRIBE`, `SETUP`, `PLAY`)
2. CCTV는 H.264/H.265 영상, AAC/Opus 오디오 등의 미디어 데이터를 RTP 패킷에 담아 MediaMTX로 전송합니다
3. MediaMTX는 이 RTP 패킷들을 수신하여 내부적으로 미디어 트랙(Track)을 구성하고 스트림을 유지합니다

### 2. WebRTC 클라이언트 연결 및 시그널링 (Signaling)

**역할**: 웹 브라우저(클라이언트)에 대해서는 **WebRTC 서버**처럼 동작합니다. MediaMTX는 자체적으로 HTTP/WebSocket 기반의 간단한 시그널링 서버를 내장하고 있습니다.

**프로세스**:
1. 사용자가 웹 페이지에 접속하면, 웹 클라이언트(JavaScript)는 MediaMTX의 특정 HTTP 엔드포인트로 WebRTC 연결을 요청합니다
2. **Offer/Answer 교환**:
   - **클라이언트 → MediaMTX (Offer)**: 클라이언트는 자신이 수신할 수 있는 코덱, 해상도 등의 정보를 담은 `SDP Offer`를 생성하여 MediaMTX에 전송합니다
   - **MediaMTX → 클라이언트 (Answer)**: MediaMTX는 수신 중인 RTSP 스트림의 코덱 정보(예: H.264, AAC)와 자신의 네트워크 정보를 바탕으로 `SDP Answer`를 생성하여 클라이언트에 응답합니다

### 3. NAT 통과 및 P2P 연결 수립 (ICE, STUN, TURN)

**역할**: MediaMTX와 클라이언트가 서로의 실제 IP 주소와 포트를 찾고, 방화벽/NAT 환경을 극복하여 데이터를 주고받을 경로를 설정합니다.

**프로세스**:
1. Offer/Answer 과정에서 교환된 SDP에는 각자의 `ICE Candidate`(네트워크 주소 후보) 정보가 포함됩니다
2. 양측은 STUN 서버를 통해 자신의 공인 IP 주소를 확인하고, 이 정보를 ICE Candidate으로 교환하여 직접 통신(P2P)을 시도합니다
3. 직접 연결이 실패하면, TURN 서버를 통해 미디어 데이터를 중계(Relay)하는 경로를 확보합니다
4. 가장 효율적인 경로가 선택되면 WebRTC 데이터 전송 채널(SRTP)이 열립니다

### 4. 미디어 데이터 변환 및 전송 (Remuxing & Streaming)

**역할**: 수신한 RTSP 스트림의 미디어 데이터를 WebRTC 규격에 맞게 **재패키징(Remuxing)**하여 전송합니다.

**프로세스**:
1. MediaMTX는 1번 단계에서 수신한 RTP 패킷에서 순수 미디어 데이터(H.264 NAL Unit, AAC ADTS 등)를 추출합니다
2. 이 데이터를 WebRTC에서 사용하는 **SRTP(Secure RTP)** 패킷으로 다시 포장합니다
3. **중요**: 만약 CCTV의 코덱(예: H.264)을 웹 브라우저가 지원한다면, 영상/음성을 다시 인코딩하는 무거운 작업(**Transcoding**) 없이 단순히 컨테이너만 바꿔주는 **Remuxing**을 수행하므로 CPU 사용량이 매우 낮고 지연 시간이 짧습니다
4. 생성된 SRTP 패킷은 3번 단계에서 수립된 경로를 통해 웹 클라이언트로 실시간 전송됩니다

---

## WHIP/WHEP 프로토콜

### 전통적인 WebRTC 시그널링의 한계

전통적인 WebRTC 시그널링은 정해진 표준이 없습니다. 개발자들은 보통 WebSocket, SIP, 또는 직접 만든 HTTP 기반 프로토콜을 사용하여 SDP Offer/Answer와 ICE Candidate 정보를 교환해야 했습니다. 이는 클라이언트와 서버가 서로의 시그널링 방식을 정확히 알고 구현해야만 통신이 가능한 구조였습니다.

### WHIP/WHEP 개요

**WHIP/WHEP**는 이 시그널링 과정을 표준화한 프로토콜입니다. 복잡한 WebSocket 연결 대신 단순한 HTTP POST/PATCH 요청을 사용합니다.

#### WHIP (WebRTC-HTTP Ingestion Protocol)

스트림을 **송출(Publishing)**하기 위한 프로토콜입니다.

- **흐름**: 클라이언트가 자신의 미디어 정보(SDP Offer)를 담아 MediaMTX의 WHIP 엔드포인트로 `HTTP POST` 요청을 보냅니다
- **서버 응답**: MediaMTX는 이 요청을 받아 자신의 미디어 정보(SDP Answer)를 `201 Created` 응답 본문에 담아 회신합니다. 이 응답에는 해당 세션을 제어할 수 있는 고유 URL이 포함됩니다
- **장점**: 어떤 클라이언트든 이 표준만 따르면 MediaMTX와 같은 WHIP 서버로 손쉽게 스트림을 보낼 수 있습니다

#### WHEP (WebRTC-HTTP Egress Protocol)

스트림을 **수신(Playback)**하기 위한 프로토콜입니다.

- **흐름**: WHIP과 유사하게, 스트림을 보고 싶은 클라이언트가 WHEP 엔드포인트로 `HTTP POST` 요청을 보냅니다
- **서버 응답**: MediaMTX는 현재 송출되고 있는 스트림에 대한 정보를 SDP Answer에 담아 응답하고, 클라이언트는 이를 통해 스트림 수신을 시작합니다

### 핵심적인 차이점

| 구분 | 전통적인 시그널링 | WHIP/WHEP |
|------|------------------|-----------|
| **프로토콜** | 표준 없음 (WebSocket, 커스텀 HTTP 등) | **표준 HTTP 기반** (POST, PATCH, DELETE) |
| **상호운용성** | 낮음 (클라이언트-서버 간 강한 종속성) | **높음** (표준을 따르는 모든 클라이언트/서버와 호환) |
| **구현 복잡도** | 높음 (상태 관리, 재연결 등 직접 구현) | **낮음** (단순한 Request-Response 모델) |

---

## Go 언어와 Pion 라이브러리

### Go 언어의 장점

1. **탁월한 동시성(Concurrency) 처리**: Go의 핵심 기능인 '고루틴(Goroutine)'은 수천, 수만 개의 동시 연결을 매우 가볍고 효율적으로 처리할 수 있게 해줍니다. 미디어 서버는 다수의 스트림 송출자와 시청자를 동시에 관리해야 하므로 이는 결정적인 장점입니다

2. **강력한 네트워킹 성능**: Go는 표준 라이브러리에서 고성능 네트워킹 기능을 기본으로 제공하여 서버 개발에 매우 적합합니다

3. **빠른 컴파일과 실행 속도**: 컴파일 언어이므로 실행 속도가 빠르며, 단일 실행 파일(Single Binary)로 컴파일되어 배포가 매우 간편합니다

### Pion 라이브러리의 역할

**Pion은 Go 언어로 작성된 순수 WebRTC 구현체**입니다. MediaMTX가 WebRTC의 복잡한 내부 동작을 직접 구현하는 대신, Pion 라이브러리를 사용하여 다음과 같은 핵심 기능들을 처리합니다:

- **ICE, STUN, TURN**: NAT 환경을 넘어 P2P 연결 경로를 찾는 과정
- **SDP**: 세션 정보를 교환하고 협상하는 과정
- **DTLS**: 미디어 채널을 암호화하기 위한 핸드셰이크 과정
- **SRTP/SRTCP**: 암호화된 미디어 데이터를 실제로 전송하는 프로토콜 처리

**결론**: MediaMTX는 Go의 동시성과 성능을 활용해 서버의 뼈대를 만들고, Pion 라이브러리를 통해 복잡한 WebRTC 프로토콜 스택을 처리하는 구조입니다.

---

## WebRTC 연결 시퀀스

### WHEP 기준 상세 연결 과정

클라이언트(시청자)가 MediaMTX로부터 스트림을 수신하는 과정을 단계별로 설명합니다.

| 단계 | 주체 | 동작 | 설명 |
|------|------|------|------|
| **1** | 클라이언트 | `RTCPeerConnection` 객체 생성 | WebRTC 연결을 관리할 객체를 만듭니다 |
| **2** | 클라이언트 | `createOffer()` 호출 및 SDP Offer 생성 | "나는 이런 코덱(H.264, Opus 등)을 받을 수 있어"라는 제안서를 생성합니다 |
| **3** | **클라이언트 → 서버** | **`HTTP POST /whep/path` (SDP Offer 포함)** | **(시그널링)** 생성된 SDP Offer를 MediaMTX의 WHEP 엔드포인트로 전송합니다 |
| **4** | 서버 (MediaMTX) | Pion으로 `RTCPeerConnection` 생성 및 Remote Description 설정 | 클라이언트의 SDP Offer를 받아 자신의 `RTCPeerConnection`에 설정합니다 |
| **5** | 서버 (MediaMTX) | `createAnswer()` 호출 및 SDP Answer 생성 | 클라이언트의 제안에 대한 응답을 생성합니다 |
| **6** | **서버 → 클라이언트** | **`HTTP 201 Created` 응답 (SDP Answer 포함)** | **(시그널링)** 생성된 SDP Answer를 클라이언트에게 보냅니다 |
| **7** | 클라이언트 & 서버 | ICE Candidate 수집 시작 | **(NAT 통과)** 각자 자신의 네트워크 환경 정보를 수집합니다 |
| **8** | **클라이언트 ↔ 서버** | **ICE Candidate 교환** | **(시그널링)** `HTTP PATCH`를 사용해 Candidate들을 교환하여 최적의 경로를 찾습니다 |
| **9** | 클라이언트 & 서버 | **DTLS 핸드셰이크** | **(보안 연결)** 찾아낸 경로로 암호화된 연결을 수립하여 암호화 키를 교환합니다 |
| **10**| **서버 → 클라이언트** | **SRTP 미디어 전송 시작** | **(미디어 스트리밍)** 암호화된 미디어 데이터를 클라이언트로 전송합니다 |

### 시퀀스 다이어그램

```
+-----------+                                                     +-----------------+
|           |                                                     |                 |
| Client    |--(1. HTTP POST with SDP Offer)--------------------->| MediaMTX Server |
| (Browser) |                                                     | (WHEP Endpoint) |
|           |<-(2. HTTP 201 with SDP Answer)----------------------|                 |
|           |                                                     |                 |
|           |                                                     |                 |
|           |--(3. ICE Candidate Exchange via HTTP PATCH)-------->|                 |
|           |<-(4. ICE Candidate Exchange via HTTP PATCH)---------|                 |
|           |                                                     |                 |
|           |                                                     |                 |
|           |============(5. DTLS Handshake)=====================>|                 |
|           |                                                     |                 |
|           |                                                     |                 |
|           |<-----------(6. SRTP Media Stream)-------------------|                 |
+-----------+                                                     +-----------------+
```

---

## PTZ 제어 구조

MediaMTX의 PTZ 제어는 WebRTC 스트림과 **별개의 채널(Out-of-Band)**을 통해 이루어집니다.

### 구조

1. **미디어 채널 (WebRTC/HLS)**:
   - 카메라 → MediaMTX → 시청자
   - 이 채널은 오직 비디오와 오디오 데이터 전송에만 사용됩니다

2. **제어 채널 (HTTP API)**:
   - 사용자/클라이언트 → MediaMTX → 카메라
   - 이 채널은 PTZ 제어 명령(좌우, 상하, 줌 등)을 전달하는 데 사용됩니다

### 동작 흐름

1. **사용자 액션**: 시청자가 웹 인터페이스에서 '오른쪽으로 이동' 버튼을 클릭합니다

2. **API 호출**: 클라이언트는 MediaMTX의 PTZ 제어 API 엔드포인트(예: `/ptz/{camera}/move`)로 `{"pan": 1, "tilt": 0, "zoom": 0}`와 같은 JSON 데이터를 담아 `HTTP POST` 요청을 보냅니다

3. **서버의 중개**: MediaMTX는 이 API 요청을 받습니다

4. **프로토콜 변환**: MediaMTX는 내부적으로 해당 스트림 소스(카메라)가 Hikvision임을 인지하고, 수신한 표준 제어 명령을 **Hikvision 카메라가 이해할 수 있는 고유한 API 형식(CGI 명령어)**으로 변환합니다

5. **카메라에 명령 전달**: MediaMTX 서버가 직접 카메라의 IP 주소로 변환된 제어 명령을 담아 HTTP 요청을 보냅니다

6. **카메라 동작 및 영상 전송**: 카메라는 명령을 수신하여 물리적으로 렌즈를 움직입니다. 이 움직임의 결과는 **카메라가 MediaMTX로 보내는 비디오 스트림에 실시간으로 반영**됩니다

7. **시청자 확인**: 시청자는 WebRTC 또는 HLS 스트림을 통해 카메라가 움직이는 것을 보게 됩니다

### PTZ API 엔드포인트

- `GET /ptz/cameras` - PTZ 지원 카메라 목록 조회
- `POST /ptz/{camera}/move` - 카메라 이동 명령
- `POST /ptz/{camera}/stop` - 카메라 이동 정지
- `GET /ptz/{camera}/status` - 현재 PTZ 상태 조회
- `GET /ptz/{camera}/presets` - 프리셋 목록 조회
- `POST /ptz/{camera}/preset/{presetId}` - 프리셋 위치로 이동

**핵심**: 제어 신호는 API를 통해 전달되고, 그 결과는 비디오 스트림을 통해 확인됩니다. 이처럼 제어와 미디어를 분리하면 각 채널이 자신의 역할에만 집중할 수 있어 시스템의 안정성과 확장성이 높아집니다.

---

## 프로토콜 및 기술 스택

### 사용되는 프로토콜

| 프로토콜 | 역할 | 사용 구간 |
|---------|------|----------|
| **RTSP** | Real Time Streaming Protocol | CCTV → MediaMTX |
| **RTP** | Real-time Transport Protocol | RTSP 내부에서 미디어 전송 |
| **WebRTC** | Web Real-Time Communication | MediaMTX → 웹 클라이언트 |
| **SRTP** | Secure RTP (암호화된 RTP) | WebRTC 미디어 전송 |
| **SDP** | Session Description Protocol | WebRTC 연결 협상 (Offer/Answer) |
| **ICE** | Interactive Connectivity Establishment | NAT 통과 및 최적 경로 탐색 |
| **STUN** | Session Traversal Utilities for NAT | 공인 IP 주소 확인 |
| **TURN** | Traversal Using Relays around NAT | P2P 실패 시 중계 서버 사용 |
| **DTLS** | Datagram Transport Layer Security | WebRTC 암호화 핸드셰이크 |
| **HTTP/HTTPS** | Hypertext Transfer Protocol | 시그널링 (WHIP/WHEP), PTZ API |
| **HLS** | HTTP Live Streaming | 대체 스트리밍 프로토콜 |

### 기술 스택

#### 서버 (MediaMTX)
- **언어**: Go
- **WebRTC 라이브러리**: Pion
- **HTTP 프레임워크**: Gin
- **설정 포맷**: YAML
- **주요 기능**:
  - RTSP 클라이언트
  - WebRTC 서버 (WHIP/WHEP)
  - HLS 서버
  - PTZ 제어 프록시
  - 동적 경로 관리

#### 클라이언트 (웹 브라우저)
- **언어**: JavaScript
- **WebRTC API**: 브라우저 내장 `RTCPeerConnection`
- **HLS 플레이어**: hls.js
- **UI 프레임워크**: HTML5, CSS3
- **프로토콜**: WHEP (WebRTC 수신), HTTP (PTZ API)

### 미디어 코덱

- **비디오**: H.264 (AVC), H.265 (HEVC)
- **오디오**: AAC, Opus, G.711

---

## 결론

MediaMTX는 Go 언어와 Pion 라이브러리를 기반으로 한 강력한 미디어 서버로, RTSP 소스를 WebRTC로 효율적으로 변환합니다. WHIP/WHEP 표준 프로토콜을 지원하여 상호운용성이 뛰어나며, Remuxing 방식으로 낮은 지연시간과 CPU 사용률을 달성합니다. PTZ 제어와 미디어 스트리밍을 분리된 채널로 관리하여 시스템의 확장성과 안정성을 확보했습니다.

이 시스템은 다음과 같은 장점을 제공합니다:
- **낮은 지연시간**: Remuxing으로 트랜스코딩 불필요
- **높은 확장성**: Go의 고루틴으로 수천 개의 동시 연결 처리
- **표준 준수**: WHIP/WHEP로 다양한 클라이언트와 호환
- **유연한 구성**: YAML 설정으로 쉬운 배포 및 관리

---

**문서 버전**: 1.0
**작성일**: 2025-12-01
**기반 기술**: MediaMTX, Go, Pion WebRTC, WHIP/WHEP
