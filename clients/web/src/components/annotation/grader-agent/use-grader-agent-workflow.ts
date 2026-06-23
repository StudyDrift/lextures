import { useCallback, useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  fetchGraderAgentConfig,
  fetchGraderAgentRun,
  postGraderAgentDryRun,
  postGraderAgentRun,
  putGraderAgentConfig,
  putSubmissionGrade,
  type GraderAgentConfigApi,
  type GraderAgentDryRunResult,
  type GraderWorkflowGraphApi,
} from '../../../lib/courses-api'
import { effectiveWorkflowGraph } from './default-graph'
import { isWorkflowRunnable, validateWorkflowGraph } from './validation'
import type { GraderWorkflowGraph, PaletteNodeType, WorkflowValidationIssue } from './types'
import { newWorkflowNodeId } from './workflow-node-id'
import { patchWorkflowNodeLabel } from './workflow-node-label'

export type RunScope = 'current' | 'ungraded' | 'all'

type UseGraderAgentWorkflowArgs = {
  open: boolean
  courseCode: string
  itemId: string
  submissionId: string | null
  onApplied?: () => void
}

export function useGraderAgentWorkflow({
  open,
  courseCode,
  itemId,
  submissionId,
  onApplied,
}: UseGraderAgentWorkflowArgs) {
  const { t } = useTranslation('common')
  const [config, setConfig] = useState<GraderAgentConfigApi | null>(null)
  const [graph, setGraph] = useState<GraderWorkflowGraph | null>(null)
  const [selectedNodeId, setSelectedNodeId] = useState<string | null>(null)
  const [dryRunning, setDryRunning] = useState(false)
  const [dryRunError, setDryRunError] = useState<string | null>(null)
  const [dryRunResult, setDryRunResult] = useState<GraderAgentDryRunResult | null>(null)
  const [hadDryRun, setHadDryRun] = useState(false)
  const [saving, setSaving] = useState(false)
  const [runScope, setRunScope] = useState<RunScope>('ungraded')
  const [runId, setRunId] = useState<string | null>(null)
  const [runProgress, setRunProgress] = useState<{ completed: number; failed: number; total: number } | null>(null)
  const [confirmOverwrite, setConfirmOverwrite] = useState(false)
  const [statusMessage, setStatusMessage] = useState('')

  const validationIssues = useMemo(() => validateWorkflowGraph(graph), [graph])
  const runnable = isWorkflowRunnable(graph)

  const loadConfig = useCallback(async () => {
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
    } else {
      const g = effectiveWorkflowGraph(null, '', false, false)
      setGraph(g)
      setSelectedNodeId(null)
    }
  }, [courseCode, itemId])

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
    if (!open || !runId) return
    const timer = window.setInterval(() => {
      void fetchGraderAgentRun(courseCode, itemId, runId)
        .then((run) => {
          setRunProgress({
            completed: run.completedCount,
            failed: run.failedCount,
            total: run.totalCount,
          })
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

  const addPaletteNode = useCallback(
    (type: PaletteNodeType, position?: { x: number; y: number }) => {
      const prefix = type === 'studentSubmission' ? 'sub' : type === 'activity' ? 'act' : 'ai'
      const id = newWorkflowNodeId(prefix)
      setSelectedNodeId(id)
      setGraph((current) => {
        if (!current) return current
        const fallback =
          type === 'studentSubmission'
            ? { x: -640, y: -80 + current.nodes.length * 40 }
            : type === 'activity'
              ? { x: -640, y: 120 + current.nodes.length * 40 }
              : { x: -320, y: 40 + current.nodes.length * 40 }
        return {
          ...current,
          nodes: [
            ...current.nodes,
            {
              id,
              type,
              position: position ?? fallback,
              data: type === 'activity' ? { assignmentItemId: itemId } : {},
            },
          ],
        }
      })
    },
    [itemId],
  )

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
    setStatusMessage('Running dry run…')
    try {
      const result = await postGraderAgentDryRun(courseCode, itemId, {
        workflowGraph: graph as GraderWorkflowGraphApi,
        submissionId,
      })
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
    if (!submissionId || !dryRunResult) return
    setSaving(true)
    try {
      await putSubmissionGrade(courseCode, itemId, submissionId, {
        pointsEarned: dryRunResult.suggestedPoints,
        rubricScores: dryRunResult.rubricScores,
        instructorComment: dryRunResult.comment,
        gradedByAi: true,
      })
      onApplied?.()
      setStatusMessage('Grade applied.')
    } catch (e) {
      setDryRunError(e instanceof Error ? e.message : 'Could not apply grade.')
    } finally {
      setSaving(false)
    }
  }

  const handleSave = async () => {
    if (!graph || config?.status === 'accepted') return
    setSaving(true)
    setDryRunError(null)
    setStatusMessage('')
    try {
      const res = await putGraderAgentConfig(courseCode, itemId, {
        prompt: config?.prompt ?? '',
        includeAssignmentContent: config?.includeAssignmentContent ?? false,
        includeRubric: config?.includeRubric ?? false,
        status: 'draft',
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
    confirmOverwrite,
    setConfirmOverwrite,
    statusMessage,
    validationIssues,
    runnable,
    updateGraph,
    updateGraderNode,
    updateAiNode,
    updateActivityNode,
    updateNodeLabel,
    addPaletteNode,
    removeNode,
    handleDryRun,
    handleApply,
    handleSave,
    handleAccept,
    handleRun,
    handleToggleAutoGrade,
  }
}

export type GraderAgentWorkflowState = ReturnType<typeof useGraderAgentWorkflow>

export function primaryValidationMessage(issues: WorkflowValidationIssue[]): string | null {
  return issues[0]?.message ?? null
}
