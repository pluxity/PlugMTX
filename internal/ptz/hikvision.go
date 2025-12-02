package ptz

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// HikvisionPTZ handles PTZ control for Hikvision cameras via ISAPI
type HikvisionPTZ struct {
	Host     string
	Username string
	Password string
	client   *http.Client
}

// NewHikvisionPTZ creates a new Hikvision PTZ controller
func NewHikvisionPTZ(host, username, password string) *HikvisionPTZ {
	return &HikvisionPTZ{
		Host:     host,
		Username: username,
		Password: password,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Move performs continuous PTZ movement
// pan: -100 to 100 (negative=left, positive=right, 0=stop)
// tilt: -100 to 100 (negative=down, positive=up, 0=stop)
// zoom: -100 to 100 (negative=zoom out, positive=zoom in, 0=stop)
func (h *HikvisionPTZ) Move(pan, tilt, zoom int) error {
	xmlData := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<PTZData>
    <pan>%d</pan>
    <tilt>%d</tilt>
    <zoom>%d</zoom>
</PTZData>`, pan, tilt, zoom)

	url := fmt.Sprintf("http://%s/ISAPI/PTZCtrl/channels/1/continuous", h.Host)
	return h.sendRequest("PUT", url, xmlData)
}

// Stop stops all PTZ movement
func (h *HikvisionPTZ) Stop() error {
	return h.Move(0, 0, 0)
}

// GetStatus gets current PTZ status
func (h *HikvisionPTZ) GetStatus() (string, error) {
	url := fmt.Sprintf("http://%s/ISAPI/PTZCtrl/channels/1/status", h.Host)
	return h.sendGetRequest(url)
}

// GetPresets gets list of available presets
func (h *HikvisionPTZ) GetPresets() (string, error) {
	url := fmt.Sprintf("http://%s/ISAPI/PTZCtrl/channels/1/presets", h.Host)
	return h.sendGetRequest(url)
}

// GotoPreset moves to a specific preset position
func (h *HikvisionPTZ) GotoPreset(presetID int) error {
	xmlData := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<PTZData>
    <AbsoluteHigh>
        <presetID>%d</presetID>
    </AbsoluteHigh>
</PTZData>`, presetID)

	url := fmt.Sprintf("http://%s/ISAPI/PTZCtrl/channels/1/presets/%d/goto", h.Host, presetID)
	return h.sendRequest("PUT", url, xmlData)
}

// sendRequest sends an HTTP request with digest authentication
func (h *HikvisionPTZ) sendRequest(method, urlStr, body string) error {
	req, err := http.NewRequest(method, urlStr, strings.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/xml")
	req.SetBasicAuth(h.Username, h.Password)

	resp, err := h.client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		// Try with digest auth
		return h.sendDigestRequest(method, urlStr, body, resp)
	}

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

// sendGetRequest sends a GET request and returns the response
func (h *HikvisionPTZ) sendGetRequest(urlStr string) (string, error) {
	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(h.Username, h.Password)

	resp, err := h.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return string(bodyBytes), nil
}

// sendDigestRequest sends a request with digest authentication
func (h *HikvisionPTZ) sendDigestRequest(method, urlStr, body string, authResp *http.Response) error {
	// Parse WWW-Authenticate header
	authHeader := authResp.Header.Get("WWW-Authenticate")
	if authHeader == "" {
		return fmt.Errorf("no WWW-Authenticate header")
	}

	// For simplicity, we'll use a basic digest auth implementation
	// In production, you might want to use a proper digest auth library
	req, err := http.NewRequest(method, urlStr, strings.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create digest request: %w", err)
	}

	req.Header.Set("Content-Type", "application/xml")

	// Parse digest challenge
	digestParams := parseDigestAuth(authHeader)

	// Calculate digest response
	ha1 := md5Hash(h.Username + ":" + digestParams["realm"] + ":" + h.Password)
	ha2 := md5Hash(method + ":" + req.URL.Path)

	response := md5Hash(ha1 + ":" + digestParams["nonce"] + ":" + ha2)

	// Build authorization header
	authHeaderValue := fmt.Sprintf(`Digest username="%s", realm="%s", nonce="%s", uri="%s", response="%s"`,
		h.Username, digestParams["realm"], digestParams["nonce"], req.URL.Path, response)

	req.Header.Set("Authorization", authHeaderValue)

	resp, err := h.client.Do(req)
	if err != nil {
		return fmt.Errorf("digest request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("digest request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

// parseDigestAuth parses the WWW-Authenticate header
func parseDigestAuth(authHeader string) map[string]string {
	params := make(map[string]string)

	// Remove "Digest " prefix
	authHeader = strings.TrimPrefix(authHeader, "Digest ")

	// Split by comma
	parts := strings.Split(authHeader, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		keyValue := strings.SplitN(part, "=", 2)
		if len(keyValue) == 2 {
			key := strings.TrimSpace(keyValue[0])
			value := strings.Trim(strings.TrimSpace(keyValue[1]), `"`)
			params[key] = value
		}
	}

	return params
}

// md5Hash calculates MD5 hash
func md5Hash(text string) string {
	hash := md5.Sum([]byte(text))
	return hex.EncodeToString(hash[:])
}

// ExtractHostFromRTSP extracts host and credentials from RTSP URL
// rtsp://username:password@host:port/path -> host, username, password
func ExtractHostFromRTSP(rtspURL string) (host, username, password string, err error) {
	u, err := url.Parse(rtspURL)
	if err != nil {
		return "", "", "", err
	}

	host = u.Host
	if u.User != nil {
		username = u.User.Username()
		password, _ = u.User.Password()
	}

	return host, username, password, nil
}
