import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { usePlatformFeatures } from '../../../context/platform-features-context'
import {
  fetchCourseCanvasLink,
  fetchGraderAgentConfig,
  fetchGraderAgentRun,
  fetchSubmissionGrade,
  postGraderAgentCancelRun,
  postGraderAgentRun,
  postGraderAgentTemplate,
  putGraderAgentConfig,
  putGraderAgentTemplate,
  putSubmissionGrade,
  streamGraderAgentDryRun,
  type CourseCanvasLinkApi,
  type GraderAgentConfigApi,
  type GraderAgentDryRunResult,
  type GraderAgentRunMode,
  type GraderAgentRunStatus,
  type GraderWorkflowGraphApi,
  type RubricDefinition,
} from '../../../lib/courses-api'
import { queueCanvasGradeSync } from '../../canvas/canvas-grade-sync'
import { buildAgentGradeApplyPayload, canvasPayloadFromSubmissionGrade } from './agent-grade-apply'
import { effectiveWorkflowGraph, synthesizeDefaultGraph } from './default-graph'
import {
  defaultRunAgentFilterState,
  runFilterFromState,
  type RunAgentFilterState,
} from './run-agent-filter-picker'
import { isWorkflowRunnable, validateWorkflowGraph } from './validation'
import type {
  CodeTestRunnerNodeData,
  ConditionalRouterNodeData,
  CriterionGraderNodeData,
  FlagForReviewNodeData,
  HumanReviewGateNodeData,
  OriginalityNodeData,
  ReferenceNodeData,
  RubricNodeData,
  ScoreAggregatorNodeData,
  GraderWorkflowGraph,
  PaletteNodeType,
  WorkflowValidationIssue,
} from './types'
import { useRubricLibraryRubrics } from './use-rubric-library-rubrics'
import { newWorkflowNodeId } from './workflow-node-id'
import { patchWorkflowNodeLabel } from './workflow-node-label'
import { paletteNodeDefaults } from './node-descriptors'

