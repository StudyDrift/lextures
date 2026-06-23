import { normalizeWorkflowGraph } from './default-graph'
import { outputSlotSourceIsValid } from './workflow-output-slot'
import { validateRouterIssues, routerInputSourceIsValid } from './router-validation'
import { workflowPromptIsPresent } from './workflow-prompt'
import type { GraderWorkflowGraph, WorkflowValidationIssue } from './types'
import {
  HANDLE_AI_INPUT,
  HANDLE_AI_OUTPUT,
  HANDLE_COMMENTS,
  HANDLE_CONTENT,
  HANDLE_CONTEXT,
  HANDLE_GRADE,
  HANDLE_RUBRIC,
  HANDLE_SUBMISSION,
  HANDLE_THEN,
  HANDLE_ELSE,
  WORKFLOW_VERSION,
  codeTestRunnerHasConfig,
  isActivityNodeType,
  isAiNodeType,
  isCodeTestRunnerNodeType,
  isConditionalRouterNodeType,
  isStudentSubmissionNodeType,
} from './types'

const MAX_NODES = 50
const MAX_EDGES = 100

function aiInputSourceIsValid(
  sourceType: string,
  sourceHandle: string,
): boolean {
  if (isStudentSubmissionNodeType(sourceType) && sourceHandle === HANDLE_SUBMISSION) return true
  if (isActivityNodeType(sourceType) && (sourceHandle === HANDLE_CONTENT || sourceHandle === HANDLE_RUBRIC)) {
    return true
  }
  if (isAiNodeType(sourceType) && sourceHandle === HANDLE_AI_OUTPUT) return true
  if (isConditionalRouterNodeType(sourceType) && (sourceHandle === HANDLE_THEN || sourceHandle === HANDLE_ELSE)) {
    return true
  }
  return false
}

function aiInputEdgeExists(
  edges: GraderWorkflowGraph['edges'],
  target: string,
  source: string,
  sourceHandle: string,
): boolean {
  return edges.some(
    (edge) =>
      edge.target === target &&
      edge.targetHandle === HANDLE_AI_INPUT &&
      edge.source === source &&
      (edge.sourceHandle ?? '') === sourceHandle,
  )
}

function aiInputHasUpstreamType(
  edges: GraderWorkflowGraph['edges'],
  nodeById: Map<string, GraderWorkflowGraph['nodes'][number]>,
  target: string,
  matches: (type: string) => boolean,
): boolean {
  return edges.some((edge) => {
    if (edge.target !== target || edge.targetHandle !== HANDLE_AI_INPUT) return false
    const upstream = nodeById.get(edge.source)
    return Boolean(upstream && matches(upstream.type))
  })
}

