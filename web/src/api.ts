// API client for gvid daemon

const API_BASE = '/api/v1';

export type Status = 'pending' | 'in_progress' | 'done' | 'blocked';
export type Priority = 'critical' | 'high' | 'medium' | 'low';

export interface IssueSummary {
  id: string;
  title: string;
  status: Status;
  priority: Priority;
}

export interface Issue {
  id: string;
  title: string;
  description: string;
  status: Status;
  priority: Priority;
  parent?: IssueSummary;
  children: IssueSummary[];
  blocks: IssueSummary[];
  blocked_by: IssueSummary[];
  done_when: string[];
  created_at: string;
  updated_at: string;
}

export interface Column {
  status: Status;
  label: string;
  count: number;
  issues: IssueSummary[];
}

export interface BoardResponse {
  columns: Column[];
  total: number;
}

export interface HealthResponse {
  status: string;
  beads_initialized: boolean;
  version: string;
  bd_version?: string;
  error?: string;
}

// Graph Types
export type EdgeType =
  | 'blocks'
  | 'blocked_by'
  | 'parent'
  | 'child'
  | 'waits_for'
  | 'waited_by'
  | 'conditional_blocks'
  | 'relates_to'
  | 'duplicates'
  | 'mentions'
  | 'derived_from'
  | 'supersedes'
  | 'implements'
  | 'unknown';

export interface GraphNode {
  id: string;
  title: string;
  status: Status;
  priority: Priority;
}

export interface GraphEdge {
  from: string;
  to: string;
  type: EdgeType;
}

export interface GraphStats {
  node_count: number;
  edge_count: number;
  max_depth: number;
}

export interface GraphResponse {
  nodes: GraphNode[];
  edges: GraphEdge[];
  stats: GraphStats;
}

export async function fetchHealth(): Promise<HealthResponse> {
  const res = await fetch(`${API_BASE}/health`);
  if (!res.ok) throw new Error(`HTTP ${res.status}`);
  return res.json();
}

export async function fetchBoard(): Promise<BoardResponse> {
  const res = await fetch(`${API_BASE}/board`);
  if (!res.ok) throw new Error(`HTTP ${res.status}`);
  return res.json();
}

export async function fetchIssue(id: string): Promise<Issue> {
  const res = await fetch(`${API_BASE}/issues/${encodeURIComponent(id)}`);
  if (!res.ok) throw new Error(`HTTP ${res.status}`);
  return res.json();
}

export async function fetchGraph(format: 'json' | 'dot' = 'json'): Promise<GraphResponse | string> {
  const res = await fetch(`${API_BASE}/graph?format=${format}`);
  if (!res.ok) throw new Error(`HTTP ${res.status}`);
  if (format === 'dot') {
    return res.text();
  }
  return res.json();
}

export async function fetchGraphJSON(): Promise<GraphResponse> {
  const res = await fetch(`${API_BASE}/graph?format=json`);
  if (!res.ok) throw new Error(`HTTP ${res.status}`);
  return res.json();
}

export async function fetchGraphDOT(): Promise<string> {
  const res = await fetch(`${API_BASE}/graph?format=dot`);
  if (!res.ok) throw new Error(`HTTP ${res.status}`);
  return res.text();
}

// Gas Town Types

export type AgentRole = 'mayor' | 'deacon' | 'witness' | 'refinery' | 'crew' | 'polecat';
export type AgentStatus = 'active' | 'idle' | 'stuck' | 'offline' | 'unknown';

export interface Agent {
  role: AgentRole;
  name: string;
  rig?: string;
  status: AgentStatus;
  session?: string;
  molecule?: string;
  hook_attached?: boolean;
  last_active?: string;
  compaction?: number;
  work_dir?: string;
}

export interface Rig {
  name: string;
  path: string;
  remote?: string;
  witness?: Agent;
  refinery?: Agent;
  polecats: Agent[];
  crew: Agent[];
}

export type ConvoyStatus = 'pending' | 'in_progress' | 'complete' | 'blocked' | 'failed';

export interface Convoy {
  id: string;
  title: string;
  status: ConvoyStatus;
  priority?: string;
  rig?: string;
  issues: string[];
  progress: number;
  total: number;
  completed: number;
  blocked: number;
  in_progress: number;
  created_at?: string;
  updated_at?: string;
  subscribers?: string[];
  agents?: string[];
}

export interface Town {
  root: string;
  name?: string;
  rigs: Rig[];
  mayor?: Agent;
  deacon?: Agent;
  convoys: Convoy[];
}

