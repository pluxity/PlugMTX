package webrtc

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"

	"github.com/bluenviron/mediamtx/internal/ptz"
)

// PTZConfig holds PTZ configuration for a camera
type PTZConfig struct {
	Host     string `json:"host"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// PTZMoveRequest represents a PTZ move command
type PTZMoveRequest struct {
	Pan  int `json:"pan"`
	Tilt int `json:"tilt"`
	Zoom int `json:"zoom"`
}

// PTZResponse represents a PTZ API response
type PTZResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
}

// PathConfig represents a single path configuration in mediamtx.yml
type PathConfig struct {
	Source string `yaml:"source"`
	PTZ    bool   `yaml:"ptz"`
}

// FullConfig represents the complete mediamtx.yml structure
type FullConfig struct {
	APIAddress    string                `yaml:"apiAddress"`
	HLSAddress    string                `yaml:"hlsAddress"`
	RTSPAddress   string                `yaml:"rtspAddress"`
	WebRTCAddress string                `yaml:"webrtcAddress"`
	Paths         map[string]PathConfig `yaml:"paths"`
}

// loadPTZCameras dynamically loads PTZ camera configurations from mediamtx.yml
func loadPTZCameras() (map[string]PTZConfig, error) {
	configPaths := []string{
		"/app/mediamtx.yml",
		"./mediamtx.yml",
		"/etc/mediamtx.yml",
	}

	var configData []byte
	var err error

	for _, path := range configPaths {
		configData, err = os.ReadFile(path)
		if err == nil {
			break
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to read mediamtx.yml: %w", err)
	}

	var config FullConfig
	if err := yaml.Unmarshal(configData, &config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	ptzCameras := make(map[string]PTZConfig)

	for name, pathConfig := range config.Paths {
		// Only process paths with ptz: true
		if !pathConfig.PTZ {
			continue
		}

		// Parse RTSP URL to extract host, username, password
		parsedURL, err := url.Parse(pathConfig.Source)
		if err != nil {
			continue
		}

		host := parsedURL.Hostname()
		username := ""
		password := ""

		if parsedURL.User != nil {
			username = parsedURL.User.Username()
			password, _ = parsedURL.User.Password()
		}

		if host != "" && username != "" {
			ptzCameras[name] = PTZConfig{
				Host:     host,
				Username: username,
				Password: password,
			}
		}
	}

	return ptzCameras, nil
}

// getPTZConfig retrieves PTZ configuration for a specific camera
func getPTZConfig(cameraName string) (PTZConfig, bool) {
	cameras, err := loadPTZCameras()
	if err != nil {
		return PTZConfig{}, false
	}
	config, exists := cameras[cameraName]
	return config, exists
}

func (s *httpServer) onPTZMove(ctx *gin.Context) {
	cameraName := ctx.Param("camera")

	config, exists := getPTZConfig(cameraName)
	if !exists {
		ctx.JSON(http.StatusNotFound, PTZResponse{
			Success: false,
			Message: fmt.Sprintf("PTZ not configured for camera: %s", cameraName),
		})
		return
	}

	var req PTZMoveRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, PTZResponse{
			Success: false,
			Message: fmt.Sprintf("Invalid request: %v", err),
		})
		return
	}

	ptzController := ptz.NewHikvisionPTZ(config.Host, config.Username, config.Password)
	err := ptzController.Move(req.Pan, req.Tilt, req.Zoom)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, PTZResponse{
			Success: false,
			Message: fmt.Sprintf("PTZ move failed: %v", err),
		})
		return
	}

	ctx.JSON(http.StatusOK, PTZResponse{
		Success: true,
		Message: "PTZ move command sent successfully",
	})
}

func (s *httpServer) onPTZStop(ctx *gin.Context) {
	cameraName := ctx.Param("camera")

	config, exists := getPTZConfig(cameraName)
	if !exists {
		ctx.JSON(http.StatusNotFound, PTZResponse{
			Success: false,
			Message: fmt.Sprintf("PTZ not configured for camera: %s", cameraName),
		})
		return
	}

	ptzController := ptz.NewHikvisionPTZ(config.Host, config.Username, config.Password)
	err := ptzController.Stop()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, PTZResponse{
			Success: false,
			Message: fmt.Sprintf("PTZ stop failed: %v", err),
		})
		return
	}

	ctx.JSON(http.StatusOK, PTZResponse{
		Success: true,
		Message: "PTZ stopped successfully",
	})
}

func (s *httpServer) onPTZStatus(ctx *gin.Context) {
	cameraName := ctx.Param("camera")

	config, exists := getPTZConfig(cameraName)
	if !exists {
		ctx.JSON(http.StatusNotFound, PTZResponse{
			Success: false,
			Message: fmt.Sprintf("PTZ not configured for camera: %s", cameraName),
		})
		return
	}

	ptzController := ptz.NewHikvisionPTZ(config.Host, config.Username, config.Password)
	status, err := ptzController.GetStatus()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, PTZResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to get PTZ status: %v", err),
		})
		return
	}

	ctx.JSON(http.StatusOK, PTZResponse{
		Success: true,
		Data:    status,
	})
}

func (s *httpServer) onPTZPresets(ctx *gin.Context) {
	cameraName := ctx.Param("camera")

	config, exists := getPTZConfig(cameraName)
	if !exists {
		ctx.JSON(http.StatusNotFound, PTZResponse{
			Success: false,
			Message: fmt.Sprintf("PTZ not configured for camera: %s", cameraName),
		})
		return
	}

	ptzController := ptz.NewHikvisionPTZ(config.Host, config.Username, config.Password)
	presets, err := ptzController.GetPresets()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, PTZResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to get presets: %v", err),
		})
		return
	}

	ctx.JSON(http.StatusOK, PTZResponse{
		Success: true,
		Data:    presets,
	})
}

func (s *httpServer) onPTZGotoPreset(ctx *gin.Context) {
	cameraName := ctx.Param("camera")
	presetIDStr := ctx.Param("presetId")

	config, exists := getPTZConfig(cameraName)
	if !exists {
		ctx.JSON(http.StatusNotFound, PTZResponse{
			Success: false,
			Message: fmt.Sprintf("PTZ not configured for camera: %s", cameraName),
		})
		return
	}

	presetID, err := strconv.Atoi(presetIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, PTZResponse{
			Success: false,
			Message: "Invalid preset ID",
		})
		return
	}

	ptzController := ptz.NewHikvisionPTZ(config.Host, config.Username, config.Password)
	err = ptzController.GotoPreset(presetID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, PTZResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to go to preset: %v", err),
		})
		return
	}

	ctx.JSON(http.StatusOK, PTZResponse{
		Success: true,
		Message: fmt.Sprintf("Moving to preset %d", presetID),
	})
}

func (s *httpServer) onPTZList(ctx *gin.Context) {
	ptzCameras, err := loadPTZCameras()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, PTZResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to load PTZ cameras: %v", err),
		})
		return
	}

	cameras := make([]string, 0, len(ptzCameras))
	for name := range ptzCameras {
		cameras = append(cameras, name)
	}

	ctx.JSON(http.StatusOK, PTZResponse{
		Success: true,
		Data:    cameras,
	})
}

// PortConfig holds port configuration for frontend
type PortConfig struct {
	WebRTC int `json:"webrtc"`
	API    int `json:"api"`
	HLS    int `json:"hls"`
	RTSP   int `json:"rtsp"`
}

// ConfigYAML represents the minimal structure we need from mediamtx.yml
type ConfigYAML struct {
	APIAddress    string `yaml:"apiAddress"`
	HLSAddress    string `yaml:"hlsAddress"`
	RTSPAddress   string `yaml:"rtspAddress"`
	WebRTCAddress string `yaml:"webrtcAddress"`
}

// onConfigPorts returns the configured ports for all services
func (s *httpServer) onConfigPorts(ctx *gin.Context) {
	// Parse WebRTC port from current server address
	webrtcPort := parsePort(s.address)

	// Try to read actual configuration from mediamtx.yml
	apiPort, hlsPort, rtspPort, err := readConfigPorts()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, PTZResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to read port configuration: %v", err),
		})
		return
	}

	// Use MediaMTX default ports if not configured
	if apiPort == 0 {
		apiPort = 9997 // MediaMTX default API port
	}
	if hlsPort == 0 {
		hlsPort = 8888 // MediaMTX default HLS port
	}
	if rtspPort == 0 {
		rtspPort = 8554 // MediaMTX default RTSP port
	}

	ctx.JSON(http.StatusOK, PortConfig{
		WebRTC: webrtcPort,
		API:    apiPort,
		HLS:    hlsPort,
		RTSP:   rtspPort,
	})
}

// readConfigPorts reads port configuration from mediamtx.yml
func readConfigPorts() (apiPort, hlsPort, rtspPort int, err error) {
	// Try common config file locations
	configPaths := []string{
		"/app/mediamtx.yml",
		"./mediamtx.yml",
		"/etc/mediamtx.yml",
	}

	var configData []byte
	var readErr error

	for _, path := range configPaths {
		configData, readErr = os.ReadFile(path)
		if readErr == nil {
			break
		}
	}

	if readErr != nil {
		return 0, 0, 0, fmt.Errorf("failed to read mediamtx.yml from any location: %w", readErr)
	}

	var config ConfigYAML
	if err := yaml.Unmarshal(configData, &config); err != nil {
		return 0, 0, 0, fmt.Errorf("failed to parse YAML configuration: %w", err)
	}

	// Parse ports from configuration (0 if not configured - optional services)
	apiPort = parsePort(config.APIAddress)
	hlsPort = parsePort(config.HLSAddress)
	rtspPort = parsePort(config.RTSPAddress)

	return apiPort, hlsPort, rtspPort, nil
}

// parsePort extracts port number from address string (e.g., ":8119" -> 8119)
func parsePort(address string) int {
	if address == "" {
		return 0
	}
	// Remove leading colon if present
	if address[0] == ':' {
		address = address[1:]
	}
	// Try to parse as integer
	port := 0
	fmt.Sscanf(address, "%d", &port)
	return port
}
