// Package api contains the API server.
package api

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gopkg.in/yaml.v3"

	"github.com/bluenviron/mediamtx/internal/auth"
	"github.com/bluenviron/mediamtx/internal/conf"
	"github.com/bluenviron/mediamtx/internal/conf/jsonwrapper"
	"github.com/bluenviron/mediamtx/internal/defs"
	"github.com/bluenviron/mediamtx/internal/logger"
	"github.com/bluenviron/mediamtx/internal/protocols/httpp"
	"github.com/bluenviron/mediamtx/internal/ptz"
	"github.com/bluenviron/mediamtx/internal/recordstore"
	"github.com/bluenviron/mediamtx/internal/servers/hls"
	"github.com/bluenviron/mediamtx/internal/servers/rtmp"
	"github.com/bluenviron/mediamtx/internal/servers/rtsp"
	"github.com/bluenviron/mediamtx/internal/servers/srt"
	"github.com/bluenviron/mediamtx/internal/servers/webrtc"
)

// PTZ related types
type PTZConfig struct {
	Protocol string `json:"protocol"` // 프로토콜: "onvif", "isapi", "hikvision"
	Host     string `json:"host"`
	PTZPort  int    `json:"ptzPort"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type PTZMoveRequest struct {
	Pan  int `json:"pan"`
	Tilt int `json:"tilt"`
	Zoom int `json:"zoom"`
}

type PTZResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
}

type PathConfig struct {
	Source    string `yaml:"source"`
	PTZ       bool   `yaml:"ptz"`       // PTZ enabled/disabled
	PTZSource string `yaml:"ptzSource"` // PTZ URL: onvif://user:pass@host:port or hikvision://user:pass@host:port
}

type FullConfig struct {
	APIAddress    string                `yaml:"apiAddress"`
	HLSAddress    string                `yaml:"hlsAddress"`
	RTSPAddress   string                `yaml:"rtspAddress"`
	WebRTCAddress string                `yaml:"webrtcAddress"`
	Paths         map[string]PathConfig `yaml:"paths"`
}

func interfaceIsEmpty(i any) bool {
	return reflect.ValueOf(i).Kind() != reflect.Ptr || reflect.ValueOf(i).IsNil()
}

func sortedKeys(paths map[string]*conf.Path) []string {
	ret := make([]string, len(paths))
	i := 0
	for name := range paths {
		ret[i] = name
		i++
	}
	sort.Strings(ret)
	return ret
}

// parsePTZURL PTZ URL을 파싱하여 프로토콜, 호스트, 포트, 사용자명, 비밀번호 추출
// 지원되는 URL 형식:
//   - ptz://user:pass@host:port (기본값: ONVIF)
//   - onvif://user:pass@host:port (ONVIF 프로토콜)
//   - isapi://user:pass@host:port (Hikvision ISAPI 프로토콜)
//   - hikvision://user:pass@host:port (Hikvision ISAPI 프로토콜)
func parsePTZURL(ptzURL string) (protocol string, host string, port int, username string, password string, err error) {
	if ptzURL == "" {
		return "", "", 0, "", "", fmt.Errorf("PTZ URL is empty")
	}

	// URL 프로토콜 추출
	var restURL string
	if strings.HasPrefix(ptzURL, "ptz://") {
		protocol = "ptz"
		restURL = strings.TrimPrefix(ptzURL, "ptz://")
	} else if strings.HasPrefix(ptzURL, "onvif://") {
		protocol = "onvif"
		restURL = strings.TrimPrefix(ptzURL, "onvif://")
	} else if strings.HasPrefix(ptzURL, "isapi://") {
		protocol = "isapi"
		restURL = strings.TrimPrefix(ptzURL, "isapi://")
	} else if strings.HasPrefix(ptzURL, "hikvision://") {
		protocol = "hikvision"
		restURL = strings.TrimPrefix(ptzURL, "hikvision://")
	} else {
		return "", "", 0, "", "", fmt.Errorf("PTZ URL must start with ptz://, onvif://, isapi://, or hikvision://")
	}

	// 마지막 @를 찾아 userinfo와 host:port 분리
	// 비밀번호에 @ 문자가 포함된 경우 처리
	lastAtIndex := strings.LastIndex(restURL, "@")
	if lastAtIndex == -1 {
		return "", "", 0, "", "", fmt.Errorf("PTZ URL must contain @ separator")
	}

	userinfo := restURL[:lastAtIndex]
	hostPort := restURL[lastAtIndex+1:]

	// userinfo를 사용자명과 비밀번호로 분리
	colonIndex := strings.Index(userinfo, ":")
	if colonIndex != -1 {
		username = userinfo[:colonIndex]
		password = userinfo[colonIndex+1:]
	} else {
		username = userinfo
	}

	// host:port 파싱
	parsed, parseErr := url.Parse("http://" + hostPort)
	if parseErr != nil {
		return "", "", 0, "", "", fmt.Errorf("invalid host:port in PTZ URL: %w", parseErr)
	}

	host = parsed.Hostname()
	portStr := parsed.Port()
	port = 80 // 기본 포트
	if portStr != "" {
		port, parseErr = strconv.Atoi(portStr)
		if parseErr != nil {
			return "", "", 0, "", "", fmt.Errorf("invalid port in PTZ URL: %w", parseErr)
		}
	}

	return protocol, host, port, username, password, nil
}

func paramName(ctx *gin.Context) (string, bool) {
	name := ctx.Param("name")

	if len(name) < 2 || name[0] != '/' {
		return "", false
	}

	return name[1:], true
}

func recordingsOfPath(
	pathConf *conf.Path,
	pathName string,
) *defs.APIRecording {
	ret := &defs.APIRecording{
		Name: pathName,
	}

	segments, _ := recordstore.FindSegments(pathConf, pathName, nil, nil)

	ret.Segments = make([]*defs.APIRecordingSegment, len(segments))

	for i, seg := range segments {
		ret.Segments[i] = &defs.APIRecordingSegment{
			Start: seg.Start,
		}
	}

	return ret
}

type apiAuthManager interface {
	Authenticate(req *auth.Request) *auth.Error
	RefreshJWTJWKS()
}

type apiParent interface {
	logger.Writer
	APIConfigSet(conf *conf.Conf)
}

// API is an API server.
type API struct {
	Version        string
	Started        time.Time
	Address        string
	Encryption     bool
	ServerKey      string
	ServerCert     string
	AllowOrigins   []string
	TrustedProxies conf.IPNetworks
	ReadTimeout    conf.Duration
	WriteTimeout   conf.Duration
	Conf           *conf.Conf
	AuthManager    apiAuthManager
	PathManager    defs.APIPathManager
	RTSPServer     defs.APIRTSPServer
	RTSPSServer    defs.APIRTSPServer
	RTMPServer     defs.APIRTMPServer
	RTMPSServer    defs.APIRTMPServer
	HLSServer      defs.APIHLSServer
	WebRTCServer   defs.APIWebRTCServer
	SRTServer      defs.APISRTServer
	Parent         apiParent

	ptzCameras map[string]PTZConfig // PTZ camera configuration cache
	httpServer *httpp.Server
	mutex      sync.RWMutex
}

// Initialize initializes API.
func (a *API) Initialize() error {
	// Load PTZ camera configurations once at startup
	ptzCameras, err := loadPTZCameras()
	if err != nil {
		// Log warning but don't fail server startup
		a.Log(logger.Warn, "failed to load PTZ cameras: %v", err)
		a.ptzCameras = make(map[string]PTZConfig)
	} else {
		a.ptzCameras = ptzCameras
		a.Log(logger.Info, "loaded %d PTZ camera(s)", len(ptzCameras))
	}

	router := gin.New()
	router.SetTrustedProxies(a.TrustedProxies.ToTrustedProxies()) //nolint:errcheck

	router.Use(a.middlewarePreflightRequests)
	router.Use(a.middlewareAuth)

	group := router.Group("/v3")

	group.GET("/info", a.onInfo)

	group.POST("/auth/jwks/refresh", a.onAuthJwksRefresh)

	group.GET("/config/global/get", a.onConfigGlobalGet)
	group.PATCH("/config/global/patch", a.onConfigGlobalPatch)

	group.GET("/config/pathdefaults/get", a.onConfigPathDefaultsGet)
	group.PATCH("/config/pathdefaults/patch", a.onConfigPathDefaultsPatch)

	group.GET("/config/paths/list", a.onConfigPathsList)
	group.GET("/config/paths/get/*name", a.onConfigPathsGet)
	group.POST("/config/paths/add/*name", a.onConfigPathsAdd)
	group.PATCH("/config/paths/patch/*name", a.onConfigPathsPatch)
	group.POST("/config/paths/replace/*name", a.onConfigPathsReplace)
	group.DELETE("/config/paths/delete/*name", a.onConfigPathsDelete)

	group.GET("/paths/list", a.onPathsList)
	group.GET("/paths/get/*name", a.onPathsGet)

	if !interfaceIsEmpty(a.HLSServer) {
		group.GET("/hlsmuxers/list", a.onHLSMuxersList)
		group.GET("/hlsmuxers/get/*name", a.onHLSMuxersGet)
	}

	if !interfaceIsEmpty(a.RTSPServer) {
		group.GET("/rtspconns/list", a.onRTSPConnsList)
		group.GET("/rtspconns/get/:id", a.onRTSPConnsGet)
		group.GET("/rtspsessions/list", a.onRTSPSessionsList)
		group.GET("/rtspsessions/get/:id", a.onRTSPSessionsGet)
		group.POST("/rtspsessions/kick/:id", a.onRTSPSessionsKick)
	}

	if !interfaceIsEmpty(a.RTSPSServer) {
		group.GET("/rtspsconns/list", a.onRTSPSConnsList)
		group.GET("/rtspsconns/get/:id", a.onRTSPSConnsGet)
		group.GET("/rtspssessions/list", a.onRTSPSSessionsList)
		group.GET("/rtspssessions/get/:id", a.onRTSPSSessionsGet)
		group.POST("/rtspssessions/kick/:id", a.onRTSPSSessionsKick)
	}

	if !interfaceIsEmpty(a.RTMPServer) {
		group.GET("/rtmpconns/list", a.onRTMPConnsList)
		group.GET("/rtmpconns/get/:id", a.onRTMPConnsGet)
		group.POST("/rtmpconns/kick/:id", a.onRTMPConnsKick)
	}

	if !interfaceIsEmpty(a.RTMPSServer) {
		group.GET("/rtmpsconns/list", a.onRTMPSConnsList)
		group.GET("/rtmpsconns/get/:id", a.onRTMPSConnsGet)
		group.POST("/rtmpsconns/kick/:id", a.onRTMPSConnsKick)
	}

	if !interfaceIsEmpty(a.WebRTCServer) {
		group.GET("/webrtcsessions/list", a.onWebRTCSessionsList)
		group.GET("/webrtcsessions/get/:id", a.onWebRTCSessionsGet)
		group.POST("/webrtcsessions/kick/:id", a.onWebRTCSessionsKick)
	}

	if !interfaceIsEmpty(a.SRTServer) {
		group.GET("/srtconns/list", a.onSRTConnsList)
		group.GET("/srtconns/get/:id", a.onSRTConnsGet)
		group.POST("/srtconns/kick/:id", a.onSRTConnsKick)
	}

	group.GET("/recordings/list", a.onRecordingsList)
	group.GET("/recordings/get/*name", a.onRecordingsGet)
	group.DELETE("/recordings/deletesegment", a.onRecordingDeleteSegment)

	// PTZ API routes
	ptzGroup := group.Group("/ptz")
	{
		ptzGroup.GET("/cameras", a.onPTZList)
		ptzGroup.POST("/:camera/move", a.onPTZMove)
		ptzGroup.POST("/:camera/move/relative", a.onPTZRelativeMove)
		ptzGroup.POST("/:camera/stop", a.onPTZStop)
		ptzGroup.POST("/:camera/focus", a.onPTZFocus)
		ptzGroup.GET("/:camera/focus", a.onPTZGetFocus)
		ptzGroup.POST("/:camera/iris", a.onPTZIris)
		ptzGroup.GET("/:camera/iris", a.onPTZGetIris)
		ptzGroup.GET("/:camera/status", a.onPTZStatus)
		ptzGroup.GET("/:camera/presets", a.onPTZPresets)
		ptzGroup.POST("/:camera/presets/:presetId", a.onPTZGotoPreset)
		ptzGroup.PUT("/:camera/presets/:presetId", a.onPTZSetPreset)
		ptzGroup.DELETE("/:camera/presets/:presetId", a.onPTZDeletePreset)
	}

	a.httpServer = &httpp.Server{
		Address:      a.Address,
		AllowOrigins: a.AllowOrigins,
		ReadTimeout:  time.Duration(a.ReadTimeout),
		WriteTimeout: time.Duration(a.WriteTimeout),
		Encryption:   a.Encryption,
		ServerCert:   a.ServerCert,
		ServerKey:    a.ServerKey,
		Handler:      router,
		Parent:       a,
	}
	err = a.httpServer.Initialize()
	if err != nil {
		return err
	}

	a.Log(logger.Info, "listener opened on "+a.Address)

	return nil
}

// Close closes the API.
func (a *API) Close() {
	a.Log(logger.Info, "listener is closing")
	a.httpServer.Close()
}

// Log implements logger.Writer.
func (a *API) Log(level logger.Level, format string, args ...any) {
	a.Parent.Log(level, "[API] "+format, args...)
}

func (a *API) writeError(ctx *gin.Context, status int, err error) {
	// show error in logs
	a.Log(logger.Error, err.Error())

	// add error to response
	ctx.JSON(status, &defs.APIError{
		Error: err.Error(),
	})
}

func (a *API) middlewarePreflightRequests(ctx *gin.Context) {
	if ctx.Request.Method == http.MethodOptions &&
		ctx.Request.Header.Get("Access-Control-Request-Method") != "" {
		ctx.Header("Access-Control-Allow-Methods", "OPTIONS, PUT, GET, POST, PATCH, DELETE")
		ctx.Header("Access-Control-Allow-Headers", "Authorization, Content-Type")
		ctx.AbortWithStatus(http.StatusNoContent)
		return
	}
}

func (a *API) middlewareAuth(ctx *gin.Context) {
	req := &auth.Request{
		Action:      conf.AuthActionAPI,
		Query:       ctx.Request.URL.RawQuery,
		Credentials: httpp.Credentials(ctx.Request),
		IP:          net.ParseIP(ctx.ClientIP()),
	}

	err := a.AuthManager.Authenticate(req)
	if err != nil {
		if err.AskCredentials {
			ctx.Header("WWW-Authenticate", `Basic realm="mediamtx"`)
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		a.Log(logger.Info, "connection %v failed to authenticate: %v", httpp.RemoteAddr(ctx), err.Wrapped)

		// wait some seconds to delay brute force attacks
		<-time.After(auth.PauseAfterError)

		ctx.AbortWithStatus(http.StatusUnauthorized)
		return
	}
}

func (a *API) onConfigGlobalGet(ctx *gin.Context) {
	a.mutex.RLock()
	c := a.Conf
	a.mutex.RUnlock()

	ctx.JSON(http.StatusOK, c.Global())
}

func (a *API) onConfigGlobalPatch(ctx *gin.Context) {
	var c conf.OptionalGlobal
	err := jsonwrapper.Decode(ctx.Request.Body, &c)
	if err != nil {
		a.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	a.mutex.Lock()
	defer a.mutex.Unlock()

	newConf := a.Conf.Clone()

	newConf.PatchGlobal(&c)

	err = newConf.Validate(nil)
	if err != nil {
		a.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	a.Conf = newConf

	// since reloading the configuration can cause the shutdown of the API,
	// call it in a goroutine
	go a.Parent.APIConfigSet(newConf)

	ctx.Status(http.StatusOK)
}

func (a *API) onConfigPathDefaultsGet(ctx *gin.Context) {
	a.mutex.RLock()
	c := a.Conf
	a.mutex.RUnlock()

	ctx.JSON(http.StatusOK, c.PathDefaults)
}

func (a *API) onConfigPathDefaultsPatch(ctx *gin.Context) {
	var p conf.OptionalPath
	err := jsonwrapper.Decode(ctx.Request.Body, &p)
	if err != nil {
		a.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	a.mutex.Lock()
	defer a.mutex.Unlock()

	newConf := a.Conf.Clone()

	newConf.PatchPathDefaults(&p)

	err = newConf.Validate(nil)
	if err != nil {
		a.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	a.Conf = newConf
	a.Parent.APIConfigSet(newConf)

	ctx.Status(http.StatusOK)
}

func (a *API) onConfigPathsList(ctx *gin.Context) {
	a.mutex.RLock()
	c := a.Conf
	a.mutex.RUnlock()

	data := &defs.APIPathConfList{
		Items: make([]*conf.Path, len(c.Paths)),
	}

	for i, key := range sortedKeys(c.Paths) {
		data.Items[i] = c.Paths[key]
	}

	data.ItemCount = len(data.Items)
	pageCount, err := paginate(&data.Items, ctx.Query("itemsPerPage"), ctx.Query("page"))
	if err != nil {
		a.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	data.PageCount = pageCount

	ctx.JSON(http.StatusOK, data)
}

func (a *API) onConfigPathsGet(ctx *gin.Context) {
	confName, ok := paramName(ctx)
	if !ok {
		a.writeError(ctx, http.StatusBadRequest, fmt.Errorf("invalid name"))
		return
	}

	a.mutex.RLock()
	c := a.Conf
	a.mutex.RUnlock()

	p, ok := c.Paths[confName]
	if !ok {
		a.writeError(ctx, http.StatusNotFound, fmt.Errorf("path configuration not found"))
		return
	}

	ctx.JSON(http.StatusOK, p)
}

func (a *API) onConfigPathsAdd(ctx *gin.Context) { //nolint:dupl
	confName, ok := paramName(ctx)
	if !ok {
		a.writeError(ctx, http.StatusBadRequest, fmt.Errorf("invalid name"))
		return
	}

	var p conf.OptionalPath
	err := jsonwrapper.Decode(ctx.Request.Body, &p)
	if err != nil {
		a.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	a.mutex.Lock()
	defer a.mutex.Unlock()

	newConf := a.Conf.Clone()

	err = newConf.AddPath(confName, &p)
	if err != nil {
		a.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	err = newConf.Validate(nil)
	if err != nil {
		a.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	a.Conf = newConf
	a.Parent.APIConfigSet(newConf)

	ctx.Status(http.StatusOK)
}

func (a *API) onConfigPathsPatch(ctx *gin.Context) { //nolint:dupl
	confName, ok := paramName(ctx)
	if !ok {
		a.writeError(ctx, http.StatusBadRequest, fmt.Errorf("invalid name"))
		return
	}

	var p conf.OptionalPath
	err := jsonwrapper.Decode(ctx.Request.Body, &p)
	if err != nil {
		a.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	a.mutex.Lock()
	defer a.mutex.Unlock()

	newConf := a.Conf.Clone()

	err = newConf.PatchPath(confName, &p)
	if err != nil {
		if errors.Is(err, conf.ErrPathNotFound) {
			a.writeError(ctx, http.StatusNotFound, err)
		} else {
			a.writeError(ctx, http.StatusBadRequest, err)
		}
		return
	}

	err = newConf.Validate(nil)
	if err != nil {
		a.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	a.Conf = newConf
	a.Parent.APIConfigSet(newConf)

	ctx.Status(http.StatusOK)
}

func (a *API) onConfigPathsReplace(ctx *gin.Context) { //nolint:dupl
	confName, ok := paramName(ctx)
	if !ok {
		a.writeError(ctx, http.StatusBadRequest, fmt.Errorf("invalid name"))
		return
	}

	var p conf.OptionalPath
	err := jsonwrapper.Decode(ctx.Request.Body, &p)
	if err != nil {
		a.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	a.mutex.Lock()
	defer a.mutex.Unlock()

	newConf := a.Conf.Clone()

	err = newConf.ReplacePath(confName, &p)
	if err != nil {
		if errors.Is(err, conf.ErrPathNotFound) {
			a.writeError(ctx, http.StatusNotFound, err)
		} else {
			a.writeError(ctx, http.StatusBadRequest, err)
		}
		return
	}

	err = newConf.Validate(nil)
	if err != nil {
		a.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	a.Conf = newConf
	a.Parent.APIConfigSet(newConf)

	ctx.Status(http.StatusOK)
}

func (a *API) onConfigPathsDelete(ctx *gin.Context) {
	confName, ok := paramName(ctx)
	if !ok {
		a.writeError(ctx, http.StatusBadRequest, fmt.Errorf("invalid name"))
		return
	}

	a.mutex.Lock()
	defer a.mutex.Unlock()

	newConf := a.Conf.Clone()

	err := newConf.RemovePath(confName)
	if err != nil {
		if errors.Is(err, conf.ErrPathNotFound) {
			a.writeError(ctx, http.StatusNotFound, err)
		} else {
			a.writeError(ctx, http.StatusBadRequest, err)
		}
		return
	}

	err = newConf.Validate(nil)
	if err != nil {
		a.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	a.Conf = newConf
	a.Parent.APIConfigSet(newConf)

	ctx.Status(http.StatusOK)
}

func (a *API) onInfo(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, &defs.APIInfo{
		Version: a.Version,
		Started: a.Started,
	})
}

func (a *API) onAuthJwksRefresh(ctx *gin.Context) {
	a.AuthManager.RefreshJWTJWKS()
	ctx.Status(http.StatusOK)
}

func (a *API) onPathsList(ctx *gin.Context) {
	data, err := a.PathManager.APIPathsList()
	if err != nil {
		a.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	data.ItemCount = len(data.Items)
	pageCount, err := paginate(&data.Items, ctx.Query("itemsPerPage"), ctx.Query("page"))
	if err != nil {
		a.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	data.PageCount = pageCount

	ctx.JSON(http.StatusOK, data)
}

func (a *API) onPathsGet(ctx *gin.Context) {
	pathName, ok := paramName(ctx)
	if !ok {
		a.writeError(ctx, http.StatusBadRequest, fmt.Errorf("invalid name"))
		return
	}

	data, err := a.PathManager.APIPathsGet(pathName)
	if err != nil {
		if errors.Is(err, conf.ErrPathNotFound) {
			a.writeError(ctx, http.StatusNotFound, err)
		} else {
			a.writeError(ctx, http.StatusInternalServerError, err)
		}
		return
	}

	ctx.JSON(http.StatusOK, data)
}

func (a *API) onRTSPConnsList(ctx *gin.Context) {
	data, err := a.RTSPServer.APIConnsList()
	if err != nil {
		a.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	data.ItemCount = len(data.Items)
	pageCount, err := paginate(&data.Items, ctx.Query("itemsPerPage"), ctx.Query("page"))
	if err != nil {
		a.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	data.PageCount = pageCount

	ctx.JSON(http.StatusOK, data)
}

func (a *API) onRTSPConnsGet(ctx *gin.Context) {
	uuid, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		a.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	data, err := a.RTSPServer.APIConnsGet(uuid)
	if err != nil {
		if errors.Is(err, rtsp.ErrConnNotFound) {
			a.writeError(ctx, http.StatusNotFound, err)
		} else {
			a.writeError(ctx, http.StatusInternalServerError, err)
		}
		return
	}

	ctx.JSON(http.StatusOK, data)
}

func (a *API) onRTSPSessionsList(ctx *gin.Context) {
	data, err := a.RTSPServer.APISessionsList()
	if err != nil {
		a.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	data.ItemCount = len(data.Items)
	pageCount, err := paginate(&data.Items, ctx.Query("itemsPerPage"), ctx.Query("page"))
	if err != nil {
		a.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	data.PageCount = pageCount

	ctx.JSON(http.StatusOK, data)
}

func (a *API) onRTSPSessionsGet(ctx *gin.Context) {
	uuid, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		a.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	data, err := a.RTSPServer.APISessionsGet(uuid)
	if err != nil {
		if errors.Is(err, rtsp.ErrSessionNotFound) {
			a.writeError(ctx, http.StatusNotFound, err)
		} else {
			a.writeError(ctx, http.StatusInternalServerError, err)
		}
		return
	}

	ctx.JSON(http.StatusOK, data)
}

func (a *API) onRTSPSessionsKick(ctx *gin.Context) {
	uuid, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		a.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	err = a.RTSPServer.APISessionsKick(uuid)
	if err != nil {
		if errors.Is(err, rtsp.ErrSessionNotFound) {
			a.writeError(ctx, http.StatusNotFound, err)
		} else {
			a.writeError(ctx, http.StatusInternalServerError, err)
		}
		return
	}

	ctx.Status(http.StatusOK)
}

func (a *API) onRTSPSConnsList(ctx *gin.Context) {
	data, err := a.RTSPSServer.APIConnsList()
	if err != nil {
		a.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	data.ItemCount = len(data.Items)
	pageCount, err := paginate(&data.Items, ctx.Query("itemsPerPage"), ctx.Query("page"))
	if err != nil {
		a.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	data.PageCount = pageCount

	ctx.JSON(http.StatusOK, data)
}

func (a *API) onRTSPSConnsGet(ctx *gin.Context) {
	uuid, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		a.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	data, err := a.RTSPSServer.APIConnsGet(uuid)
	if err != nil {
		if errors.Is(err, rtsp.ErrConnNotFound) {
			a.writeError(ctx, http.StatusNotFound, err)
		} else {
			a.writeError(ctx, http.StatusInternalServerError, err)
		}
		return
	}

	ctx.JSON(http.StatusOK, data)
}

func (a *API) onRTSPSSessionsList(ctx *gin.Context) {
	data, err := a.RTSPSServer.APISessionsList()
	if err != nil {
		a.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	data.ItemCount = len(data.Items)
	pageCount, err := paginate(&data.Items, ctx.Query("itemsPerPage"), ctx.Query("page"))
	if err != nil {
		a.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	data.PageCount = pageCount

	ctx.JSON(http.StatusOK, data)
}

func (a *API) onRTSPSSessionsGet(ctx *gin.Context) {
	uuid, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		a.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	data, err := a.RTSPSServer.APISessionsGet(uuid)
	if err != nil {
		if errors.Is(err, rtsp.ErrSessionNotFound) {
			a.writeError(ctx, http.StatusNotFound, err)
		} else {
			a.writeError(ctx, http.StatusInternalServerError, err)
		}
		return
	}

	ctx.JSON(http.StatusOK, data)
}

func (a *API) onRTSPSSessionsKick(ctx *gin.Context) {
	uuid, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		a.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	err = a.RTSPSServer.APISessionsKick(uuid)
	if err != nil {
		if errors.Is(err, rtsp.ErrSessionNotFound) {
			a.writeError(ctx, http.StatusNotFound, err)
		} else {
			a.writeError(ctx, http.StatusInternalServerError, err)
		}
		return
	}

	ctx.Status(http.StatusOK)
}

func (a *API) onRTMPConnsList(ctx *gin.Context) {
	data, err := a.RTMPServer.APIConnsList()
	if err != nil {
		a.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	data.ItemCount = len(data.Items)
	pageCount, err := paginate(&data.Items, ctx.Query("itemsPerPage"), ctx.Query("page"))
	if err != nil {
		a.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	data.PageCount = pageCount

	ctx.JSON(http.StatusOK, data)
}

func (a *API) onRTMPConnsGet(ctx *gin.Context) {
	uuid, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		a.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	data, err := a.RTMPServer.APIConnsGet(uuid)
	if err != nil {
		if errors.Is(err, rtmp.ErrConnNotFound) {
			a.writeError(ctx, http.StatusNotFound, err)
		} else {
			a.writeError(ctx, http.StatusInternalServerError, err)
		}
		return
	}

	ctx.JSON(http.StatusOK, data)
}

func (a *API) onRTMPConnsKick(ctx *gin.Context) {
	uuid, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		a.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	err = a.RTMPServer.APIConnsKick(uuid)
	if err != nil {
		if errors.Is(err, rtmp.ErrConnNotFound) {
			a.writeError(ctx, http.StatusNotFound, err)
		} else {
			a.writeError(ctx, http.StatusInternalServerError, err)
		}
		return
	}

	ctx.Status(http.StatusOK)
}

func (a *API) onRTMPSConnsList(ctx *gin.Context) {
	data, err := a.RTMPSServer.APIConnsList()
	if err != nil {
		a.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	data.ItemCount = len(data.Items)
	pageCount, err := paginate(&data.Items, ctx.Query("itemsPerPage"), ctx.Query("page"))
	if err != nil {
		a.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	data.PageCount = pageCount

	ctx.JSON(http.StatusOK, data)
}

func (a *API) onRTMPSConnsGet(ctx *gin.Context) {
	uuid, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		a.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	data, err := a.RTMPSServer.APIConnsGet(uuid)
	if err != nil {
		if errors.Is(err, rtmp.ErrConnNotFound) {
			a.writeError(ctx, http.StatusNotFound, err)
		} else {
			a.writeError(ctx, http.StatusInternalServerError, err)
		}
		return
	}

	ctx.JSON(http.StatusOK, data)
}

func (a *API) onRTMPSConnsKick(ctx *gin.Context) {
	uuid, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		a.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	err = a.RTMPSServer.APIConnsKick(uuid)
	if err != nil {
		if errors.Is(err, rtmp.ErrConnNotFound) {
			a.writeError(ctx, http.StatusNotFound, err)
		} else {
			a.writeError(ctx, http.StatusInternalServerError, err)
		}
		return
	}

	ctx.Status(http.StatusOK)
}

func (a *API) onHLSMuxersList(ctx *gin.Context) {
	data, err := a.HLSServer.APIMuxersList()
	if err != nil {
		a.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	data.ItemCount = len(data.Items)
	pageCount, err := paginate(&data.Items, ctx.Query("itemsPerPage"), ctx.Query("page"))
	if err != nil {
		a.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	data.PageCount = pageCount

	ctx.JSON(http.StatusOK, data)
}

func (a *API) onHLSMuxersGet(ctx *gin.Context) {
	pathName, ok := paramName(ctx)
	if !ok {
		a.writeError(ctx, http.StatusBadRequest, fmt.Errorf("invalid name"))
		return
	}

	data, err := a.HLSServer.APIMuxersGet(pathName)
	if err != nil {
		if errors.Is(err, hls.ErrMuxerNotFound) {
			a.writeError(ctx, http.StatusNotFound, err)
		} else {
			a.writeError(ctx, http.StatusInternalServerError, err)
		}
		return
	}

	ctx.JSON(http.StatusOK, data)
}

func (a *API) onWebRTCSessionsList(ctx *gin.Context) {
	data, err := a.WebRTCServer.APISessionsList()
	if err != nil {
		a.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	data.ItemCount = len(data.Items)
	pageCount, err := paginate(&data.Items, ctx.Query("itemsPerPage"), ctx.Query("page"))
	if err != nil {
		a.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	data.PageCount = pageCount

	ctx.JSON(http.StatusOK, data)
}

func (a *API) onWebRTCSessionsGet(ctx *gin.Context) {
	uuid, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		a.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	data, err := a.WebRTCServer.APISessionsGet(uuid)
	if err != nil {
		if errors.Is(err, webrtc.ErrSessionNotFound) {
			a.writeError(ctx, http.StatusNotFound, err)
		} else {
			a.writeError(ctx, http.StatusInternalServerError, err)
		}
		return
	}

	ctx.JSON(http.StatusOK, data)
}

func (a *API) onWebRTCSessionsKick(ctx *gin.Context) {
	uuid, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		a.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	err = a.WebRTCServer.APISessionsKick(uuid)
	if err != nil {
		if errors.Is(err, webrtc.ErrSessionNotFound) {
			a.writeError(ctx, http.StatusNotFound, err)
		} else {
			a.writeError(ctx, http.StatusInternalServerError, err)
		}
		return
	}

	ctx.Status(http.StatusOK)
}

func (a *API) onSRTConnsList(ctx *gin.Context) {
	data, err := a.SRTServer.APIConnsList()
	if err != nil {
		a.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	data.ItemCount = len(data.Items)
	pageCount, err := paginate(&data.Items, ctx.Query("itemsPerPage"), ctx.Query("page"))
	if err != nil {
		a.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	data.PageCount = pageCount

	ctx.JSON(http.StatusOK, data)
}

func (a *API) onSRTConnsGet(ctx *gin.Context) {
	uuid, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		a.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	data, err := a.SRTServer.APIConnsGet(uuid)
	if err != nil {
		if errors.Is(err, srt.ErrConnNotFound) {
			a.writeError(ctx, http.StatusNotFound, err)
		} else {
			a.writeError(ctx, http.StatusInternalServerError, err)
		}
		return
	}

	ctx.JSON(http.StatusOK, data)
}

func (a *API) onSRTConnsKick(ctx *gin.Context) {
	uuid, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		a.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	err = a.SRTServer.APIConnsKick(uuid)
	if err != nil {
		if errors.Is(err, srt.ErrConnNotFound) {
			a.writeError(ctx, http.StatusNotFound, err)
		} else {
			a.writeError(ctx, http.StatusInternalServerError, err)
		}
		return
	}

	ctx.Status(http.StatusOK)
}

func (a *API) onRecordingsList(ctx *gin.Context) {
	a.mutex.RLock()
	c := a.Conf
	a.mutex.RUnlock()

	pathNames := recordstore.FindAllPathsWithSegments(c.Paths)

	data := defs.APIRecordingList{}

	data.ItemCount = len(pathNames)
	pageCount, err := paginate(&pathNames, ctx.Query("itemsPerPage"), ctx.Query("page"))
	if err != nil {
		a.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	data.PageCount = pageCount

	data.Items = make([]*defs.APIRecording, len(pathNames))

	for i, pathName := range pathNames {
		pathConf, _, _ := conf.FindPathConf(c.Paths, pathName)
		data.Items[i] = recordingsOfPath(pathConf, pathName)
	}

	ctx.JSON(http.StatusOK, data)
}

func (a *API) onRecordingsGet(ctx *gin.Context) {
	pathName, ok := paramName(ctx)
	if !ok {
		a.writeError(ctx, http.StatusBadRequest, fmt.Errorf("invalid name"))
		return
	}

	a.mutex.RLock()
	c := a.Conf
	a.mutex.RUnlock()

	pathConf, _, err := conf.FindPathConf(c.Paths, pathName)
	if err != nil {
		a.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	ctx.JSON(http.StatusOK, recordingsOfPath(pathConf, pathName))
}

func (a *API) onRecordingDeleteSegment(ctx *gin.Context) {
	pathName := ctx.Query("path")

	start, err := time.Parse(time.RFC3339, ctx.Query("start"))
	if err != nil {
		a.writeError(ctx, http.StatusBadRequest, fmt.Errorf("invalid 'start' parameter: %w", err))
		return
	}

	a.mutex.RLock()
	c := a.Conf
	a.mutex.RUnlock()

	pathConf, _, err := conf.FindPathConf(c.Paths, pathName)
	if err != nil {
		a.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	pathFormat := recordstore.PathAddExtension(
		strings.ReplaceAll(pathConf.RecordPath, "%path", pathName),
		pathConf.RecordFormat,
	)

	segmentPath := recordstore.Path{
		Start: start,
	}.Encode(pathFormat)

	err = os.Remove(segmentPath)
	if err != nil {
		a.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	ctx.Status(http.StatusOK)
}

// ReloadConf is called by core.
func (a *API) ReloadConf(conf *conf.Conf) {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	a.Conf = conf
}

// PTZ related functions

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
		// PTZ가 활성화되고 PTZSource가 설정된 경로만 처리
		if !pathConfig.PTZ || pathConfig.PTZSource == "" {
			continue
		}

		// PTZ URL 파싱: protocol://user:pass@host:port
		protocol, host, port, username, password, err := parsePTZURL(pathConfig.PTZSource)
		if err != nil {
			fmt.Printf("Warning: Failed to parse PTZ URL for %s: %v\n", name, err)
			continue
		}

		if host != "" && username != "" {
			ptzCameras[name] = PTZConfig{
				Protocol: protocol,
				Host:     host,
				PTZPort:  port,
				Username: username,
				Password: password,
			}
		}
	}

	return ptzCameras, nil
}

// getPTZConfig 캐시에서 특정 카메라의 PTZ 설정 조회
func (a *API) getPTZConfig(cameraName string) (PTZConfig, bool) {
	a.mutex.RLock()
	defer a.mutex.RUnlock()
	config, exists := a.ptzCameras[cameraName]
	return config, exists
}

// createPTZController PTZ 설정으로부터 컨트롤러 생성 및 연결
func createPTZController(config PTZConfig) (ptz.Controller, error) {
	controller, err := ptz.NewController(ptz.ControllerConfig{
		Protocol: config.Protocol,
		Host:     config.Host,
		Port:     config.PTZPort,
		Username: config.Username,
		Password: config.Password,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create PTZ controller: %w", err)
	}

	if err := controller.Connect(); err != nil {
		return nil, fmt.Errorf("failed to connect to PTZ camera: %w", err)
	}

	return controller, nil
}

func (a *API) onPTZMove(ctx *gin.Context) {
	cameraName := ctx.Param("camera")

	config, exists := a.getPTZConfig(cameraName)
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

	ptzController, err := createPTZController(config)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, PTZResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to create PTZ controller: %v", err),
		})
		return
	}

	err = ptzController.Move(req.Pan, req.Tilt, req.Zoom)
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

func (a *API) onPTZRelativeMove(ctx *gin.Context) {
	cameraName := ctx.Param("camera")

	config, exists := a.getPTZConfig(cameraName)
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

	ptzController, err := createPTZController(config)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, PTZResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to create PTZ controller: %v", err),
		})
		return
	}

	err = ptzController.RelativeMove(req.Pan, req.Tilt, req.Zoom)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, PTZResponse{
			Success: false,
			Message: fmt.Sprintf("PTZ relative move failed: %v", err),
		})
		return
	}

	ctx.JSON(http.StatusOK, PTZResponse{
		Success: true,
		Message: "PTZ relative move command sent successfully",
	})
}

func (a *API) onPTZStop(ctx *gin.Context) {
	cameraName := ctx.Param("camera")

	config, exists := a.getPTZConfig(cameraName)
	if !exists {
		ctx.JSON(http.StatusNotFound, PTZResponse{
			Success: false,
			Message: fmt.Sprintf("PTZ not configured for camera: %s", cameraName),
		})
		return
	}

	ptzController, err := createPTZController(config)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, PTZResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to create PTZ controller: %v", err),
		})
		return
	}
	err = ptzController.Stop()
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

func (a *API) onPTZFocus(ctx *gin.Context) {
	cameraName := ctx.Param("camera")

	config, exists := a.getPTZConfig(cameraName)
	if !exists {
		ctx.JSON(http.StatusNotFound, PTZResponse{
			Success: false,
			Message: fmt.Sprintf("PTZ not configured for camera: %s", cameraName),
		})
		return
	}

	var req struct {
		Speed int `json:"speed"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, PTZResponse{
			Success: false,
			Message: "Invalid request body",
		})
		return
	}

	ptzController, err := createPTZController(config)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, PTZResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to create PTZ controller: %v", err),
		})
		return
	}
	err = ptzController.Focus(req.Speed)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, PTZResponse{
			Success: false,
			Message: fmt.Sprintf("Focus adjustment failed: %v", err),
		})
		return
	}

	ctx.JSON(http.StatusOK, PTZResponse{
		Success: true,
		Message: "Focus adjustment command sent successfully",
	})
}

