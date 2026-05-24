package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateSessionToken_LengthAndUniqueness(t *testing.T) {
	t1, err := GenerateSessionToken()
	if err != nil {
		t.Fatalf("GenerateSessionToken: %v", err)
	}
	t2, err := GenerateSessionToken()
	if err != nil {
		t.Fatalf("GenerateSessionToken: %v", err)
	}
	if t1.Value() == t2.Value() {
		t.Error("two consecutive tokens should not be equal — entropy source broken")
	}
	// 32 bytes -> 64 hex chars
	if len(t1.Value()) != 64 {
		t.Errorf("token length: got %d, want 64", len(t1.Value()))
	}
	for _, c := range t1.Value() {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			t.Errorf("token contains non-hex byte %q", c)
			break
		}
	}
}

func TestSessionToken_Equal_ConstantTime(t *testing.T) {
	tok, err := GenerateSessionToken()
	if err != nil {
		t.Fatalf("GenerateSessionToken: %v", err)
	}
	if !tok.Equal(tok.Value()) {
		t.Error("token should equal its own Value()")
	}
	if tok.Equal("wrong") {
		t.Error("token should not equal an obviously-wrong string")
	}
	if tok.Equal("") {
		t.Error("token should not equal empty string")
	}
	var nilTok *SessionToken
	if nilTok.Equal("anything") {
		t.Error("nil token should never equal anything")
	}
}

func TestSessionToken_Persist_Mode0600_AtomicWrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "token")
	tok, err := GenerateSessionToken()
	if err != nil {
		t.Fatalf("GenerateSessionToken: %v", err)
	}
	abs, err := tok.Persist(path)
	if err != nil {
		t.Fatalf("Persist: %v", err)
	}
	if abs == "" {
		t.Error("Persist should return non-empty resolved path")
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if mode := info.Mode().Perm(); mode != 0o600 {
		t.Errorf("token file mode: got %o, want 600", mode)
	}
	// Parent dir should be 0700 because we MkdirAll'd it.
	parentInfo, err := os.Stat(filepath.Dir(path))
	if err != nil {
		t.Fatalf("Stat parent: %v", err)
	}
	if mode := parentInfo.Mode().Perm(); mode != 0o700 {
		t.Errorf("token dir mode: got %o, want 700", mode)
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if strings.TrimSpace(string(contents)) != tok.Value() {
		t.Error("token file contents do not match token value")
	}
	// Re-persist should overwrite atomically without leaving temp files.
	if _, err := tok.Persist(path); err != nil {
		t.Errorf("repeat Persist: %v", err)
	}
	entries, _ := os.ReadDir(filepath.Dir(path))
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".gvid-token-") {
			t.Errorf("temp file left behind: %s", e.Name())
		}
	}
}

func TestSessionToken_Persist_NilReceiver(t *testing.T) {
	var tok *SessionToken
	if _, err := tok.Persist("/tmp/whatever"); err == nil {
		t.Error("nil receiver Persist should return error")
	}
}

func TestIsLoopbackHost(t *testing.T) {
	cases := []struct {
		host    string
		want    bool
		comment string
	}{
		{"localhost", true, "canonical loopback name"},
		{"127.0.0.1", true, "IPv4 loopback"},
		{"127.0.0.42", true, "127.0.0.0/8 loopback range"},
		{"::1", true, "IPv6 loopback"},
		{"[::1]", true, "bracketed IPv6 loopback"},
		{"", false, "empty host binds 0.0.0.0 in net/http — MUST reject (PR #12 Gemini security finding)"},
		{"0.0.0.0", false, "bind-all IPv4 — must reject"},
		{"::", false, "bind-all IPv6 — must reject"},
		{"192.168.1.10", false, "private LAN — must reject"},
		{"10.0.0.1", false, "private LAN — must reject"},
		{"169.254.169.254", false, "link-local — must reject (cloud metadata)"},
		{"203.0.113.50", false, "TEST-NET-3 public range — must reject"},
		{"my-host.example", false, "hostname other than 'localhost' — must reject without DNS"},
	}
	for _, tc := range cases {
		t.Run(tc.host+"/"+tc.comment, func(t *testing.T) {
			got := IsLoopbackHost(tc.host)
			if got != tc.want {
				t.Errorf("IsLoopbackHost(%q) = %v, want %v (%s)", tc.host, got, tc.want, tc.comment)
			}
		})
	}
}

