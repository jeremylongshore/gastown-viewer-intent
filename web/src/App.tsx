import { useEffect, useState } from 'react';
import type { BoardResponse, Issue, Column, IssueSummary, Town, TownStatus, Agent, Rig, Molecule, Convoy, DoltSyncState, DoltHealth, HumanFlagsResponse } from './api';
import { fetchBoard, fetchIssue, fetchTown, fetchTownStatus, fetchMolecules, fetchConvoys, fetchSync, fetchHumanFlags } from './api';
import DependencyGraph from './components/DependencyGraph';
import './App.css';

type ViewMode = 'beads' | 'graph' | 'gastown' | 'triage';

// SyncPill renders the dolt sync state in the header (council Q0 Surface 2).
// Color binds to the server-derived health field; the UI does not compute
// the color itself so that the server stays the source of truth and the
// rule does not drift between front-end and back-end.
function SyncPill({ state }: { state: DoltSyncState | null }) {
  // Loading / not-yet-fetched: render a neutral gray pill so the header
  // doesn't visually claim something it doesn't know yet.
  if (!state) {
    return (
      <span
        className="sync-pill"
        title="Sync state loading…"
        style={pillStyle('unknown')}
      >
        Sync ○
      </span>
    );
  }
  const labels: Record<DoltHealth, string> = {
    green: 'Synced',
    yellow: 'Push pending',
    red: 'Server down',
    unknown: 'Sync ?',
  };
  const remoteSummary = state.remotes.length === 0
    ? 'no remotes'
    : `${state.remotes.length} remote${state.remotes.length === 1 ? '' : 's'}`;
  const tooltip = state.error
    ? `Sync state unknown — ${state.error}`
    : `${labels[state.health]} (${remoteSummary})`;
  return (
    <span
      className="sync-pill"
      title={tooltip}
      style={pillStyle(state.health)}
    >
      ● {labels[state.health]}
    </span>
  );
}

// pillStyle maps a DoltHealth to the inline color tuple the SyncPill renders.
// Kept inline (vs. .css) because the same colors are read by the JSX title
// computation just above — co-locating reduces drift.
function pillStyle(health: DoltHealth): React.CSSProperties {
  const palette: Record<DoltHealth, { bg: string; fg: string }> = {
    green:   { bg: '#10b981', fg: '#ffffff' },
    yellow:  { bg: '#f59e0b', fg: '#1f2937' },
    red:     { bg: '#ef4444', fg: '#ffffff' },
    unknown: { bg: '#6b7280', fg: '#ffffff' },
  };
  const p = palette[health];
  return {
    backgroundColor: p.bg,
    color: p.fg,
    padding: '4px 10px',
    borderRadius: 999,
    fontSize: 12,
    fontWeight: 600,
    marginLeft: 12,
  };
}

// TriageBanner is the persistent screen-share warning required by
// 005-PP-POLICY-memories-classification §4 rendering rules. The Triage
// panel does not itself surface memory content (only issue titles flagged
// by automation), but it lives in the same view-mode arc and the banner
// reinforces the discipline before any future panel that does. Banner
// stays for the lifetime of the view — engineer navigates away to
// dismiss.
function ScreenShareBanner() {
  return (
    <div
      role="status"
      style={{
        backgroundColor: '#fef3c7',
        color: '#78350f',
        padding: '8px 16px',
        borderBottom: '1px solid #f59e0b',
        fontSize: 13,
        fontWeight: 600,
      }}
    >
      Triage queue may surface confidential issue titles — close before
      screen-sharing or use the redaction toggle (when memories panel
      ships in gastown-fp0).
    </div>
  );
}

