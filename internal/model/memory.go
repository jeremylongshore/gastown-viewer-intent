package model

// Memory is a single entry from the bd persistent memory layer. Each entry
// is identified by a stable kebab-case key (either explicit via
// `bd remember --key ...` or auto-derived from content) and carries a
// free-text content string.
//
// The viewer treats memories as a regulated surface. See
// 000-docs/005-PP-POLICY-memories-classification-2026-05-24.md for the
// classification rules; the Redacted, RedactionMarkers, and RevealAvailable
// fields are populated by internal/api/memoryredact.go before any memory
// crosses the HTTP boundary.
type Memory struct {
	// Key is the canonical identifier — what `bd recall <key>` resolves.
	Key string `json:"key"`

	// Content is the memory body as stored in bd. When the response was
	// served with reveal=false (the default), partner-name (Class A) and
	// secret-pattern (Class B) matches in Content are replaced with
	// `[REDACTED ...]` placeholders by the redaction layer before the
	// response leaves the daemon. The raw bytes are NEVER included in the
	// rendered HTML when redaction is in effect.
	Content string `json:"content"`

	// Redacted is true when the redaction layer modified Content. The UI
	// uses this to decide whether to render the reveal button.
	Redacted bool `json:"redacted,omitempty"`

	// RedactionMarkers reports the classes of redaction applied. The UI
	// uses this for the reveal-button tooltip ("contains partner names" vs.
	// "contains a possible API token") so the engineer knows what they're
	// revealing before they click.
	RedactionMarkers []string `json:"redaction_markers,omitempty"`
}

// MemoriesResponse is the response body of GET /api/v1/memories. Memories
// is a sorted (by Key) slice so the UI gets stable ordering across polls.
// SchemaVersion is mirrored from bd so the viewer can refuse to render a
// version it does not understand (defense against future bd memory-schema
// changes that might add new sensitive fields).
type MemoriesResponse struct {
	Memories      []Memory `json:"memories"`
	Count         int      `json:"count"`
	SchemaVersion int      `json:"schema_version"`
}