func TestOriginAllowlistMiddleware_NoOriginPassesThrough(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})
	mw := OriginAllowlistMiddleware([]string{"http://localhost:5173"})(next)

	req := httptest.NewRequest("GET", "/api/v1/health", nil)
	// Deliberately no Origin header set — native client.
	w := httptest.NewRecorder()
	mw.ServeHTTP(w, req)

	if !called {
		t.Error("native client (no Origin) should pass through to handler")
	}
	if w.Code != http.StatusOK {
		t.Errorf("status: got %d, want 200", w.Code)
	}
}

func TestOriginAllowlistMiddleware_AllowedOriginPasses(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})
	mw := OriginAllowlistMiddleware([]string{"http://localhost:5173"})(next)

	req := httptest.NewRequest("GET", "/api/v1/health", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	w := httptest.NewRecorder()
	mw.ServeHTTP(w, req)

	if !called {
		t.Error("allowed origin should pass through")
	}
	if w.Code != http.StatusOK {
		t.Errorf("status: got %d, want 200", w.Code)
	}
}

func TestOriginAllowlistMiddleware_DisallowedOriginRejected(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})
	mw := OriginAllowlistMiddleware([]string{"http://localhost:5173"})(next)

	// Simulates a drive-by from any other origin — including DNS-rebound
	// attackers and other localhost ports.
	for _, badOrigin := range []string{
		"http://evil.example",
		"https://attacker.local",
		"http://localhost:9999", // wrong port, even on localhost
		"http://127.0.0.1:5173", // raw IP variant of the allowed origin
		"null",                  // sandboxed iframes / file:// pages send "null"
	} {
		t.Run(badOrigin, func(t *testing.T) {
			called = false
			req := httptest.NewRequest("GET", "/api/v1/health", nil)
			req.Header.Set("Origin", badOrigin)
			w := httptest.NewRecorder()
			mw.ServeHTTP(w, req)

			if called {
				t.Errorf("handler must NOT be called for disallowed origin %q", badOrigin)
			}
			if w.Code != http.StatusForbidden {
				t.Errorf("status for %q: got %d, want 403", badOrigin, w.Code)
			}
			if !strings.Contains(w.Body.String(), "ORIGIN_REJECTED") {
				t.Errorf("response body should mention ORIGIN_REJECTED, got %q", w.Body.String())
			}
		})
	}
}

func TestOriginAllowlistMiddleware_WildcardAllowsAny(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})
	mw := OriginAllowlistMiddleware([]string{"*"})(next)

	req := httptest.NewRequest("GET", "/api/v1/health", nil)
	req.Header.Set("Origin", "http://anywhere.example")
	w := httptest.NewRecorder()
	mw.ServeHTTP(w, req)

	if !called {
		t.Error(`wildcard "*" allowlist should pass any origin`)
	}
}

func TestRequireTokenMiddleware_RejectsMissingToken(t *testing.T) {
	tok, _ := GenerateSessionToken()
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler must NOT be reached without token")
	})
	mw := RequireTokenMiddleware(tok)(next)
	req := httptest.NewRequest("POST", "/api/v1/whatever", nil)
	w := httptest.NewRecorder()
	mw.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status: got %d, want 401", w.Code)
	}
	if !strings.Contains(w.Body.String(), "TOKEN_REQUIRED") {
		t.Errorf("body should mention TOKEN_REQUIRED, got %q", w.Body.String())
	}
}