func (a *API) onPTZIris(ctx *gin.Context) {
	cameraName := ctx.Param("camera")

	config, exists := a.getPTZConfig(cameraName)
	if !exists {
		ctx.JSON(http.StatusNotFound, PTZResponse{
			Success: false,
			Message: fmt.Sprintf("PTZ not configured for camera: %s", cameraName),
		})
		return
	}

	var req struct {
		Speed int `json:"speed"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, PTZResponse{
			Success: false,
			Message: "Invalid request body",
		})
		return
	}

	ptzController, err := createPTZController(config)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, PTZResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to create PTZ controller: %v", err),
		})
		return
	}
	err = ptzController.Iris(req.Speed)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, PTZResponse{
			Success: false,
			Message: fmt.Sprintf("Iris adjustment failed: %v", err),
		})
		return
	}

	ctx.JSON(http.StatusOK, PTZResponse{
		Success: true,
		Message: "Iris adjustment command sent successfully",
	})
}

func (a *API) onPTZStatus(ctx *gin.Context) {
	cameraName := ctx.Param("camera")

	config, exists := a.getPTZConfig(cameraName)
	if !exists {
		ctx.JSON(http.StatusNotFound, PTZResponse{
			Success: false,
			Message: fmt.Sprintf("PTZ not configured for camera: %s", cameraName),
		})
		return
	}

	ptzController, err := createPTZController(config)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, PTZResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to create PTZ controller: %v", err),
		})
		return
	}
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

func (a *API) onPTZPresets(ctx *gin.Context) {
	cameraName := ctx.Param("camera")

	config, exists := a.getPTZConfig(cameraName)
	if !exists {
		ctx.JSON(http.StatusNotFound, PTZResponse{
			Success: false,
			Message: fmt.Sprintf("PTZ not configured for camera: %s", cameraName),
		})
		return
	}

	ptzController, err := createPTZController(config)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, PTZResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to create PTZ controller: %v", err),
		})
		return
	}
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

func (a *API) onPTZGotoPreset(ctx *gin.Context) {
	cameraName := ctx.Param("camera")
	presetIDStr := ctx.Param("presetId")

	config, exists := a.getPTZConfig(cameraName)
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

	ptzController, err := createPTZController(config)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, PTZResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to create PTZ controller: %v", err),
		})
		return
	}
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

func (a *API) onPTZSetPreset(ctx *gin.Context) {
	cameraName := ctx.Param("camera")
	presetIDStr := ctx.Param("presetId")

	config, exists := a.getPTZConfig(cameraName)
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

	var req struct {
		Name string `json:"name"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, PTZResponse{
			Success: false,
			Message: "Invalid request body: name is required",
		})
		return
	}

	if req.Name == "" {
		req.Name = fmt.Sprintf("Preset%d", presetID)
	}

	ptzController, err := createPTZController(config)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, PTZResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to create PTZ controller: %v", err),
		})
		return
	}
	err = ptzController.SetPreset(presetID, req.Name)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, PTZResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to set preset: %v", err),
		})
		return
	}

	// Get the updated preset list to return the created preset
	presets, err := ptzController.GetPresets()
	if err != nil {
		// If we can't get presets, still return success but with message only
		ctx.JSON(http.StatusOK, PTZResponse{
			Success: true,
			Message: fmt.Sprintf("Preset %d saved as '%s'", presetID, req.Name),
		})
		return
	}

	// 생성된 프리셋을 목록에서 찾기
	var createdPreset *ptz.Preset
	for i := range presets {
		if presets[i].ID == presetID {
			createdPreset = &presets[i]
			break
		}
	}

	if createdPreset != nil {
		ctx.JSON(http.StatusOK, PTZResponse{
			Success: true,
			Message: fmt.Sprintf("Preset %d saved as '%s'", presetID, req.Name),
			Data:    createdPreset,
		})
	} else {
		ctx.JSON(http.StatusOK, PTZResponse{
			Success: true,
			Message: fmt.Sprintf("Preset %d saved as '%s'", presetID, req.Name),
		})
	}
}

