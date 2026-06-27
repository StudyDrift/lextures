import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { usePlatformFeatures } from '../../../context/platform-features-context'
import {
  fetchCourseCanvasLink,
  fetchGraderAgentConfig,
  fetchGraderAgentRun,
  fetchGraderAgentRunEstimate,
  fetchSubmissionGrade,
  postGraderAgentCancelRun,
  postGraderAgentAIBuild,
  postGraderAgentRun,
  postGraderAgentTemplate,
  putGraderAgentConfig,
  putGraderAgentTemplate,
  putSubmissionGrade,
  streamGraderAgentDryRun,
  type CourseCanvasLinkApi,
  type GraderAgentConfigApi,
  type GraderAgentDryRunResult,
  type GraderAgentRunCostEstimate,
  type GraderAgentRunMode,
  type GraderAgentRunStatus,
  type GraderWorkflowGraphApi,
  type RubricDefinition,
} from '../../../lib/courses-api'
import { queueCanvasGradeSync } from '../../canvas/canvas-grade-sync'
import { buildAgentGradeApplyPayload, canvasPayloadFromSubmissionGrade } from './agent-grade-apply'
import { effectiveWorkflowGraph, normalizeWorkflowGraph, synthesizeDefaultGraph } from './default-graph'
import {
  createGroupFromSelection,
  groupNodeData,
  ungroupNode,
  type GraderGroupNodeData,
} from './group-utils'
import {
  defaultRunAgentFilterState,
  runFilterFromState,
  type RunAgentFilterState,
} from './run-agent-filter-picker'
import { isWorkflowRunnable, validateWorkflowGraph } from './validation'
import type {
  GradingAgentItemKind,
  GraderWorkflowGraph,
  PaletteNodeType,
  WorkflowValidationIssue,
} from './types'
import type { QuizQuestionSlot } from './quiz-question-slots'
import { useRubricLibraryRubrics } from './use-rubric-library-rubrics'
import { newWorkflowNodeId } from './workflow-node-id'
import { patchWorkflowNodeLabel } from './workflow-node-label'
import { paletteNodeDefaults } from './node-descriptors'

export type RunScope = 'current' | 'ungraded' | 'all'

/** One level of drill-in navigation into a group's subgraph. */
export type GroupNavEntry = {
  parentGraph: GraderWorkflowGraph
  groupId: string
  label: string
}

/** Writes an edited subgraph back into its group node, pruning ports (and parent
 *  edges) that reference internal nodes which no longer exist. */
function writeSubgraphBack(
  parent: GraderWorkflowGraph,
  groupId: string,
  subgraph: GraderWorkflowGraph,
): GraderWorkflowGraph {
  const internalIds = new Set(subgraph.nodes.map((n) => n.id))
  let prunedPortIds = new Set<string>()
  const nodes = parent.nodes.map((n) => {
    if (n.id !== groupId || n.type !== 'group') return n
    const gd = groupNodeData(n)
    const inputs = gd.inputs.filter((p) => internalIds.has(p.nodeId))
    const outputs = gd.outputs.filter((p) => internalIds.has(p.nodeId))
    prunedPortIds = new Set(
      [...gd.inputs, ...gd.outputs]
        .filter((p) => !internalIds.has(p.nodeId))
        .map((p) => p.id),
    )
    const data: GraderGroupNodeData = { ...gd, subgraph, inputs, outputs }
    return { ...n, data: data as unknown as Record<string, unknown> }
  })
  const edges =
    prunedPortIds.size === 0
      ? parent.edges
      : parent.edges.filter((e) => {
          if (e.source === groupId && e.sourceHandle && prunedPortIds.has(e.sourceHandle)) return false
          if (e.target === groupId && e.targetHandle && prunedPortIds.has(e.targetHandle)) return false
          return true
        })
  return { ...parent, nodes, edges }
}

/** Folds the active (deepest) graph back through the nav stack to the root graph. */
function foldNavToRoot(active: GraderWorkflowGraph, navStack: GroupNavEntry[]): GraderWorkflowGraph {
  let current = active
  for (let i = navStack.length - 1; i >= 0; i--) {
    current = writeSubgraphBack(navStack[i].parentGraph, navStack[i].groupId, current)
  }
  return current
}

function graderAgentConfigPutPayload(
  config: GraderAgentConfigApi | null | undefined,
  graph: GraderWorkflowGraphApi,
  overrides: {
    status?: GraderAgentConfigApi['status']
    autoGradeNew?: boolean
    postPolicy?: 'draft' | 'auto_post'
    confidenceFloor?: number | null
  } = {},
) {
  const floor = overrides.confidenceFloor !== undefined ? overrides.confidenceFloor : config?.confidenceFloor
  return {
    prompt: config?.prompt ?? '',
    includeAssignmentContent: config?.includeAssignmentContent ?? false,
    includeRubric: config?.includeRubric ?? false,
    status: overrides.status ?? config?.status ?? 'draft',
    autoGradeNew: overrides.autoGradeNew ?? config?.autoGradeNew ?? false,
    postPolicy: overrides.postPolicy ?? config?.postPolicy ?? 'draft',
    confidenceFloor: typeof floor === 'number' && floor > 0 ? floor : null,
    workflowGraph: graph,
  }
}

