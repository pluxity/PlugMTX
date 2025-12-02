# MediaMTX Docker Deployment Guide

## Overview
This guide covers deploying MediaMTX with PTZ support using Docker and Docker Compose.

## Prerequisites
- Docker Engine 20.10+
- Docker Compose 2.0+
- 2GB RAM minimum
- Network access to camera streams

## Quick Start

### 1. Build and Run with Docker Compose
```bash
# Build and start the container
docker-compose up -d

# View logs
docker-compose logs -f

# Stop the container
docker-compose down
```

### 2. Access Dashboards
Once running, access the web interfaces:

- **WebRTC Dashboard**: http://localhost:8889/dashboard
- **HLS Dashboard**: http://localhost:8889/dashboard-hls
- **PTZ Control**: http://localhost:8889/ptz
- **API**: http://localhost:9997/v3/paths/list

## Configuration

### Method 1: Using docker-compose.yml (Recommended)

The `docker-compose.yml` file provides the easiest deployment method.

**Default Configuration:**
- Network mode: `host` (best for RTSP/WebRTC)
- Configuration: Mounts `mediamtx.yml` from host
- Recordings: Saved to `./recordings` directory
- Auto-restart: Enabled
- Health checks: Enabled

**Edit Configuration:**
```bash
# Edit your mediamtx.yml
nano mediamtx.yml

# Restart container to apply changes
docker-compose restart
```

### Method 2: Using Docker CLI

```bash
# Build the image
docker build -t mediamtx-ptz:latest .

# Run the container
docker run -d \
  --name mediamtx \
  --network host \
  -v $(pwd)/mediamtx.yml:/app/mediamtx.yml:ro \
  -v $(pwd)/recordings:/app/recordings \
  --restart unless-stopped \
  mediamtx-ptz:latest
```

### Method 3: Port Mapping (Alternative to Host Network)

If you cannot use host networking:

```yaml
# In docker-compose.yml, comment out network_mode and use:
ports:
  - "8889:8889"     # WebRTC/Dashboard
  - "9997:9997"     # API
  - "8888:8888"     # HLS
  - "8554:8554"     # RTSP
  - "1935:1935"     # RTMP
  - "8890:8890"     # SRT
  - "8189:8189/udp" # WebRTC UDP
```

## Ports Explanation

| Port | Protocol | Purpose |
|------|----------|---------|
| 8889 | TCP | WebRTC server & Dashboards |
| 9997 | TCP | API server |
| 8888 | TCP | HLS server |
| 8554 | TCP | RTSP server |
| 1935 | TCP | RTMP server |
| 8890 | TCP | SRT server |
| 8189 | UDP | WebRTC media |

## Volume Mounts

### Configuration File
```yaml
volumes:
  - ./mediamtx.yml:/app/mediamtx.yml:ro
```
- Mounts your local `mediamtx.yml` into the container
- Read-only (`:ro`) prevents accidental modification
- Edit locally and restart container to apply changes

### Recordings Directory
```yaml
volumes:
  - ./recordings:/app/recordings
```
- Persists recordings to host filesystem
- Survives container restarts
- Can be backed up easily

## Environment Variables

Override configuration using environment variables:

```yaml
environment:
  - TZ=Asia/Seoul
  - MTX_API=yes
  - MTX_APIADDRESS=:9997
  - MTX_WEBRTC=yes
  - MTX_WEBRTCADDRESS=:8889
  - MTX_HLS=yes
  - MTX_HLSADDRESS=:8888
```

**Common Environment Variables:**
- `TZ` - Timezone (e.g., `Asia/Seoul`, `UTC`, `America/New_York`)
- `MTX_*` - Any MediaMTX config can be overridden with `MTX_` prefix

## PTZ Configuration

PTZ camera credentials are compiled into the binary. To change PTZ cameras:

1. Edit `internal/servers/webrtc/ptz_handler.go`:
```go
var ptzCameras = map[string]PTZConfig{
    "CCTV-TEST1": {
        Host:     "192.168.10.53",
        Username: "admin",
        Password: "live0416",
    },
}
```

2. Rebuild the Docker image:
```bash
docker-compose build
docker-compose up -d
```

## Resource Limits

Configure CPU and memory limits in `docker-compose.yml`:

```yaml
deploy:
  resources:
    limits:
      cpus: '2'        # Maximum 2 CPU cores
      memory: 2G       # Maximum 2GB RAM
    reservations:
      cpus: '0.5'      # Reserve 0.5 CPU
      memory: 512M     # Reserve 512MB RAM
```

Adjust based on:
- Number of streams
- Encoding/transcoding needs
- Concurrent viewers

## Networking

### Host Network Mode (Recommended)
```yaml
network_mode: host
```

**Advantages:**
- Best performance for RTSP/WebRTC
- No NAT issues
- Simplified port management

**Disadvantages:**
- Less isolated
- Ports conflict with host services

### Bridge Network Mode
```yaml
networks:
  - mediamtx-network

networks:
  mediamtx-network:
    driver: bridge
```

**Advantages:**
- Better isolation
- Can run multiple instances

**Disadvantages:**
- May have NAT/firewall issues with WebRTC
- Requires explicit port mapping

## Health Checks

The container includes a health check:

```yaml
healthcheck:
  test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:9997/v3/config/global"]
  interval: 30s
  timeout: 10s
  retries: 3
  start_period: 10s
```

**Check container health:**
```bash
docker ps
# Look for "healthy" status

# View health check logs
docker inspect mediamtx | grep -A 10 Health
```

## Logging

### View Logs
```bash
# Follow logs
docker-compose logs -f

# Last 100 lines
docker-compose logs --tail=100

# Specific service logs
docker logs mediamtx
```

