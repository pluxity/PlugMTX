# MediaMTX Maximum Performance Tuning Plan
**Date**: 2025-12-04
**Server**: 27.102.205.67
**Environment**: 64 RTSP streams, WebRTC streaming
**Current State**: Optimized (NACK removed, mDNS disabled, RTCP 3s, buffer pooling)

---

## Executive Summary

**Current Performance**:
- CPU: 66.57s / 180s (36.95%)
- Memory: 28.31 MB
- Goroutines: 2,688
- Build ID: d3e67514e17069eda1ffba5c706b9f68dfa65789

**Target Performance** (Maximum Tuning):
- CPU: <25% (-32% improvement)
- Memory: <20 MB (-29% improvement)
- Goroutines: <2,000 (-26% improvement)

**Status**: Ready for aggressive optimization phase

---

## Current Bottleneck Analysis

### 1. CPU Hotspots (Flat Time)

| Function | Time | % | Category | Optimization Potential |
|----------|------|---|----------|----------------------|
| syscall.Syscall6 | 23.73s | 35.65% | Network I/O | ⭐⭐⭐ High (packet batching) |
| crypto/sha1.blockAVX2 | 2.19s | 3.29% | SRTP HMAC | ⭐⭐ Medium (GCM cipher) |
| crypto/sha1.blockGeneric | 2.03s | 3.05% | SRTP HMAC | ⭐⭐ Medium (GCM cipher) |
| runtime.findObject | 1.60s | 2.40% | GC | ⭐⭐ Medium (reduce allocations) |
| runtime.memmove | 1.27s | 1.91% | Memory | ⭐⭐ Medium (zero-copy) |
| runtime.scanobject | 0.99s | 1.49% | GC | ⭐ Low (GOGC tuning) |

**Total Addressable CPU**: ~30s (45% of total) with aggressive optimizations

### 2. Memory Hotspots (InUse Space)

| Function | Memory | % | Category | Optimization Potential |
|----------|--------|---|----------|----------------------|
| InterleavedFrame | 13.33 MB | 47.09% | RTSP Buffer | ⭐⭐⭐ High (buffer reuse) |
| runtime.malg | 2.05 MB | 7.24% | Goroutine Stacks | ⭐⭐ Medium (pooling) |
| dtls.init.func1 | 1.55 MB | 5.47% | DTLS | ⭐ Low (one-time init) |
| srtp.session.start.func1 | 1.55 MB | 5.47% | SRTP | ⭐ Low (session setup) |
| rtph264.joinFragments | 0.59 MB | 2.09% | H.264 NAL | ⭐⭐ Medium (buffer reuse) |

**Total Addressable Memory**: ~16 MB (56% of total) with buffer optimizations

### 3. Goroutine Analysis

**Current**: 2,688 goroutines
- Per-stream overhead: ~42 goroutines per stream (64 streams)
- Categories:
  - RTSP readers/writers: ~1,280 goroutines
  - WebRTC peer connections: ~1,280 goroutines
  - RTCP senders/receivers: ~128 goroutines

**Optimization Potential**: Worker pool pattern could reduce to ~1,500 goroutines

---

## Maximum Tuning Strategies

### Strategy 1: Packet Batching (sendmmsg) ⭐⭐⭐
**Target**: syscall.Syscall6 (35.65% CPU)

**Problem**:
- Each RTP packet triggers individual syscall.Syscall6 (sendto/recvfrom)
- 64 streams × ~50 packets/sec = 3,200 syscalls/sec
- Context switching overhead dominates CPU

**Solution**: Batch multiple packets into single syscall using sendmmsg()

**Implementation**:
```go
// File: internal/protocols/webrtc/outgoing_track.go (new function)

import (
    "golang.org/x/sys/unix"
)

type packetBatcher struct {
    messages []unix.Mmsghdr
    buffers  [][]byte
    batchSize int
    mu       sync.Mutex
}

func newPacketBatcher(size int) *packetBatcher {
    return &packetBatcher{
        messages:  make([]unix.Mmsghdr, size),
        buffers:   make([][]byte, size),
        batchSize: size,
    }
}

func (pb *packetBatcher) sendBatch(fd int, packets [][]byte) error {
    pb.mu.Lock()
    defer pb.mu.Unlock()

    n := len(packets)
    if n > pb.batchSize {
        n = pb.batchSize
    }

    for i := 0; i < n; i++ {
        pb.messages[i].Msghdr.Iov = &unix.Iovec{
            Base: &packets[i][0],
            Len:  uint64(len(packets[i])),
        }
    }

    sent, err := unix.Sendmmsg(fd, pb.messages[:n], 0)
    if err != nil {
        return err
    }

    // Handle partial sends
    if sent < n {
        return fmt.Errorf("partial send: %d/%d", sent, n)
    }

    return nil
}
```

