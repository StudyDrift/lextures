import { normalizeWorkflowGraph } from './default-graph'
import { aggregatorInputSourceIsValid, detectRubricMergeCriterionConflicts } from './aggregator-validation'
import { flagSinkSourceIsValid, graphHasFlagSink } from './flag-sink-validation'
import { gateInputSourceIsValid } from './gate-validation'
import { outputSlotSourceIsValid } from './workflow-output-slot'
import { validateRouterIssues, routerInputSourceIsValid } from './router-validation'
import { workflowPromptIsPresent } from './workflow-prompt'
import { criterionGraderRubric } from './criterion-grader-rubric'
import type { RubricDefinition } from '../../../lib/courses-api'
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
  HANDLE_REASON,
  HANDLE_REPORT,
  HANDLE_SCORE,
  HANDLE_FLAG,
  HANDLE_REFERENCE,
  WORKFLOW_VERSION,
  isFlagForReviewNodeType,
  isHumanReviewGateNodeType,
  isOriginalityNodeType,
  isReferenceNodeType,
  isRubricNodeType,
  referenceHasSource,
  rubricHasSource,
  codeTestRunnerHasConfig,
  isActivityNodeType,
  isAiNodeType,
  isCodeTestRunnerNodeType,
  isConditionalRouterNodeType,
  isCriterionGraderNodeType,
  isScoreAggregatorNodeType,
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
  if (isOriginalityNodeType(sourceType) && (sourceHandle === HANDLE_SCORE || sourceHandle === HANDLE_REPORT)) {
    return true
  }
  if (isReferenceNodeType(sourceType) && sourceHandle === HANDLE_REFERENCE) {
    return true
  }
  if (isRubricNodeType(sourceType) && sourceHandle === HANDLE_RUBRIC) {
    return true
  }
  return false
}

function rubricOutputSourceIsValid(sourceType: string, sourceHandle: string): boolean {
  return isRubricNodeType(sourceType) && sourceHandle === HANDLE_RUBRIC
}

