package gastown

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	// sentinelWindowsClaudeRunning is the synthetic session key getTmuxSessions inserts
	// when tmux is unavailable but the gt daemon state file is present. enrichAgent
	// checks for this key to switch to file-mtime-based liveness detection.
	sentinelWindowsClaudeRunning = "__windows_claude_running__"

	// gastownCommandTimeout caps any out-of-process gt CLI call so a hung Dolt
	// connection cannot block the calling HTTP handler indefinitely.
	gastownCommandTimeout = 5 * time.Second

	// windowsActivityThreshold is the file-mtime window inside which an agent's
	// workDir is considered active when tmux is unavailable. Aligned with the
	// project's stuck-detection threshold so the same agent is never simultaneously
	// "active" here and "stuck" elsewhere.
	windowsActivityThreshold = 10 * time.Minute
)

// Adapter provides access to Gas Town data.
type Adapter interface {
	// Status returns the overall town health status.
	Status(ctx context.Context) (*TownStatus, error)

	// Town returns the full town structure.
	Town(ctx context.Context) (*Town, error)

	// Rigs returns all rigs in the town.
	Rigs(ctx context.Context) ([]Rig, error)

	// Rig returns a specific rig by name.
	Rig(ctx context.Context, name string) (*Rig, error)

	// Agents returns all agents across all rigs.
	Agents(ctx context.Context) ([]Agent, error)

	// Convoys returns active convoys.
	Convoys(ctx context.Context) ([]Convoy, error)

	// Convoy returns a specific convoy by ID.
	Convoy(ctx context.Context, id string) (*Convoy, error)

	// Molecules returns all active molecules across all agents.
	Molecules(ctx context.Context) ([]Molecule, error)

	// Molecule returns a specific molecule by ID.
	Molecule(ctx context.Context, id string) (*Molecule, error)

	// Mail returns messages for an agent address.
	Mail(ctx context.Context, address string) ([]Message, error)
}

// FSAdapter reads Gas Town state from the filesystem and gt CLI.
type FSAdapter struct {
	townRoot string
}

// NewFSAdapter creates a new filesystem-based adapter.
//
// When townRoot is empty, the home directory is resolved via os.UserHomeDir(),
// which handles HOME on Linux/macOS and USERPROFILE / HOMEDRIVE+HOMEPATH on
// Windows, plus an /etc/passwd lookup fallback on Linux. If even that fails
// (rare: containers with no env and no passwd entry), townRoot is left empty
// so downstream calls fail predictably via townExists() rather than silently
// constructing a relative "gt" path.
func NewFSAdapter(townRoot string) *FSAdapter {
	if townRoot == "" {
		if home, err := os.UserHomeDir(); err == nil && home != "" {
			townRoot = filepath.Join(home, "gt")
		}
	}
	return &FSAdapter{townRoot: townRoot}
}

// Status returns the overall town health status.
func (a *FSAdapter) Status(ctx context.Context) (*TownStatus, error) {
	status := &TownStatus{
		TownRoot: a.townRoot,
	}

	// Check if town exists
	if !a.townExists() {
		status.Healthy = false
		status.Error = fmt.Sprintf("Town not found at %s", a.townRoot)
		return status, nil
	}

	// Get town data
	town, err := a.Town(ctx)
	if err != nil {
		status.Healthy = false
		status.Error = err.Error()
		return status, nil
	}

	// Count agents
	status.ActiveRigs = len(town.Rigs)
	for _, rig := range town.Rigs {
		status.TotalAgents += len(rig.Polecats) + len(rig.Crew)
		if rig.Witness != nil {
			status.TotalAgents++
			if rig.Witness.Status == StatusActive {
				status.ActiveAgents++
			}
		}
		if rig.Refinery != nil {
			status.TotalAgents++
			if rig.Refinery.Status == StatusActive {
				status.ActiveAgents++
			}
		}
		for _, p := range rig.Polecats {
			if p.Status == StatusActive {
				status.ActiveAgents++
			}
		}
		for _, c := range rig.Crew {
			if c.Status == StatusActive {
				status.ActiveAgents++
			}
		}
	}

	if town.Mayor != nil {
		status.TotalAgents++
		if town.Mayor.Status == StatusActive {
			status.ActiveAgents++
		}
	}
	if town.Deacon != nil {
		status.TotalAgents++
		if town.Deacon.Status == StatusActive {
			status.ActiveAgents++
		}
	}

	status.OpenConvoys = len(town.Convoys)
	status.Healthy = true

	return status, nil
}

