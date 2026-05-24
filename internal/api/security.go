package api

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// SessionToken is a high-entropy random token generated at daemon startup and
// persisted to a 0600-mode file. State-changing endpoints (the human-triage
// POSTs that ship in a later burst — see gastown-3uf) will require the bearer
// to present this token via either `Authorization: Bearer <token>` or the
// `X-Gvid-Token: <token>` header. Read-only endpoints continue to function
// without a token so the existing web UI and `curl` workflows are not broken.
//
// Threat model: protects against same-machine processes (other user-level
// daemons, untrusted MCP servers, drive-by markdown previews that POST to
// localhost) that lack filesystem access to the token file. Does NOT protect
// against an attacker with read access to the token file — at that point the
// attacker is already in-process or has user-shell access and the dashboard
// is the least of the user's problems.
type SessionToken struct {
	raw string
}

// GenerateSessionToken returns a fresh 32-byte (256-bit) token encoded as a
// 64-character hex string. crypto/rand is the entropy source; failure to read
// from it returns an error so the caller can fail-fast at startup rather than
// silently using a low-entropy fallback.
func GenerateSessionToken() (*SessionToken, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return nil, fmt.Errorf("session-token entropy read: %w", err)
	}
	return &SessionToken{raw: hex.EncodeToString(buf)}, nil
}

// Value returns the token's wire representation. Callers should log this
// ONLY at the token file's path (the file itself) — never to stdout/stderr.
func (t *SessionToken) Value() string {
	return t.raw
}

// Equal compares two tokens in constant time. Callers should never use
// `==` on token strings directly because timing attacks on the rejection path
// could leak token bytes one at a time.
func (t *SessionToken) Equal(candidate string) bool {
	if t == nil {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(t.raw), []byte(candidate)) == 1
}

// Persist writes the token to path with 0600 (owner read/write only) and
// creates the parent directory with 0700 if it does not exist. Returns the
// resolved absolute path on success so callers can log it.
func (t *SessionToken) Persist(path string) (string, error) {
	if t == nil {
		return "", errors.New("nil SessionToken")
	}
	if path == "" {
		return "", errors.New("empty token persist path")
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("create token dir %s: %w", dir, err)
	}
	// Write atomically via a temp file in the same dir so a partial write
	// cannot leave a half-written token at the canonical path.
	tmp, err := os.CreateTemp(dir, ".gvid-token-*")
	if err != nil {
		return "", fmt.Errorf("create temp token: %w", err)
	}
	tmpPath := tmp.Name()
	if err := os.Chmod(tmpPath, 0o600); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return "", fmt.Errorf("chmod temp token: %w", err)
	}
	if _, err := tmp.WriteString(t.raw + "\n"); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return "", fmt.Errorf("write temp token: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return "", fmt.Errorf("close temp token: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return "", fmt.Errorf("rename temp token to %s: %w", path, err)
	}
	// Final mode enforcement in case umask interfered with the temp file.
	if err := os.Chmod(path, 0o600); err != nil {
		return "", fmt.Errorf("chmod final token: %w", err)
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return path, nil
	}
	return abs, nil
}

// DefaultSessionTokenPath returns ~/.config/gvid/token (or the equivalent on
// Windows via os.UserConfigDir). Falls back to a temp-dir path if the user
// config dir cannot be resolved — the daemon will still function, but the
// token will not survive a reboot.
func DefaultSessionTokenPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil || dir == "" {
		return "", fmt.Errorf("user config dir: %w", err)
	}
	return filepath.Join(dir, "gvid", "token"), nil
}

// IsLoopbackHost reports whether the host string resolves to a loopback
// address (127.0.0.0/8 or ::1) only. "localhost", "127.0.0.1", "::1", and
// "[::1]" are accepted; "0.0.0.0", "::", and any non-loopback IP are
// rejected. Hostnames other than "localhost" are conservatively rejected
// rather than attempting DNS resolution — the daemon is local-first and
// binding to a name that happens to resolve to a loopback IP today but a
// public IP tomorrow is exactly the deployment surprise we want to avoid.
func IsLoopbackHost(host string) bool {
	host = strings.TrimSpace(host)
	host = strings.Trim(host, "[]")
	switch host {
	case "", "localhost":
		return true
	}
	ip := net.ParseIP(host)
	if ip == nil {
		// Hostname other than "localhost" — reject. Resolving DNS here would
		// let `gvid --host my-public-name.example` slip past a check that
		// happens to resolve loopback today.
		return false
	}
	return ip.IsLoopback()
}