func TestRequireTokenMiddleware_AcceptsBearerHeader(t *testing.T) {
	tok, _ := GenerateSessionToken()
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})
	mw := RequireTokenMiddleware(tok)(next)
	req := httptest.NewRequest("POST", "/api/v1/whatever", nil)
	req.Header.Set("Authorization", "Bearer "+tok.Value())
	w := httptest.NewRecorder()
	mw.ServeHTTP(w, req)
	if !called {
		t.Errorf("handler must be reached with valid Bearer; got status %d, body %q", w.Code, w.Body.String())
	}
}

func TestRequireTokenMiddleware_AcceptsXGvidTokenHeader(t *testing.T) {
	tok, _ := GenerateSessionToken()
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})
	mw := RequireTokenMiddleware(tok)(next)
	req := httptest.NewRequest("POST", "/api/v1/whatever", nil)
	req.Header.Set("X-Gvid-Token", tok.Value())
	w := httptest.NewRecorder()
	mw.ServeHTTP(w, req)
	if !called {
		t.Errorf("handler must be reached with valid X-Gvid-Token; got status %d", w.Code)
	}
}

func TestRequireTokenMiddleware_RejectsWrongToken(t *testing.T) {
	tok, _ := GenerateSessionToken()
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler must NOT be reached with wrong token")
	})
	mw := RequireTokenMiddleware(tok)(next)
	req := httptest.NewRequest("POST", "/api/v1/whatever", nil)
	req.Header.Set("Authorization", "Bearer "+strings.Repeat("0", 64))
	w := httptest.NewRecorder()
	mw.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status: got %d, want 401", w.Code)
	}
	if !strings.Contains(w.Body.String(), "TOKEN_INVALID") {
		t.Errorf("body should mention TOKEN_INVALID, got %q", w.Body.String())
	}
}

func TestRequireTokenMiddleware_NilTokenFails(t *testing.T) {
	// Defensive: if the daemon somehow reached a request without generating
	// a token, the gate must refuse rather than silently bypass.
	mw := RequireTokenMiddleware(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler must NOT be reached when token is nil")
	}))
	req := httptest.NewRequest("POST", "/api/v1/whatever", nil)
	req.Header.Set("Authorization", "Bearer something")
	w := httptest.NewRecorder()
	mw.ServeHTTP(w, req)
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status: got %d, want 503", w.Code)
	}
}

func TestExtractToken_BearerPrecedenceOverXGvid(t *testing.T) {
	// When both headers are present, Authorization Bearer wins. This avoids
	// the surprise of a client setting both headers with different values
	// and the gate accepting one but logging the other.
	req := httptest.NewRequest("POST", "/", nil)
	req.Header.Set("Authorization", "Bearer aaaaaa")
	req.Header.Set("X-Gvid-Token", "bbbbbb")
	if got := extractToken(req); got != "aaaaaa" {
		t.Errorf("extractToken: got %q, want %q (Bearer should win)", got, "aaaaaa")
	}
}

func TestExtractToken_NonBearerAuthorizationIgnored(t *testing.T) {
	// `Authorization: Basic ...` is not the Bearer scheme and must not be
	// silently accepted as a token candidate.
	req := httptest.NewRequest("POST", "/", nil)
	req.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
	req.Header.Set("X-Gvid-Token", "xgvid-value")
	if got := extractToken(req); got != "xgvid-value" {
		t.Errorf("extractToken: got %q, want %q (Basic should be ignored, fall back to X-Gvid-Token)", got, "xgvid-value")
	}
}

func TestDefaultSessionTokenPath_UnderUserConfigDir(t *testing.T) {
	got, err := DefaultSessionTokenPath()
	if err != nil {
		t.Skipf("user config dir unavailable in this test env: %v", err)
	}
	if !strings.HasSuffix(got, filepath.Join("gvid", "token")) {
		t.Errorf("default token path %q should end in gvid/token", got)
	}
}