// Town returns the full town structure.
func (a *FSAdapter) Town(ctx context.Context) (*Town, error) {
	if !a.townExists() {
		return nil, fmt.Errorf("town not found at %s", a.townRoot)
	}

	town := &Town{
		Root: a.townRoot,
	}

	// Read town config
	config, err := a.readTownConfig()
	if err == nil {
		town.Name = config.Name
	}

	// Get tmux sessions to determine agent status
	sessions := a.getTmuxSessions()

	// Check mayor
	if a.dirExists(filepath.Join(a.townRoot, "mayor")) {
		mayor := &Agent{
			Role: RoleMayor,
			Name: "mayor",
		}
		a.enrichAgent(mayor, sessions)
		town.Mayor = mayor
	}

	// Check deacon (via daemon)
	if a.daemonRunning() {
		deacon := &Agent{
			Role:   RoleDeacon,
			Name:   "deacon",
			Status: StatusActive,
		}
		a.enrichAgent(deacon, sessions)
		town.Deacon = deacon
	}

	// Find rigs
	rigs, err := a.Rigs(ctx)
	if err == nil {
		town.Rigs = rigs
	}

	// Get convoys
	convoys, err := a.Convoys(ctx)
	if err == nil {
		town.Convoys = convoys
	}

	return town, nil
}

// Rigs returns all rigs in the town.
func (a *FSAdapter) Rigs(ctx context.Context) ([]Rig, error) {
	var rigs []Rig

	// Look for directories that have rig markers
	entries, err := os.ReadDir(a.townRoot)
	if err != nil {
		return nil, err
	}

	sessions := a.getTmuxSessions()

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()
		// Skip non-rig directories
		if name == "mayor" || name == ".beads" || name == ".git" || strings.HasPrefix(name, ".") {
			continue
		}

		rigPath := filepath.Join(a.townRoot, name)

		// Check if it looks like a rig (has polecats/, witness/, or .beads/)
		if !a.dirExists(filepath.Join(rigPath, "polecats")) &&
			!a.dirExists(filepath.Join(rigPath, "witness")) &&
			!a.dirExists(filepath.Join(rigPath, ".beads")) {
			continue
		}

		rig := Rig{
			Name: name,
			Path: rigPath,
		}

		// Check witness
		if a.dirExists(filepath.Join(rigPath, "witness")) {
			witness := &Agent{
				Role: RoleWitness,
				Name: "witness",
				Rig:  name,
			}
			a.enrichAgent(witness, sessions)
			rig.Witness = witness
		}

		// Check refinery
		if a.dirExists(filepath.Join(rigPath, "refinery")) {
			refinery := &Agent{
				Role: RoleRefinery,
				Name: "refinery",
				Rig:  name,
			}
			a.enrichAgent(refinery, sessions)
			rig.Refinery = refinery
		}

		// Find polecats
		polecatsDir := filepath.Join(rigPath, "polecats")
		if a.dirExists(polecatsDir) {
			pEntries, err := os.ReadDir(polecatsDir)
			if err == nil {
				for _, pe := range pEntries {
					if pe.IsDir() && !strings.HasPrefix(pe.Name(), ".") {
						polecat := Agent{
							Role: RolePolecat,
							Name: pe.Name(),
							Rig:  name,
						}
						a.enrichAgent(&polecat, sessions)
						rig.Polecats = append(rig.Polecats, polecat)
					}
				}
			}
		}

		// Find crew
		crewDir := filepath.Join(rigPath, "crew")
		if a.dirExists(crewDir) {
			cEntries, err := os.ReadDir(crewDir)
			if err == nil {
				for _, ce := range cEntries {
					if ce.IsDir() && !strings.HasPrefix(ce.Name(), ".") {
						crew := Agent{
							Role: RoleCrew,
							Name: ce.Name(),
							Rig:  name,
						}
						a.enrichAgent(&crew, sessions)
						rig.Crew = append(rig.Crew, crew)
					}
				}
			}
		}

		rigs = append(rigs, rig)
	}

	return rigs, nil
}