**Changes Required**:
- Modify `writePacketRTP()` in Pion library to support batching
- Accumulate packets for 1ms before batch send
- Handle UDP socket file descriptor extraction

**Expected Impact**:
- CPU: -10% to -15% (reduce syscalls from 3,200/sec to 320/sec)
- Latency: +1-2ms (batching delay)

**Difficulty**: ⚠️ High (requires Pion library fork)
**Risk**: ⚠️⚠️ High (UDP packet ordering, error handling complexity)
**Priority**: P1 (highest CPU impact)

---

### Strategy 2: SRTP GCM Cipher Mode ⭐⭐
**Target**: crypto/sha1 (6.34% CPU)

**Problem**:
- Current: SRTP using AES-128-CTR + HMAC-SHA1 (separate encryption + authentication)
- SHA1 HMAC requires extra CPU cycles for each packet
- 64 streams × 50 packets/sec = 3,200 HMAC operations/sec

**Solution**: Switch to AES-GCM (combined encryption + authentication)

**Implementation**:
```go
// File: internal/protocols/webrtc/peer_connection.go

func registerInterceptors(mediaEngine *webrtc.MediaEngine, interceptorRegistry *interceptor.Registry) error {
    // Force GCM cipher suite
    err := webrtc.ConfigureTWCCHeaderExtensionReceiverID(mediaEngine, interceptorRegistry)
    if err != nil {
        return err
    }

    return nil
}

func (co *PeerConnection) Start() error {
    settingsEngine := webrtc.SettingEngine{}

    // Set GCM cipher suite priority
    settingsEngine.SetSRTPProtectionProfiles(
        webrtc.SRTP_AEAD_AES_128_GCM,  // GCM first (hardware accelerated)
        webrtc.SRTP_AES128_CM_HMAC_SHA1_80, // Fallback
    )

    // ... rest of setup
}
```

**Expected Impact**:
- CPU: -3% to -5% (hardware AES-GCM acceleration)
- Compatibility: May require browser support check

**Difficulty**: ⚠️ Medium (configuration change only)
**Risk**: ⚠️ Medium (browser compatibility, some older browsers may not support GCM)
**Priority**: P2 (good CPU/risk ratio)

---

### Strategy 3: InterleavedFrame Buffer Pool ⭐⭐⭐
**Target**: InterleavedFrame (47.09% memory - 13.33 MB)

**Problem**:
- RTSP interleaved frames allocate new buffers frequently
- Large frames (up to 64KB for I-frames) cause heap fragmentation
- GC pressure from frequent allocations

**Solution**: Tiered buffer pool with size classes

**Implementation**:
```go
// File: internal/protocols/rtsp/interleaved_frame.go (or create new file)

var interleavedBufferPools = []*sync.Pool{
    // Small packets (RTP): 1500 bytes
    &sync.Pool{
        New: func() interface{} {
            buf := make([]byte, 1500)
            return &buf
        },
    },
    // Medium packets: 8KB
    &sync.Pool{
        New: func() interface{} {
            buf := make([]byte, 8192)
            return &buf
        },
    },
    // Large packets (I-frames): 64KB
    &sync.Pool{
        New: func() interface{} {
            buf := make([]byte, 65536)
            return &buf
        },
    },
}

func getInterleavedBuffer(size int) *[]byte {
    var pool *sync.Pool

    switch {
    case size <= 1500:
        pool = interleavedBufferPools[0]
    case size <= 8192:
        pool = interleavedBufferPools[1]
    default:
        pool = interleavedBufferPools[2]
    }

    bufPtr := pool.Get().(*[]byte)
    buf := *bufPtr
    return &buf[:size] // Slice to actual size
}

func putInterleavedBuffer(buf *[]byte, originalSize int) {
    var pool *sync.Pool

    switch {
    case originalSize <= 1500:
        pool = interleavedBufferPools[0]
    case originalSize <= 8192:
        pool = interleavedBufferPools[1]
    default:
        pool = interleavedBufferPools[2]
    }

    pool.Put(buf)
}
```

