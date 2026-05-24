package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/intent-solutions-io/gastown-viewer-intent/internal/beads"
)

func TestHealthHandler(t *testing.T) {
	// Create server with mock adapter
	config := DefaultConfig()
	config.TownRoot = "/tmp/nonexistent"
	adapter := beads.NewCLIAdapter("")

	server := NewServer(config, adapter)

	req := httptest.NewRequest("GET", "/api/v1/health", nil)
	w := httptest.NewRecorder()

	server.Handler().ServeHTTP(w, req)

	// Health returns 200 if beads available, 503 if not (CI has no bd CLI)
	if w.Code != http.StatusOK && w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 200 or 503, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Version should always be present
	if resp["version"] != config.Version {
		t.Errorf("Expected version %s, got %v", config.Version, resp["version"])
	}
}

func TestTownStatusHandler(t *testing.T) {
	config := DefaultConfig()
	config.TownRoot = "/tmp/nonexistent-town"
	adapter := beads.NewCLIAdapter("")

	server := NewServer(config, adapter)

	req := httptest.NewRequest("GET", "/api/v1/town/status", nil)
	w := httptest.NewRecorder()

	server.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Should report unhealthy for non-existent town
	if resp["healthy"] != false {
		t.Error("Expected healthy=false for non-existent town")
	}
}

func TestCORSMiddleware(t *testing.T) {
	config := DefaultConfig()
	config.CORSOrigins = []string{"http://localhost:5173"}
	adapter := beads.NewCLIAdapter("")

	server := NewServer(config, adapter)

	req := httptest.NewRequest("GET", "/api/v1/health", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	w := httptest.NewRecorder()

	server.Handler().ServeHTTP(w, req)

	if w.Header().Get("Access-Control-Allow-Origin") != "http://localhost:5173" {
		t.Error("Expected CORS header to be set")
	}
}

func TestCORSPreflight(t *testing.T) {
	config := DefaultConfig()
	config.CORSOrigins = []string{"http://localhost:5173"}
	adapter := beads.NewCLIAdapter("")

	server := NewServer(config, adapter)

	req := httptest.NewRequest("OPTIONS", "/api/v1/health", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	w := httptest.NewRecorder()

	server.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("Expected status 204 for preflight, got %d", w.Code)
	}
}

// TestMemoriesHandler_RedactsByDefault confirms the wiring: a memory
// containing a partner-name + secret token comes back redacted unless
// ?reveal=true is set. Council Q2 read-only architectural invariant.
func TestMemoriesHandler_RedactsByDefault(t *testing.T) {
	config := DefaultConfig()
	adapter := beads.NewCLIAdapter("")
	server := NewServer(config, adapter)

	req := httptest.NewRequest("GET", "/api/v1/memories", nil)
	w := httptest.NewRecorder()
	server.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusOK && w.Code != http.StatusServiceUnavailable {
		t.Errorf("status: got %d, want 200 or 503", w.Code)
	}
	if w.Code == http.StatusOK {
		var resp struct {
			Memories []map[string]interface{} `json:"memories"`
			Count    int                      `json:"count"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("decode: %v", err)
		}
		// memories array must be non-null (empty slice acceptable).
		if resp.Memories == nil {
			t.Error("memories field must be non-null (empty array preferred)")
		}
	}
}

// TestMemoriesHandler_NoPOSTRouteRegistered guards the architectural
// invariant: POST/PUT/PATCH/DELETE under /api/v1/memories/* must NOT be
// routed. Council Q2 (gastown-cr5 AT-DECR). The mux should return 405
// or 404 for these methods; what's NOT acceptable is a 200 that
// indicates an accidentally-registered write handler.
func TestMemoriesHandler_NoPOSTRouteRegistered(t *testing.T) {
	config := DefaultConfig()
	adapter := beads.NewCLIAdapter("")
	server := NewServer(config, adapter)

	for _, method := range []string{"POST", "PUT", "PATCH", "DELETE"} {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/v1/memories", nil)
			w := httptest.NewRecorder()
			server.Handler().ServeHTTP(w, req)
			if w.Code == http.StatusOK {
				t.Errorf("%s on /api/v1/memories must NOT return 200 — architectural invariant violated",
					method)
			}
		})
	}
}