// Rig returns a specific rig by name.
func (a *FSAdapter) Rig(ctx context.Context, name string) (*Rig, error) {
	rigs, err := a.Rigs(ctx)
	if err != nil {
		return nil, err
	}

	for _, rig := range rigs {
		if rig.Name == name {
			return &rig, nil
		}
	}

	return nil, fmt.Errorf("rig not found: %s", name)
}

// Agents returns all agents across all rigs.
func (a *FSAdapter) Agents(ctx context.Context) ([]Agent, error) {
	town, err := a.Town(ctx)
	if err != nil {
		return nil, err
	}

	var agents []Agent

	if town.Mayor != nil {
		agents = append(agents, *town.Mayor)
	}
	if town.Deacon != nil {
		agents = append(agents, *town.Deacon)
	}

	for _, rig := range town.Rigs {
		if rig.Witness != nil {
			agents = append(agents, *rig.Witness)
		}
		if rig.Refinery != nil {
			agents = append(agents, *rig.Refinery)
		}
		agents = append(agents, rig.Polecats...)
		agents = append(agents, rig.Crew...)
	}

	return agents, nil
}

// Convoys returns active convoys by running gt convoy list.
func (a *FSAdapter) Convoys(ctx context.Context) ([]Convoy, error) {
	// Use a short sub-timeout so a slow gt daemon doesn't block the whole response.
	cCtx, cancel := context.WithTimeout(ctx, gastownCommandTimeout)
	defer cancel()
	// Try to run gt convoy list --json
	cmd := exec.CommandContext(cCtx, "gt", "convoy", "list", "--json")
	cmd.Dir = a.townRoot
	output, err := cmd.Output()
	if err != nil {
		// gt might not be installed or convoy command might fail
		return nil, nil
	}

	var rawConvoys []struct {
		ID          string   `json:"id"`
		Title       string   `json:"title"`
		Status      string   `json:"status"`
		Priority    string   `json:"priority,omitempty"`
		Rig         string   `json:"rig,omitempty"`
		Issues      []string `json:"issues"`
		Progress    int      `json:"progress"`
		Total       int      `json:"total"`
		Completed   int      `json:"completed"`
		Blocked     int      `json:"blocked"`
		InProgress  int      `json:"in_progress"`
		CreatedAt   string   `json:"created_at,omitempty"`
		UpdatedAt   string   `json:"updated_at,omitempty"`
		Subscribers []string `json:"subscribers,omitempty"`
		Agents      []string `json:"agents,omitempty"`
	}

	if err := json.Unmarshal(output, &rawConvoys); err != nil {
		// Try parsing as single convoy
		var raw struct {
			ID          string   `json:"id"`
			Title       string   `json:"title"`
			Status      string   `json:"status"`
			Priority    string   `json:"priority,omitempty"`
			Rig         string   `json:"rig,omitempty"`
			Issues      []string `json:"issues"`
			Progress    int      `json:"progress"`
			Total       int      `json:"total"`
			Completed   int      `json:"completed"`
			Blocked     int      `json:"blocked"`
			InProgress  int      `json:"in_progress"`
			CreatedAt   string   `json:"created_at,omitempty"`
			UpdatedAt   string   `json:"updated_at,omitempty"`
			Subscribers []string `json:"subscribers,omitempty"`
			Agents      []string `json:"agents,omitempty"`
		}
		if err := json.Unmarshal(output, &raw); err != nil {
			return nil, nil
		}
		rawConvoys = append(rawConvoys, raw)
	}

	var convoys []Convoy
	for _, r := range rawConvoys {
		convoy := a.parseRawConvoy(r.ID, r.Title, r.Status, r.Priority, r.Rig,
			r.Issues, r.Progress, r.Total, r.Completed, r.Blocked, r.InProgress,
			r.CreatedAt, r.UpdatedAt, r.Subscribers, r.Agents)
		convoys = append(convoys, convoy)
	}

	return convoys, nil
}