export interface TownStatus {
  healthy: boolean;
  town_root: string;
  active_agents: number;
  total_agents: number;
  active_rigs: number;
  open_convoys: number;
  error?: string;
}

export interface AgentsResponse {
  agents: Agent[];
  total: number;
  active: number;
  offline: number;
}

export interface RigsResponse {
  rigs: Rig[];
  total: number;
}

export interface ConvoysResponse {
  convoys: Convoy[];
  total: number;
  in_progress: number;
  pending: number;
  complete: number;
  blocked: number;
}

// Gas Town API calls

export async function fetchTownStatus(): Promise<TownStatus> {
  const res = await fetch(`${API_BASE}/town/status`);
  if (!res.ok) throw new Error(`HTTP ${res.status}`);
  return res.json();
}

export async function fetchTown(): Promise<Town> {
  const res = await fetch(`${API_BASE}/town`);
  if (!res.ok) throw new Error(`HTTP ${res.status}`);
  return res.json();
}

export async function fetchRigs(): Promise<RigsResponse> {
  const res = await fetch(`${API_BASE}/town/rigs`);
  if (!res.ok) throw new Error(`HTTP ${res.status}`);
  return res.json();
}

export async function fetchAgents(): Promise<AgentsResponse> {
  const res = await fetch(`${API_BASE}/town/agents`);
  if (!res.ok) throw new Error(`HTTP ${res.status}`);
  return res.json();
}

export async function fetchConvoys(): Promise<ConvoysResponse> {
  const res = await fetch(`${API_BASE}/town/convoys`);
  if (!res.ok) throw new Error(`HTTP ${res.status}`);
  return res.json();
}

export async function fetchConvoy(id: string): Promise<Convoy> {
  const res = await fetch(`${API_BASE}/town/convoys/${encodeURIComponent(id)}`);
  if (!res.ok) throw new Error(`HTTP ${res.status}`);
  return res.json();
}

// Molecule Types

export type MoleculeStatus = 'pending' | 'in_progress' | 'complete' | 'blocked' | 'failed';

export interface MoleculeStep {
  index: number;
  id: string;
  description: string;
  status: string;
  needs?: string[];
  started_at?: string;
  completed_at?: string;
}

export interface Molecule {
  id: string;
  title: string;
  status: MoleculeStatus;
  steps: MoleculeStep[];
  current_step: number;
  progress: number;
  total: number;
  formula?: string;
  agent?: string;
  rig?: string;
  created_at?: string;
  updated_at?: string;
}

export interface MoleculesResponse {
  molecules: Molecule[];
  total: number;
  in_progress: number;
  pending: number;
  complete: number;
  blocked: number;
}

// Molecule API calls

export async function fetchMolecules(): Promise<MoleculesResponse> {
  const res = await fetch(`${API_BASE}/town/molecules`);
  if (!res.ok) throw new Error(`HTTP ${res.status}`);
  return res.json();
}

export async function fetchMolecule(id: string): Promise<Molecule> {
  const res = await fetch(`${API_BASE}/town/molecules/${encodeURIComponent(id)}`);
  if (!res.ok) throw new Error(`HTTP ${res.status}`);
  return res.json();
}

// Memories — read-only by architectural invariant (Council Q2). The
// redaction layer runs server-side; clients pass `reveal=true` to get
// the raw content. UI bins should NEVER persist a reveal across
// reloads — that's per 005-PP-POLICY § 4.

export interface Memory {
  key: string;
  content: string;
  redacted?: boolean;
  redaction_markers?: string[];
}

export interface MemoriesResponse {
  memories: Memory[];
  count: number;
  schema_version: number;
}

export async function fetchMemories(reveal = false): Promise<MemoriesResponse> {
  const q = reveal ? '?reveal=true' : '';
  const res = await fetch(`${API_BASE}/memories${q}`);
  if (!res.ok) throw new Error(`HTTP ${res.status}`);
  return res.json();
}

export async function fetchMemory(key: string, reveal = false): Promise<Memory> {
  const q = reveal ? '?reveal=true' : '';
  const res = await fetch(`${API_BASE}/memories/${encodeURIComponent(key)}${q}`);
  if (!res.ok) throw new Error(`HTTP ${res.status}`);
  return res.json();
}

export async function searchMemories(query: string, reveal = false): Promise<MemoriesResponse> {
  const params = new URLSearchParams();
  if (query) params.set('q', query);
  if (reveal) params.set('reveal', 'true');
  const qs = params.toString();
  const res = await fetch(`${API_BASE}/memories/search${qs ? '?' + qs : ''}`);
  if (!res.ok) throw new Error(`HTTP ${res.status}`);
  return res.json();
}