export type NodeExecutionStatus = 'idle' | 'running' | 'success' | 'error' | 'skipped'

export type DryRunLogEntry = {
  message: string
  level: 'info' | 'warn' | 'error'
}

export type NodeDryRunDetail = {
  compiledPrompt?: string
  compiledSystemPrompt?: string
  compiledInput?: string
  compiledOutput?: string
}

export type GraderAgentWorkflowSeed = {
  prompt: string
  includeAssignmentContent: boolean
  includeRubric: boolean
  workflowGraph?: GraderWorkflowGraphApi
}

export type GraderAgentTemplateMode = {
  name: string
  templateId?: string | null
}

type UseGraderAgentWorkflowArgs = {
  open: boolean
  courseCode: string
  itemId: string
  itemKind?: GradingAgentItemKind
  quizQuestionSlots?: QuizQuestionSlot[]
  submissionId: string | null
  rubric?: RubricDefinition | null
  seedWorkflow?: GraderAgentWorkflowSeed | null
  templateMode?: GraderAgentTemplateMode | null
  onApplied?: () => void
  onSubmissionGraded?: (submissionId: string) => void
}

export function useGraderAgentWorkflow({
  open,
  courseCode,
  itemId,
  itemKind = 'assignment',
  quizQuestionSlots = [],
  submissionId,
  rubric = null,
  seedWorkflow = null,
  templateMode = null,
  onApplied,
  onSubmissionGraded,
}: UseGraderAgentWorkflowArgs) {
  const { t } = useTranslation('common')
  const { graderAgentSuggestModeEnabled, graderAgentRunFiltersEnabled, graderAgentCostEstimateEnabled, graderAgentCancelRunEnabled } =
    usePlatformFeatures()
  const [savedTemplateId, setSavedTemplateId] = useState<string | null>(templateMode?.templateId ?? null)
  const [config, setConfig] = useState<GraderAgentConfigApi | null>(null)
  const [graph, setGraph] = useState<GraderWorkflowGraph>(() => synthesizeDefaultGraph('', false, false))
  const graphRef = useRef(graph)
  graphRef.current = graph
  // Drill-in navigation: each entry captures the parent graph + the group entered.
  // `graph` is always the currently-edited level; the root is folded back for
  // validation and persistence so groups are transparent to the rest of the engine.
  const [navStack, setNavStack] = useState<GroupNavEntry[]>([])
  const [selectedNodeId, setSelectedNodeId] = useState<string | null>(null)
  const [dryRunning, setDryRunning] = useState(false)
  const [batchRunning, setBatchRunning] = useState(false)
  const [cancellingRun, setCancellingRun] = useState(false)
  const [dryRunError, setDryRunError] = useState<string | null>(null)
  const [dryRunResult, setDryRunResult] = useState<GraderAgentDryRunResult | null>(null)
  const [hadDryRun, setHadDryRun] = useState(false)
  const [saving, setSaving] = useState(false)
  const [aiBuilding, setAiBuilding] = useState(false)
  const [applyingSubmissionId, setApplyingSubmissionId] = useState<string | null>(null)
  const [syncingSubmissionIds, setSyncingSubmissionIds] = useState<ReadonlySet<string>>(() => new Set())
  const [runScope, setRunScope] = useState<RunScope>('ungraded')
  const [runMode, setRunMode] = useState<GraderAgentRunMode>(
    graderAgentSuggestModeEnabled ? 'suggest' : 'apply',
  )
  const [lastRunMode, setLastRunMode] = useState<GraderAgentRunMode>('apply')
  const [runId, setRunId] = useState<string | null>(null)
  const [runProgress, setRunProgress] = useState<{ completed: number; failed: number; total: number } | null>(null)
  const [runResults, setRunResults] = useState<GraderAgentRunStatus['results']>([])
  const [confirmOverwrite, setConfirmOverwrite] = useState(false)
  const [runFilterState, setRunFilterState] = useState<RunAgentFilterState>(defaultRunAgentFilterState)
  const [runCostEstimate, setRunCostEstimate] = useState<GraderAgentRunCostEstimate | null>(null)
  const [runCostEstimateLoading, setRunCostEstimateLoading] = useState(false)
  const [budgetUsd, setBudgetUsd] = useState<number | null>(null)
  const [statusMessage, setStatusMessage] = useState('')
  const [nodeExecutionStates, setNodeExecutionStates] = useState<Record<string, NodeExecutionStatus>>({})
  const [dryRunLogs, setDryRunLogs] = useState<DryRunLogEntry[]>([])
  const [dryRunConsoleOpen, setDryRunConsoleOpen] = useState(false)
  const [nodeDryRunDetails, setNodeDryRunDetails] = useState<Record<string, NodeDryRunDetail>>({})
  const [canvasLink, setCanvasLink] = useState<CourseCanvasLinkApi | null>(null)
  const canvasSyncAbortRef = useRef<(() => void) | null>(null)
  const canvasSyncedSubmissionIdsRef = useRef<Set<string>>(new Set())
  const lastRunProgressLogRef = useRef<string | null>(null)

  const { libraryRubrics, setLibraryRubricAvailability } = useRubricLibraryRubrics(graph)
  const validationOptions = useMemo(
    () => ({
      rubric,
      assignmentItemId: itemId,
      itemKind,
      quizQuestionSlots,
      libraryRubrics,
    }),
    [rubric, itemId, itemKind, quizQuestionSlots, libraryRubrics],
  )
  // The folded root graph is the source of truth for validation and persistence,
  // regardless of how deep the user has drilled into groups.
  const rootGraph = useMemo(
    () => (navStack.length === 0 ? graph : foldNavToRoot(graph, navStack)),
    [graph, navStack],
  )
  const validationIssues = useMemo(
    () => validateWorkflowGraph(rootGraph, validationOptions),
    [rootGraph, validationOptions],
  )
  const runnable = isWorkflowRunnable(rootGraph, validationOptions)

  useEffect(() => {
    if (!open) {
      setApplyingSubmissionId(null)
      setSyncingSubmissionIds(new Set())
      return
    }
    setSavedTemplateId(templateMode?.templateId ?? null)
  }, [open, templateMode?.templateId])

  useEffect(() => {
    if (!open) {
      setRunFilterState(defaultRunAgentFilterState)
    }
  }, [open])

  useEffect(() => {
    setRunMode(graderAgentSuggestModeEnabled ? 'suggest' : 'apply')
  }, [graderAgentSuggestModeEnabled])

  useEffect(() => {
    if (!open || !graderAgentCostEstimateEnabled) {
      setRunCostEstimate(null)
      setRunCostEstimateLoading(false)
      return
    }
    let cancelled = false
    setRunCostEstimateLoading(true)
    const runFilter = graderAgentRunFiltersEnabled ? runFilterFromState(runFilterState) : undefined
    void fetchGraderAgentRunEstimate(courseCode, itemId, {
      scope: runScope,
      submissionId: runScope === 'current' ? submissionId ?? undefined : undefined,
      overwrite: runScope === 'all' && confirmOverwrite,
      filter: runFilter,
    })
      .then((estimate) => {
        if (!cancelled) setRunCostEstimate(estimate)
      })
      .catch(() => {
        if (!cancelled) setRunCostEstimate(null)
      })
      .finally(() => {
        if (!cancelled) setRunCostEstimateLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [
    open,
    graderAgentCostEstimateEnabled,
    graderAgentRunFiltersEnabled,
    courseCode,
    itemId,
    runScope,
    submissionId,
    confirmOverwrite,
    runFilterState,
  ])

  const loadConfig = useCallback(async () => {
    if (templateMode) {
      setConfig(null)
      if (seedWorkflow) {
        const g = effectiveWorkflowGraph(
          seedWorkflow.workflowGraph as GraderWorkflowGraph | undefined,
          seedWorkflow.prompt,
          seedWorkflow.includeAssignmentContent,
          seedWorkflow.includeRubric,
          itemKind,
        )
        setGraph(g)
        setNavStack([])
        setSelectedNodeId(null)
      } else {
        const g = effectiveWorkflowGraph(null, '', false, false, itemKind)
        setGraph(g)
        setNavStack([])
        setSelectedNodeId(null)
      }
      return
    }
    const res = await fetchGraderAgentConfig(courseCode, itemId, itemKind)
    const c = res.config
    setConfig(c)
    if (c) {
      const g = effectiveWorkflowGraph(
        c.workflowGraph as GraderWorkflowGraph | undefined,
        c.prompt,
        c.includeAssignmentContent,
        c.includeRubric,
        itemKind,
      )
      setGraph(g)
      setSelectedNodeId(null)
      setHadDryRun(c.status === 'accepted')
    } else if (seedWorkflow) {
      const g = effectiveWorkflowGraph(
        seedWorkflow.workflowGraph as GraderWorkflowGraph | undefined,
        seedWorkflow.prompt,
        seedWorkflow.includeAssignmentContent,
        seedWorkflow.includeRubric,
        itemKind,
      )
      setGraph(g)
      setSelectedNodeId(null)
    } else {
      const g = effectiveWorkflowGraph(null, '', false, false, itemKind)
      setGraph(g)
      setSelectedNodeId(null)
    }
  }, [courseCode, itemId, itemKind, seedWorkflow, templateMode])

  useEffect(() => {
    if (!open) return
    let cancelled = false
    void (async () => {
      try {
        await loadConfig()
      } catch (e) {
        if (!cancelled) setDryRunError(e instanceof Error ? e.message : 'Could not load grading agent.')
      }
    })()
    return () => {
      cancelled = true
    }
  }, [open, loadConfig])

  useEffect(() => {
    if (!open) return
    let cancelled = false
    void (async () => {
      try {
        const link = await fetchCourseCanvasLink(courseCode)
        if (!cancelled) setCanvasLink(link)
      } catch {
        if (!cancelled) setCanvasLink({ linked: false, gradeSyncEnabled: false })
      }
    })()
    return () => {
      cancelled = true
    }
  }, [open, courseCode])

  useEffect(() => {
    canvasSyncAbortRef.current?.()
    canvasSyncAbortRef.current = null
  }, [submissionId])

  useEffect(() => {
    return () => {
      canvasSyncAbortRef.current?.()
    }
  }, [])

  const appendRunLog = useCallback((message: string, level: DryRunLogEntry['level'] = 'info') => {
    setDryRunLogs((prev) => [...prev, { message, level }])
  }, [])

  const setBatchNodeExecutionRunning = useCallback(() => {
    if (!graph) return
    const states: Record<string, NodeExecutionStatus> = {}
    for (const node of graph.nodes) {
      if (node.type !== 'output') {
        states[node.id] = 'running'
      }
    }
    setNodeExecutionStates(states)
  }, [graph])

  const resetBatchNodeExecution = useCallback(() => {
    setNodeExecutionStates({})
  }, [])

  const addSyncingSubmission = useCallback((id: string) => {
    setSyncingSubmissionIds((prev) => {
      if (prev.has(id)) return prev
      const next = new Set(prev)
      next.add(id)
      return next
    })
  }, [])

  const removeSyncingSubmission = useCallback((id: string) => {
    setSyncingSubmissionIds((prev) => {
      if (!prev.has(id)) return prev
      const next = new Set(prev)
      next.delete(id)
      return next
    })
  }, [])

  const finishAppliedSubmission = useCallback(
    (id: string) => {
      removeSyncingSubmission(id)
      setApplyingSubmissionId((current) => (current === id ? null : current))
      onSubmissionGraded?.(id)
    },
    [onSubmissionGraded, removeSyncingSubmission],
  )

  const syncAppliedResultToCanvas = useCallback(
    async (submissionId: string) => {
      if (!canvasLink?.linked || !canvasLink.gradeSyncEnabled) return
      if (canvasSyncedSubmissionIdsRef.current.has(submissionId)) return
      canvasSyncedSubmissionIdsRef.current.add(submissionId)
      addSyncingSubmission(submissionId)
      try {
        const grade = await fetchSubmissionGrade(courseCode, itemId, submissionId)
        const syncHandle = queueCanvasGradeSync({
          courseCode,
          itemId,
          submissionId,
          canvasLink,
          gradePayload: canvasPayloadFromSubmissionGrade(grade),
          onComplete: () => {
            finishAppliedSubmission(submissionId)
            onApplied?.()
          },
          onError: (message) => {
            finishAppliedSubmission(submissionId)
            appendRunLog(message, 'warn')
          },
        })
        if (syncHandle) {
          canvasSyncAbortRef.current = syncHandle.abort
        } else {
          finishAppliedSubmission(submissionId)
        }
      } catch (e) {
        finishAppliedSubmission(submissionId)
        appendRunLog(
          e instanceof Error ? e.message : 'Could not load grade for Canvas sync.',
          'warn',
        )
      }
    },
    [addSyncingSubmission, appendRunLog, canvasLink, courseCode, finishAppliedSubmission, itemId, onApplied],
  )

  const processRunStatus = useCallback(
    async (run: GraderAgentRunStatus) => {
      setRunProgress({
        completed: run.completedCount,
        failed: run.failedCount,
        total: run.totalCount,
      })
      setRunResults(run.results)

      const progressKey = `${run.completedCount}:${run.failedCount}:${run.totalCount}`
      if (progressKey !== lastRunProgressLogRef.current) {
        lastRunProgressLogRef.current = progressKey
        appendRunLog(
          t('gradingAgent.run.progress', {
            completed: run.completedCount,
            failed: run.failedCount,
            total: run.totalCount,
          }),
        )
      }

      for (const result of run.results) {
        if (result.status !== 'applied') continue
        void syncAppliedResultToCanvas(result.submissionId)
      }

      setStatusMessage(`${run.completedCount} / ${run.totalCount} complete`)

      if (run.status === 'done' || run.status === 'error' || run.status === 'failed' || run.status === 'cancelled' || run.status === 'budget_exceeded') {
        setBatchRunning(false)
        setCancellingRun(false)
        resetBatchNodeExecution()
        const appliedCount = run.results.filter((result) => result.status === 'applied').length
        const suggestedCount = run.results.filter((result) => result.status === 'suggested').length
        const costNote =
          run.costUsd != null && run.costUsd > 0
            ? t('gradingAgent.run.cost.actualSummary', { cost: run.costUsd.toFixed(4) })
            : null
        const skippedCount = run.results.filter((result) => result.status === 'skipped').length
        if (run.status === 'cancelled') {
          appendRunLog(
            t('gradingAgent.run.cancel.complete', {
              applied: appliedCount,
              suggested: suggestedCount,
              skipped: skippedCount,
            }),
          )
          setStatusMessage(
            costNote
              ? `${t('gradingAgent.run.cancel.complete', { applied: appliedCount, suggested: suggestedCount, skipped: skippedCount })} ${costNote}`
              : t('gradingAgent.run.cancel.complete', {
                  applied: appliedCount,
                  suggested: suggestedCount,
                  skipped: skippedCount,
                }),
          )
        } else if (run.status === 'error' || run.status === 'failed') {
          appendRunLog(t('gradingAgent.run.failed'), 'error')
          setStatusMessage(t('gradingAgent.run.failed'))
        } else if (run.status === 'budget_exceeded') {
          appendRunLog(t('gradingAgent.run.cost.budgetExceeded'), 'warn')
          setStatusMessage(
            costNote
              ? `${t('gradingAgent.run.cost.budgetExceeded')} ${costNote}`
              : t('gradingAgent.run.cost.budgetExceeded'),
          )
        } else if (lastRunMode === 'suggest') {
          appendRunLog(
            t('gradingAgent.run.completeSuggest', {
              suggested: suggestedCount,
              failed: run.failedCount,
            }),
          )
          setStatusMessage(
            costNote
              ? `${t('gradingAgent.run.completeSuggest', { suggested: suggestedCount, failed: run.failedCount })} ${costNote}`
              : t('gradingAgent.run.completeSuggest', { suggested: suggestedCount, failed: run.failedCount }),
          )
        } else {
          appendRunLog(
            t('gradingAgent.run.complete', {
              applied: appliedCount,
              failed: run.failedCount,
            }),
          )
          setStatusMessage(
            costNote
              ? `${t('gradingAgent.run.complete', { applied: appliedCount, failed: run.failedCount })} ${costNote}`
              : t('gradingAgent.run.complete', { applied: appliedCount, failed: run.failedCount }),
          )
        }
        return true
      }
      return false
    },
    [appendRunLog, lastRunMode, resetBatchNodeExecution, syncAppliedResultToCanvas, t],
  )

  useEffect(() => {
    if (!open || !runId) return
    let cancelled = false
    let finalized = false
    let timer: number | undefined

    const finalize = () => {
      if (finalized) return
      finalized = true
      if (timer !== undefined) window.clearInterval(timer)
      onApplied?.()
    }

    const poll = async () => {
      try {
        const run = await fetchGraderAgentRun(courseCode, itemId, runId)
        if (cancelled) return
        const finished = await processRunStatus(run)
        if (finished) finalize()
      } catch {
        if (!cancelled) {
          appendRunLog(t('gradingAgent.run.pollFailed'), 'warn')
        }
      }
    }

    void poll()
    timer = window.setInterval(() => {
      void poll()
    }, 1500)
    return () => {
      cancelled = true
      if (timer !== undefined) window.clearInterval(timer)
    }
  }, [open, runId, courseCode, itemId, processRunStatus, appendRunLog, t, onApplied])

  const updateGraph = useCallback((next: GraderWorkflowGraph) => {
    setGraph(next)
  }, [])

  const updateNodeData = useCallback(<T extends object>(nodeId: string, patch: Partial<T>) => {
    setGraph((prev) => {
      if (!prev) return prev
      return {
        ...prev,
        nodes: prev.nodes.map((n) =>
          n.id === nodeId ? { ...n, data: { ...n.data, ...patch } } : n,
        ),
      }
    })
  }, [])

  const addPaletteNode = useCallback(
    (type: PaletteNodeType, position?: { x: number; y: number }) => {
      if (!graph) return
      const descriptor = paletteNodeDefaults(type, {
        itemId,
        nodeCount: graph.nodes.length,
      })
      const id = newWorkflowNodeId(descriptor.idPrefix)
      setSelectedNodeId(id)
      setGraph((current) => {
        if (!current) return current
        return {
          ...current,
          nodes: [
            ...current.nodes,
            {
              id,
              type,
              position: position ?? descriptor.position,
              data: descriptor.data,
            },
          ],
        }
      })
    },
    [graph, itemId],
  )

  const handleAIBuild = useCallback(
    async (instruction: string): Promise<boolean> => {
      const trimmed = instruction.trim()
      if (!trimmed || aiBuilding) return false
      setAiBuilding(true)
      setDryRunError(null)
      setStatusMessage(t('gradingAgent.aiBuilder.generating'))
      try {
        const quizSlots =
          itemKind === 'quiz'
            ? quizQuestionSlots.map((s) => ({
                index: s.index,
                label: s.label,
                questionType: s.questionType,
                maxPoints: s.maxPoints,
              }))
            : undefined
        const result = await postGraderAgentAIBuild(
          courseCode,
          itemId,
          {
            instruction: trimmed,
            currentGraph: rootGraph as GraderWorkflowGraphApi,
            quizSlots,
          },
          itemKind,
        )
        setGraph(normalizeWorkflowGraph(result.workflowGraph as GraderWorkflowGraph))
        setNavStack([])
        setSelectedNodeId(null)
        setDryRunResult(null)
        setHadDryRun(false)
        setStatusMessage(result.summary || t('gradingAgent.aiBuilder.success'))
        return true
      } catch (e) {
        setDryRunError(e instanceof Error ? e.message : t('gradingAgent.aiBuilder.error'))
        setStatusMessage('')
        return false
      } finally {
        setAiBuilding(false)
      }
    },
    [aiBuilding, courseCode, itemId, itemKind, graph, quizQuestionSlots, t],
  )

  const enterGroup = useCallback(
    (groupId: string) => {
      const node = graph.nodes.find((n) => n.id === groupId && n.type === 'group')
      if (!node) return
      const gd = groupNodeData(node)
      setNavStack((prev) => [...prev, { parentGraph: graph, groupId, label: gd.label ?? 'Group' }])
      setGraph(gd.subgraph)
      setSelectedNodeId(null)
    },
    [graph],
  )

  const exitToDepth = useCallback(
    (depth: number) => {
      if (depth >= navStack.length) return
      let current = graphRef.current
      for (let i = navStack.length - 1; i >= depth; i--) {
        current = writeSubgraphBack(navStack[i].parentGraph, navStack[i].groupId, current)
      }
      setGraph(current)
      setNavStack(navStack.slice(0, depth))
      setSelectedNodeId(null)
    },
    [navStack],
  )

  const exitGroup = useCallback(() => {
    exitToDepth(Math.max(0, navStack.length - 1))
  }, [exitToDepth, navStack.length])

  const groupSelection = useCallback(
    (nodeIds: string[], label?: string): boolean => {
      const result = createGroupFromSelection(graph, nodeIds, label)
      if (!result) return false
      setGraph(result.graph)
      setSelectedNodeId(result.groupId)
      return true
    },
    [graph],
  )

  const ungroup = useCallback(
    (groupId: string) => {
      setGraph(ungroupNode(graph, groupId))
      setSelectedNodeId(null)
    },
    [graph],
  )

  const refreshRunResults = useCallback(async () => {
    if (!runId) return
    const run = await fetchGraderAgentRun(courseCode, itemId, runId)
    setRunResults(run.results)
    setRunProgress({
      completed: run.completedCount,
      failed: run.failedCount,
      total: run.totalCount,
    })
    onApplied?.()
  }, [courseCode, itemId, onApplied, runId])

  const updateNodeLabel = useCallback((nodeId: string, label: string | null) => {
    setGraph((prev) => {
      if (!prev) return prev
      return {
        ...prev,
        nodes: prev.nodes.map((n) =>
          n.id === nodeId ? { ...n, data: patchWorkflowNodeLabel(n.data, label) } : n,
        ),
      }
    })
  }, [])

  const removeNode = useCallback((nodeId: string) => {
    if (nodeId === 'output') return
    setGraph((prev) => {
      if (!prev) return prev
      return {
        ...prev,
        nodes: prev.nodes.filter((n) => n.id !== nodeId),
        edges: prev.edges.filter((e) => e.source !== nodeId && e.target !== nodeId),
      }
    })
    setSelectedNodeId((current) => (current === nodeId ? null : current))
  }, [])

  const removeEdge = useCallback((edgeId: string) => {
    setGraph((prev) => {
      if (!prev) return prev
      return {
        ...prev,
        edges: prev.edges.filter((e) => e.id !== edgeId),
      }
    })
  }, [])

  const resetDryRunVisualState = useCallback(() => {
    setNodeExecutionStates({})
    setDryRunLogs([])
    setNodeDryRunDetails({})
  }, [])

  const prevSubmissionIdRef = useRef<string | null>(null)
  useEffect(() => {
    if (!open) {
      prevSubmissionIdRef.current = null
      return
    }
    if (prevSubmissionIdRef.current !== null && prevSubmissionIdRef.current !== submissionId) {
      setDryRunResult(null)
      setDryRunError(null)
      setHadDryRun(false)
      resetDryRunVisualState()
    }
    prevSubmissionIdRef.current = submissionId
  }, [open, submissionId, resetDryRunVisualState])

  const handleDryRun = async () => {
    if (!submissionId || !graph) {
      setDryRunError('Open a submission before dry running.')
      return
    }
    if (!runnable) {
      setDryRunError(validationIssues[0]?.message ?? 'Fix workflow validation errors first.')
      return
    }
    setDryRunning(true)
    setDryRunError(null)
    setDryRunResult(null)
    setDryRunConsoleOpen(true)
    resetDryRunVisualState()
    setStatusMessage('Running dry run…')
    try {
      const result = await streamGraderAgentDryRun(
        courseCode,
        itemId,
        {
          workflowGraph: rootGraph as GraderWorkflowGraphApi,
          submissionId,
        },
        (event) => {
          if (event.type === 'log' && event.message) {
            setDryRunLogs((prev) => [
              ...prev,
              { message: event.message!, level: event.level ?? 'info' },
            ])
          }
          if (event.type === 'node_start' && event.nodeId) {
            setNodeExecutionStates((prev) => ({ ...prev, [event.nodeId!]: 'running' }))
          }
          if (event.type === 'node_complete' && event.nodeId) {
            const status = event.status === 'error' ? 'error' : event.status === 'skipped' ? 'skipped' : 'success'
            setNodeExecutionStates((prev) => ({
              ...prev,
              [event.nodeId!]: status,
            }))
            if (
              event.status === 'success' &&
              (event.compiledPrompt ||
                event.compiledSystemPrompt ||
                event.compiledInput ||
                event.compiledOutput)
            ) {
              setNodeDryRunDetails((prev) => ({
                ...prev,
                [event.nodeId!]: {
                  compiledPrompt: event.compiledPrompt,
                  compiledSystemPrompt: event.compiledSystemPrompt,
                  compiledInput: event.compiledInput,
                  compiledOutput: event.compiledOutput,
                },
              }))
            }
          }
          if (event.type === 'result' && event.result) {
            setDryRunResult(event.result)
          }
        },
      )
      setDryRunResult(result)
      setHadDryRun(true)
      setStatusMessage('Dry run complete.')
    } catch (e) {
      setDryRunError(e instanceof Error ? e.message : 'Dry run failed.')
      setStatusMessage('')
    } finally {
      setDryRunning(false)
    }
  }

  const handleApply = async () => {
    if (!submissionId || !dryRunResult || dryRunResult.flagged || dryRunResult.held?.wouldHold) return
    if (applyingSubmissionId === submissionId) return
    const built = buildAgentGradeApplyPayload(dryRunResult, rubric)
    if (!built.ok) {
      setDryRunError(built.error)
      return
    }
    setApplyingSubmissionId(submissionId)
    setDryRunError(null)
    canvasSyncAbortRef.current?.()
    canvasSyncAbortRef.current = null
    try {
      await putSubmissionGrade(courseCode, itemId, submissionId, built.gradeBody)
      onApplied?.()
      let startedCanvasSync = false
      if (canvasLink) {
        const syncHandle = queueCanvasGradeSync({
          courseCode,
          itemId,
          submissionId,
          canvasLink,
          gradePayload: built.canvasPayload,
          onComplete: () => {
            finishAppliedSubmission(submissionId)
            setStatusMessage('Grade applied and synced to Canvas.')
            onApplied?.()
          },
          onError: (message) => {
            finishAppliedSubmission(submissionId)
            setStatusMessage(message)
          },
        })
        if (syncHandle) {
          canvasSyncAbortRef.current = syncHandle.abort
          addSyncingSubmission(submissionId)
          setStatusMessage('Grade applied. Syncing to Canvas…')
          startedCanvasSync = true
        }
      }
      if (!startedCanvasSync) {
        finishAppliedSubmission(submissionId)
        setStatusMessage('Grade applied.')
      }
    } catch (e) {
      setApplyingSubmissionId(null)
      removeSyncingSubmission(submissionId)
      setDryRunError(e instanceof Error ? e.message : 'Could not apply grade.')
      setStatusMessage('')
    }
  }

  const handleSave = async () => {
    if (!graph) return
    setSaving(true)
    setDryRunError(null)
    setStatusMessage('')
    try {
      if (templateMode) {
        const body = {
          name: templateMode.name.trim(),
          prompt: '',
          includeAssignmentContent: false,
          includeRubric: false,
          workflowGraph: rootGraph as GraderWorkflowGraphApi,
        }
        if (savedTemplateId) {
          await putGraderAgentTemplate(courseCode, savedTemplateId, body)
        } else {
          const res = await postGraderAgentTemplate(courseCode, body)
          setSavedTemplateId(res.template.id)
        }
        setStatusMessage(t('gradingAgent.save.templateSaved'))
        return
      }
      const status = config?.status === 'accepted' ? 'accepted' : config?.status === 'archived' ? 'archived' : 'draft'
      const res = await putGraderAgentConfig(
        courseCode,
        itemId,
        graderAgentConfigPutPayload(config, rootGraph as GraderWorkflowGraphApi, { status }),
        itemKind,
      )
      setConfig(res.config)
      const savedGraph = effectiveWorkflowGraph(
        res.config.workflowGraph as GraderWorkflowGraph | undefined,
        res.config.prompt,
        res.config.includeAssignmentContent,
        res.config.includeRubric,
        itemKind,
      )
      setGraph(savedGraph)
      setNavStack([])
      setStatusMessage(t('gradingAgent.save.saved'))
    } catch (e) {
      setDryRunError(e instanceof Error ? e.message : t('gradingAgent.error.save'))
      setStatusMessage('')
    } finally {
      setSaving(false)
    }
  }

  const handleAccept = async () => {
    if (!graph || !runnable) return
    setSaving(true)
    try {
      const res = await putGraderAgentConfig(
        courseCode,
        itemId,
        graderAgentConfigPutPayload(config, rootGraph as GraderWorkflowGraphApi, { status: 'accepted' }),
        itemKind,
      )
      setConfig(res.config)
      setHadDryRun(true)
      setStatusMessage('Agent accepted.')
    } catch (e) {
      setDryRunError(e instanceof Error ? e.message : 'Could not save agent.')
    } finally {
      setSaving(false)
    }
  }

  const handleCancelRun = async () => {
    if (!runId || cancellingRun || !graderAgentCancelRunEnabled) return
    setCancellingRun(true)
    setDryRunError(null)
    try {
      await postGraderAgentCancelRun(courseCode, itemId, runId)
      appendRunLog(t('gradingAgent.run.cancel.requested'))
      setStatusMessage(t('gradingAgent.run.cancel.cancelling'))
    } catch (e) {
      setCancellingRun(false)
      setDryRunError(e instanceof Error ? e.message : t('gradingAgent.run.cancel.failed'))
    }
  }

  const handleRun = async () => {
    if (runScope === 'all' && !confirmOverwrite) {
      setConfirmOverwrite(true)
      return
    }
    if (!runnable) {
      setDryRunError(validationIssues[0]?.message ?? 'Fix workflow validation errors first.')
      return
    }
    setSaving(true)
    setDryRunError(null)
    try {
      if (graph && config?.status !== 'accepted') {
        await putGraderAgentConfig(
          courseCode,
          itemId,
          graderAgentConfigPutPayload(config, rootGraph as GraderWorkflowGraphApi),
          itemKind,
        )
      }
      const effectiveMode: GraderAgentRunMode = graderAgentSuggestModeEnabled ? runMode : 'apply'
      const runFilter = graderAgentRunFiltersEnabled ? runFilterFromState(runFilterState) : undefined
      const res = await postGraderAgentRun(courseCode, itemId, {
        scope: runScope,
        mode: effectiveMode,
        submissionId: runScope === 'current' ? submissionId ?? undefined : undefined,
        overwrite: runScope === 'all',
        authoredVia: 'canvas',
        filter: runFilter,
        budgetUsd: graderAgentCostEstimateEnabled && budgetUsd != null ? budgetUsd : undefined,
      })
      setLastRunMode(res.mode ?? effectiveMode)
      canvasSyncedSubmissionIdsRef.current = new Set()
      lastRunProgressLogRef.current = null
      setRunId(res.runId)
      setRunProgress({ completed: 0, failed: 0, total: res.totalCount })
      setConfirmOverwrite(false)
      setBatchRunning(true)
      setDryRunConsoleOpen(true)
      resetDryRunVisualState()
      setBatchNodeExecutionRunning()
      appendRunLog(t('gradingAgent.run.starting', { total: res.totalCount }))
      setStatusMessage(res.targetSummary ?? t('gradingAgent.run.starting', { total: res.totalCount }))
    } catch (e) {
      setBatchRunning(false)
      resetBatchNodeExecution()
      setDryRunError(e instanceof Error ? e.message : t('gradingAgent.error.run'))
      setStatusMessage('')
    } finally {
      setSaving(false)
    }
  }

  const handleToggleAutoGrade = async (enabled: boolean) => {
    if (!config || config.status !== 'accepted' || !graph) return
    const res = await putGraderAgentConfig(
      courseCode,
      itemId,
      graderAgentConfigPutPayload(config, rootGraph as GraderWorkflowGraphApi, {
        status: 'accepted',
        autoGradeNew: enabled,
      }),
      itemKind,
    )
    setConfig(res.config)
  }

  const handleTogglePostPolicy = async (autoPost: boolean) => {
    if (!config || config.status !== 'accepted' || !graph) return
    const res = await putGraderAgentConfig(
      courseCode,
      itemId,
      graderAgentConfigPutPayload(config, rootGraph as GraderWorkflowGraphApi, {
        status: 'accepted',
        postPolicy: autoPost ? 'auto_post' : 'draft',
      }),
      itemKind,
    )
    setConfig(res.config)
  }

  const handleSetConfidenceFloor = async (floor: number | null) => {
    if (!config || !graph) return
    const res = await putGraderAgentConfig(
      courseCode,
      itemId,
      graderAgentConfigPutPayload(config, rootGraph as GraderWorkflowGraphApi, { confidenceFloor: floor }),
      itemKind,
    )
    setConfig(res.config)
  }

  const handleSaveAsTemplate = async (name: string) => {
    if (!graph) return
    setSaving(true)
    setDryRunError(null)
    setStatusMessage('')
    try {
      await postGraderAgentTemplate(courseCode, {
        name,
        prompt: config?.prompt ?? '',
        includeAssignmentContent: config?.includeAssignmentContent ?? false,
        includeRubric: config?.includeRubric ?? false,
        workflowGraph: rootGraph as GraderWorkflowGraphApi,
      })
      setStatusMessage(t('gradingAgent.save.templateSaved'))
    } catch (e) {
      setDryRunError(e instanceof Error ? e.message : t('gradingAgent.error.saveTemplate'))
      setStatusMessage('')
      throw e
    } finally {
      setSaving(false)
    }
  }

  return {
    config,
    graph,
    selectedNodeId,
    setSelectedNodeId,
    dryRunning,
    batchRunning,
    cancellingRun,
    cancelRunEnabled: graderAgentCancelRunEnabled === true,
    dryRunError,
    dryRunResult,
    setDryRunResult,
    hadDryRun,
    saving,
    aiBuilding,
    handleAIBuild,
    applyingSubmissionId,
    syncingSubmissionIds,
    runScope,
    setRunScope,
    runMode,
    setRunMode,
    runProgress,
    runResults,
    confirmOverwrite,
    setConfirmOverwrite,
    runFilterState,
    setRunFilterState,
    runCostEstimate,
    runCostEstimateLoading,
    budgetUsd,
    setBudgetUsd,
    graderAgentCostEstimateEnabled,
    statusMessage,
    validationIssues,
    runnable,
    updateGraph,
    updateNodeData,
    setLibraryRubricAvailability,
    refreshRunResults,
    updateNodeLabel,
    addPaletteNode,
    navStack,
    enterGroup,
    exitGroup,
    exitToDepth,
    groupSelection,
    ungroup,
    removeNode,
    removeEdge,
    handleDryRun,
    handleApply,
    handleSave,
    handleSaveAsTemplate,
    handleAccept,
    handleRun,
    handleCancelRun,
    handleToggleAutoGrade,
    handleTogglePostPolicy,
    handleSetConfidenceFloor,
    nodeExecutionStates,
    dryRunLogs,
    dryRunConsoleOpen,
    nodeDryRunDetails,
    itemKind,
    quizQuestionSlots,
  }
}

export type GraderAgentWorkflowState = ReturnType<typeof useGraderAgentWorkflow>

export function primaryValidationMessage(issues: WorkflowValidationIssue[]): string | null {
  return issues[0]?.message ?? null
}