// OriginAllowlistMiddleware hard-rejects requests whose Origin header is
// present and not in the allowlist. Requests without an Origin header are
// passed through — that covers native HTTP clients (curl, the TUI, the gvid
// daemon's own self-tests) which by spec do not send Origin. Browsers always
// send Origin on cross-origin fetch(), so this gate stops DNS rebinding and
// drive-by CSRF attempts from any web page Jeremy might have open in another
// tab on the same dev box.
//
// Council decision Q0 (gastown-cr5 AT-DECR, CISO binding constraint): this
// is the foundation defense before any state-mutating endpoint ships. For
// read-only endpoints it protects data exfiltration (read of confidential
// memories via DNS-rebind would return JSON to attacker JS).
func OriginAllowlistMiddleware(allowed []string) func(http.Handler) http.Handler {
	allowedSet := make(map[string]struct{}, len(allowed))
	for _, o := range allowed {
		allowedSet[o] = struct{}{}
	}
	_, allowAny := allowedSet["*"]
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin == "" {
				// Native client (curl, native app, server-to-server). Browsers
				// always set Origin; absence is a strong signal this is not a
				// cross-origin browser context.
				next.ServeHTTP(w, r)
				return
			}
			if allowAny {
				next.ServeHTTP(w, r)
				return
			}
			if _, ok := allowedSet[origin]; !ok {
				// Hard reject. Distinct from CORS — CORS only tells the
				// browser to drop the response; we want the server to refuse
				// the request entirely so confidential data never leaves the
				// process. 403 over 401 because no credential-based remedy is
				// available (the origin itself is wrong).
				writeError(w, http.StatusForbidden, "ORIGIN_REJECTED",
					fmt.Sprintf("Origin %q is not in the daemon's allowlist; "+
						"set --cors-origin or run the web UI from a permitted origin", origin))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireTokenMiddleware demands `Authorization: Bearer <token>` OR
// `X-Gvid-Token: <token>` on every request. The constant-time SessionToken.Equal
// is used to compare so timing attacks on the rejection path cannot leak
// bytes. Wrapped only around state-mutating endpoints (none in this burst —
// reserved for the human-triage POSTs in gastown-3uf and any future memories
// write-path, though Q2 of the AT-DECR makes the latter an architectural
// invariant: read-only-forever).
func RequireTokenMiddleware(token *SessionToken) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if token == nil {
				// Defensive: a nil token indicates a startup bug (token not
				// generated). Better to refuse than to silently bypass.
				writeError(w, http.StatusServiceUnavailable, "TOKEN_NOT_READY",
					"Session token not initialized; daemon misconfigured")
				return
			}
			candidate := extractToken(r)
			if candidate == "" {
				writeError(w, http.StatusUnauthorized, "TOKEN_REQUIRED",
					"State-changing endpoint requires Authorization: Bearer <token> or X-Gvid-Token header")
				return
			}
			if !token.Equal(candidate) {
				writeError(w, http.StatusUnauthorized, "TOKEN_INVALID",
					"Token does not match the daemon's session token")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// extractToken pulls a candidate token from either the Authorization Bearer
// scheme or the X-Gvid-Token header. The Authorization header takes
// precedence when both are present — by convention that header is the
// canonical place clients learn about, and accepting two values for the same
// secret without a tiebreaker would invite confusion.
func extractToken(r *http.Request) string {
	if auth := r.Header.Get("Authorization"); auth != "" {
		if strings.HasPrefix(auth, "Bearer ") {
			return strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
		}
	}
	return strings.TrimSpace(r.Header.Get("X-Gvid-Token"))
}
