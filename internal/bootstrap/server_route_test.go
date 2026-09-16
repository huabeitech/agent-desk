package bootstrap

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"agent-desk/internal/pkg/config"
)

func TestNewServerRegistersGinRoutes(t *testing.T) {
	config.SetCurrent(&config.Config{
		Storage: config.StorageConfig{
			Local: config.LocalStorageConfig{
				Root:    "storage",
				BaseURL: "/storage",
			},
		},
	})

	app, err := NewServer()
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}

	routes := make(map[string]bool)
	for _, route := range app.Routes() {
		routes[route.Method+" "+route.Path] = true
	}

	expected := []string{
		http.MethodPost + " /api/auth/login",
		http.MethodGet + " /api/config",
		http.MethodGet + " /api/avatar/user/:userId",
		http.MethodGet + " /api/avatar/agent/:agentProfileId",
		http.MethodGet + " /api/health",
		http.MethodGet + " /api/auth/oidc_login",
		http.MethodGet + " /api/auth/oidc_callback",
		http.MethodGet + " /api/auth/callback/custom",
		http.MethodPost + " /api/auth/oidc_exchange",
		http.MethodGet + " /api/auth/profile",
		http.MethodPost + " /api/webhooks/org-sync",
		http.MethodGet + " /api/dashboard/organization/my_list",
		http.MethodPost + " /api/dashboard/organization/switch",
		http.MethodGet + " /api/dashboard/user/list",
		http.MethodGet + " /api/dashboard/user/:id",
		http.MethodPost + " /api/dashboard/user/create",
		http.MethodPost + " /api/dashboard/conversation/send_message",
		http.MethodGet + " /api/dashboard/ai-workflow/default-definition",
		http.MethodGet + " /api/dashboard/ai-workflow/template/list",
		http.MethodGet + " /api/dashboard/ai-workflow/run/list",
		http.MethodGet + " /api/dashboard/ai-workflow/run/:id",
		http.MethodGet + " /api/dashboard/agent-run/metrics",
		http.MethodPost + " /api/dashboard/agent-run/evaluate",
		http.MethodGet + " /api/dashboard/agent-run/:id",
		http.MethodPost + " /api/dashboard/ai-agent/rollback_rollout",
		http.MethodPost + " /api/dashboard/channel/rollback_ai_agent_rollout",
		http.MethodPost + " /api/dashboard/agent-run/quality_feedback",
		http.MethodGet + " /api/dashboard/agent-run/list",
		http.MethodGet + " /api/ws/dashboard",
		http.MethodGet + " /api/ws/open",
	}
	for _, route := range expected {
		if !routes[route] {
			t.Fatalf("expected route %s to be registered", route)
		}
	}
}

func TestNewServerHealthEndpointIsPublic(t *testing.T) {
	config.SetCurrent(&config.Config{
		Storage: config.StorageConfig{
			Local: config.LocalStorageConfig{
				Root:    "storage",
				BaseURL: "/storage",
			},
		},
	})

	app, err := NewServer()
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}

	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/health", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var body struct {
		Success bool `json:"success"`
		Data    struct {
			Status string `json:"status"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if !body.Success {
		t.Fatalf("success=false, body=%s", rec.Body.String())
	}
	if body.Data.Status != "ok" {
		t.Fatalf("status=%q want ok", body.Data.Status)
	}
}

func TestNewServerExposesPublicConfig(t *testing.T) {
	config.SetCurrent(&config.Config{
		Language: "zh-CN",
		Storage: config.StorageConfig{
			Local: config.LocalStorageConfig{
				Root:    "storage",
				BaseURL: "/storage",
			},
		},
		WxWork: config.WxWorkConfig{
			Enabled: true,
		},
		OIDC: config.OIDCConfig{
			Enabled:      false,
			ClientSecret: "must-not-leak",
		},
	})

	app, err := NewServer()
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}

	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/config", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d want %d", rec.Code, http.StatusOK)
	}

	var body struct {
		Success bool `json:"success"`
		Data    struct {
			Language             string `json:"language"`
			PasswordLoginEnabled bool   `json:"passwordLoginEnabled"`
			WxWorkEnabled        bool   `json:"wxworkEnabled"`
			OIDCEnabled          bool   `json:"oidcEnabled"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if !body.Success {
		t.Fatalf("success=false, body=%s", rec.Body.String())
	}
	if body.Data.Language != "zh-CN" {
		t.Fatalf("language=%q want zh-CN", body.Data.Language)
	}
	if !body.Data.PasswordLoginEnabled {
		t.Fatalf("passwordLoginEnabled=false want true")
	}
	if !body.Data.WxWorkEnabled {
		t.Fatalf("wxworkEnabled=false want true")
	}
	if body.Data.OIDCEnabled {
		t.Fatalf("oidcEnabled=true want false")
	}
	if strings.Contains(rec.Body.String(), "must-not-leak") {
		t.Fatalf("response leaked sensitive OIDC config: %s", rec.Body.String())
	}
}

