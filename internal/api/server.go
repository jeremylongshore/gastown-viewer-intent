// Package api provides the HTTP API server for Gastown Viewer Intent.
package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/intent-solutions-io/gastown-viewer-intent/internal/beads"
	"github.com/intent-solutions-io/gastown-viewer-intent/internal/gastown"
)

// Config holds server configuration.
type Config struct {
	Port        int
	Host        string
	CORSOrigins []string
	Version     string
	TownRoot    string // Gas Town workspace root (default: ~/gt)

	// SessionTokenPath is the filesystem location where the generated
	// per-process session token will be persisted at startup. Empty means
	// "use DefaultSessionTokenPath() (~/.config/gvid/token)". The token file
	// is always written with mode 0600.
	SessionTokenPath string

	// DisableLoopbackCheck bypasses the startup-time loopback bind check.
	// Off-by-default; enabling this allows binding to 0.0.0.0 / non-loopback
	// addresses. Intended ONLY for ephemeral container test environments;
	// hard-flagged as unsafe in the log line at startup.
	DisableLoopbackCheck bool
}

// DefaultConfig returns configuration with sensible defaults.
func DefaultConfig() Config {
	return Config{
		Port:        7070,
		Host:        "localhost",
		CORSOrigins: []string{"http://localhost:5173"},
		Version:     "0.1.0",
		TownRoot:    "", // Empty means use default ~/gt
	}
}

// Server is the HTTP API server.
type Server struct {
	config    Config
	adapter   beads.Adapter
	gtAdapter gastown.Adapter
	mux       *http.ServeMux
	sse       *SSEBroker

	// sessionToken is the per-process bearer token required by
	// RequireTokenMiddleware on state-mutating endpoints. Generated at Start()
	// time, persisted to config.SessionTokenPath, and never logged in plaintext.
	sessionToken *SessionToken
}

// NewServer creates a new API server.
func NewServer(config Config, adapter beads.Adapter) *Server {
	s := &Server{
		config:    config,
		adapter:   adapter,
		gtAdapter: gastown.NewFSAdapter(config.TownRoot),
		mux:       http.NewServeMux(),
		sse:       NewSSEBroker(),
	}
	s.registerRoutes()
	return s
}

// registerRoutes sets up all API endpoints.
func (s *Server) registerRoutes() {
	// Health check
	s.mux.HandleFunc("GET /api/v1/health", s.handleHealth)

	// Beads - Issues
	s.mux.HandleFunc("GET /api/v1/issues", s.handleListIssues)
	s.mux.HandleFunc("GET /api/v1/issues/{id}", s.handleGetIssue)

	// Beads - Board
	s.mux.HandleFunc("GET /api/v1/board", s.handleBoard)

	// Beads - Graph
	s.mux.HandleFunc("GET /api/v1/graph", s.handleGraph)

	// SSE Events
	s.mux.HandleFunc("GET /api/v1/events", s.handleEvents)

	// Gas Town - Town
	s.mux.HandleFunc("GET /api/v1/town", s.handleTown)
	s.mux.HandleFunc("GET /api/v1/town/status", s.handleTownStatus)

	// Gas Town - Rigs
	s.mux.HandleFunc("GET /api/v1/town/rigs", s.handleRigs)
	s.mux.HandleFunc("GET /api/v1/town/rigs/{name}", s.handleRig)

	// Gas Town - Agents
	s.mux.HandleFunc("GET /api/v1/town/agents", s.handleAgents)

	// Gas Town - Convoys
	s.mux.HandleFunc("GET /api/v1/town/convoys", s.handleConvoys)
	s.mux.HandleFunc("GET /api/v1/town/convoys/{id}", s.handleConvoy)

	// Gas Town - Molecules
	s.mux.HandleFunc("GET /api/v1/town/molecules", s.handleMolecules)
	s.mux.HandleFunc("GET /api/v1/town/molecules/{id}", s.handleMolecule)

	// Gas Town - Mail
	s.mux.HandleFunc("GET /api/v1/town/mail/{address}", s.handleMail)

	// Static files — catch-all after API routes
	s.serveStaticFiles()
}

