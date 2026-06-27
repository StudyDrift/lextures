import { WORKFLOW_VERSION } from './types'
import type { GraderWorkflowEdge, GraderWorkflowGraph, GraderWorkflowNode } from './types'
import { newWorkflowNodeId } from './workflow-node-id'

/** A group boundary port mapping a group handle to an internal node handle. */
export type GraderGroupPort = {
  id: string
  label?: string
  nodeId: string
  handle: string
}

/** Stored in a group node's `data`. The nested subgraph plus auto-derived ports. */
export type GraderGroupNodeData = {
  label?: string
  subgraph: GraderWorkflowGraph
  inputs: GraderGroupPort[]
  outputs: GraderGroupPort[]
}

const MAX_GROUP_EXPANSIONS = 500

export function isGroupNode(node: { type: string }): boolean {
  return node.type === 'group'
}

export function graphContainsGroup(graph: GraderWorkflowGraph): boolean {
  return graph.nodes.some((n) => n.type === 'group')
}

export function groupNodeData(node: GraderWorkflowNode): GraderGroupNodeData {
  const data = (node.data ?? {}) as Partial<GraderGroupNodeData>
  return {
    label: typeof data.label === 'string' ? data.label : undefined,
    subgraph: data.subgraph ?? { version: WORKFLOW_VERSION, nodes: [], edges: [] },
    inputs: Array.isArray(data.inputs) ? data.inputs : [],
    outputs: Array.isArray(data.outputs) ? data.outputs : [],
  }
}

/**
 * Recursively inlines every group node, mirroring the server's FlattenWorkflowGraph.
 * Internal ids are prefixed with `groupId/` and boundary edges are rewired through ports.
 */
export function flattenWorkflowGraph(graph: GraderWorkflowGraph): GraderWorkflowGraph {
  let cur: GraderWorkflowGraph = {
    version: graph.version,
    nodes: [...graph.nodes],
    edges: [...graph.edges],
  }
  for (let i = 0; i < MAX_GROUP_EXPANSIONS; i++) {
    const gi = cur.nodes.findIndex((n) => n.type === 'group')
    if (gi < 0) return { ...cur, version: cur.version || WORKFLOW_VERSION }
    cur = expandGroupNode(cur, gi)
  }
  return cur
}

function expandGroupNode(graph: GraderWorkflowGraph, gi: number): GraderWorkflowGraph {
  const gnode = graph.nodes[gi]
  const gd = groupNodeData(gnode)
  const prefix = `${gnode.id}/`

  const nodes: GraderWorkflowNode[] = []
  graph.nodes.forEach((n, idx) => {
    if (idx !== gi) nodes.push(n)
  })
  for (const m of gd.subgraph.nodes) {
    nodes.push({ ...m, id: prefix + m.id })
  }

  const inById = new Map(gd.inputs.map((p) => [p.id, p]))
  const outById = new Map(gd.outputs.map((p) => [p.id, p]))

  const edges: GraderWorkflowEdge[] = []
  for (const e of gd.subgraph.edges) {
    edges.push({ ...e, id: prefix + e.id, source: prefix + e.source, target: prefix + e.target })
  }
  for (const e of graph.edges) {
    const srcIs = e.source === gnode.id
    const tgtIs = e.target === gnode.id
    if (!srcIs && !tgtIs) {
      edges.push(e)
      continue
    }
    const ne: GraderWorkflowEdge = { ...e, id: `${prefix}boundary/${e.id}` }
    if (srcIs) {
      const p = outById.get((e.sourceHandle ?? '').trim())
      if (!p) continue
      ne.source = prefix + p.nodeId
      ne.sourceHandle = p.handle
    }
    if (tgtIs) {
      const p = inById.get((e.targetHandle ?? '').trim())
      if (!p) continue
      ne.target = prefix + p.nodeId
      ne.targetHandle = p.handle
    }
    edges.push(ne)
  }

  return { version: graph.version, nodes, edges }
}

export type CreateGroupResult = {
  graph: GraderWorkflowGraph
  groupId: string
}

/**
 * Collapses the selected nodes into a single group node. Ports are auto-derived
 * from edges crossing the selection boundary. Returns null when the selection is
 * not groupable (fewer than two nodes, or it includes the fixed output node).
 */
