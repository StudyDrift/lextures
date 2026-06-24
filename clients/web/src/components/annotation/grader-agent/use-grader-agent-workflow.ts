import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  fetchCourseCanvasLink,
  fetchGraderAgentConfig,
  fetchGraderAgentRun,
  postGraderAgentRun,
  postGraderAgentTemplate,
  putGraderAgentConfig,
  putGraderAgentTemplate,
  putSubmissionGrade,
  streamGraderAgentDryRun,
  type CourseCanvasLinkApi,
  type GraderAgentConfigApi,
  type GraderAgentDryRunResult,
  type GraderAgentRunStatus,
  type GraderWorkflowGraphApi,
  type RubricDefinition,
} from '../../../lib/courses-api'
import { queueCanvasGradeSync } from '../../canvas/canvas-grade-sync'
import { buildAgentGradeApplyPayload } from './agent-grade-apply'
import { effectiveWorkflowGraph, synthesizeDefaultGraph } from './default-graph'
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
import {
  defaultCodeTestRunnerNodeData,
  defaultConditionalRouterNodeData,
  defaultFlagForReviewNodeData,
  defaultHumanReviewGateNodeData,
  defaultOriginalityNodeData,
  defaultReferenceNodeData,
  defaultRubricNodeData,
  defaultScoreAggregatorNodeData,
} from './types'
import { useRubricLibraryRubrics } from './use-rubric-library-rubrics'
import { newWorkflowNodeId } from './workflow-node-id'
import { patchWorkflowNodeLabel } from './workflow-node-label'

