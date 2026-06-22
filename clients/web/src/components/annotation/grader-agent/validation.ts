import type { GraderWorkflowGraph, WorkflowValidationIssue } from './types'
import { WORKFLOW_VERSION } from './types'

const MAX_NODES = 50
const MAX_EDGES = 100

function graderPrompt(data: Record<string, unknown>): string {
  return typeof data.prompt === 'string' ? data.prompt.trim() : ''
}

function hasCycle(adj: Map<string, string[]>, nodeIds: string[]): boolean {
  const state = new Map<string, 0 | 1 | 2>()
  const visit = (u: string): boolean => {
    const s = state.get(u) ?? 0
    if (s === 1) return true
    if (s === 2) return false
    state.set(u, 1)
    for (const v of adj.get(u) ?? []) {
      if (visit(v)) return true
    }
    state.set(u, 2)
    return false
  }
  for (const id of nodeIds) {
    if ((state.get(id) ?? 0) === 0 && visit(id)) return true
  }
  return false
}

/** Client-side validator mirroring server workflow rules. */
export function validateWorkflowGraph(graph: GraderWorkflowGraph | null | undefined): WorkflowValidationIssue[] {
  const issues: WorkflowValidationIssue[] = []
  if (!graph) {
    issues.push({ field: 'workflowGraph', message: 'Workflow graph is required.' })
    return issues
  }
  if (graph.version !== WORKFLOW_VERSION) {
    issues.push({ field: 'workflowGraph.version', message: 'Unsupported workflow graph version.' })
    return issues
  }
  if (graph.nodes.length > MAX_NODES) {
    issues.push({ field: 'workflowGraph.nodes', message: `Graph exceeds ${MAX_NODES} node limit.` })
  }
  if (graph.edges.length > MAX_EDGES) {
    issues.push({ field: 'workflowGraph.edges', message: `Graph exceeds ${MAX_EDGES} edge limit.` })
  }

  const nodeById = new Map(graph.nodes.map((n) => [n.id, n]))
  let outputCount = 0
  for (const n of graph.nodes) {
    if (n.type === 'output') outputCount++
  }
  if (outputCount !== 1) {
    issues.push({ field: 'workflowGraph.nodes', message: 'Graph must contain exactly one output node.' })
  }

  const outputSlots = new Set<string>()
  const adj = new Map<string, string[]>()
  for (const e of graph.edges) {
    const src = nodeById.get(e.source)
    const tgt = nodeById.get(e.target)
    if (!src || !tgt) {
      issues.push({ field: 'workflowGraph.edges', message: 'Edge references unknown node.' })
      continue
    }
    if (tgt.type === 'output') {
      const slot = e.targetHandle ?? ''
      if (slot !== 'grade' && slot !== 'comments') {
        issues.push({ field: 'output', message: 'Output node edges must target grade or comments slots.' })
      } else if (e.sourceHandle !== slot) {
        issues.push({
          field: `output.${slot}`,
          message: 'Grade sources must connect to the grade slot; comments sources to the comments slot.',
        })
      } else if (outputSlots.has(slot)) {
        issues.push({ field: `output.${slot}`, message: 'Each output slot accepts at most one inbound edge.' })
      } else {
        outputSlots.add(slot)
      }
    }
    if (tgt.type === 'grader') {
      if (e.targetHandle === 'context' && src.type !== 'assignmentContext') {
        issues.push({ field: `node:${tgt.id}`, message: 'Context input must come from an assignment context node.' })
      }
      if (e.targetHandle === 'submission' && src.type !== 'submission') {
        issues.push({ field: `node:${tgt.id}`, message: 'Submission input must come from a submission node.' })
      }
    }
    const list = adj.get(e.source) ?? []
    list.push(e.target)
    adj.set(e.source, list)
  }

  if (!outputSlots.has('grade')) {
    issues.push({ field: 'output.grade', message: 'Connect the grade slot before running.' })
  }

  if (hasCycle(adj, graph.nodes.map((n) => n.id))) {
    issues.push({ field: 'workflowGraph.edges', message: 'Workflow graph must be acyclic.' })
  }

  for (const n of graph.nodes) {
    if (n.type === 'grader' && graderPrompt(n.data) === '') {
      issues.push({ field: `node:${n.id}.prompt`, message: 'Grader node prompt is required.' })
    }
  }

  return issues
}

export function isWorkflowRunnable(graph: GraderWorkflowGraph | null | undefined): boolean {
  return validateWorkflowGraph(graph).length === 0
}

export function connectionIsValid(
  graph: GraderWorkflowGraph,
  source: string,
  sourceHandle: string | null | undefined,
  target: string,
  targetHandle: string | null | undefined,
): boolean {
  const nodeById = new Map(graph.nodes.map((n) => [n.id, n]))
  const src = nodeById.get(source)
  const tgt = nodeById.get(target)
  if (!src || !tgt) return false
  const sh = sourceHandle ?? ''
  const th = targetHandle ?? ''
  if (tgt.type === 'output') {
    if (sh !== 'grade' && sh !== 'comments') return false
    return sh === th && !graph.edges.some((e) => e.target === target && e.targetHandle === th)
  }
  if (tgt.type === 'grader') {
    if (th === 'context') return src.type === 'assignmentContext'
    if (th === 'submission') return src.type === 'submission'
  }
  return false
}