export function createGroupFromSelection(
  graph: GraderWorkflowGraph,
  selectedIds: string[],
  label?: string,
): CreateGroupResult | null {
  const selected = new Set(selectedIds)
  const members = graph.nodes.filter((n) => selected.has(n.id))
  if (members.length < 2) return null
  if (members.some((n) => n.type === 'output')) return null

  const internalEdges: GraderWorkflowEdge[] = []
  const inputPorts: GraderGroupPort[] = []
  const outputPorts: GraderGroupPort[] = []
  const inputKey = new Map<string, GraderGroupPort>()
  const outputKey = new Map<string, GraderGroupPort>()
  const externalEdges: GraderWorkflowEdge[] = []
  const boundaryIn: { edge: GraderWorkflowEdge; port: GraderGroupPort }[] = []
  const boundaryOut: { edge: GraderWorkflowEdge; port: GraderGroupPort }[] = []

  const groupId = newWorkflowNodeId('grp')

  for (const e of graph.edges) {
    const srcIn = selected.has(e.source)
    const tgtIn = selected.has(e.target)
    if (srcIn && tgtIn) {
      internalEdges.push(e)
    } else if (tgtIn && !srcIn) {
      const handle = (e.targetHandle ?? '').trim()
      const key = `${e.target}::${handle}`
      let port = inputKey.get(key)
      if (!port) {
        port = { id: newWorkflowNodeId('in'), label: portLabel(e.target, members), nodeId: e.target, handle }
        inputKey.set(key, port)
        inputPorts.push(port)
      }
      boundaryIn.push({ edge: e, port })
    } else if (srcIn && !tgtIn) {
      const handle = (e.sourceHandle ?? '').trim()
      const key = `${e.source}::${handle}`
      let port = outputKey.get(key)
      if (!port) {
        port = { id: newWorkflowNodeId('out'), label: portLabel(e.source, members), nodeId: e.source, handle }
        outputKey.set(key, port)
        outputPorts.push(port)
      }
      boundaryOut.push({ edge: e, port })
    } else {
      externalEdges.push(e)
    }
  }

  const groupNode: GraderWorkflowNode = {
    id: groupId,
    type: 'group',
    position: centroid(members),
    data: {
      label: label ?? 'Group',
      subgraph: { version: graph.version || WORKFLOW_VERSION, nodes: members, edges: internalEdges },
      inputs: inputPorts,
      outputs: outputPorts,
    } satisfies GraderGroupNodeData as unknown as Record<string, unknown>,
  }

  const rewired: GraderWorkflowEdge[] = [...externalEdges]
  for (const { edge, port } of boundaryIn) {
    rewired.push({ ...edge, id: newWorkflowNodeId('e'), target: groupId, targetHandle: port.id })
  }
  for (const { edge, port } of boundaryOut) {
    rewired.push({ ...edge, id: newWorkflowNodeId('e'), source: groupId, sourceHandle: port.id })
  }

  return {
    graph: {
      version: graph.version || WORKFLOW_VERSION,
      nodes: [...graph.nodes.filter((n) => !selected.has(n.id)), groupNode],
      edges: rewired,
    },
    groupId,
  }
}

/** Inlines a single group node back into the current graph (one level of flattening). */
export function ungroupNode(graph: GraderWorkflowGraph, groupId: string): GraderWorkflowGraph {
  const gi = graph.nodes.findIndex((n) => n.id === groupId && n.type === 'group')
  if (gi < 0) return graph
  return expandGroupNode(graph, gi)
}

function portLabel(nodeId: string, members: GraderWorkflowNode[]): string {
  const node = members.find((n) => n.id === nodeId)
  const label = node?.data && typeof (node.data as { label?: unknown }).label === 'string'
    ? ((node.data as { label?: string }).label as string)
    : undefined
  return label ?? node?.type ?? nodeId
}

function centroid(nodes: GraderWorkflowNode[]): { x: number; y: number } {
  if (nodes.length === 0) return { x: 0, y: 0 }
  const sum = nodes.reduce(
    (acc, n) => ({ x: acc.x + (n.position?.x ?? 0), y: acc.y + (n.position?.y ?? 0) }),
    { x: 0, y: 0 },
  )
  return { x: Math.round(sum.x / nodes.length), y: Math.round(sum.y / nodes.length) }
}
