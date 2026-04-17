package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	redc "red-cloud/mod"
	"reflect"
	"strings"
	"sync"
	"time"
)

// SSEClient represents a connected SSE client
type SSEClient struct {
	ch chan string
}

// SSEHub manages SSE connections
type SSEHub struct {
	mu      sync.RWMutex
	clients map[*SSEClient]struct{}
}

func newSSEHub() *SSEHub {
	return &SSEHub{
		clients: make(map[*SSEClient]struct{}),
	}
}

func (h *SSEHub) subscribe() *SSEClient {
	c := &SSEClient{ch: make(chan string, 64)}
	h.mu.Lock()
	h.clients[c] = struct{}{}
	h.mu.Unlock()
	return c
}

func (h *SSEHub) unsubscribe(c *SSEClient) {
	h.mu.Lock()
	delete(h.clients, c)
	h.mu.Unlock()
	close(c.ch)
}

func (h *SSEHub) closeAll() {
	h.mu.Lock()
	for c := range h.clients {
		close(c.ch)
		delete(h.clients, c)
	}
	h.mu.Unlock()
}

func (h *SSEHub) broadcast(name string, data interface{}) {
	payload, err := json.Marshal(map[string]interface{}{
		"event": name,
		"data":  data,
	})
	if err != nil {
		return
	}
	msg := string(payload)
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.clients {
		select {
		case c.ch <- msg:
		default:
			// drop if client is slow
		}
	}
}

// HTTPServer handles HTTP mode
type HTTPServer struct {
	app   *App
	hub   *SSEHub
	srv   *http.Server
	token string
	host  string
	port  int
	users []redc.HTTPUser
}

// RoleLevel returns the numeric level for a role (higher = more permissions)
func RoleLevel(role string) int {
	switch role {
	case "admin":
		return 3
	case "operator":
		return 2
	case "viewer":
		return 1
	default:
		return 0
	}
}