**Changes Required**:
- Identify all InterleavedFrame allocation sites
- Replace `make([]byte, size)` with pool Get/Put
- Ensure proper lifecycle management (defer Put())

**Expected Impact**:
- Memory: -8 MB to -10 MB (reduce InterleavedFrame from 13 MB to 3-5 MB)
- CPU: -2% to -3% (reduce GC pressure)

**Difficulty**: ⚠️ Medium (need to track all allocation sites)
**Risk**: ⚠️ Low (sync.Pool is safe, just need proper Put())
**Priority**: P1 (highest memory impact)

---

### Strategy 4: H.264 NAL Fragment Buffer Reuse ⭐⭐
**Target**: rtph264.joinFragments (2.09% memory - 0.59 MB)

**Problem**:
- H.264 fragmented NAL units require buffer joining
- New allocation for each fragmented packet
- Common with large I-frames split across multiple RTP packets

**Solution**: Pre-allocate NAL join buffer per stream

**Implementation**:
```go
// File: internal/protocols/rtsp/h264_reader.go (or similar)

type h264NalJoiner struct {
    buffer    []byte
    offset    int
    maxSize   int
}

func newH264NalJoiner() *h264NalJoiner {
    return &h264NalJoiner{
        buffer:  make([]byte, 128*1024), // 128KB per stream
        maxSize: 128*1024,
    }
}

func (j *h264NalJoiner) joinFragments(fragments [][]byte) []byte {
    j.offset = 0

    for _, frag := range fragments {
        copy(j.buffer[j.offset:], frag)
        j.offset += len(frag)

        if j.offset > j.maxSize-1024 {
            // Reset if approaching limit
            return j.buffer[:j.offset]
        }
    }

    return j.buffer[:j.offset]
}

func (j *h264NalJoiner) reset() {
    j.offset = 0
}
```

**Changes Required**:
- Add h264NalJoiner field to stream context
- Replace rtph264.joinFragments() calls with joiner.joinFragments()
- Ensure reset() after each complete NAL

**Expected Impact**:
- Memory: -0.5 MB (eliminate joinFragments allocations)
- CPU: -0.5% to -1% (reduce GC, faster copy)

**Difficulty**: ⚠️ Low (straightforward buffer management)
**Risk**: ⚠️ Low (buffer size is conservative)
**Priority**: P3 (small but safe improvement)

---

### Strategy 5: Goroutine Worker Pool ⭐⭐
**Target**: runtime.malg (7.24% memory - 2.05 MB), reduce goroutine count

**Problem**:
- 2,688 goroutines = 2,688 stacks = ~2 MB overhead
- Each goroutine stack starts at 2-8 KB
- Many goroutines are idle waiting on channels

**Solution**: Worker pool for packet processing

**Implementation**:
```go
// File: internal/core/worker_pool.go (new file)

type WorkerPool struct {
    workers   int
    taskQueue chan func()
    wg        sync.WaitGroup
}

func NewWorkerPool(workers int, queueSize int) *WorkerPool {
    pool := &WorkerPool{
        workers:   workers,
        taskQueue: make(chan func(), queueSize),
    }

    for i := 0; i < workers; i++ {
        pool.wg.Add(1)
        go pool.worker()
    }

    return pool
}

func (p *WorkerPool) worker() {
    defer p.wg.Done()

    for task := range p.taskQueue {
        task()
    }
}

func (p *WorkerPool) Submit(task func()) {
    p.taskQueue <- task
}

func (p *WorkerPool) Close() {
    close(p.taskQueue)
    p.wg.Wait()
}
```

**Changes Required**:
- Create shared worker pool (256 workers for 64 streams)
- Replace dedicated goroutines for packet sending with pool.Submit()
- Careful with synchronization and context

**Expected Impact**:
- Memory: -1 MB to -1.5 MB (reduce goroutine count to ~1,500)
- CPU: -1% to -2% (better cache locality, reduced context switching)

**Difficulty**: ⚠️⚠️ High (requires architectural refactoring)
**Risk**: ⚠️⚠️ High (potential deadlocks, synchronization issues)
**Priority**: P4 (good concept but high risk)

