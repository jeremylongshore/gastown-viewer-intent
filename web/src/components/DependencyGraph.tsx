import { useEffect, useRef, useState, useCallback } from 'react';
import * as d3 from 'd3';
import type { GraphResponse, EdgeType, Status } from '../api';
import { fetchGraphJSON, fetchGraphDOT } from '../api';

// Color schemes
const statusColors: Record<Status, string> = {
  pending: '#3b82f6',    // blue
  in_progress: '#eab308', // yellow
  done: '#22c55e',       // green
  blocked: '#ef4444',    // red
};

const edgeColors: Record<EdgeType, string> = {
  blocks: '#ef4444',          // red
  blocked_by: '#ef4444',      // red
  parent: '#6b7280',          // gray
  child: '#6b7280',           // gray
  waits_for: '#f97316',       // orange
  waited_by: '#f97316',       // orange
  conditional_blocks: '#a855f7', // purple
  relates_to: '#3b82f6',      // blue
  duplicates: '#ec4899',      // pink
  mentions: '#06b6d4',        // cyan
  derived_from: '#84cc16',    // lime
  supersedes: '#f59e0b',      // amber
  implements: '#22c55e',      // green
  unknown: '#9ca3af',         // gray
};

const edgeStyles: Record<EdgeType, string> = {
  blocks: 'solid',
  blocked_by: 'dashed',
  parent: 'dashed',
  child: 'dotted',
  waits_for: 'dashed',
  waited_by: 'dotted',
  conditional_blocks: 'dashed',
  relates_to: 'dotted',
  duplicates: 'dotted',
  mentions: 'dotted',
  derived_from: 'dashed',
  supersedes: 'solid',
  implements: 'solid',
  unknown: 'dotted',
};

interface SimNode extends d3.SimulationNodeDatum {
  id: string;
  title: string;
  status: Status;
}

interface SimLink extends d3.SimulationLinkDatum<SimNode> {
  type: EdgeType;
}

interface DependencyGraphProps {
  onNodeClick?: (nodeId: string) => void;
  width?: number;
  height?: number;
}