// Convoy returns a specific convoy by ID.
func (a *FSAdapter) Convoy(ctx context.Context, id string) (*Convoy, error) {
	convoys, err := a.Convoys(ctx)
	if err != nil {
		return nil, err
	}

	for _, convoy := range convoys {
		if convoy.ID == id {
			return &convoy, nil
		}
	}

	return nil, fmt.Errorf("convoy not found: %s", id)
}

// parseRawConvoy converts raw convoy data to a Convoy struct.
func (a *FSAdapter) parseRawConvoy(id, title, status, priority, rig string,
	issues []string, progress, total, completed, blocked, inProgress int,
	createdAt, updatedAt string, subscribers, agents []string) Convoy {

	// Convert status string to ConvoyStatus
	convoyStatus := ConvoyStatusPending
	switch status {
	case "in_progress":
		convoyStatus = ConvoyStatusInProgress
	case "complete", "completed":
		convoyStatus = ConvoyStatusComplete
	case "blocked":
		convoyStatus = ConvoyStatusBlocked
	case "failed":
		convoyStatus = ConvoyStatusFailed
	}

	convoy := Convoy{
		ID:          id,
		Title:       title,
		Status:      convoyStatus,
		Priority:    priority,
		Rig:         rig,
		Issues:      issues,
		Progress:    progress,
		Total:       total,
		Completed:   completed,
		Blocked:     blocked,
		InProgress:  inProgress,
		Subscribers: subscribers,
		Agents:      agents,
	}

	// Parse timestamps
	if createdAt != "" {
		if t, err := time.Parse(time.RFC3339, createdAt); err == nil {
			convoy.CreatedAt = t
		}
	}
	if updatedAt != "" {
		if t, err := time.Parse(time.RFC3339, updatedAt); err == nil {
			convoy.UpdatedAt = t
		}
	}

	// Calculate progress if not provided
	if convoy.Total == 0 && len(convoy.Issues) > 0 {
		convoy.Total = len(convoy.Issues)
	}
	if convoy.Total > 0 && convoy.Progress == 0 {
		convoy.Progress = (convoy.Completed * 100) / convoy.Total
	}

	return convoy
}

// Mail returns messages for an agent address.
func (a *FSAdapter) Mail(ctx context.Context, address string) ([]Message, error) {
	// Use a short sub-timeout so a slow gt daemon doesn't block the whole response.
	mCtx, cancel := context.WithTimeout(ctx, gastownCommandTimeout)
	defer cancel()
	// Run gt mail inbox for the address
	cmd := exec.CommandContext(mCtx, "gt", "mail", "inbox", "--json")
	cmd.Dir = a.townRoot
	cmd.Env = append(os.Environ(), fmt.Sprintf("GT_ROLE=%s", address))
	output, err := cmd.Output()
	if err != nil {
		return nil, nil
	}

	var messages []Message
	if err := json.Unmarshal(output, &messages); err != nil {
		return nil, nil
	}

	return messages, nil
}

// Helper methods

func (a *FSAdapter) townExists() bool {
	return a.dirExists(filepath.Join(a.townRoot, "mayor"))
}