function aiInputAllowsEdge(
  edges: GraderWorkflowGraph['edges'],
  nodeById: Map<string, GraderWorkflowGraph['nodes'][number]>,
  source: string,
  sourceHandle: string,
  target: string,
): boolean {
  if (aiInputEdgeExists(edges, target, source, sourceHandle)) return false
  const src = nodeById.get(source)
  if (!src || !aiInputSourceIsValid(src.type, sourceHandle)) return false
  if (isAiNodeType(src.type) && aiInputHasUpstreamType(edges, nodeById, target, isAiNodeType)) return false
  if (
    isStudentSubmissionNodeType(src.type) &&
    aiInputHasUpstreamType(edges, nodeById, target, isStudentSubmissionNodeType)
  ) {
    return false
  }
  return true
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
  const { version, nodes, edges } = normalizeWorkflowGraph(graph)
  if (version !== WORKFLOW_VERSION) {
    issues.push({ field: 'workflowGraph.version', message: 'Unsupported workflow graph version.' })
    return issues
  }
  if (nodes.length > MAX_NODES) {
    issues.push({ field: 'workflowGraph.nodes', message: `Graph exceeds ${MAX_NODES} node limit.` })
  }
  if (edges.length > MAX_EDGES) {
    issues.push({ field: 'workflowGraph.edges', message: `Graph exceeds ${MAX_EDGES} edge limit.` })
  }

  const nodeById = new Map(nodes.map((n) => [n.id, n]))
  let outputCount = 0
  for (const n of nodes) {
    if (n.type === 'output') outputCount++
  }
  if (outputCount !== 1) {
    issues.push({ field: 'workflowGraph.nodes', message: 'Graph must contain exactly one output node.' })
  }

  const outputSlots = new Set<string>()
  const adj = new Map<string, string[]>()
  for (const e of edges) {
    const src = nodeById.get(e.source)
    const tgt = nodeById.get(e.target)
    if (!src || !tgt) {
      issues.push({ field: 'workflowGraph.edges', message: 'Edge references unknown node.' })
      continue
    }
    if (tgt.type === 'output') {
      const slot = e.targetHandle ?? ''
      if (slot !== HANDLE_GRADE && slot !== HANDLE_COMMENTS) {
        issues.push({ field: 'output', message: 'Output node edges must target grade or comments slots.' })
      } else if (!outputSlotSourceIsValid(src.type, e.sourceHandle ?? '', slot)) {
        issues.push({
          field: `output.${slot}`,
          message: 'Grade slot accepts Grader, AI, Code Test Runner, or Conditional Router branch outputs; comments slot accepts Grader comments or test reports.',
        })
      } else if (outputSlots.has(slot) && slot === HANDLE_COMMENTS) {
        issues.push({ field: `output.${slot}`, message: 'Each output slot accepts at most one inbound edge.' })
      } else {
        outputSlots.add(slot)
      }
    }
    if (tgt.type === 'grader') {
      const th = e.targetHandle ?? ''
      if (th === HANDLE_CONTENT) {
        if (!isActivityNodeType(src.type) || e.sourceHandle !== HANDLE_CONTENT) {
          issues.push({ field: `node:${tgt.id}`, message: 'Content input must come from an Activity content output.' })
        }
      } else if (th === HANDLE_RUBRIC) {
        if (!isActivityNodeType(src.type) || e.sourceHandle !== HANDLE_RUBRIC) {
          issues.push({ field: `node:${tgt.id}`, message: 'Rubric input must come from an Activity rubric output.' })
        }
      } else if (th === HANDLE_SUBMISSION) {
        if (!isStudentSubmissionNodeType(src.type)) {
          issues.push({ field: `node:${tgt.id}`, message: 'Submission input must come from a Student Submission node.' })
        }
      } else if (th === HANDLE_CONTEXT) {
        if (!isActivityNodeType(src.type)) {
          issues.push({ field: `node:${tgt.id}`, message: 'Context input must come from an Activity node.' })
        }
      }
    }
    if (isAiNodeType(tgt.type)) {
      const th = e.targetHandle ?? ''
      if (th !== HANDLE_AI_INPUT) {
        issues.push({ field: `node:${tgt.id}`, message: 'AI node edges must target the input slot.' })
      } else if (!aiInputSourceIsValid(src.type, e.sourceHandle ?? '')) {
        issues.push({
          field: `node:${tgt.id}`,
          message: 'AI input must come from a submission, activity, or upstream AI output.',
        })
      }
    }
    if (isCodeTestRunnerNodeType(tgt.type)) {
      const th = e.targetHandle ?? ''
      if (th === HANDLE_SUBMISSION) {
        if (!isStudentSubmissionNodeType(src.type)) {
          issues.push({ field: `node:${tgt.id}`, message: 'Submission input must come from a Student Submission node.' })
        }
      } else {
        issues.push({ field: `node:${tgt.id}`, message: 'Code Test Runner accepts a submission input only.' })
      }
    }
    if (isConditionalRouterNodeType(tgt.type)) {
      const th = e.targetHandle ?? ''
      if (th !== HANDLE_AI_INPUT) {
        issues.push({ field: `node:${tgt.id}`, message: 'Conditional Router edges must target the input slot.' })
      } else if (!routerInputSourceIsValid(src.type, e.sourceHandle ?? '')) {
        issues.push({
          field: `node:${tgt.id}`,
          message: 'Router input must come from a submission, grade, or upstream branch output.',
        })
      }
    }
    if (isConditionalRouterNodeType(src.type) && (e.sourceHandle ?? '') !== HANDLE_THEN && (e.sourceHandle ?? '') !== HANDLE_ELSE) {
      issues.push({ field: `node:${src.id}`, message: 'Conditional Router edges must originate from then or else outputs.' })
    }
    if (isAiNodeType(src.type) && (e.sourceHandle ?? '') !== HANDLE_AI_OUTPUT) {
      issues.push({ field: `node:${src.id}`, message: 'AI node edges must originate from the output slot.' })
    }
    const list = adj.get(e.source) ?? []
    list.push(e.target)
    adj.set(e.source, list)
  }

  if (!outputSlots.has(HANDLE_GRADE)) {
    issues.push({ field: 'output.grade', message: 'Connect the grade slot before running.' })
  }

  issues.push(...validateRouterIssues(graph, nodeById))

  if (hasCycle(adj, nodes.map((n) => n.id))) {
    issues.push({ field: 'workflowGraph.edges', message: 'Workflow graph must be acyclic.' })
  }

  for (const n of nodes) {
    if (n.type === 'grader' && !workflowPromptIsPresent(n.data)) {
      issues.push({ field: `node:${n.id}.prompt`, message: 'Grader node prompt is required.' })
    }
    if (isAiNodeType(n.type) && !workflowPromptIsPresent(n.data)) {
      issues.push({ field: `node:${n.id}.prompt`, message: 'AI node prompt is required.' })
    }
    if (isCodeTestRunnerNodeType(n.type) && !codeTestRunnerHasConfig(n.data)) {
      issues.push({ field: `node:${n.id}.testCases`, message: 'Add at least one test case or select a test suite.' })
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
  const { nodes, edges } = normalizeWorkflowGraph(graph)
  const nodeById = new Map(nodes.map((n) => [n.id, n]))
  const src = nodeById.get(source)
  const tgt = nodeById.get(target)
  if (!src || !tgt) return false
  const sh = sourceHandle ?? ''
  const th = targetHandle ?? ''
  if (tgt.type === 'output') {
    if (th !== HANDLE_GRADE && th !== HANDLE_COMMENTS) return false
    if (!outputSlotSourceIsValid(src.type, sh, th)) return false
    return !edges.some((e) => e.target === target && e.targetHandle === th)
  }
  if (tgt.type === 'grader') {
    if (th === HANDLE_CONTENT) return isActivityNodeType(src.type) && sh === HANDLE_CONTENT
    if (th === HANDLE_RUBRIC) return isActivityNodeType(src.type) && sh === HANDLE_RUBRIC
    if (th === HANDLE_SUBMISSION) return isStudentSubmissionNodeType(src.type)
    if (th === HANDLE_CONTEXT) return isActivityNodeType(src.type)
  }
  if (isAiNodeType(tgt.type)) {
    if (th !== HANDLE_AI_INPUT) return false
    return aiInputAllowsEdge(edges, nodeById, source, sh, target)
  }
  if (isCodeTestRunnerNodeType(tgt.type)) {
    return th === HANDLE_SUBMISSION && isStudentSubmissionNodeType(src.type)
  }
  if (isConditionalRouterNodeType(tgt.type)) {
    if (th !== HANDLE_AI_INPUT) return false
    return routerInputSourceIsValid(src.type, sh)
  }
  return false
}