// TriagePanel renders the read-only human-needed bead queue. Respond and
// dismiss actions are intentionally absent — they require POST handlers
// that ship in a later bead behind the auth token gate. The panel
// surfaces the queue so the engineer knows to alt-tab to a terminal and
// run `bd human respond <id>` / `bd human dismiss <id>` for now. That is
// the CLI-passthrough pattern Council Q2 endorsed.
function TriagePanel({
  flags,
  onIssueClick,
}: {
  flags: HumanFlagsResponse | null;
  onIssueClick: (id: string) => void;
}) {
  if (!flags) {
    return <div className="loading">Loading triage queue…</div>;
  }
  if (flags.count === 0) {
    return (
      <div className="triage-empty" style={{ padding: 24, color: '#6b7280' }}>
        <h3 style={{ marginTop: 0 }}>Triage queue empty</h3>
        <p>
          No beads carry the <code>human</code> label right now. Anything an
          AI agent or automation flags for human decision will appear here.
        </p>
      </div>
    );
  }
  return (
    <div className="triage">
      <ScreenShareBanner />
      <div style={{ padding: 16 }}>
        <h2 style={{ marginTop: 0 }}>Triage ({flags.count})</h2>
        <p style={{ color: '#6b7280', fontSize: 13 }}>
          Read-only view — respond / dismiss actions require the auth-token
          gate that ships in a later bead. For now: click an issue to view
          details, then run <code>bd human respond &lt;id&gt;</code> or{' '}
          <code>bd human dismiss &lt;id&gt;</code> in a terminal.
        </p>
        <ul style={{ padding: 0, listStyle: 'none' }}>
          {flags.flags.map((issue) => (
            <li
              key={issue.id}
              onClick={() => onIssueClick(issue.id)}
              style={{
                padding: '12px 16px',
                marginBottom: 8,
                backgroundColor: '#1f2937',
                borderRadius: 6,
                cursor: 'pointer',
                color: '#f3f4f6',
              }}
            >
              <div style={{ fontFamily: 'monospace', fontSize: 12, color: '#9ca3af' }}>
                {issue.id}
              </div>
              <div style={{ marginTop: 4, fontWeight: 600 }}>{issue.title}</div>
              <div style={{ marginTop: 4, fontSize: 12, color: '#9ca3af' }}>
                {issue.priority} · {issue.status}
              </div>
            </li>
          ))}
        </ul>
      </div>
    </div>
  );
}

function StatusBadge({ status }: { status: string }) {
  const colors: Record<string, string> = {
    pending: '#6b7280',
    in_progress: '#f59e0b',
    done: '#10b981',
    blocked: '#ef4444',
    active: '#10b981',
    offline: '#6b7280',
    idle: '#f59e0b',
    stuck: '#ef4444',
  };
  return (
    <span
      className="status-badge"
      style={{ backgroundColor: colors[status] || '#6b7280' }}
    >
      {status.replace('_', ' ')}
    </span>
  );
}

function IssueCard({
  issue,
  onClick,
}: {
  issue: IssueSummary;
  onClick: () => void;
}) {
  return (
    <div className="issue-card" onClick={onClick}>
      <div className="issue-title">{issue.title}</div>
      <div className="issue-meta">
        <StatusBadge status={issue.status} />
        <span className="issue-priority">{issue.priority}</span>
      </div>
    </div>
  );
}

function BoardColumn({
  column,
  onIssueClick,
}: {
  column: Column;
  onIssueClick: (id: string) => void;
}) {
  return (
    <div className="board-column">
      <div className="column-header">
        <span className="column-title">{column.label}</span>
        <span className="column-count">{column.count}</span>
      </div>
      <div className="column-issues">
        {column.issues.map((issue) => (
          <IssueCard
            key={issue.id}
            issue={issue}
            onClick={() => onIssueClick(issue.id)}
          />
        ))}
      </div>
    </div>
  );
}