function referenceContentSourceIsValid(sourceType: string, sourceHandle: string): boolean {
  return isReferenceNodeType(sourceType) && sourceHandle === HANDLE_REFERENCE
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

export type ValidateWorkflowGraphOptions = {
  rubric?: RubricDefinition | null
  assignmentItemId?: string
  /** itemId → has rubric; used to validate library-mode Rubric nodes. */
  libraryRubrics?: Record<string, boolean>
}

/** Client-side validator mirroring server workflow rules. */
export function validateWorkflowGraph(
  graph: GraderWorkflowGraph | null | undefined,
  options: ValidateWorkflowGraphOptions = {},
): WorkflowValidationIssue[] {
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
          message:
            'Grade slot accepts Grader, Criterion Grader, AI, Code Test Runner, Human Review Gate, or Conditional Router branch outputs; comments slot accepts Grader or Criterion Grader comments or test reports.',
        })
      } else if (outputSlots.has(slot) && slot === HANDLE_COMMENTS) {
        issues.push({ field: `output.${slot}`, message: 'Each output slot accepts at most one inbound edge.' })
      } else {
        outputSlots.add(slot)
      }
    }
    if (tgt.type === 'grader' || isCriterionGraderNodeType(tgt.type)) {
      const th = e.targetHandle ?? ''
      if (th === HANDLE_CONTENT) {
        if (
          !(isActivityNodeType(src.type) && e.sourceHandle === HANDLE_CONTENT) &&
          !referenceContentSourceIsValid(src.type, e.sourceHandle ?? '')
        ) {
          issues.push({
            field: `node:${tgt.id}`,
            message: 'Content input must come from an Activity content output or Reference Material.',
          })
        }
      } else if (th === HANDLE_RUBRIC) {
        const sh = e.sourceHandle ?? ''
        if (
          !(isActivityNodeType(src.type) && sh === HANDLE_RUBRIC) &&
          !rubricOutputSourceIsValid(src.type, sh)
        ) {
          issues.push({
            field: `node:${tgt.id}`,
            message: 'Rubric input must come from an Activity or Rubric rubric output.',
          })
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
          message: 'AI input must come from a submission, activity, reference, or upstream AI output.',
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
    if (isFlagForReviewNodeType(tgt.type)) {
      const th = e.targetHandle ?? ''
      if (
        th !== HANDLE_REASON &&
        th !== HANDLE_COMMENTS &&
        th !== HANDLE_REPORT &&
        th !== HANDLE_GRADE &&
        th !== HANDLE_FLAG
      ) {
        issues.push({
          field: `node:${tgt.id}`,
          message: 'Flag for Review accepts reason, comments, report, grade, or flag inputs only.',
        })
      } else if (!flagSinkSourceIsValid(src.type, e.sourceHandle ?? '', th)) {
        issues.push({
          field: `node:${tgt.id}.${th}`,
          message: 'Invalid source for this Flag for Review input slot.',
        })
      }
    }
    if (isOriginalityNodeType(tgt.type)) {
      const th = e.targetHandle ?? ''
      if (th !== HANDLE_SUBMISSION) {
        issues.push({
          field: `node:${tgt.id}`,
          message: 'Originality Check accepts a submission input only.',
        })
      } else if (!isStudentSubmissionNodeType(src.type)) {
        issues.push({
          field: `node:${tgt.id}`,
          message: 'Submission input must come from a Student Submission node.',
        })
      }
    }
    if (isHumanReviewGateNodeType(tgt.type)) {
      const th = e.targetHandle ?? ''
      if (th !== HANDLE_COMMENTS && th !== HANDLE_REPORT && th !== HANDLE_GRADE && th !== HANDLE_FLAG) {
        issues.push({
          field: `node:${tgt.id}`,
          message: 'Human Review Gate accepts grade (required), comments, report, or flag inputs only.',
        })
      } else if (!gateInputSourceIsValid(src.type, e.sourceHandle ?? '', th)) {
        issues.push({
          field: `node:${tgt.id}.${th}`,
          message: 'Invalid source for this Human Review Gate input slot.',
        })
      }
    }
    if (isScoreAggregatorNodeType(tgt.type)) {
      const th = e.targetHandle ?? ''
      if (th !== HANDLE_GRADE) {
        issues.push({
          field: `node:${tgt.id}`,
          message: 'Score Aggregator accepts grade inputs only.',
        })
      } else if (!aggregatorInputSourceIsValid(src.type, e.sourceHandle ?? '')) {
        issues.push({
          field: `node:${tgt.id}.${th}`,
          message: 'Invalid grade source for Score Aggregator.',
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

  if (!outputSlots.has(HANDLE_GRADE) && !graphHasFlagSink(nodes)) {
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
    if (isOriginalityNodeType(n.type)) {
      const hasSubmissionInput = edges.some(
        (edge) => edge.target === n.id && (edge.targetHandle ?? '') === HANDLE_SUBMISSION,
      )
      if (!hasSubmissionInput) {
        issues.push({
          field: `node:${n.id}.submission`,
          message: 'Connect a submission input to the Originality Check.',
        })
      }
    }
    if (isHumanReviewGateNodeType(n.type)) {
      const hasGradeInput = edges.some(
        (edge) => edge.target === n.id && (edge.targetHandle ?? '') === HANDLE_GRADE,
      )
      if (!hasGradeInput) {
        issues.push({
          field: `node:${n.id}.grade`,
          message: 'Connect a grade input to the Human Review Gate.',
        })
      }
    }
    if (isReferenceNodeType(n.type) && !referenceHasSource(n.data)) {
      issues.push({
        field: `node:${n.id}.text`,
        message: 'Add reference text or select a course file.',
      })
    }
    if (isRubricNodeType(n.type)) {
      const rubricOpts = {
        assignmentHasRubric: Boolean(options.rubric?.criteria?.length),
        libraryRubrics: options.libraryRubrics,
      }
      if (!rubricHasSource(n.data, rubricOpts)) {
        const source = typeof n.data.source === 'string' ? n.data.source : 'assignment'
        if (source === 'library') {
          issues.push({
            field: `node:${n.id}.rubricAssignmentItemId`,
            message: 'Select an assignment that has a rubric.',
          })
        } else if (source === 'inline') {
          issues.push({
            field: `node:${n.id}.rubric`,
            message: 'Add at least one rubric criterion.',
          })
        } else {
          issues.push({
            field: `node:${n.id}.source`,
            message: 'This assignment has no rubric.',
          })
        }
      }
    }
    if (isScoreAggregatorNodeType(n.type)) {
      const hasGradeInput = edges.some(
        (edge) => edge.target === n.id && (edge.targetHandle ?? '') === HANDLE_GRADE,
      )
      if (!hasGradeInput) {
        issues.push({
          field: `node:${n.id}.grade`,
          message: 'Connect at least one grade input to the Score Aggregator.',
        })
      }
      const mode = typeof n.data.mode === 'string' ? n.data.mode : 'sum'
      if (mode === 'rubricMerge') {
        const criterionIds = edges
          .filter((edge) => edge.target === n.id && (edge.targetHandle ?? '') === HANDLE_GRADE)
          .map((edge) => nodeById.get(edge.source))
          .filter((src) => src && isCriterionGraderNodeType(src.type))
          .map((src) => (typeof src!.data.criterionId === 'string' ? src!.data.criterionId.trim() : ''))
        if (detectRubricMergeCriterionConflicts(criterionIds).length > 0) {
          issues.push({
            field: `node:${n.id}.mode`,
            message: 'rubricMerge: each criterion may be scored only once across inputs.',
          })
        }
      }
      if (mode === 'weightedSum') {
        const weights = n.data.weights
        if (weights && typeof weights === 'object') {
          let sum = 0
          for (const value of Object.values(weights as Record<string, unknown>)) {
            if (typeof value === 'number' && Number.isFinite(value)) sum += value
          }
          if (sum > 0 && Math.abs(sum - 1) > 0.01) {
            issues.push({
              field: `node:${n.id}.weights`,
              message: 'Weighted sum weights do not total 1.0 — consider normalizing.',
            })
          }
        }
      }
    }
    if (isCriterionGraderNodeType(n.type)) {
      if (!workflowPromptIsPresent(n.data)) {
        issues.push({ field: `node:${n.id}.prompt`, message: 'Criterion Grader prompt is required.' })
      }
      const criterionId = typeof n.data.criterionId === 'string' ? n.data.criterionId.trim() : ''
      if (!criterionId) {
        issues.push({ field: `node:${n.id}.criterionId`, message: 'Select a rubric criterion.' })
      } else {
        const rubric = criterionGraderRubric(graph, n.id, options.rubric, options.assignmentItemId ?? '')
        if (rubric?.criteria?.length) {
          const known = rubric.criteria.some((criterion) => criterion.id === criterionId)
          if (!known) {
            issues.push({
              field: `node:${n.id}.criterionId`,
              message: 'Selected criterion is not in the wired rubric.',
            })
          }
        }
      }
    }
  }

  return issues
}

export function isWorkflowRunnable(
  graph: GraderWorkflowGraph | null | undefined,
  options: ValidateWorkflowGraphOptions = {},
): boolean {
  return validateWorkflowGraph(graph, options).length === 0
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
  if (tgt.type === 'grader' || isCriterionGraderNodeType(tgt.type)) {
    if (th === HANDLE_CONTENT) {
      return (isActivityNodeType(src.type) && sh === HANDLE_CONTENT) || referenceContentSourceIsValid(src.type, sh)
    }
    if (th === HANDLE_RUBRIC) {
      return (isActivityNodeType(src.type) && sh === HANDLE_RUBRIC) || rubricOutputSourceIsValid(src.type, sh)
    }
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
  if (isFlagForReviewNodeType(tgt.type)) {
    if (
      th !== HANDLE_REASON &&
      th !== HANDLE_COMMENTS &&
      th !== HANDLE_REPORT &&
      th !== HANDLE_GRADE &&
      th !== HANDLE_FLAG
    ) {
      return false
    }
    return flagSinkSourceIsValid(src.type, sh, th)
  }
  if (isOriginalityNodeType(tgt.type)) {
    return th === HANDLE_SUBMISSION && isStudentSubmissionNodeType(src.type)
  }
  if (isHumanReviewGateNodeType(tgt.type)) {
    if (th !== HANDLE_COMMENTS && th !== HANDLE_REPORT && th !== HANDLE_GRADE && th !== HANDLE_FLAG) {
      return false
    }
    return gateInputSourceIsValid(src.type, sh, th)
  }
  if (isScoreAggregatorNodeType(tgt.type)) {
    if (th !== HANDLE_GRADE) return false
    return aggregatorInputSourceIsValid(src.type, sh)
  }
  return false
}