func (a *API) onPTZDeletePreset(ctx *gin.Context) {
	cameraName := ctx.Param("camera")
	presetIDStr := ctx.Param("presetId")

	config, exists := a.getPTZConfig(cameraName)
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

	ptzController, err := createPTZController(config)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, PTZResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to create PTZ controller: %v", err),
		})
		return
	}
	err = ptzController.DeletePreset(presetID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, PTZResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to delete preset: %v", err),
		})
		return
	}

	ctx.JSON(http.StatusOK, PTZResponse{
		Success: true,
		Message: fmt.Sprintf("Preset %d deleted", presetID),
	})
}

func (a *API) onPTZList(ctx *gin.Context) {
	a.mutex.RLock()
	cameras := make([]string, 0, len(a.ptzCameras))
	for name := range a.ptzCameras {
		cameras = append(cameras, name)
	}
	a.mutex.RUnlock()

	ctx.JSON(http.StatusOK, PTZResponse{
		Success: true,
		Data:    cameras,
	})
}

func (a *API) onPTZGetFocus(ctx *gin.Context) {
	cameraName := ctx.Param("camera")

	config, exists := a.getPTZConfig(cameraName)
	if !exists {
		ctx.JSON(http.StatusNotFound, PTZResponse{
			Success: false,
			Message: fmt.Sprintf("PTZ not configured for camera: %s", cameraName),
		})
		return
	}

	ptzController, err := createPTZController(config)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, PTZResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to create PTZ controller: %v", err),
		})
		return
	}
	imageSettings, err := ptzController.GetImageSettings()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, PTZResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to get focus settings: %v", err),
		})
		return
	}

	ctx.JSON(http.StatusOK, PTZResponse{
		Success: true,
		Data:    imageSettings,
	})
}

func (a *API) onPTZGetIris(ctx *gin.Context) {
	cameraName := ctx.Param("camera")

	config, exists := a.getPTZConfig(cameraName)
	if !exists {
		ctx.JSON(http.StatusNotFound, PTZResponse{
			Success: false,
			Message: fmt.Sprintf("PTZ not configured for camera: %s", cameraName),
		})
		return
	}

	ptzController, err := createPTZController(config)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, PTZResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to create PTZ controller: %v", err),
		})
		return
	}
	imageSettings, err := ptzController.GetImageSettings()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, PTZResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to get iris settings: %v", err),
		})
		return
	}

	ctx.JSON(http.StatusOK, PTZResponse{
		Success: true,
		Data:    imageSettings,
	})
}
