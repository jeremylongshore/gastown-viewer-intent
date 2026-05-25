package model

// DoltHealth is the derived pill color the header sync widget renders.
// Computed from the raw DoltSyncState (running + remote-status) rather than
// pushed to the web client raw — the UI does not need to make this call.
type DoltHealth string

const (
	// DoltHealthGreen means the dolt server is up and every configured remote
	// reports status "ok". Safe to push or pull at will.
	DoltHealthGreen DoltHealth = "green"

	// DoltHealthYellow means the dolt server is up but at least one remote
	// reports something other than "ok" (auth failure, network error,
	// permission issue). Read traffic still works; pushes may fail.
	DoltHealthYellow DoltHealth = "yellow"

	// DoltHealthRed means the local dolt server is not running. Every bd
	// command that touches state will fail until the server is restarted —
	// silent desync risk this widget exists to surface.
	DoltHealthRed DoltHealth = "red"

	// DoltHealthUnknown is used when `bd dolt status --json` itself failed
	// (bd missing, beads uninitialized, command timed out). Distinct from
	// Red because the daemon literally does not know whether the server is
	// up; the UI should style this differently (gray pill, not red).
	DoltHealthUnknown DoltHealth = "unknown"
)

// DoltRemote is a single configured remote on the local dolt store.
// Fields map to the JSON shape `bd dolt remote list --json` emits per row:
// `{name, sql_url, cli_url, status}`. The viewer surfaces only what the
// header widget needs; full remote URLs are kept off the wire to avoid
// leaking the user's DoltHub workspace name into screen-recordings.
type DoltRemote struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

// DoltSyncState is the response body of GET /api/v1/sync. It composes the
// raw output of `bd dolt status --json` and `bd dolt remote list --json`
// into a single response with a derived Health pill the UI can render
// directly without inspecting the lower-level fields.
type DoltSyncState struct {
	// Health is the derived UI pill color. See DoltHealth constants for the
	// computation rule. UI components should bind to this field, not to
	// Running / Remotes / etc. directly.
	Health DoltHealth `json:"health"`

	// Running is true when the embedded dolt SQL server is up.
	Running bool `json:"running"`

	// Port is the local dolt server port. Useful for ad-hoc debugging; not
	// rendered in the UI. Zero when the server is not running.
	Port int `json:"port,omitempty"`

	// SchemaVersion is the dolt schema version reported by bd. Surfaced for
	// human-debugging only; the UI does not branch on this value.
	SchemaVersion int `json:"schema_version,omitempty"`

	// Remotes lists every configured remote and its current status string
	// from `bd dolt remote list --json`. The UI renders just the count and
	// the worst status.
	Remotes []DoltRemote `json:"remotes"`

	// Error is populated when the underlying `bd dolt status` command
	// failed. Health is DoltHealthUnknown when this is non-empty; the UI
	// shows this string in a tooltip.
	Error string `json:"error,omitempty"`
}

// HumanFlagsResponse is the response body of GET /api/v1/human. It lists
// every bead carrying the "human" label — beads that an AI agent or
// automation flagged for human decision. The list is read-only in this
// burst; respond/dismiss POSTs are deferred to a future bead behind the
// auth token gate from gastown-hu4 (see THREAT_MODEL.md).
type HumanFlagsResponse struct {
	// Flags is the list of human-needed issues in priority then age order.
	// Empty array (NOT nil) when nothing is flagged so the UI does not need
	// to defend against null.
	Flags []Issue `json:"flags"`

	// Count is len(Flags). Carried in the response so the UI can render a
	// badge without parsing the array.
	Count int `json:"count"`
}