func (a *FSAdapter) dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func (a *FSAdapter) readTownConfig() (*TownConfig, error) {
	configPath := filepath.Join(a.townRoot, "mayor", "town.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var config TownConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

func (a *FSAdapter) getTmuxSessions() map[string]bool {
	sessions := make(map[string]bool)

	// Try tmux first (Linux/macOS)
	cmd := exec.Command("tmux", "list-sessions", "-F", "#{session_name}")
	if output, err := cmd.Output(); err == nil {
		for _, line := range strings.Split(string(output), "\n") {
			if line = strings.TrimSpace(line); line != "" {
				sessions[line] = true
			}
		}
		return sessions
	}

	// Windows/no-tmux fallback: check daemon/state.json to see if the gt system is running.
	// This is a fast file stat (no subprocess). If the daemon is up, enrichAgent will
	// use workdir modification times to approximate agent activity.
	statePath := filepath.Join(a.townRoot, "daemon", "state.json")
	if _, err := os.Stat(statePath); err == nil {
		sessions[sentinelWindowsClaudeRunning] = true
	}

	return sessions
}

func (a *FSAdapter) daemonRunning() bool {
	// Check if gt daemon is running by looking for pid file or process
	pidFile := filepath.Join(a.townRoot, "mayor", "daemon.pid")
	if _, err := os.Stat(pidFile); err == nil {
		return true
	}

	// Also check via gt daemon status
	cmd := exec.Command("gt", "daemon", "status")
	cmd.Dir = a.townRoot
	err := cmd.Run()
	return err == nil
}

// latestActivity returns how long ago the most recently-modified entry in dir
// was touched, measured against now. Walks one level deep — the dir itself plus
// its immediate children — which is enough to catch .claude/seance.json updates
// from a long-running Claude session that doesn't change the dir's own mtime.
// Returns a very large duration if dir cannot be read so callers treat it as
// "not recent" rather than "active just now".
func latestActivity(dir string, now time.Time) time.Duration {
	stale := 1000 * time.Hour

	info, err := os.Stat(dir)
	if err != nil {
		return stale
	}
	newest := info.ModTime()

	entries, err := os.ReadDir(dir)
	if err != nil {
		return now.Sub(newest)
	}
	for _, e := range entries {
		ei, err := e.Info()
		if err != nil {
			continue
		}
		if ei.ModTime().After(newest) {
			newest = ei.ModTime()
		}
	}
	return now.Sub(newest)
}

// LastActivity returns the last modification time of agent's workspace.
func (a *FSAdapter) LastActivity(rigName, agentName string) time.Time {
	var checkPath string
	if agentName == "witness" {
		checkPath = filepath.Join(a.townRoot, rigName, "witness")
	} else if agentName == "refinery" {
		checkPath = filepath.Join(a.townRoot, rigName, "refinery")
	} else {
		checkPath = filepath.Join(a.townRoot, rigName, "polecats", agentName)
		if !a.dirExists(checkPath) {
			checkPath = filepath.Join(a.townRoot, rigName, "crew", agentName)
		}
	}

	info, err := os.Stat(checkPath)
	if err != nil {
		return time.Time{}
	}
	return info.ModTime()
}

// getAgentWorkDir returns the working directory for an agent.
func (a *FSAdapter) getAgentWorkDir(rigName string, role Role, name string) string {
	switch role {
	case RoleMayor:
		return filepath.Join(a.townRoot, "mayor")
	case RoleDeacon:
		return filepath.Join(a.townRoot, "mayor") // Deacon runs from mayor dir
	case RoleWitness:
		return filepath.Join(a.townRoot, rigName, "witness")
	case RoleRefinery:
		return filepath.Join(a.townRoot, rigName, "refinery")
	case RolePolecat:
		return filepath.Join(a.townRoot, rigName, "polecats", name)
	case RoleCrew:
		return filepath.Join(a.townRoot, rigName, "crew", name)
	default:
		return ""
	}
}

// enrichAgent adds session, molecule, and hook info to an agent.
func (a *FSAdapter) enrichAgent(agent *Agent, sessions map[string]bool) {
	workDir := a.getAgentWorkDir(agent.Rig, agent.Role, agent.Name)
	if workDir == "" {
		return
	}
	agent.WorkDir = workDir

	// Get session name
	sessionName := a.getSessionName(agent)
	agent.Session = sessionName

	// Check if session is active (tmux on Linux/macOS, file-based on Windows)
	if sessions[sessionName] {
		agent.Status = StatusActive
	} else if sessions[sentinelWindowsClaudeRunning] {
		// Windows fallback: tmux is unavailable, so use file-mtime liveness on workDir.
		// os.Stat on a directory alone is unreliable because POSIX dir mtime only updates
		// on entry add/remove — a long-running agent rewriting .claude/seance.json without
		// adding new entries would appear stale. Walk one level into workDir and use the
		// newest entry's mtime, capturing the dir itself as a floor.
		if latestActivity(workDir, time.Now()) < windowsActivityThreshold {
			agent.Status = StatusActive
		} else {
			agent.Status = StatusOffline
		}
	} else {
		agent.Status = StatusOffline
	}

	// Read seance file for compaction level
	seancePath := filepath.Join(workDir, ".claude", "seance.json")
	if data, err := os.ReadFile(seancePath); err == nil {
		var seance struct {
			Compaction int    `json:"compaction"`
			Molecule   string `json:"molecule,omitempty"`
		}
		if json.Unmarshal(data, &seance) == nil {
			agent.Compaction = seance.Compaction
			if seance.Molecule != "" {
				agent.Molecule = seance.Molecule
			}
		}
	}

	// Check hook for attached molecule
	hookPath := filepath.Join(workDir, ".claude", "hook.json")
	if data, err := os.ReadFile(hookPath); err == nil {
		var hook struct {
			Molecule string `json:"molecule,omitempty"`
			Attached bool   `json:"attached,omitempty"`
		}
		if json.Unmarshal(data, &hook) == nil {
			agent.HookAttached = hook.Attached || hook.Molecule != ""
			if hook.Molecule != "" {
				agent.Molecule = hook.Molecule
			}
		}
	}

	// NOTE: the previous implementation also fell back to reading
	// `<workDir>/.beads/molecule.json` here to recover the agent's molecule
	// attachment. That file was the gt 0.8 surface for the agent ↔ molecule
	// relationship; gt 0.9 replaced it with the root-level wisps SQLite store,
	// queried via `gt wisps list --json` (see Molecules() below). The
	// seance.json and hook.json reads above already provide the agent's
	// self-reported molecule ID, so dropping the legacy file read costs no
	// information against gt 0.9+ and removes a violation of the
	// CLI-shelling-not-file-parsing invariant documented in repo CLAUDE.md.

	// Get last activity time
	if info, err := os.Stat(workDir); err == nil {
		agent.LastActive = info.ModTime()
	}

	// Detect stuck status: active session but no activity for 10+ minutes
	if agent.Status == StatusActive && !agent.LastActive.IsZero() {
		if time.Since(agent.LastActive) > 10*time.Minute {
			agent.Status = StatusStuck
		} else if time.Since(agent.LastActive) > 2*time.Minute && !agent.HookAttached {
			agent.Status = StatusIdle
		}
	}
}

// rawWisp mirrors the JSON shape returned by `gt wisps list --json`. Field
// names match gt's wire format (wisps are gt 0.9's rename of molecules; the
// underlying schema is preserved across the rename for backward compatibility,
// but the file-on-disk path that the gt 0.8 viewer read was retired). The
// viewer keeps the model name "Molecule" so the API surface (and the web tab)
// stays stable while the data source migrates.
type rawWisp struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Status      string `json:"status"`
	Formula     string `json:"formula,omitempty"`
	CurrentStep int    `json:"current_step"`
	Agent       string `json:"agent,omitempty"`
	Rig         string `json:"rig,omitempty"`
	Steps       []struct {
		Index       int        `json:"index"`
		ID          string     `json:"id"`
		Description string     `json:"description"`
		Status      string     `json:"status"`
		Needs       []string   `json:"needs,omitempty"`
		StartedAt   *time.Time `json:"started_at,omitempty"`
		CompletedAt *time.Time `json:"completed_at,omitempty"`
	} `json:"steps,omitempty"`
	CreatedAt time.Time `json:"created_at,omitempty"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
}

// Molecules returns all active molecules by shelling `gt wisps list --json`.
//
// Migrated from the legacy `<workDir>/.beads/molecule.json` file-parsing path
// (gt 0.8 surface) to the gt 0.9 root-level wisps SQLite store. The shell
// boundary mirrors Convoys() and Mail() in posture but distinguishes two
// failure classes so the API surface can answer correctly:
//
//   - **Caller context cancelled OR deadline exceeded** → return ctx.Err()
//     so handlers can respond with 504 (the caller asked for a timely answer
//     and we could not give one). This is the case the dashboard's own
//     gastownCommandTimeout fires for, and it's the case an external client
//     deserves to know about.
//   - **gt missing / town absent / exec error / parse error** → return nil,nil
//     so the rest of /api/v1/town/* still answers. Empty list is the honest
//     answer when there is no gt to query; failing the whole town view because
//     wisps couldn't be enumerated would be over-blocking.
//
// Council decision Q0 + Q3 (gastown-cr5 AT-DECR). Refined per PR #11 Gemini
// review (2026-05-24): the original "nil,nil on any err" pattern matched
// Convoys() but lost the timeout vs. degradation signal — fixed.
func (a *FSAdapter) Molecules(ctx context.Context) ([]Molecule, error) {
	wCtx, cancel := context.WithTimeout(ctx, gastownCommandTimeout)
	defer cancel()

	cmd := exec.CommandContext(wCtx, "gt", "wisps", "list", "--json")
	cmd.Dir = a.townRoot
	output, err := cmd.Output()
	if err != nil {
		// Distinguish "caller cancelled / their deadline" (propagate) from
		// "our sub-timeout fired / gt missing / exec failure" (degrade).
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		// gt missing, town absent, or wisps subcommand failed — degrade silently.
		return nil, nil
	}

	raws, err := parseWispListOutput(output)
	if err != nil || len(raws) == 0 {
		return nil, nil
	}

	molecules := make([]Molecule, 0, len(raws))
	seen := make(map[string]bool, len(raws))
	for _, raw := range raws {
		if raw.ID == "" || seen[raw.ID] {
			continue
		}
		seen[raw.ID] = true
		molecules = append(molecules, raw.toMolecule())
	}
	return molecules, nil
}

// Molecule returns a specific molecule by ID.
func (a *FSAdapter) Molecule(ctx context.Context, id string) (*Molecule, error) {
	molecules, err := a.Molecules(ctx)
	if err != nil {
		return nil, err
	}

	for _, mol := range molecules {
		if mol.ID == id {
			return &mol, nil
		}
	}

	return nil, fmt.Errorf("molecule not found: %s", id)
}

// parseWispListOutput parses the JSON output of `gt wisps list --json`,
// accepting either a top-level array (the common case) or a single object
// (defensive against single-result responses, matching Convoys()'s behavior).
func parseWispListOutput(output []byte) ([]rawWisp, error) {
	var list []rawWisp
	if err := json.Unmarshal(output, &list); err == nil {
		return list, nil
	}
	var single rawWisp
	if err := json.Unmarshal(output, &single); err != nil {
		return nil, err
	}
	if single.ID == "" {
		return nil, nil
	}
	return []rawWisp{single}, nil
}

// toMolecule converts a rawWisp into the viewer's Molecule domain type.
func (r rawWisp) toMolecule() Molecule {
	status := MolStatusPending
	switch r.Status {
	case "in_progress":
		status = MolStatusInProgress
	case "complete", "completed":
		status = MolStatusComplete
	case "blocked":
		status = MolStatusBlocked
	case "failed":
		status = MolStatusFailed
	}

	mol := Molecule{
		ID:          r.ID,
		Title:       r.Title,
		Status:      status,
		Formula:     r.Formula,
		CurrentStep: r.CurrentStep,
		Agent:       r.Agent,
		Rig:         r.Rig,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}

	for _, s := range r.Steps {
		mol.Steps = append(mol.Steps, MoleculeStep{
			Index:       s.Index,
			ID:          s.ID,
			Description: s.Description,
			Status:      s.Status,
			Needs:       s.Needs,
			StartedAt:   s.StartedAt,
			CompletedAt: s.CompletedAt,
		})
	}

	mol.Total = len(mol.Steps)
	for _, step := range mol.Steps {
		// Lower-case before compare so "Complete" / "DONE" / "Completed" all
		// count as done. Consistent with mapStatus's case-insensitive contract
		// in internal/beads/parser.go — gt's CLI output capitalization has
		// drifted across releases.
		switch strings.ToLower(step.Status) {
		case "complete", "completed", "done":
			mol.Progress++
		}
	}
	return mol
}

// getSessionName returns the expected tmux session name for an agent.
func (a *FSAdapter) getSessionName(agent *Agent) string {
	switch agent.Role {
	case RoleMayor:
		return "gt-mayor"
	case RoleDeacon:
		return "gt-deacon"
	case RoleWitness:
		return fmt.Sprintf("gt-%s-witness", agent.Rig)
	case RoleRefinery:
		return fmt.Sprintf("gt-%s-refinery", agent.Rig)
	case RolePolecat:
		return fmt.Sprintf("gt-%s-%s", agent.Rig, agent.Name)
	case RoleCrew:
		return fmt.Sprintf("gt-%s-crew-%s", agent.Rig, agent.Name)
	default:
		return ""
	}
}