func TestNewServerDoesNotExposeLegacyAuthOptions(t *testing.T) {
	config.SetCurrent(&config.Config{
		Storage: config.StorageConfig{
			Local: config.LocalStorageConfig{
				Root:    "storage",
				BaseURL: "/storage",
			},
		},
	})

	app, err := NewServer()
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}

	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/auth/options", nil))

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d want %d, body=%s", rec.Code, http.StatusNotFound, rec.Body.String())
	}
}

func TestNewServerPasswordLoginDisabled(t *testing.T) {
	disabled := false
	config.SetCurrent(&config.Config{
		Auth: config.AuthConfig{
			PasswordLoginEnabled: &disabled,
		},
		Storage: config.StorageConfig{
			Local: config.LocalStorageConfig{
				Root:    "storage",
				BaseURL: "/storage",
			},
		},
	})

	app, err := NewServer()
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}

	// 1. /api/config should return passwordLoginEnabled=false
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/config", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d want %d", rec.Code, http.StatusOK)
	}
	var configBody struct {
		Data struct {
			PasswordLoginEnabled bool `json:"passwordLoginEnabled"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &configBody); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if configBody.Data.PasswordLoginEnabled {
		t.Fatalf("expected passwordLoginEnabled=false, got true")
	}

	// 2. /api/auth/login should be rejected
	rec = httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"username":"admin","password":"secret"}`))
	req.Header.Set("Content-Type", "application/json")
	app.ServeHTTP(rec, req)

	var loginBody struct {
		Success bool `json:"success"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &loginBody); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if loginBody.Success {
		t.Fatalf("expected login failure when password login is disabled")
	}
}

func TestNewServerSeparatesAPIStaticAndSPA(t *testing.T) {
	for _, rootDir := range []string{"web/out", "../web/out", "../../web/out"} {
		_ = os.MkdirAll(rootDir, 0o755)
		_ = os.WriteFile(filepath.Join(rootDir, "index.html"), []byte("<html>spa</html>"), 0o644)
		defer func(d string) {
			_ = os.Remove(filepath.Join(d, "index.html"))
		}(rootDir)
	}

	config.SetCurrent(&config.Config{
		Storage: config.StorageConfig{
			Local: config.LocalStorageConfig{
				Root:    "storage",
				BaseURL: "/storage",
			},
		},
	})

	app, err := NewServer()
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}

	tests := []struct {
		path        string
		wantStatus  int
		contentType string
	}{
		{path: "/api/not-exists", wantStatus: http.StatusNotFound, contentType: "application/json"},
		{path: "/dashboard/not-exists", wantStatus: http.StatusOK, contentType: "text/html"},
		{path: "/support/docs/runtime-authored-slug", wantStatus: http.StatusOK, contentType: "text/html"},
		{path: "/support/community/posts/12345", wantStatus: http.StatusOK, contentType: "text/html"},
		{path: "/support/community/categories/product", wantStatus: http.StatusOK, contentType: "text/html"},
		{path: "/support/community/posts/new", wantStatus: http.StatusOK, contentType: "text/html"},
	}

	for _, tt := range tests {
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tt.path, nil))

		if rec.Code != tt.wantStatus {
			t.Fatalf("%s status=%d want %d", tt.path, rec.Code, tt.wantStatus)
		}
		if !strings.Contains(rec.Header().Get("Content-Type"), tt.contentType) {
			t.Fatalf("%s Content-Type=%q want %q", tt.path, rec.Header().Get("Content-Type"), tt.contentType)
		}
	}
}

func TestNewServerHardensStoredAssetResponses(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"screenshot.png", "archive.zip", "legacy-page.html"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte("payload"), 0o644); err != nil {
			t.Fatalf("WriteFile(%s) error = %v", name, err)
		}
	}

	config.SetCurrent(&config.Config{
		Storage: config.StorageConfig{
			Local: config.LocalStorageConfig{
				Root:    root,
				BaseURL: "/storage",
			},
		},
	})

	app, err := NewServer()
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}

	tests := []struct {
		path           string
		wantStatus     int
		contentType    string
		wantAttachment bool
	}{
		{path: "/storage/screenshot.png", wantStatus: http.StatusOK, contentType: "image/png"},
		{path: "/storage/archive.zip", wantStatus: http.StatusOK, wantAttachment: true},
		// A file planted before the upload policy existed must still not render.
		{path: "/storage/legacy-page.html", wantStatus: http.StatusOK, wantAttachment: true},
		{path: "/storage/missing.png", wantStatus: http.StatusNotFound},
	}

	for _, tt := range tests {
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tt.path, nil))

		if rec.Code != tt.wantStatus {
			t.Fatalf("%s status=%d want %d", tt.path, rec.Code, tt.wantStatus)
		}
		if got := rec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
			t.Fatalf("%s X-Content-Type-Options=%q want nosniff", tt.path, got)
		}
		if tt.contentType != "" && !strings.Contains(rec.Header().Get("Content-Type"), tt.contentType) {
			t.Fatalf("%s Content-Type=%q want %q", tt.path, rec.Header().Get("Content-Type"), tt.contentType)
		}
		got := rec.Header().Get("Content-Disposition")
		if tt.wantAttachment && got != "attachment" {
			t.Fatalf("%s Content-Disposition=%q want attachment", tt.path, got)
		}
		if !tt.wantAttachment && got != "" {
			t.Fatalf("%s Content-Disposition=%q want empty so the asset renders inline", tt.path, got)
		}
	}
}

func TestNewServerAllowsConfiguredCORSOrigin(t *testing.T) {
	config.SetCurrent(&config.Config{
		Server: config.ServerConfig{
			CORS: config.CORSConfig{
				AllowedOrigins: []string{"https://console.example.com"},
			},
		},
		Storage: config.StorageConfig{
			Local: config.LocalStorageConfig{
				Root:    "storage",
				BaseURL: "/storage",
			},
		},
	})

	app, err := NewServer()
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/api/auth/login", nil)
	req.Header.Set("Origin", "https://console.example.com")
	req.Header.Set("Access-Control-Request-Method", http.MethodPost)
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status=%d want %d", rec.Code, http.StatusNoContent)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://console.example.com" {
		t.Fatalf("Access-Control-Allow-Origin=%q want %q", got, "https://console.example.com")
	}
	if got := rec.Header().Get("Access-Control-Allow-Methods"); !strings.Contains(got, http.MethodPost) {
		t.Fatalf("Access-Control-Allow-Methods=%q should contain %q", got, http.MethodPost)
	}
	if got := rec.Header().Get("Vary"); got != "Origin" {
		t.Fatalf("Vary=%q want %q", got, "Origin")
	}
}

func TestNewServerRejectsUnconfiguredCORSOrigin(t *testing.T) {
	config.SetCurrent(&config.Config{
		Server: config.ServerConfig{
			CORS: config.CORSConfig{
				AllowedOrigins: []string{"https://console.example.com"},
			},
		},
		Storage: config.StorageConfig{
			Local: config.LocalStorageConfig{
				Root:    "storage",
				BaseURL: "/storage",
			},
		},
	})

	app, err := NewServer()
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/api/auth/login", nil)
	req.Header.Set("Origin", "https://evil.example.com")
	req.Header.Set("Access-Control-Request-Method", http.MethodPost)
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d want %d", rec.Code, http.StatusForbidden)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("Access-Control-Allow-Origin=%q want empty", got)
	}
}

// TestNewServerPublicConfigEndpointsAllowAnyOrigin 验证嵌入式 SDK 挂件预拉取的
// 两个公开只读接口对任意来源放行（含带 X-Channel-Id 头触发的预检请求），
// 同时非公开接口仍受白名单约束。
func TestNewServerPublicConfigEndpointsAllowAnyOrigin(t *testing.T) {
	config.SetCurrent(&config.Config{
		Language: "zh-CN",
		Server: config.ServerConfig{
			CORS: config.CORSConfig{
				// 白名单为空：公开接口仍应放行，非公开接口应被拒。
				AllowedOrigins: nil,
			},
		},
		Storage: config.StorageConfig{
			Local: config.LocalStorageConfig{
				Root:    "storage",
				BaseURL: "/storage",
			},
		},
	})

	app, err := NewServer()
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}

	cases := []struct {
		name       string
		method     string
		path       string
		origin     string
		extraHdrs  map[string]string
		wantStatus int
		wantACAO   string
	}{
		{
			name:       "GET /api/config with arbitrary origin",
			method:     http.MethodGet,
			path:       "/api/config",
			origin:     "http://localhost:5201",
			wantStatus: http.StatusOK,
			wantACAO:   "*",
		},
		{
			name:       "OPTIONS /api/config preflight passes for arbitrary origin",
			method:     http.MethodOptions,
			path:       "/api/config",
			origin:     "http://localhost:5201",
			extraHdrs:  map[string]string{"Access-Control-Request-Method": http.MethodGet},
			wantStatus: http.StatusNoContent,
			wantACAO:   "*",
		},
		{
			name:       "OPTIONS /api/channel/config preflight passes with X-Channel-Id request header",
			method:     http.MethodOptions,
			path:       "/api/channel/config",
			origin:     "http://localhost:5201",
			extraHdrs:  map[string]string{"Access-Control-Request-Method": http.MethodGet, "Access-Control-Request-Headers": "X-Channel-Id"},
			wantStatus: http.StatusNoContent,
			wantACAO:   "*",
		},
		{
			name:       "GET /api/config without Origin header still works (same-origin)",
			method:     http.MethodGet,
			path:       "/api/config",
			origin:     "",
			wantStatus: http.StatusOK,
			wantACAO:   "*",
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(tt.method, tt.path, nil)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			for k, v := range tt.extraHdrs {
				req.Header.Set(k, v)
			}
			app.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status=%d want %d, body=%s", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if got := rec.Header().Get("Access-Control-Allow-Origin"); got != tt.wantACAO {
				t.Fatalf("Access-Control-Allow-Origin=%q want %q", got, tt.wantACAO)
			}
			if got := rec.Header().Get("Access-Control-Allow-Headers"); !strings.Contains(got, "X-Channel-Id") {
				t.Fatalf("Access-Control-Allow-Headers=%q should contain X-Channel-Id", got)
			}
		})
	}

	// 回归：非公开接口在白名单为空时，预检应被拒。
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/api/auth/login", nil)
	req.Header.Set("Origin", "http://localhost:5201")
	req.Header.Set("Access-Control-Request-Method", http.MethodPost)
	app.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("non-public preflight status=%d want %d", rec.Code, http.StatusForbidden)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("non-public Access-Control-Allow-Origin=%q want empty", got)
	}
}

func TestNewServerEchoesRequestID(t *testing.T) {
	config.SetCurrent(&config.Config{
		Storage: config.StorageConfig{
			Local: config.LocalStorageConfig{
				Root:    "storage",
				BaseURL: "/storage",
			},
		},
	})

	app, err := NewServer()
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/not-exists", nil)
	req.Header.Set("X-Request-Id", "trace-123")
	app.ServeHTTP(rec, req)

	if got := rec.Header().Get("X-Request-Id"); got != "trace-123" {
		t.Fatalf("X-Request-Id=%q want %q", got, "trace-123")
	}
}

func TestNewServerGeneratesRequestID(t *testing.T) {
	config.SetCurrent(&config.Config{
		Storage: config.StorageConfig{
			Local: config.LocalStorageConfig{
				Root:    "storage",
				BaseURL: "/storage",
			},
		},
	})

	app, err := NewServer()
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}

	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/not-exists", nil))

	if got := rec.Header().Get("X-Request-Id"); got == "" {
		t.Fatalf("X-Request-Id should be generated")
	}
}