export default function DependencyGraph({
  onNodeClick,
  width = 800,
  height = 600
}: DependencyGraphProps) {
  const svgRef = useRef<SVGSVGElement>(null);
  const [graph, setGraph] = useState<GraphResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [selectedEdgeTypes, setSelectedEdgeTypes] = useState<Set<EdgeType>>(
    new Set(['blocks', 'blocked_by', 'parent', 'child'])
  );

  const loadGraph = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);
      const data = await fetchGraphJSON();
      setGraph(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load graph');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadGraph();
  }, [loadGraph]);

  useEffect(() => {
    if (!graph || !svgRef.current) return;

    const svg = d3.select(svgRef.current);
    svg.selectAll('*').remove();

    // Filter edges by selected types
    const filteredEdges = graph.edges.filter(e => selectedEdgeTypes.has(e.type));

    // Create simulation data
    const nodes: SimNode[] = graph.nodes.map(n => ({ ...n }));
    const nodeMap = new Map(nodes.map(n => [n.id, n]));

    const links: SimLink[] = filteredEdges
      .filter(e => nodeMap.has(e.from) && nodeMap.has(e.to))
      .map(e => ({
        source: nodeMap.get(e.from)!,
        target: nodeMap.get(e.to)!,
        type: e.type,
      }));

    // Create zoom behavior
    const zoom = d3.zoom<SVGSVGElement, unknown>()
      .scaleExtent([0.1, 4])
      .on('zoom', (event) => {
        container.attr('transform', event.transform);
      });

    svg.call(zoom);

    const container = svg.append('g');

    // Arrow markers for each edge type
    const defs = svg.append('defs');
    Object.entries(edgeColors).forEach(([type, color]) => {
      defs.append('marker')
        .attr('id', `arrow-${type}`)
        .attr('viewBox', '0 -5 10 10')
        .attr('refX', 20)
        .attr('refY', 0)
        .attr('markerWidth', 6)
        .attr('markerHeight', 6)
        .attr('orient', 'auto')
        .append('path')
        .attr('fill', color)
        .attr('d', 'M0,-5L10,0L0,5');
    });

    // Create simulation
    const simulation = d3.forceSimulation(nodes)
      .force('link', d3.forceLink<SimNode, SimLink>(links)
        .id(d => d.id)
        .distance(150))
      .force('charge', d3.forceManyBody().strength(-400))
      .force('center', d3.forceCenter(width / 2, height / 2))
      .force('collision', d3.forceCollide().radius(50));

    // Draw links
    const link = container.append('g')
      .selectAll('line')
      .data(links)
      .join('line')
      .attr('stroke', d => edgeColors[d.type])
      .attr('stroke-width', d => d.type === 'blocks' ? 2 : 1.5)
      .attr('stroke-dasharray', d => {
        const style = edgeStyles[d.type];
        if (style === 'dashed') return '8,4';
        if (style === 'dotted') return '2,4';
        return null;
      })
      .attr('marker-end', d => `url(#arrow-${d.type})`);

    // Draw nodes
    const node = container.append('g')
      .selectAll<SVGGElement, SimNode>('g')
      .data(nodes)
      .join('g')
      .attr('cursor', 'pointer');

    // Add drag behavior
    const dragBehavior = d3.drag<SVGGElement, SimNode>()
      .on('start', (event, d) => {
        if (!event.active) simulation.alphaTarget(0.3).restart();
        d.fx = d.x;
        d.fy = d.y;
      })
      .on('drag', (event, d) => {
        d.fx = event.x;
        d.fy = event.y;
      })
      .on('end', (event) => {
        if (!event.active) simulation.alphaTarget(0);
        // d.fx/d.fy remain set so the node stays where the user placed it
      });

    node.call(dragBehavior);

    // Node background
    node.append('rect')
      .attr('width', 120)
      .attr('height', 40)
      .attr('x', -60)
      .attr('y', -20)
      .attr('rx', 8)
      .attr('fill', d => statusColors[d.status])
      .attr('stroke', '#fff')
      .attr('stroke-width', 2);

    // Node label
    node.append('text')
      .attr('text-anchor', 'middle')
      .attr('dy', '0.35em')
      .attr('fill', '#fff')
      .attr('font-size', '11px')
      .attr('font-weight', 'bold')
      .text(d => d.title.length > 15 ? d.title.slice(0, 15) + '...' : d.title);

    // Node ID (smaller, below)
    node.append('text')
      .attr('text-anchor', 'middle')
      .attr('dy', '1.5em')
      .attr('fill', 'rgba(255,255,255,0.7)')
      .attr('font-size', '9px')
      .text(d => d.id);

    // Click handler
    node.on('click', (_, d) => {
      if (onNodeClick) onNodeClick(d.id);
    });

    // Tooltip
    node.append('title')
      .text(d => `${d.title}\nID: ${d.id}\nStatus: ${d.status}`);

    // Update positions on tick
    simulation.on('tick', () => {
      link
        .attr('x1', d => (d.source as SimNode).x!)
        .attr('y1', d => (d.source as SimNode).y!)
        .attr('x2', d => (d.target as SimNode).x!)
        .attr('y2', d => (d.target as SimNode).y!);

      node.attr('transform', d => `translate(${d.x},${d.y})`);
    });

    return () => {
      simulation.stop();
    };
  }, [graph, selectedEdgeTypes, width, height, onNodeClick]);

  const handleExportDOT = async () => {
    try {
      const dot = await fetchGraphDOT();
      const blob = new Blob([dot], { type: 'text/vnd.graphviz' });
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = 'dependencies.dot';
      a.click();
      URL.revokeObjectURL(url);
    } catch (err) {
      console.error('Failed to export DOT:', err);
    }
  };

  const toggleEdgeType = (type: EdgeType) => {
    setSelectedEdgeTypes(prev => {
      const next = new Set(prev);
      if (next.has(type)) {
        next.delete(type);
      } else {
        next.add(type);
      }
      return next;
    });
  };

  const edgeTypesInGraph = graph?.edges
    ? [...new Set(graph.edges.map(e => e.type))]
    : [];

  if (loading) {
    return (
      <div className="graph-loading">
        Loading dependency graph...
      </div>
    );
  }

  if (error) {
    return (
      <div className="graph-error">
        <p>Error: {error}</p>
        <button onClick={loadGraph}>Retry</button>
      </div>
    );
  }

  return (
    <div className="dependency-graph">
      <div className="graph-toolbar">
        <div className="graph-filters">
          <span>Show edges:</span>
          {edgeTypesInGraph.map(type => (
            <label key={type} className="edge-filter">
              <input
                type="checkbox"
                checked={selectedEdgeTypes.has(type)}
                onChange={() => toggleEdgeType(type)}
              />
              <span
                className="edge-type-badge"
                style={{
                  backgroundColor: edgeColors[type],
                  opacity: selectedEdgeTypes.has(type) ? 1 : 0.4
                }}
              >
                {type.replace('_', ' ')}
              </span>
            </label>
          ))}
        </div>
        <div className="graph-actions">
          <button onClick={loadGraph} title="Refresh">
            Refresh
          </button>
          <button onClick={handleExportDOT} title="Export to DOT format">
            Export DOT
          </button>
        </div>
      </div>
      <div className="graph-stats">
        {graph?.stats && (
          <span>
            {graph.stats.node_count} nodes, {graph.stats.edge_count} edges
          </span>
        )}
      </div>
      <svg
        ref={svgRef}
        width={width}
        height={height}
        className="graph-svg"
      />
      <div className="graph-legend">
        <div className="legend-section">
          <strong>Status:</strong>
          {Object.entries(statusColors).map(([status, color]) => (
            <span key={status} className="legend-item">
              <span className="legend-color" style={{ backgroundColor: color }} />
              {status.replace('_', ' ')}
            </span>
          ))}
        </div>
      </div>
    </div>
  );
}