function IssueDetail({
  issue,
  onClose,
}: {
  issue: Issue;
  onClose: () => void;
}) {
  return (
    <div className="issue-detail-overlay" onClick={onClose}>
      <div className="issue-detail" onClick={(e) => e.stopPropagation()}>
        <button className="close-btn" onClick={onClose}>
          &times;
        </button>
        <h2>{issue.title}</h2>
        <div className="issue-detail-meta">
          <StatusBadge status={issue.status} />
          <span className="issue-priority">[{issue.priority}]</span>
          <span className="issue-id">{issue.id}</span>
        </div>

        {issue.description && (
          <div className="issue-section">
            <h3>Description</h3>
            <p className="issue-description">{issue.description}</p>
          </div>
        )}

        {issue.done_when && issue.done_when.length > 0 && (
          <div className="issue-section">
            <h3>Done When</h3>
            <ul>
              {issue.done_when.map((item, i) => (
                <li key={i}>{item}</li>
              ))}
            </ul>
          </div>
        )}

        {issue.blocks && issue.blocks.length > 0 && (
          <div className="issue-section">
            <h3>Blocks</h3>
            <ul>
              {issue.blocks.map((dep) => (
                <li key={dep.id}>
                  {dep.title} <span className="dep-id">({dep.id})</span>
                </li>
              ))}
            </ul>
          </div>
        )}

        {issue.blocked_by && issue.blocked_by.length > 0 && (
          <div className="issue-section">
            <h3>Blocked By</h3>
            <ul>
              {issue.blocked_by.map((dep) => (
                <li key={dep.id}>
                  {dep.title} <span className="dep-id">({dep.id})</span>
                </li>
              ))}
            </ul>
          </div>
        )}

        {issue.children && issue.children.length > 0 && (
          <div className="issue-section">
            <h3>Children</h3>
            <ul>
              {issue.children.map((child) => (
                <li key={child.id}>
                  {child.title} <span className="dep-id">({child.id})</span>
                </li>
              ))}
            </ul>
          </div>
        )}
      </div>
    </div>
  );
}

// Gas Town Components

function formatTimeAgo(dateStr?: string): string {
  if (!dateStr) return '';
  const date = new Date(dateStr);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffMins = Math.floor(diffMs / 60000);

  if (diffMins < 1) return 'just now';
  if (diffMins < 60) return `${diffMins}m ago`;
  const diffHours = Math.floor(diffMins / 60);
  if (diffHours < 24) return `${diffHours}h ago`;
  const diffDays = Math.floor(diffHours / 24);
  return `${diffDays}d ago`;
}

function AgentCard({ agent }: { agent: Agent }) {
  const roleIcons: Record<string, string> = {
    mayor: '👑',
    deacon: '⚙️',
    witness: '👁️',
    refinery: '🏭',
    polecat: '🦨',
    crew: '👷',
  };

  return (
    <div className={`agent-card ${agent.status === 'stuck' ? 'agent-stuck' : ''}`}>
      <div className="agent-icon">{roleIcons[agent.role] || '🤖'}</div>
      <div className="agent-info">
        <div className="agent-name">
          {agent.name}
          {agent.hook_attached && <span className="hook-indicator" title="Work attached">🪝</span>}
        </div>
        <div className="agent-meta">
          <StatusBadge status={agent.status} />
          <span className="agent-role">{agent.role}</span>
          {agent.rig && <span className="agent-rig">{agent.rig}</span>}
        </div>
        {(agent.molecule || agent.last_active) && (
          <div className="agent-details">
            {agent.molecule && (
              <span className="agent-molecule" title="Current molecule">
                📋 {agent.molecule}
              </span>
            )}
            {agent.last_active && (
              <span className="agent-activity" title="Last activity">
                {formatTimeAgo(agent.last_active)}
              </span>
            )}
          </div>
        )}
      </div>
    </div>
  );
}

function RigCard({ rig }: { rig: Rig }) {
  const agentCount = (rig.polecats?.length || 0) + (rig.crew?.length || 0) +
    (rig.witness ? 1 : 0) + (rig.refinery ? 1 : 0);
  const activeCount = [
    ...(rig.polecats || []),
    ...(rig.crew || []),
    rig.witness,
    rig.refinery
  ].filter(a => a && a.status === 'active').length;

  return (
    <div className="rig-card">
      <div className="rig-header">
        <span className="rig-name">{rig.name}</span>
        <span className="rig-stats">{activeCount}/{agentCount} active</span>
      </div>
      <div className="rig-agents">
        {rig.witness && <AgentCard agent={rig.witness} />}
        {rig.refinery && <AgentCard agent={rig.refinery} />}
        {rig.polecats?.map((p, i) => <AgentCard key={`p-${i}`} agent={p} />)}
        {rig.crew?.map((c, i) => <AgentCard key={`c-${i}`} agent={c} />)}
      </div>
    </div>
  );
}