// methodMinRole defines the minimum role required for each method.
// Methods not listed default to "admin" for safety.
var methodMinRole = map[string]string{
	// === Viewer: read-only ===
	"GetConfig": "viewer", "GetVersion": "viewer", "CheckForUpdates": "viewer",
	"GetLanguage": "viewer", "GetShowWelcomeDialog": "viewer",
	"GetNotificationEnabled": "viewer", "GetDisableRightClick": "viewer",
	"GetSpotMonitorEnabled": "viewer", "GetSpotAutoRecoverEnabled": "viewer",
	"GetWebhookConfig": "viewer", "GetAllCaseTags": "viewer", "GetAllTagNames": "viewer",
	"GetHTTPServerStatus": "viewer",
	"ListCases": "viewer", "GetCaseOutputs": "viewer", "GetCasePlanPreview": "viewer",
	"GetResourceSummary": "viewer", "GetBalances": "viewer", "GetBills": "viewer",
	"GetTotalRuntime": "viewer", "GetPredictedMonthlyCost": "viewer",
	"ListProfiles": "viewer", "GetActiveProfile": "viewer",
	"GetProvidersConfig": "viewer", "GetCurrentProject": "viewer", "ListProjects": "viewer",
	"ListTemplates": "viewer", "ListAllTemplates": "viewer", "GetTemplateVariables": "viewer",
	"FetchRegistryTemplates": "viewer", "FetchTemplateReadme": "viewer",
	"GetTemplateFiles": "viewer", "GetTemplateMetadata": "viewer", "GetBaseTemplates": "viewer",
	"ListUserdataTemplates": "viewer", "ListComposeTemplates": "viewer",
	"ListCustomDeployments": "viewer", "GetDeploymentHistory": "viewer",
	"GetDeploymentPlanPreview": "viewer",
	"GetCostEstimate": "viewer",
	"ListPlugins": "viewer", "GetPluginConfig": "viewer", "FetchPluginRegistry": "viewer",
	"GetMCPStatus": "viewer",
	"ListScheduledTasks": "viewer", "ListCaseScheduledTasks": "viewer",
	"ListAllScheduledTasks": "viewer", "GetScheduledTask": "viewer",
	"GetAgentMemories": "viewer",
	"GetF8xCatalog": "viewer", "GetF8xCategories": "viewer", "GetF8xPresets": "viewer",
	"GetF8xStatus": "viewer", "GetF8xInstallHistory": "viewer", "GetF8xRunningTasks": "viewer",
	"RefreshF8xCatalog": "viewer",
	"GetSSHInfoForCase": "viewer", "GetSSHInfosForCase": "viewer",
	"ListPortForwards": "viewer", "ListRemoteFiles": "viewer",
	"GetRemoteFileContent": "viewer",
	"HasActiveOperations": "viewer",
	"ValidateTemplate": "viewer", "ValidateDeploymentConfig": "viewer",
	"EstimateDeploymentCost": "viewer",
	"GetProviderRegions": "viewer", "GetInstanceTypes": "viewer",
	"GetTerraformMirrorConfig": "viewer",
	"GetAuditLogs": "viewer", "ExportAuditLogs": "viewer",
	// MCP read
	"MCPGetCostEstimate": "viewer", "MCPGetBalances": "viewer",
	"MCPGetResourceSummary": "viewer", "MCPGetPredictedMonthlyCost": "viewer",
	"MCPGetBills": "viewer", "MCPGetTotalRuntime": "viewer",
	"MCPListCustomDeployments": "viewer", "MCPListProjects": "viewer",
	"MCPListProfiles": "viewer", "MCPGetActiveProfile": "viewer",
	"MCPListScheduledTasks": "viewer",

	// === Operator: create + operate ===
	"StartCase": "operator", "StopCase": "operator",
	"CreateCase": "operator", "CreateAndRunCase": "operator",
	"DeployCase": "operator", "CloneCase": "operator",
	"CreateCustomDeployment": "operator", "StartCustomDeployment": "operator",
	"StopCustomDeployment": "operator", "CloneCustomDeployment": "operator",
	"BatchStartCustomDeployments": "operator", "BatchStopCustomDeployments": "operator",
	"ComposePreview": "operator", "ComposeUp": "operator", "ComposeDown": "operator",
	"SelectComposeFile": "operator",
	"ExecCommand": "operator", "ExecUserdata": "operator",
	"UploadUserdataScript": "operator", "UploadFile": "operator", "DownloadFile": "operator",
	"StartSSHTerminal": "operator", "StartSSHTerminalInstance": "operator",
	"StartSSHTerminalDirect": "operator",
	"WriteToTerminal": "operator", "ResizeTerminal": "operator", "CloseTerminal": "operator",
	"CreateRemoteDirectory": "operator", "WriteRemoteFileContent": "operator",
	"RenameRemoteFile": "operator",
	"StartPortForward": "operator", "StopPortForward": "operator",
	"AIChatStream": "operator", "SmartAgentChatStream": "operator",
	"AgentChatStream": "operator", "DeployAgentChatStream": "operator",
	"TroubleshootAgentChatStream": "operator", "OrchestratorStream": "operator",
	"StopAgentStream": "operator",
	"SubmitAskUserResponse": "operator", "ExportChatLog": "operator",
	"AIRecommendTemplates": "operator", "AIGenerateTemplate": "operator",
	"AICostOptimization": "operator", "RecommendTemplates": "operator",
	"AnalyzeDeploymentError": "operator", "AnalyzeCaseError": "operator",
	"PullTemplate": "operator", "CreateLocalTemplate": "operator",
	"SaveTemplateFiles": "operator", "CopyTemplate": "operator",
	"ExportTemplates": "operator", "ImportTemplates": "operator",
	"CopyFileTo": "operator",
	"SaveConfigTemplate": "operator", "LoadConfigTemplate": "operator",
	"ListConfigTemplates": "operator",
	"ScheduleTask": "operator", "ScheduleTaskWithRepeat": "operator",
	"ScheduleTaskFull": "operator", "CancelScheduledTask": "operator",
	"SetCaseTags": "operator",
	"SetActiveProfile": "operator", "SwitchProject": "operator",
	"InstallPlugin": "operator", "EnablePlugin": "operator",
	"DisablePlugin": "operator", "UpdatePlugin": "operator",
	"SavePluginConfig": "operator",
	// MCP write
	"MCPComposePreview": "operator", "MCPComposeUp": "operator", "MCPComposeDown": "operator",
	"MCPStartCustomDeployment": "operator", "MCPStopCustomDeployment": "operator",
	"MCPSwitchProject": "operator", "MCPSetActiveProfile": "operator",
	"MCPScheduleTask": "operator", "MCPCancelScheduledTask": "operator",
	"MCPSaveTemplateFiles": "operator", "MCPSaveComposeFile": "operator",
	"EnsureF8x": "operator", "RunF8xInstall": "operator",

	// === Admin: destructive + system config ===
	// RemoveCase, Delete*, Clear*, Batch*Delete*, system config, user management
	// Not listed here → defaults to "admin"
}