---

### Strategy 6: GOGC Tuning ⭐
**Target**: GC overhead (runtime.findObject 2.40%, scanobject 1.49%)

**Problem**:
- Default GOGC=100 triggers GC when heap doubles
- With 28 MB heap, GC runs frequently
- GC overhead ~4% CPU

**Solution**: Increase GOGC to reduce GC frequency

**Implementation**:
```bash
# Set environment variable before starting MediaMTX
export GOGC=200  # GC when heap grows 200% (triple size)

# Or in code
debug.SetGCPercent(200)
```

**Expected Impact**:
- CPU: -1% to -2% (fewer GC cycles)
- Memory: +5 MB to +10 MB (trade memory for CPU)

**Difficulty**: ⚠️ Very Low (single config change)
**Risk**: ⚠️ Very Low (easily reversible)
**Priority**: P2 (quick win, very safe)

---

### Strategy 7: Zero-Copy Packet Forwarding ⭐⭐⭐
**Target**: runtime.memmove (1.91% CPU), overall packet path

**Problem**:
- Packet data copied multiple times in pipeline:
  - RTSP source → internal buffer
  - Internal buffer → WebRTC track
  - WebRTC track → UDP socket
- Each copy = memmove() call

**Solution**: Use io.Reader/Writer interface with shared buffers

**Implementation**:
```go
// File: internal/protocols/webrtc/outgoing_track.go

type zerocopySender struct {
    track     *webrtc.TrackLocalStaticRTP
    sharedBuf []byte
}

func (s *zerocopySender) writeRTP(packet *rtp.Packet) error {
    // Serialize directly into shared buffer
    n, err := packet.MarshalTo(s.sharedBuf)
    if err != nil {
        return err
    }

    // Send without additional copy
    return s.track.WriteRTP(s.sharedBuf[:n])
}
```

**Changes Required**:
- Refactor packet pipeline to use slice references instead of copies
- Ensure buffer lifetime is managed correctly
- Use sync.Pool for shared buffers

**Expected Impact**:
- CPU: -3% to -5% (eliminate unnecessary copies)
- Memory: -2 MB (fewer intermediate buffers)

**Difficulty**: ⚠️⚠️⚠️ Very High (requires deep refactoring)
**Risk**: ⚠️⚠️⚠️ Very High (buffer lifetime bugs, race conditions)
**Priority**: P4 (high impact but very risky)

---

## Optimization Roadmap

### Phase 1: Quick Wins (Low Risk, High Impact) - Week 1
**Target**: -10% CPU, -5 MB memory

1. ✅ **GOGC Tuning** (P2)
   - Effort: 5 minutes
   - Impact: -1-2% CPU
   - Risk: Very Low

2. ✅ **SRTP GCM Cipher** (P2)
   - Effort: 2 hours (testing browser compatibility)
   - Impact: -3-5% CPU
   - Risk: Medium (fallback available)

3. ✅ **H.264 NAL Buffer Reuse** (P3)
   - Effort: 4 hours
   - Impact: -0.5 MB, -1% CPU
   - Risk: Low

### Phase 2: Medium Risk (High Impact) - Week 2
**Target**: -15% CPU, -8 MB memory

4. ✅ **InterleavedFrame Buffer Pool** (P1)
   - Effort: 1 day (identify all allocation sites)
   - Impact: -8-10 MB, -2-3% CPU
   - Risk: Low

5. ⚠️ **Packet Batching (sendmmsg)** (P1)
   - Effort: 3-5 days (Pion fork, testing)
   - Impact: -10-15% CPU
   - Risk: High (UDP semantics, error handling)

### Phase 3: High Risk (Research Required) - Week 3+
**Target**: -5% CPU, -1 MB memory

6. ⚠️⚠️ **Goroutine Worker Pool** (P4)
   - Effort: 1 week (architectural refactoring)
   - Impact: -1-2% CPU, -1-1.5 MB
   - Risk: High (synchronization issues)

7. ⚠️⚠️⚠️ **Zero-Copy Forwarding** (P4)
   - Effort: 2 weeks (deep refactoring)
   - Impact: -3-5% CPU, -2 MB
   - Risk: Very High (buffer lifetime bugs)

---

## Expected Performance Targets