function MoleculeCard({ molecule }: { molecule: Molecule }) {
  const statusIcons: Record<string, string> = {
    pending: '⏳',
    in_progress: '🔄',
    complete: '✅',
    blocked: '🚫',
    failed: '❌',
  };

  const progressPercent = molecule.total > 0
    ? Math.round((molecule.progress / molecule.total) * 100)
    : 0;

  return (
    <div className={`molecule-card ${molecule.status === 'blocked' || molecule.status === 'failed' ? 'molecule-blocked' : ''}`}>
      <div className="molecule-header">
        <span className="molecule-icon">{statusIcons[molecule.status] || '📋'}</span>
        <div className="molecule-title-area">
          <div className="molecule-title">{molecule.title || molecule.id}</div>
          {molecule.formula && (
            <span className="molecule-formula" title="Formula template">
              📐 {molecule.formula}
            </span>
          )}
        </div>
      </div>

      <div className="molecule-progress-bar">
        <div
          className="molecule-progress-fill"
          style={{ width: `${progressPercent}%` }}
        />
      </div>

      <div className="molecule-meta">
        <StatusBadge status={molecule.status} />
        <span className="molecule-step-count">
          Step {molecule.current_step + 1} of {molecule.total}
        </span>
        <span className="molecule-progress-pct">{progressPercent}%</span>
      </div>

      {molecule.agent && (
        <div className="molecule-context">
          <span className="molecule-agent" title="Assigned agent">
            🤖 {molecule.agent}
          </span>
          {molecule.rig && (
            <span className="molecule-rig" title="Rig">
              📦 {molecule.rig}
            </span>
          )}
        </div>
      )}

      {molecule.steps && molecule.steps.length > 0 && (
        <div className="molecule-steps">
          {molecule.steps.slice(0, 5).map((step, i) => (
            <div
              key={step.id || i}
              className={`molecule-step ${step.status === 'complete' || step.status === 'done' ? 'step-complete' : ''} ${i === molecule.current_step ? 'step-current' : ''}`}
            >
              <span className="step-indicator">
                {step.status === 'complete' || step.status === 'done' ? '✓' : i === molecule.current_step ? '▸' : '○'}
              </span>
              <span className="step-description">{step.description || step.id}</span>
            </div>
          ))}
          {molecule.steps.length > 5 && (
            <div className="molecule-step-more">
              +{molecule.steps.length - 5} more steps
            </div>
          )}
        </div>
      )}
    </div>
  );
}

function ConvoyCard({ convoy }: { convoy: Convoy }) {
  const statusIcons: Record<string, string> = {
    pending: '⏳',
    in_progress: '🚚',
    complete: '✅',
    blocked: '🚫',
    failed: '❌',
  };

  const progressPercent = convoy.total > 0
    ? Math.round((convoy.completed / convoy.total) * 100)
    : convoy.progress;

  const priorityColors: Record<string, string> = {
    critical: '#ef4444',
    high: '#f59e0b',
    medium: '#3b82f6',
    low: '#6b7280',
  };

  return (
    <div className={`convoy-card-enhanced ${convoy.status === 'blocked' || convoy.status === 'failed' ? 'convoy-blocked' : ''}`}>
      <div className="convoy-header-enhanced">
        <span className="convoy-icon">{statusIcons[convoy.status] || '📦'}</span>
        <div className="convoy-title-area">
          <div className="convoy-title-enhanced">{convoy.title}</div>
          <div className="convoy-subtitle">
            <span className="convoy-id-badge">{convoy.id}</span>
            {convoy.rig && <span className="convoy-rig">📦 {convoy.rig}</span>}
            {convoy.priority && (
              <span
                className="convoy-priority-badge"
                style={{ backgroundColor: priorityColors[convoy.priority] || '#6b7280' }}
              >
                {convoy.priority}
              </span>
            )}
          </div>
        </div>
      </div>

      <div className="convoy-progress-bar">
        <div
          className="convoy-progress-fill"
          style={{ width: `${progressPercent}%` }}
        />
      </div>

      <div className="convoy-stats">
        <div className="convoy-stat">
          <span className="stat-value">{convoy.completed}</span>
          <span className="stat-label">Done</span>
        </div>
        <div className="convoy-stat">
          <span className="stat-value">{convoy.in_progress}</span>
          <span className="stat-label">Active</span>
        </div>
        <div className="convoy-stat">
          <span className="stat-value">{convoy.blocked}</span>
          <span className="stat-label">Blocked</span>
        </div>
        <div className="convoy-stat">
          <span className="stat-value">{convoy.total - convoy.completed - convoy.in_progress - convoy.blocked}</span>
          <span className="stat-label">Pending</span>
        </div>
      </div>

      <div className="convoy-meta">
        <StatusBadge status={convoy.status} />
        <span className="convoy-progress-text">{progressPercent}% complete</span>
        <span className="convoy-issue-count">{convoy.total} issues</span>
      </div>

      {convoy.agents && convoy.agents.length > 0 && (
        <div className="convoy-agents">
          <span className="convoy-agents-label">Agents:</span>
          {convoy.agents.slice(0, 3).map((agent, i) => (
            <span key={i} className="convoy-agent-badge">🤖 {agent}</span>
          ))}
          {convoy.agents.length > 3 && (
            <span className="convoy-agent-more">+{convoy.agents.length - 3} more</span>
          )}
        </div>
      )}
    </div>
  );
}