// resolveUser finds the user by token and returns their role.
// Returns ("admin", true) for the master token, ("", false) if not found.
func (s *HTTPServer) resolveUser(r *http.Request) (string, string, bool) {
	token := ""
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		token = strings.TrimPrefix(auth, "Bearer ")
	} else {
		token = r.URL.Query().Get("token")
	}
	if token == "" {
		return "", "", false
	}
	// Master token = admin
	if s.token != "" && token == s.token {
		return "admin", "admin", true
	}
	// Check user tokens
	for _, u := range s.users {
		if u.Token == token {
			return u.Username, u.Role, true
		}
	}
	return "", "", false
}

// NewHTTPServer creates a new HTTP server instance
func NewHTTPServer(app *App, host string, port int, token string, users []redc.HTTPUser) *HTTPServer {
	return &HTTPServer{
		app:   app,
		hub:   newSSEHub(),
		token: token,
		host:  host,
		port:  port,
		users: users,
	}
}

// broadcast sends an event to all SSE clients
func (s *HTTPServer) broadcast(name string, data interface{}) {
	s.hub.broadcast(name, data)
}

// GenerateToken generates a random token
func GenerateToken() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// auditLog records an operation to the audit store
func (s *HTTPServer) auditLog(username, role, method string, args []json.RawMessage, ip string, success bool, errMsg string) {
	if s.app.auditStore == nil {
		return
	}
	// Extract first 2 args for context (e.g. caseID, name)
	argsStr := ""
	if len(args) > 0 {
		preview := make([]json.RawMessage, 0, 2)
		for i, a := range args {
			if i >= 2 {
				break
			}
			preview = append(preview, a)
		}
		if b, err := json.Marshal(preview); err == nil {
			argsStr = string(b)
		}
	}
	// Strip port from RemoteAddr
	clientIP := ip
	if idx := strings.LastIndex(ip, ":"); idx > 0 {
		clientIP = ip[:idx]
	}
	go s.app.auditStore.Log(username, role, method, argsStr, clientIP, success, errMsg)
}

// isAuditableMethod returns true for methods that should be recorded in the audit log.
// Read-only methods (Get*, List*, Fetch*, Has*, Validate*, Estimate*) are excluded.
// High-frequency terminal I/O (WriteToTerminal, ResizeTerminal) is also excluded.
func isAuditableMethod(method string) bool {
	for _, prefix := range []string{"Get", "List", "Fetch", "Has", "Validate", "Estimate", "Check"} {
		if strings.HasPrefix(method, prefix) {
			return false
		}
	}
	switch method {
	case "WriteToTerminal", "ResizeTerminal", "SubmitAskUserResponse":
		return false
	}
	return true
}