### Log Configuration
```yaml
logging:
  driver: "json-file"
  options:
    max-size: "10m"    # Max log file size
    max-file: "3"      # Keep 3 log files
```

## Updating

### Update MediaMTX
```bash
# Rebuild image
docker-compose build --no-cache

# Recreate container
docker-compose up -d
```

### Update Configuration Only
```bash
# Edit mediamtx.yml
nano mediamtx.yml

# Restart container
docker-compose restart
```

## Troubleshooting

### Container Won't Start
```bash
# Check logs
docker-compose logs

# Check configuration
docker-compose config

# Validate mediamtx.yml
./mediamtx --check-config mediamtx.yml
```

### Cannot Access Dashboards
```bash
# Check if container is running
docker ps | grep mediamtx

# Check if ports are listening
docker exec mediamtx netstat -tulpn

# Test API
curl http://localhost:9997/v3/config/global
```

### WebRTC Not Working
1. **Check host network mode:**
```yaml
network_mode: host
```

2. **Check firewall:**
```bash
# Allow WebRTC ports
sudo ufw allow 8889/tcp
sudo ufw allow 8189/udp
```

3. **Check ICE servers in mediamtx.yml:**
```yaml
webrtcICEServers2:
  - url: stun:stun.l.google.com:19302
```

### Streams Not Appearing
```bash
# Check if cameras are reachable from container
docker exec mediamtx ping 192.168.10.53

# Check API paths list
curl http://localhost:9997/v3/paths/list

# Check mediamtx.yml configuration
docker exec mediamtx cat /app/mediamtx.yml
```

### PTZ Not Working
```bash
# Check PTZ cameras list
curl http://localhost:8889/ptz/cameras

# Test PTZ move
curl -X POST http://localhost:8889/ptz/CCTV-TEST1/move \
  -H "Content-Type: application/json" \
  -d '{"pan":0,"tilt":40,"zoom":0}'

# Check camera connectivity
docker exec mediamtx ping 192.168.10.53
```

## Production Deployment

### Using Docker Swarm

1. **Initialize swarm:**
```bash
docker swarm init
```

2. **Deploy stack:**
```bash
docker stack deploy -c docker-compose.yml mediamtx
```

3. **Scale services:**
```bash
docker service scale mediamtx_mediamtx=3
```

### Using Kubernetes

Convert docker-compose to Kubernetes:
```bash
# Install kompose
curl -L https://github.com/kubernetes/kompose/releases/download/v1.28.0/kompose-linux-amd64 -o kompose
chmod +x kompose
sudo mv kompose /usr/local/bin/

# Convert
kompose convert -f docker-compose.yml

# Apply
kubectl apply -f .
```

### Reverse Proxy (Nginx)

```nginx
# nginx.conf
server {
    listen 80;
    server_name mediamtx.example.com;

    location / {
        proxy_pass http://localhost:8889;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }

    location /api/ {
        proxy_pass http://localhost:9997/;
    }
}
```

## Backup and Restore

### Backup Configuration
```bash
# Backup mediamtx.yml
cp mediamtx.yml mediamtx.yml.backup

# Backup recordings
tar -czf recordings-backup.tar.gz recordings/
```

### Restore
```bash
# Restore configuration
cp mediamtx.yml.backup mediamtx.yml

# Restore recordings
tar -xzf recordings-backup.tar.gz

# Restart container
docker-compose restart
```

## Security Best Practices

1. **Run as non-root user** (already configured in Dockerfile)

2. **Use read-only filesystem where possible:**
```yaml
volumes:
  - ./mediamtx.yml:/app/mediamtx.yml:ro
```

3. **Limit resources:**
```yaml
deploy:
  resources:
    limits:
      cpus: '2'
      memory: 2G
```

4. **Enable authentication in mediamtx.yml:**
```yaml
authMethod: internal
authInternalUsers:
  - user: admin
    pass: your-secure-password
    permissions:
      - action: api
      - action: publish
      - action: read
```

5. **Use secrets for sensitive data:**
```bash
# Create Docker secret
echo "your-password" | docker secret create camera_password -

# Reference in compose
secrets:
  - camera_password
```

## Monitoring

### Prometheus Metrics
```yaml
# In mediamtx.yml
metrics: yes
metricsAddress: :9998
```

```yaml
# In docker-compose.yml
ports:
  - "9998:9998"  # Prometheus metrics
```

### Grafana Dashboard
1. Add Prometheus data source
2. Import MediaMTX dashboard
3. Monitor streams, bandwidth, errors

## Performance Tuning

### For Many Streams (>20)
```yaml
deploy:
  resources:
    limits:
      cpus: '4'
      memory: 4G
```

### For High Bandwidth
```yaml
# In mediamtx.yml
readBufferCount: 2048
```

### For Low Latency
```yaml
# Use WebRTC
webrtc: yes

# Optimize HLS
hlsSegmentDuration: 1s
hlsPartDuration: 200ms
```

## Common Commands Cheat Sheet

```bash
# Start
docker-compose up -d

# Stop
docker-compose down

# Restart
docker-compose restart

# Rebuild
docker-compose build --no-cache

# Logs
docker-compose logs -f

# Shell access
docker exec -it mediamtx sh

# Check health
docker ps

# Update and restart
docker-compose pull && docker-compose up -d

# Clean up
docker-compose down -v
docker system prune -a
```

## Support

For issues:
1. Check logs: `docker-compose logs`
2. Verify configuration: `docker-compose config`
3. Test connectivity: `docker exec mediamtx ping <camera-ip>`
4. Check MediaMTX documentation
5. Report issues on GitHub