function TownView({ town, status, molecules, convoys }: { town: Town | null; status: TownStatus | null; molecules: Molecule[]; convoys: Convoy[] }) {
  if (!town) {
    return (
      <div className="town-empty">
        <h2>Gas Town Not Found</h2>
        <p>No Gas Town workspace found at {status?.town_root || '~/gt'}</p>
        <p>Run <code>gt install ~/gt</code> to create one.</p>
      </div>
    );
  }

  return (
    <div className="town-view">
      {/* Town Status Bar */}
      <div className="town-status-bar">
        <div className="status-item">
          <span className="status-label">Status</span>
          <StatusBadge status={status?.healthy ? 'active' : 'offline'} />
        </div>
        <div className="status-item">
          <span className="status-label">Agents</span>
          <span className="status-value">{status?.active_agents || 0}/{status?.total_agents || 0}</span>
        </div>
        <div className="status-item">
          <span className="status-label">Rigs</span>
          <span className="status-value">{status?.active_rigs || 0}</span>
        </div>
        <div className="status-item">
          <span className="status-label">Convoys</span>
          <span className="status-value">{status?.open_convoys || 0}</span>
        </div>
        <div className="status-item">
          <span className="status-label">Molecules</span>
          <span className="status-value">{molecules.length}</span>
        </div>
      </div>

      {/* Active Molecules */}
      {molecules.length > 0 && (
        <div className="town-molecules">
          <h3>Active Molecules ({molecules.length})</h3>
          <div className="molecules-grid">
            {molecules.map((mol) => (
              <MoleculeCard key={mol.id} molecule={mol} />
            ))}
          </div>
        </div>
      )}

      {/* Town-level agents */}
      <div className="town-agents">
        <h3>Town Agents</h3>
        <div className="agents-grid">
          {town.mayor && <AgentCard agent={town.mayor} />}
          {town.deacon && <AgentCard agent={town.deacon} />}
        </div>
      </div>

      {/* Rigs */}
      <div className="town-rigs">
        <h3>Rigs ({town.rigs?.length || 0})</h3>
        {town.rigs?.length === 0 ? (
          <p className="empty-message">No rigs configured. Run <code>gt rig add &lt;name&gt;</code></p>
        ) : (
          <div className="rigs-grid">
            {town.rigs?.map((rig) => <RigCard key={rig.name} rig={rig} />)}
          </div>
        )}
      </div>

      {/* Convoys */}
      {convoys.length > 0 && (
        <div className="town-convoys">
          <h3>Active Convoys ({convoys.length})</h3>
          <div className="convoys-grid">
            {convoys.map((convoy) => (
              <ConvoyCard key={convoy.id} convoy={convoy} />
            ))}
          </div>
        </div>
      )}
    </div>
  );
}

