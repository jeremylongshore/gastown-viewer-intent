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

	// Beads - Memories — READ-ONLY per AT-DECR Q2 architectural invariant.
	// Zero state-mutating endpoints under /api/v1/memories/* by design.
	// Redaction applied per 000-docs/005-PP-POLICY-memories-classification-2026-05-24.md.
	s.mux.HandleFunc("GET /api/v1/memories", s.handleMemories)
	s.mux.HandleFunc("GET /api/v1/memories/search", s.handleMemoriesSearch)
	s.mux.HandleFunc("GET /api/v1/memories/{key}", s.handleMemory)

	// Static files — catch-all after API routes
	s.serveStaticFiles()
}

// Handler returns the HTTP handler with middleware applied.
func (s *Server) Handler() http.Handler {
	return s.corsMiddleware(s.loggingMiddleware(s.mux))
}

// Start starts the HTTP server.
func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
	log.Printf("Starting Gastown Viewer Intent daemon on %s", addr)
	log.Printf("API: http://%s/api/v1/", addr)

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