export type RunScope = 'current' | 'ungraded' | 'all'

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
  submissionId,
  rubric = null,
  seedWorkflow = null,
  templateMode = null,
  onApplied,
  onSubmissionGraded,
}: UseGraderAgentWorkflowArgs) {
  const { t } = useTranslation('common')
  const { graderAgentSuggestModeEnabled, graderAgentRunFiltersEnabled, graderAgentCancelRunEnabled } =
    usePlatformFeatures()
  const [savedTemplateId, setSavedTemplateId] = useState<string | null>(templateMode?.templateId ?? null)
  const [config, setConfig] = useState<GraderAgentConfigApi | null>(null)
  const [graph, setGraph] = useState<GraderWorkflowGraph>(() => synthesizeDefaultGraph('', false, false))
  const [selectedNodeId, setSelectedNodeId] = useState<string | null>(null)
  const [dryRunning, setDryRunning] = useState(false)
  const [batchRunning, setBatchRunning] = useState(false)
  const [cancellingRun, setCancellingRun] = useState(false)
  const [dryRunError, setDryRunError] = useState<string | null>(null)
  const [dryRunResult, setDryRunResult] = useState<GraderAgentDryRunResult | null>(null)
  const [hadDryRun, setHadDryRun] = useState(false)
  const [saving, setSaving] = useState(false)
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
    () => ({ rubric, assignmentItemId: itemId, libraryRubrics }),
    [rubric, itemId, libraryRubrics],
  )
  const validationIssues = useMemo(
    () => validateWorkflowGraph(graph, validationOptions),
    [graph, validationOptions],
  )
  const runnable = isWorkflowRunnable(graph, validationOptions)

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

  const loadConfig = useCallback(async () => {
    if (templateMode) {
      setConfig(null)
      if (seedWorkflow) {
        const g = effectiveWorkflowGraph(
          seedWorkflow.workflowGraph as GraderWorkflowGraph | undefined,
          seedWorkflow.prompt,
          seedWorkflow.includeAssignmentContent,
          seedWorkflow.includeRubric,
        )
        setGraph(g)
        setSelectedNodeId(null)
      } else {
        const g = effectiveWorkflowGraph(null, '', false, false)
        setGraph(g)
        setSelectedNodeId(null)
      }
      return
    }
    const res = await fetchGraderAgentConfig(courseCode, itemId)
    const c = res.config
    setConfig(c)
    if (c) {
      const g = effectiveWorkflowGraph(
        c.workflowGraph as GraderWorkflowGraph | undefined,
        c.prompt,
        c.includeAssignmentContent,
        c.includeRubric,
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
      )
      setGraph(g)
      setSelectedNodeId(null)
    } else {
      const g = effectiveWorkflowGraph(null, '', false, false)
      setGraph(g)
      setSelectedNodeId(null)
    }
  }, [courseCode, itemId, seedWorkflow, templateMode])

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

      if (run.status === 'done' || run.status === 'error' || run.status === 'failed' || run.status === 'cancelled') {
        setBatchRunning(false)
        setCancellingRun(false)
        resetBatchNodeExecution()
        const appliedCount = run.results.filter((result) => result.status === 'applied').length
        const suggestedCount = run.results.filter((result) => result.status === 'suggested').length
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
            t('gradingAgent.run.cancel.complete', {
              applied: appliedCount,
              suggested: suggestedCount,
              skipped: skippedCount,
            }),
          )
        } else if (run.status === 'error' || run.status === 'failed') {
          appendRunLog(t('gradingAgent.run.failed'), 'error')
          setStatusMessage(t('gradingAgent.run.failed'))
        } else if (lastRunMode === 'suggest') {
          appendRunLog(
            t('gradingAgent.run.completeSuggest', {
              suggested: suggestedCount,
              failed: run.failedCount,
            }),
          )
          setStatusMessage(
            t('gradingAgent.run.completeSuggest', { suggested: suggestedCount, failed: run.failedCount }),
          )
        } else {
          appendRunLog(
            t('gradingAgent.run.complete', {
              applied: appliedCount,
              failed: run.failedCount,
            }),
          )
          setStatusMessage(t('gradingAgent.run.complete', { applied: appliedCount, failed: run.failedCount }))
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

  const updateGraderNode = useCallback(
    (nodeId: string, patch: { prompt?: string; modelId?: string | null }) => {
      if (!graph) return
      setGraph({
        ...graph,
        nodes: graph.nodes.map((n) =>
          n.id === nodeId ? { ...n, data: { ...n.data, ...patch } } : n,
        ),
      })
    },
    [graph],
  )

  const updateAiNode = useCallback(
    (nodeId: string, patch: { prompt?: string }) => {
      if (!graph) return
      setGraph({
        ...graph,
        nodes: graph.nodes.map((n) =>
          n.id === nodeId ? { ...n, data: { ...n.data, ...patch } } : n,
        ),
      })
    },
    [graph],
  )

  const updateCriterionGraderNode = useCallback(
    (nodeId: string, patch: Partial<CriterionGraderNodeData>) => {
      if (!graph) return
      setGraph({
        ...graph,
        nodes: graph.nodes.map((n) =>
          n.id === nodeId ? { ...n, data: { ...n.data, ...patch } } : n,
        ),
      })
    },
    [graph],
  )

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

  const updateCodeTestRunnerNode = useCallback(
    (nodeId: string, patch: Partial<CodeTestRunnerNodeData>) => {
      if (!graph) return
      setGraph({
        ...graph,
        nodes: graph.nodes.map((n) =>
          n.id === nodeId ? { ...n, data: { ...n.data, ...patch } } : n,
        ),
      })
    },
    [graph],
  )

  const updateConditionalRouterNode = useCallback(
    (nodeId: string, patch: Partial<ConditionalRouterNodeData>) => {
      if (!graph) return
      setGraph({
        ...graph,
        nodes: graph.nodes.map((n) =>
          n.id === nodeId ? { ...n, data: { ...n.data, ...patch } } : n,
        ),
      })
    },
    [graph],
  )

  const updateFlagForReviewNode = useCallback(
    (nodeId: string, patch: Partial<FlagForReviewNodeData>) => {
      if (!graph) return
      setGraph({
        ...graph,
        nodes: graph.nodes.map((n) =>
          n.id === nodeId ? { ...n, data: { ...n.data, ...patch } } : n,
        ),
      })
    },
    [graph],
  )

  const updateOriginalityNode = useCallback(
    (nodeId: string, patch: Partial<OriginalityNodeData>) => {
      if (!graph) return
      setGraph({
        ...graph,
        nodes: graph.nodes.map((n) =>
          n.id === nodeId ? { ...n, data: { ...n.data, ...patch } } : n,
        ),
      })
    },
    [graph],
  )

  const updateReferenceNode = useCallback(
    (nodeId: string, patch: Partial<ReferenceNodeData>) => {
      if (!graph) return
      setGraph({
        ...graph,
        nodes: graph.nodes.map((n) =>
          n.id === nodeId ? { ...n, data: { ...n.data, ...patch } } : n,
        ),
      })
    },
    [graph],
  )

  const updateRubricNode = useCallback(
    (nodeId: string, patch: Partial<RubricNodeData>) => {
      if (!graph) return
      setGraph({
        ...graph,
        nodes: graph.nodes.map((n) =>
          n.id === nodeId ? { ...n, data: { ...n.data, ...patch } } : n,
        ),
      })
    },
    [graph],
  )

  const updateHumanReviewGateNode = useCallback(
    (nodeId: string, patch: Partial<HumanReviewGateNodeData>) => {
      if (!graph) return
      setGraph({
        ...graph,
        nodes: graph.nodes.map((n) =>
          n.id === nodeId ? { ...n, data: { ...n.data, ...patch } } : n,
        ),
      })
    },
    [graph],
  )

  const updateScoreAggregatorNode = useCallback(
    (nodeId: string, patch: Partial<ScoreAggregatorNodeData>) => {
      if (!graph) return
      setGraph({
        ...graph,
        nodes: graph.nodes.map((n) =>
          n.id === nodeId ? { ...n, data: { ...n.data, ...patch } } : n,
        ),
      })
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

  const updateActivityNode = useCallback(
    (nodeId: string, patch: { assignmentItemId?: string | null }) => {
      if (!graph) return
      setGraph({
        ...graph,
        nodes: graph.nodes.map((n) =>
          n.id === nodeId ? { ...n, data: { ...n.data, ...patch } } : n,
        ),
      })
    },
    [graph],
  )

  const updateNodeLabel = useCallback(
    (nodeId: string, label: string | null) => {
      if (!graph) return
      setGraph({
        ...graph,
        nodes: graph.nodes.map((n) =>
          n.id === nodeId ? { ...n, data: patchWorkflowNodeLabel(n.data, label) } : n,
        ),
      })
    },
    [graph],
  )

  const removeNode = useCallback(
    (nodeId: string) => {
      if (!graph || nodeId === 'output') return
      setGraph({
        ...graph,
        nodes: graph.nodes.filter((n) => n.id !== nodeId),
        edges: graph.edges.filter((e) => e.source !== nodeId && e.target !== nodeId),
      })
      if (selectedNodeId === nodeId) setSelectedNodeId(null)
    },
    [graph, selectedNodeId],
  )

  const removeEdge = useCallback(
    (edgeId: string) => {
      if (!graph) return
      setGraph({
        ...graph,
        edges: graph.edges.filter((e) => e.id !== edgeId),
      })
    },
    [graph],
  )

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
          workflowGraph: graph as GraderWorkflowGraphApi,
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
          workflowGraph: graph as GraderWorkflowGraphApi,
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
        graderAgentConfigPutPayload(config, graph as GraderWorkflowGraphApi, { status }),
      )
      setConfig(res.config)
      const savedGraph = effectiveWorkflowGraph(
        res.config.workflowGraph as GraderWorkflowGraph | undefined,
        res.config.prompt,
        res.config.includeAssignmentContent,
        res.config.includeRubric,
      )
      setGraph(savedGraph)
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
        graderAgentConfigPutPayload(config, graph as GraderWorkflowGraphApi, { status: 'accepted' }),
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
          graderAgentConfigPutPayload(config, graph as GraderWorkflowGraphApi),
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
      graderAgentConfigPutPayload(config, graph as GraderWorkflowGraphApi, {
        status: 'accepted',
        autoGradeNew: enabled,
      }),
    )
    setConfig(res.config)
  }

  const handleTogglePostPolicy = async (autoPost: boolean) => {
    if (!config || config.status !== 'accepted' || !graph) return
    const res = await putGraderAgentConfig(
      courseCode,
      itemId,
      graderAgentConfigPutPayload(config, graph as GraderWorkflowGraphApi, {
        status: 'accepted',
        postPolicy: autoPost ? 'auto_post' : 'draft',
      }),
    )
    setConfig(res.config)
  }

  const handleSetConfidenceFloor = async (floor: number | null) => {
    if (!config || !graph) return
    const res = await putGraderAgentConfig(
      courseCode,
      itemId,
      graderAgentConfigPutPayload(config, graph as GraderWorkflowGraphApi, { confidenceFloor: floor }),
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
        workflowGraph: graph as GraderWorkflowGraphApi,
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
    statusMessage,
    validationIssues,
    runnable,
    updateGraph,
    updateGraderNode,
    updateAiNode,
    updateCriterionGraderNode,
    updateActivityNode,
    updateCodeTestRunnerNode,
    updateConditionalRouterNode,
    updateFlagForReviewNode,
    updateHumanReviewGateNode,
    updateScoreAggregatorNode,
    updateOriginalityNode,
    updateReferenceNode,
    updateRubricNode,
    setLibraryRubricAvailability,
    refreshRunResults,
    updateNodeLabel,
    addPaletteNode,
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
  }
}

export type GraderAgentWorkflowState = ReturnType<typeof useGraderAgentWorkflow>

export function primaryValidationMessage(issues: WorkflowValidationIssue[]): string | null {
  return issues[0]?.message ?? null
}