### After Phase 1 (Quick Wins):
- CPU: 36.95% → **26-30%** (-19-30% reduction)
- Memory: 28.31 MB → **23-24 MB** (-15-18% reduction)
- Risk: Low
- Timeline: 1 week

### After Phase 2 (Medium Risk):
- CPU: 26-30% → **18-22%** (-40-51% total reduction)
- Memory: 23-24 MB → **15-17 MB** (-40-47% total reduction)
- Risk: Medium
- Timeline: 2-3 weeks

### After Phase 3 (High Risk):
- CPU: 18-22% → **15-18%** (-51-59% total reduction)
- Memory: 15-17 MB → **13-15 MB** (-47-54% total reduction)
- Risk: High
- Timeline: 4-6 weeks

---

## Risk Assessment Matrix

| Strategy | Impact | Difficulty | Risk | Recommendation |
|----------|--------|------------|------|----------------|
| Packet Batching (sendmmsg) | ⭐⭐⭐ | ⚠️⚠️⚠️ | ⚠️⚠️ | **Conditional**: Test in staging first |
| InterleavedFrame Pool | ⭐⭐⭐ | ⚠️⚠️ | ⚠️ | **Recommended**: Safe and high impact |
| SRTP GCM Cipher | ⭐⭐ | ⚠️⚠️ | ⚠️ | **Recommended**: Test browser compatibility |
| H.264 NAL Reuse | ⭐⭐ | ⚠️ | ⚠️ | **Recommended**: Safe improvement |
| GOGC Tuning | ⭐ | ⚠️ | ⚠️ | **Highly Recommended**: Zero risk |
| Worker Pool | ⭐⭐ | ⚠️⚠️⚠️ | ⚠️⚠️ | **Optional**: Good for long-term architecture |
| Zero-Copy | ⭐⭐⭐ | ⚠️⚠️⚠️ | ⚠️⚠️⚠️ | **Research Only**: Too risky for production |

---

## Implementation Priority

### Immediate (Do Now):
1. **GOGC=200** - Zero risk, immediate deployment
2. **InterleavedFrame Buffer Pool** - High impact, low risk

### Short-term (1-2 weeks):
3. **SRTP GCM Cipher** - Test browser compatibility first
4. **H.264 NAL Buffer Reuse** - Safe incremental improvement

### Medium-term (2-4 weeks):
5. **Packet Batching** - Prototype and benchmark in staging

### Long-term (Research):
6. **Worker Pool** - Consider for next major refactor
7. **Zero-Copy** - Research project, not production-ready

---

## Monitoring and Validation

### Performance Metrics to Track:
- **CPU Usage**: Target <25% (current: 36.95%)
- **Memory Usage**: Target <20 MB (current: 28.31 MB)
- **Goroutine Count**: Target <2,000 (current: 2,688)
- **Latency**: P95 latency should remain <100ms
- **Packet Loss**: Should remain <0.1%

### Validation Process:
1. Deploy optimization to staging environment
2. Profile for 3 minutes with 64 streams
3. Compare CPU/memory/goroutines against baseline
4. Monitor for 24 hours for stability
5. Check WebRTC metrics (packet loss, jitter, latency)
6. Deploy to production if all metrics improve

### Rollback Plan:
- Keep previous binary version (`mediamtx_optimized.exe`)
- Have Docker image ready for quick rollback
- Monitor first 1 hour closely after deployment

---

## Conclusion

**Recommended Approach**: Start with Phase 1 (Quick Wins)

**Expected Results**:
- **CPU**: 36.95% → 26-30% after Phase 1, → 18-22% after Phase 2
- **Memory**: 28.31 MB → 23-24 MB after Phase 1, → 15-17 MB after Phase 2
- **Timeline**: 2-3 weeks for significant improvements

**Key Insight**: The biggest remaining bottleneck is syscall overhead (35.65%). Packet batching (sendmmsg) has the highest potential impact but also highest risk. Starting with safer optimizations (GOGC, GCM, buffer pools) can achieve 30-40% improvement with much lower risk.

**Next Steps**:
1. Review this plan and approve Phase 1 strategies
2. Implement GOGC=200 immediately (5 minutes)
3. Begin InterleavedFrame buffer pool implementation (1 day)
4. Test SRTP GCM compatibility with target browsers (2 hours)