export type RunScope = 'current' | 'ungraded' | 'all'

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
}: UseGraderAgentWorkflowArgs) {
  const { t } = useTranslation('common')
  const [savedTemplateId, setSavedTemplateId] = useState<string | null>(templateMode?.templateId ?? null)
  const [config, setConfig] = useState<GraderAgentConfigApi | null>(null)
  const [graph, setGraph] = useState<GraderWorkflowGraph>(() => synthesizeDefaultGraph('', false, false))
  const [selectedNodeId, setSelectedNodeId] = useState<string | null>(null)
  const [dryRunning, setDryRunning] = useState(false)
  const [dryRunError, setDryRunError] = useState<string | null>(null)
  const [dryRunResult, setDryRunResult] = useState<GraderAgentDryRunResult | null>(null)
  const [hadDryRun, setHadDryRun] = useState(false)
  const [saving, setSaving] = useState(false)
  const [runScope, setRunScope] = useState<RunScope>('ungraded')
  const [runId, setRunId] = useState<string | null>(null)
  const [runProgress, setRunProgress] = useState<{ completed: number; failed: number; total: number } | null>(null)
  const [runResults, setRunResults] = useState<GraderAgentRunStatus['results']>([])
  const [confirmOverwrite, setConfirmOverwrite] = useState(false)
  const [statusMessage, setStatusMessage] = useState('')
  const [nodeExecutionStates, setNodeExecutionStates] = useState<Record<string, NodeExecutionStatus>>({})
  const [dryRunLogs, setDryRunLogs] = useState<DryRunLogEntry[]>([])
  const [dryRunConsoleOpen, setDryRunConsoleOpen] = useState(false)
  const [nodeDryRunDetails, setNodeDryRunDetails] = useState<Record<string, NodeDryRunDetail>>({})
  const [canvasLink, setCanvasLink] = useState<CourseCanvasLinkApi | null>(null)
  const canvasSyncAbortRef = useRef<(() => void) | null>(null)

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
    if (!open) return
    setSavedTemplateId(templateMode?.templateId ?? null)
  }, [open, templateMode?.templateId])

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

  useEffect(() => {
    if (!open || !runId) return
    const timer = window.setInterval(() => {
      void fetchGraderAgentRun(courseCode, itemId, runId)
        .then((run) => {
          setRunProgress({
            completed: run.completedCount,
            failed: run.failedCount,
            total: run.totalCount,
          })
          setRunResults(run.results)
          setStatusMessage(`${run.completedCount} / ${run.totalCount} complete`)
          if (run.status === 'done' || run.status === 'error') {
            window.clearInterval(timer)
            onApplied?.()
          }
        })
        .catch(() => undefined)
    }, 1500)
    return () => window.clearInterval(timer)
  }, [open, runId, courseCode, itemId, onApplied])

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
      const prefix =
        type === 'studentSubmission'
          ? 'sub'
          : type === 'activity'
            ? 'act'
            : type === 'codeTestRunner'
              ? 'ctr'
              : type === 'conditionalRouter'
                ? 'rtr'
                : type === 'flagForReview'
                  ? 'flag'
                  : type === 'humanReviewGate'
                    ? 'gate'
                    : type === 'scoreAggregator'
                      ? 'agg'
                      : type === 'originality'
                      ? 'orig'
                      : type === 'reference'
                        ? 'ref'
                        : type === 'rubric'
                          ? 'rub'
                          : type === 'criterionGrader'
                    ? 'cg'
                    : 'ai'
      const id = newWorkflowNodeId(prefix)
      setSelectedNodeId(id)
      setGraph((current) => {
        if (!current) return current
        const fallback =
          type === 'studentSubmission'
            ? { x: -640, y: -80 + current.nodes.length * 40 }
            : type === 'activity'
              ? { x: -640, y: 120 + current.nodes.length * 40 }
              : type === 'codeTestRunner'
                ? { x: -320, y: -40 + current.nodes.length * 40 }
                : type === 'conditionalRouter'
                  ? { x: -320, y: 80 + current.nodes.length * 40 }
                  : type === 'flagForReview'
                    ? { x: 160, y: 80 + current.nodes.length * 40 }
                    : type === 'humanReviewGate'
                      ? { x: 0, y: 40 + current.nodes.length * 40 }
                      : type === 'scoreAggregator'
                        ? { x: 0, y: 0 + current.nodes.length * 40 }
                        : type === 'originality'
                        ? { x: -160, y: 120 + current.nodes.length * 40 }
                        : type === 'reference'
                          ? { x: -640, y: 200 + current.nodes.length * 40 }
                          : type === 'rubric'
                            ? { x: -640, y: 280 + current.nodes.length * 40 }
                            : type === 'criterionGrader'
                      ? { x: -320, y: 0 + current.nodes.length * 40 }
                      : { x: -320, y: 40 + current.nodes.length * 40 }
        const data =
          type === 'activity'
            ? { assignmentItemId: itemId }
            : type === 'codeTestRunner'
              ? defaultCodeTestRunnerNodeData()
              : type === 'conditionalRouter'
                ? defaultConditionalRouterNodeData()
                : type === 'flagForReview'
                  ? defaultFlagForReviewNodeData()
                  : type === 'humanReviewGate'
                    ? defaultHumanReviewGateNodeData()
                    : type === 'scoreAggregator'
                      ? defaultScoreAggregatorNodeData()
                      : type === 'originality'
                      ? defaultOriginalityNodeData()
                      : type === 'reference'
                        ? defaultReferenceNodeData()
                        : type === 'rubric'
                          ? defaultRubricNodeData()
                          : type === 'criterionGrader'
                    ? { prompt: '' }
                    : {}
        return {
          ...current,
          nodes: [
            ...current.nodes,
            {
              id,
              type,
              position: position ?? fallback,
              data,
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
    const built = buildAgentGradeApplyPayload(dryRunResult, rubric)
    if (!built.ok) {
      setDryRunError(built.error)
      return
    }
    setSaving(true)
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
            setStatusMessage('Grade applied and synced to Canvas.')
            onApplied?.()
          },
          onError: (message) => {
            setStatusMessage(message)
          },
        })
        if (syncHandle) {
          canvasSyncAbortRef.current = syncHandle.abort
          setStatusMessage('Grade applied. Syncing to Canvas…')
          startedCanvasSync = true
        }
      }
      if (!startedCanvasSync) {
        setStatusMessage('Grade applied.')
      }
    } catch (e) {
      setDryRunError(e instanceof Error ? e.message : 'Could not apply grade.')
      setStatusMessage('')
    } finally {
      setSaving(false)
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
      const res = await putGraderAgentConfig(courseCode, itemId, {
        prompt: config?.prompt ?? '',
        includeAssignmentContent: config?.includeAssignmentContent ?? false,
        includeRubric: config?.includeRubric ?? false,
        status,
        autoGradeNew: config?.autoGradeNew ?? false,
        workflowGraph: graph as GraderWorkflowGraphApi,
      })
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
      const res = await putGraderAgentConfig(courseCode, itemId, {
        prompt: config?.prompt ?? '',
        includeAssignmentContent: config?.includeAssignmentContent ?? false,
        includeRubric: config?.includeRubric ?? false,
        status: 'accepted',
        autoGradeNew: config?.autoGradeNew ?? false,
        workflowGraph: graph as GraderWorkflowGraphApi,
      })
      setConfig(res.config)
      setHadDryRun(true)
      setStatusMessage('Agent accepted.')
    } catch (e) {
      setDryRunError(e instanceof Error ? e.message : 'Could not save agent.')
    } finally {
      setSaving(false)
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
        await putGraderAgentConfig(courseCode, itemId, {
          prompt: config?.prompt ?? '',
          includeAssignmentContent: config?.includeAssignmentContent ?? false,
          includeRubric: config?.includeRubric ?? false,
          status: config?.status ?? 'draft',
          autoGradeNew: config?.autoGradeNew ?? false,
          workflowGraph: graph as GraderWorkflowGraphApi,
        })
      }
      const res = await postGraderAgentRun(courseCode, itemId, {
        scope: runScope,
        submissionId: runScope === 'current' ? submissionId ?? undefined : undefined,
        overwrite: runScope === 'all',
        authoredVia: 'canvas',
      })
      setRunId(res.runId)
      setRunProgress({ completed: 0, failed: 0, total: res.totalCount })
      setConfirmOverwrite(false)
      setStatusMessage('Run started.')
    } catch (e) {
      setDryRunError(e instanceof Error ? e.message : 'Could not start run.')
    } finally {
      setSaving(false)
    }
  }

  const handleToggleAutoGrade = async (enabled: boolean) => {
    if (!config || config.status !== 'accepted' || !graph) return
    const res = await putGraderAgentConfig(courseCode, itemId, {
      prompt: config.prompt,
      includeAssignmentContent: config.includeAssignmentContent,
      includeRubric: config.includeRubric,
      status: 'accepted',
      autoGradeNew: enabled,
      workflowGraph: graph as GraderWorkflowGraphApi,
    })
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
    dryRunError,
    dryRunResult,
    setDryRunResult,
    hadDryRun,
    saving,
    runScope,
    setRunScope,
    runProgress,
    runResults,
    confirmOverwrite,
    setConfirmOverwrite,
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
    handleToggleAutoGrade,
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
