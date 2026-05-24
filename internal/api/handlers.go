package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/intent-solutions-io/gastown-viewer-intent/internal/beads"
	"github.com/intent-solutions-io/gastown-viewer-intent/internal/model"
)

// HealthResponse is the response for GET /api/v1/health.
type HealthResponse struct {
	Status           string `json:"status"`
	BeadsInitialized bool   `json:"beads_initialized"`
	Version          string `json:"version"`
	BDVersion        string `json:"bd_version,omitempty"`
	Error            string `json:"error,omitempty"`
}

// handleHealth handles GET /api/v1/health.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	resp := HealthResponse{
		Version: s.config.Version,
	}

	// Check if beads is initialized
	initialized, err := s.adapter.IsInitialized(ctx)
	if err != nil {
		if beads.IsBDNotFoundError(err) {
			resp.Status = "error"
			resp.BeadsInitialized = false
			resp.Error = "bd CLI not found in PATH. Install from https://github.com/intent-solutions-io/beads"
			writeJSON(w, http.StatusServiceUnavailable, resp)
			return
		}
		resp.Status = "error"
		resp.Error = err.Error()
		writeJSON(w, http.StatusInternalServerError, resp)
		return
	}

	resp.BeadsInitialized = initialized

	if !initialized {
		resp.Status = "error"
		resp.Error = "Beads not initialized. Run 'bd init' in your project directory."
		writeJSON(w, http.StatusServiceUnavailable, resp)
		return
	}

	// Get bd version
	version, err := s.adapter.Version(ctx)
	if err == nil {
		resp.BDVersion = version
	}

	resp.Status = "ok"
	writeJSON(w, http.StatusOK, resp)
}

// handleListIssues handles GET /api/v1/issues.
func (s *Server) handleListIssues(w http.ResponseWriter, r *http.Request) {
	if !s.checkBeadsInitialized(w, r) {
		return
	}

	ctx := r.Context()
	query := r.URL.Query()

	filter := model.NewIssueFilter()
	filter.Status = query.Get("status")
	filter.Parent = query.Get("parent")
	filter.Search = query.Get("search")

	if limitStr := query.Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			filter.Limit = limit
		}
	}

	if offsetStr := query.Get("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil && offset >= 0 {
			filter.Offset = offset
		}
	}

	issues, err := s.adapter.ListIssues(ctx, filter)
	if err != nil {
		handleAdapterError(w, err)
		return
	}

	resp := model.IssueListResponse{
		Issues: issues,
		Total:  len(issues),
		Limit:  filter.Limit,
		Offset: filter.Offset,
	}

	writeJSON(w, http.StatusOK, resp)
}

// handleGetIssue handles GET /api/v1/issues/{id}.
func (s *Server) handleGetIssue(w http.ResponseWriter, r *http.Request) {
	if !s.checkBeadsInitialized(w, r) {
		return
	}

	ctx := r.Context()
	id := r.PathValue("id")

	if id == "" {
		writeError(w, http.StatusBadRequest, "INVALID_PARAM", "issue ID required")
		return
	}

	issue, err := s.adapter.GetIssue(ctx, id)
	if err != nil {
		handleAdapterError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, issue)
}

// handleBoard handles GET /api/v1/board.
func (s *Server) handleBoard(w http.ResponseWriter, r *http.Request) {
	if !s.checkBeadsInitialized(w, r) {
		return
	}

	ctx := r.Context()

	board, err := s.adapter.Board(ctx)
	if err != nil {
		handleAdapterError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, board)
}

// GraphResponse extends model.Graph with format-specific output.
type GraphResponse struct {
	model.Graph
	Format string `json:"-"`
}

// handleGraph handles GET /api/v1/graph.
func (s *Server) handleGraph(w http.ResponseWriter, r *http.Request) {
	if !s.checkBeadsInitialized(w, r) {
		return
	}

	ctx := r.Context()
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "json"
	}

	graph, err := s.adapter.Graph(ctx)
	if err != nil {
		handleAdapterError(w, err)
		return
	}

	switch format {
	case "dot":
		w.Header().Set("Content-Type", "text/vnd.graphviz; charset=utf-8")
		w.Header().Set("Content-Disposition", "inline; filename=\"dependencies.dot\"")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(graph.ToDOT()))
		return
	case "svg":
		// SVG format placeholder - would require graphviz binary
		writeError(w, http.StatusNotImplemented, "NOT_IMPLEMENTED",
			"SVG export requires graphviz. Use DOT format and convert with 'dot -Tsvg'")
		return
	}

	writeJSON(w, http.StatusOK, graph)
}

// ErrorResponse is the standard error response format.
type ErrorResponse struct {
	Error   string                 `json:"error"`
	Code    string                 `json:"code"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// writeJSON writes a JSON response.
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

// writeError writes an error response.
func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, ErrorResponse{
		Error: message,
		Code:  code,
	})
}

// handleAdapterError converts adapter errors to HTTP responses.
func handleAdapterError(w http.ResponseWriter, err error) {
	if beads.IsBDNotFoundError(err) {
		writeError(w, http.StatusServiceUnavailable, "BD_NOT_FOUND", err.Error())
		return
	}

	if beads.IsNotInitializedError(err) {
		writeError(w, http.StatusServiceUnavailable, "BEADS_NOT_INIT", err.Error())
		return
	}

	if beads.IsNotFoundError(err) {
		writeError(w, http.StatusNotFound, "ISSUE_NOT_FOUND", err.Error())
		return
	}

	if beads.IsParseError(err) {
		writeError(w, http.StatusInternalServerError, "PARSE_ERROR", err.Error())
		return
	}

	writeError(w, http.StatusInternalServerError, "BD_ERROR", err.Error())
}

// handleSync handles GET /api/v1/sync. Returns the composed dolt server +
// remote status used by the header sync pill (council Q0 Surface 2). The
// handler does NOT call checkBeadsInitialized because a not-initialized
// daemon should still respond with Health=unknown rather than 503'ing —
// the whole point of the pill is to surface the desync condition.
//
// DoltSyncState's contract is "never errors" (failures are encoded as
// Health=unknown in the body). We still defend against a Go error here
// because the contract is a documented invariant, not a type-system
// guarantee — a future bug that violates it should produce a 500 with a
// useful body rather than a panic or empty response.
func (s *Server) handleSync(w http.ResponseWriter, r *http.Request) {
	state, err := s.adapter.DoltSyncState(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "BD_ERROR",
			"DoltSyncState should not error per its contract; got: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, state)
}

// handleHumanFlags handles GET /api/v1/human. Returns the list of beads
// flagged for human decision (council Q0 Surface 3 — READ-VIEW only;
// respond/dismiss POST handlers are explicitly deferred to a future bead
// per the 2026-05-23 AT-DECR).
func (s *Server) handleHumanFlags(w http.ResponseWriter, r *http.Request) {
	if !s.checkBeadsInitialized(w, r) {
		return
	}
	flags, err := s.adapter.HumanFlags(r.Context())
	if err != nil {
		handleAdapterError(w, err)
		return
	}
	resp := model.HumanFlagsResponse{
		Flags: flags,
		Count: len(flags),
	}
	writeJSON(w, http.StatusOK, resp)
}