function App() {
  const [viewMode, setViewMode] = useState<ViewMode>('beads');
  const [board, setBoard] = useState<BoardResponse | null>(null);
  const [selectedIssue, setSelectedIssue] = useState<Issue | null>(null);
  const [town, setTown] = useState<Town | null>(null);
  const [townStatus, setTownStatus] = useState<TownStatus | null>(null);
  const [molecules, setMolecules] = useState<Molecule[]>([]);
  const [convoys, setConvoys] = useState<Convoy[]>([]);
  const [sync, setSync] = useState<DoltSyncState | null>(null);
  const [humanFlags, setHumanFlags] = useState<HumanFlagsResponse | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    loadData();
    const interval = setInterval(loadData, 5000);
    return () => clearInterval(interval);
  }, []);

  async function loadData() {
    try {
      const [boardData, townData, statusData, moleculesData, convoysData, syncData, humanData] = await Promise.all([
        fetchBoard().catch(() => null),
        fetchTown().catch(() => null),
        fetchTownStatus().catch(() => null),
        fetchMolecules().catch(() => null),
        fetchConvoys().catch(() => null),
        fetchSync().catch(() => null),
        fetchHumanFlags().catch(() => null),
      ]);
      if (boardData) setBoard(boardData);
      setTown(townData);
      setTownStatus(statusData);
      setMolecules(moleculesData?.molecules || []);
      setConvoys(convoysData?.convoys || []);
      setSync(syncData);
      setHumanFlags(humanData);
      setError(null);
    } catch {
      setError('Failed to connect to daemon. Is gvid running on localhost:7070?');
    } finally {
      setLoading(false);
    }
  }

  async function handleIssueClick(id: string) {
    try {
      const issue = await fetchIssue(id);
      setSelectedIssue(issue);
    } catch (e) {
      console.error('Failed to fetch issue:', e);
    }
  }

  if (loading) {
    return (
      <div className="app">
        <div className="loading">Loading...</div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="app">
        <div className="error">
          <h2>Connection Error</h2>
          <p>{error}</p>
          <button onClick={loadData}>Retry</button>
        </div>
      </div>
    );
  }

  return (
    <div className="app">
      <header className="app-header">
        <h1>
          Gastown Viewer Intent
          <SyncPill state={sync} />
        </h1>
        <div className="view-tabs">
          <button
            className={`tab ${viewMode === 'beads' ? 'active' : ''}`}
            onClick={() => setViewMode('beads')}
          >
            Board ({board?.total || 0})
          </button>
          <button
            className={`tab ${viewMode === 'graph' ? 'active' : ''}`}
            onClick={() => setViewMode('graph')}
          >
            Graph
          </button>
          <button
            className={`tab ${viewMode === 'gastown' ? 'active' : ''}`}
            onClick={() => setViewMode('gastown')}
          >
            Gas Town {townStatus?.healthy ? '●' : '○'}
          </button>
          <button
            className={`tab ${viewMode === 'triage' ? 'active' : ''}`}
            onClick={() => setViewMode('triage')}
          >
            Triage ({humanFlags?.count ?? 0})
          </button>
        </div>
      </header>

      {viewMode === 'beads' && (
        <div className="board">
          {board?.columns.map((column) => (
            <BoardColumn
              key={column.status}
              column={column}
              onIssueClick={handleIssueClick}
            />
          ))}
        </div>
      )}

      {viewMode === 'graph' && (
        <DependencyGraph
          onNodeClick={handleIssueClick}
          width={window.innerWidth - 32}
          height={window.innerHeight - 200}
        />
      )}

      {viewMode === 'gastown' && (
        <TownView town={town} status={townStatus} molecules={molecules} convoys={convoys} />
      )}

      {viewMode === 'triage' && (
        <TriagePanel flags={humanFlags} onIssueClick={handleIssueClick} />
      )}

      {selectedIssue && (
        <IssueDetail
          issue={selectedIssue}
          onClose={() => setSelectedIssue(null)}
        />
      )}
    </div>
  );
}

export default App;