// Handler returns the HTTP handler with middleware applied. The order is
// outside-in: incoming requests hit OriginAllowlist first (cheap, fast reject
// of cross-origin browser attacks), then CORS for response headers, then the
// logging middleware, then the mux. RequireTokenMiddleware is intentionally
// NOT applied at this layer — it is wrapped per-route on state-mutating
// endpoints when they ship, so read-only endpoints (the entire surface this
// burst) remain accessible to native clients without a token.
func (s *Server) Handler() http.Handler {
	originAllowlist := OriginAllowlistMiddleware(s.config.CORSOrigins)
	return originAllowlist(s.corsMiddleware(s.loggingMiddleware(s.mux)))
}

// Start starts the HTTP server.
//
// Two pre-flight checks before binding:
//
//  1. Loopback bind enforcement: the daemon refuses to bind a non-loopback
//     host unless Config.DisableLoopbackCheck is explicitly set. This
//     prevents the most common deployment surprise — a default-bind on
//     0.0.0.0 exposing the dashboard to any network the dev box is on.
//  2. Session token generation: a fresh 256-bit token is generated and
//     persisted to Config.SessionTokenPath (default ~/.config/gvid/token,
//     mode 0600). The token's existence is reported via the daemon log but
//     never the value itself.
func (s *Server) Start() error {
	if !s.config.DisableLoopbackCheck && !IsLoopbackHost(s.config.Host) {
		return fmt.Errorf("refusing to bind non-loopback host %q; pass --host=localhost "+
			"(or set Config.DisableLoopbackCheck for ephemeral containers ONLY)",
			s.config.Host)
	}
	if s.config.DisableLoopbackCheck {
		log.Printf("WARNING: loopback bind check is DISABLED; binding %s on a "+
			"non-loopback address exposes the dashboard to the network",
			s.config.Host)
	}

	tokenPath := s.config.SessionTokenPath
	if tokenPath == "" {
		var err error
		tokenPath, err = DefaultSessionTokenPath()
		if err != nil {
			return fmt.Errorf("resolve session token path: %w", err)
		}
	}
	tok, err := GenerateSessionToken()
	if err != nil {
		return fmt.Errorf("generate session token: %w", err)
	}
	resolvedPath, err := tok.Persist(tokenPath)
	if err != nil {
		return fmt.Errorf("persist session token to %s: %w", tokenPath, err)
	}
	s.sessionToken = tok

	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
	log.Printf("Starting Gastown Viewer Intent daemon on %s", addr)
	log.Printf("API: http://%s/api/v1/", addr)
	log.Printf("Session token: %s (mode 0600; required by future state-changing endpoints)", resolvedPath)

	// Start SSE broker
	go s.sse.Start()

	server := &http.Server{
		Addr:         addr,
		Handler:      s.Handler(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return server.ListenAndServe()
}

// SessionToken returns the in-process token (or nil if Start() has not yet
// run). Test helpers use this; production code paths should reach for the
// token via RequireTokenMiddleware only.
func (s *Server) SessionToken() *SessionToken { return s.sessionToken }

// corsMiddleware adds CORS headers for development.
func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		// Check if origin is allowed
		allowed := false
		for _, o := range s.config.CORSOrigins {
			if o == origin || o == "*" {
				allowed = true
				break
			}
		}

		if allowed {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		}

		// Handle preflight
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// loggingMiddleware logs requests.
func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}

// checkBeadsInitialized verifies beads is ready, returns false and writes error if not.
func (s *Server) checkBeadsInitialized(w http.ResponseWriter, r *http.Request) bool {
	ctx := r.Context()
	initialized, err := s.adapter.IsInitialized(ctx)
	if err != nil {
		if beads.IsBDNotFoundError(err) {
			writeError(w, http.StatusServiceUnavailable, "BD_NOT_FOUND", err.Error())
			return false
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return false
	}
	if !initialized {
		writeError(w, http.StatusServiceUnavailable, "BEADS_NOT_INIT",
			"Beads not initialized. Run 'bd init' in your project directory.")
		return false
	}
	return true
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	s.sse.Stop()
	return nil
}