// Start starts the HTTP server
func (s *HTTPServer) Start(staticFS fs.FS) error {
	mux := http.NewServeMux()

	// Auth middleware helper — returns (username, role, ok)
	checkAuth := func(r *http.Request) (string, string, bool) {
		if s.token == "" && len(s.users) == 0 {
			return "anonymous", "admin", true
		}
		return s.resolveUser(r)
	}

	// POST /api/call — dispatch to App methods
	mux.HandleFunc("/api/call", func(w http.ResponseWriter, r *http.Request) {
		username, role, ok := checkAuth(r)
		if !ok {
			http.Error(w, "Unauthorized", 401)
			return
		}
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", 405)
			return
		}

		var req struct {
			Method string            `json:"method"`
			Args   []json.RawMessage `json:"args"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		// Permission check
		requiredRole := "admin" // default: admin for unlisted methods
		if r, ok := methodMinRole[req.Method]; ok {
			requiredRole = r
		}
		if RoleLevel(role) < RoleLevel(requiredRole) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(403)
			errMsg := fmt.Sprintf("权限不足: 用户 %s (%s) 无权执行 %s，需要 %s 权限", username, role, req.Method, requiredRole)
			// Audit: permission denied
			s.auditLog(username, role, req.Method, req.Args, r.RemoteAddr, false, errMsg)
			json.NewEncoder(w).Encode(map[string]interface{}{"error": errMsg})
			return
		}

		result, err := s.dispatch(req.Method, req.Args)

		// Audit: log write operations (operator+admin methods)
		if isAuditableMethod(req.Method) {
			errMsg := ""
			if err != nil {
				errMsg = err.Error()
			}
			s.auditLog(username, role, req.Method, req.Args, r.RemoteAddr, err == nil, errMsg)
		}

		w.Header().Set("Content-Type", "application/json")
		if err != nil {
			json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{"result": result})
	})

	// GET /api/events — SSE stream
	mux.HandleFunc("/api/events", func(w http.ResponseWriter, r *http.Request) {
		_, _, ok := checkAuth(r)
		if !ok {
			http.Error(w, "Unauthorized", 401)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		client := s.hub.subscribe()
		defer s.hub.unsubscribe(client)

		// Send ping immediately to confirm connection
		fmt.Fprintf(w, "data: {\"event\":\"connected\"}\n\n")
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}

		for {
			select {
			case msg, ok := <-client.ch:
				if !ok {
					return
				}
				fmt.Fprintf(w, "data: %s\n\n", msg)
				if f, ok := w.(http.Flusher); ok {
					f.Flush()
				}
			case <-r.Context().Done():
				return
			}
		}
	})

	// Login check endpoint — returns role info
	mux.HandleFunc("/api/auth", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		username, role, ok := checkAuth(r)
		if ok {
			json.NewEncoder(w).Encode(map[string]interface{}{"ok": true, "username": username, "role": role})
		} else {
			w.WriteHeader(401)
			json.NewEncoder(w).Encode(map[string]interface{}{"ok": false})
		}
	})

	// File upload endpoint for browser mode
	mux.HandleFunc("/api/upload", func(w http.ResponseWriter, r *http.Request) {
		_, role, ok := checkAuth(r)
		if !ok {
			http.Error(w, "Unauthorized", 401)
			return
		}
		if RoleLevel(role) < RoleLevel("operator") {
			http.Error(w, "Forbidden: requires operator role", 403)
			return
		}
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", 405)
			return
		}
		r.ParseMultipartForm(32 << 20) // 32MB max
		file, header, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "No file provided", 400)
			return
		}
		defer file.Close()

		// Save to temp directory
		tmpDir := os.TempDir()
		tmpFile, err := os.CreateTemp(tmpDir, "redc-upload-*-"+header.Filename)
		if err != nil {
			http.Error(w, "Failed to create temp file", 500)
			return
		}
		defer tmpFile.Close()
		io.Copy(tmpFile, file)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"path": tmpFile.Name()})
	})

	// Static files (SPA fallback)
	subFS, err := fs.Sub(staticFS, "frontend/dist")
	if err != nil {
		return fmt.Errorf("failed to access static files: %w", err)
	}
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}
		// Try to open the file
		f, err := subFS.Open(path)
		if err != nil {
			// SPA fallback: serve index.html for unknown paths
			path = "index.html"
			f, err = subFS.Open(path)
			if err != nil {
				http.Error(w, "Not found", 404)
				return
			}
		}
		defer f.Close()

		stat, err := f.Stat()
		if err != nil || stat.IsDir() {
			// If it's a directory, try index.html inside it or fallback
			f.Close()
			path = "index.html"
			f, err = subFS.Open(path)
			if err != nil {
				http.Error(w, "Not found", 404)
				return
			}
			defer func() { /* already deferred above, but overwritten */ }()
			stat, _ = f.Stat()
		}

		// Determine content type from extension
		contentType := "application/octet-stream"
		if strings.HasSuffix(path, ".html") {
			contentType = "text/html; charset=utf-8"
		} else if strings.HasSuffix(path, ".js") {
			contentType = "application/javascript"
		} else if strings.HasSuffix(path, ".css") {
			contentType = "text/css"
		} else if strings.HasSuffix(path, ".json") {
			contentType = "application/json"
		} else if strings.HasSuffix(path, ".svg") {
			contentType = "image/svg+xml"
		} else if strings.HasSuffix(path, ".png") {
			contentType = "image/png"
		} else if strings.HasSuffix(path, ".woff2") {
			contentType = "font/woff2"
		} else if strings.HasSuffix(path, ".woff") {
			contentType = "font/woff"
		}
		w.Header().Set("Content-Type", contentType)

		if rs, ok := f.(io.ReadSeeker); ok {
			http.ServeContent(w, r, path, stat.ModTime(), rs)
		} else {
			w.Header().Set("Content-Length", fmt.Sprintf("%d", stat.Size()))
			io.Copy(w, f)
		}
	})

	addr := fmt.Sprintf("%s:%d", s.host, s.port)
	s.srv = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	go func() {
		if err := s.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("[HTTP Server] Error: %v\n", err)
		}
	}()

	return nil
}

// Stop stops the HTTP server
func (s *HTTPServer) Stop() error {
	if s.srv == nil {
		return nil
	}
	// Close all SSE clients first so Shutdown doesn't block on long-lived connections
	s.hub.closeAll()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err := s.srv.Shutdown(ctx)
	if err != nil {
		// Force close if graceful shutdown times out
		s.srv.Close()
		err = nil
	}
	return err
}

// blockedInHTTPMode lists methods that require native OS dialogs
var blockedInHTTPMode = map[string]bool{
	"SelectFile":      true,
	"SelectDirectory": true,
	"SelectSaveFile":  true,
}

// dispatch calls an App method by name using reflection
func (s *HTTPServer) dispatch(method string, args []json.RawMessage) (interface{}, error) {
	if blockedInHTTPMode[method] {
		return nil, fmt.Errorf("此功能在浏览器模式下不可用，请使用桌面应用")
	}

	appVal := reflect.ValueOf(s.app)
	m := appVal.MethodByName(method)
	if !m.IsValid() {
		return nil, fmt.Errorf("method %s not found", method)
	}

	mt := m.Type()
	if mt.NumIn() != len(args) {
		return nil, fmt.Errorf("method %s expects %d args, got %d", method, mt.NumIn(), len(args))
	}

	in := make([]reflect.Value, len(args))
	for i, arg := range args {
		paramType := mt.In(i)
		paramPtr := reflect.New(paramType)
		if err := json.Unmarshal(arg, paramPtr.Interface()); err != nil {
			return nil, fmt.Errorf("arg %d: %w", i, err)
		}
		in[i] = paramPtr.Elem()
	}

	out := m.Call(in)

	if len(out) == 0 {
		return nil, nil
	}

	// Check if last return is error
	last := out[len(out)-1]
	if last.Type().Implements(reflect.TypeOf((*error)(nil)).Elem()) {
		if !last.IsNil() {
			return nil, last.Interface().(error)
		}
		out = out[:len(out)-1]
	}

	if len(out) == 0 {
		return nil, nil
	}
	if len(out) == 1 {
		return out[0].Interface(), nil
	}

	// Multiple return values → return as array
	results := make([]interface{}, len(out))
	for i, v := range out {
		results[i] = v.Interface()
	}
	return results, nil
}
