# Multi-stage build for MediaMTX with PTZ support
FROM golang:1.25-alpine3.22 AS builder

# Install build dependencies
RUN apk add --no-cache git make

# Set working directory
WORKDIR /workspace

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Generate required files
RUN go generate ./...

# Build the application
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o mediamtx

# Final stage - minimal image
FROM alpine:3.22

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    ffmpeg

# Create non-root user
RUN addgroup -g 1000 mediamtx && \
    adduser -D -u 1000 -G mediamtx mediamtx

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /workspace/mediamtx /app/mediamtx

# Copy configuration file
COPY mediamtx.yml /app/mediamtx.yml

# Change ownership
RUN chown -R mediamtx:mediamtx /app

# Switch to non-root user
USER mediamtx

# Expose ports
# 8889 - WebRTC/Dashboard
# 9997 - API
# 8888 - HLS
# 8554 - RTSP
# 1935 - RTMP
# 8890 - SRT
EXPOSE 8889 9997 8888 8554 1935 8890

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:9997/v3/config/global || exit 1

# Run the application
ENTRYPOINT ["/app/mediamtx"]
