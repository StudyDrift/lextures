import { useCallback, useEffect, useId, useRef, useState } from 'react'
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
  type RubricDefinition,
} from '../../lib/courses-api'
import { RubricGradePicker } from '../grading/rubric-grade-picker'

type RunScope = 'current' | 'ungraded' | 'all'

type GraderAgentDrawerProps = {
  open: boolean
  onClose: () => void
  courseCode: string
  itemId: string
  submissionId: string | null
  rubric: RubricDefinition | null
  maxPoints: number | null
  onApplied?: () => void
}

export function GraderAgentDrawer({
  open,
  onClose,
  courseCode,
  itemId,
  submissionId,
  rubric,
  maxPoints,
  onApplied,
}: GraderAgentDrawerProps) {
  const { t } = useTranslation('common')
  const statusId = useId()
  const drawerRef = useRef<HTMLDivElement>(null)
  const [config, setConfig] = useState<GraderAgentConfigApi | null>(null)
  const [prompt, setPrompt] = useState('')
  const [includeContent, setIncludeContent] = useState(false)
  const [dryRunning, setDryRunning] = useState(false)
  const [dryRunError, setDryRunError] = useState<string | null>(null)
  const [dryRunResult, setDryRunResult] = useState<GraderAgentDryRunResult | null>(null)
  const [hadDryRun, setHadDryRun] = useState(false)
  const [saving, setSaving] = useState(false)
  const [runScope, setRunScope] = useState<RunScope>('ungraded')
  const [runId, setRunId] = useState<string | null>(null)
  const [runProgress, setRunProgress] = useState<{ completed: number; failed: number; total: number } | null>(null)
  const [confirmOverwrite, setConfirmOverwrite] = useState(false)

  const loadConfig = useCallback(async () => {
    const res = await fetchGraderAgentConfig(courseCode, itemId)
    const c = res.config
    setConfig(c)
    if (c) {
      setPrompt(c.prompt)
      setIncludeContent(c.includeAssignmentContent || c.includeRubric)
      setHadDryRun(c.status === 'accepted')
    }
  }, [courseCode, itemId])

  useEffect(() => {
    if (!open) return
    let cancelled = false
    void (async () => {
      try {
        await loadConfig()
      } catch (e) {
        if (!cancelled) setDryRunError(e instanceof Error ? e.message : t('gradingAgent.error.load'))
      }
    })()
    return () => {
      cancelled = true
    }
  }, [open, loadConfig, t])

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
          if (run.status === 'done' || run.status === 'error') {
            window.clearInterval(timer)
            onApplied?.()
          }
        })
        .catch(() => undefined)
    }, 1500)
    return () => window.clearInterval(timer)
  }, [open, runId, courseCode, itemId, onApplied])

  const handleDryRun = async () => {
    if (!submissionId) {
      setDryRunError(t('gradingAgent.error.noSubmission'))
      return
    }
    setDryRunning(true)
    setDryRunError(null)
    try {
      const result = await postGraderAgentDryRun(courseCode, itemId, {
        prompt,
        includeAssignmentContent: includeContent,
        includeRubric: includeContent,
        submissionId,
      })
      setDryRunResult(result)
      setHadDryRun(true)
    } catch (e) {
      setDryRunError(e instanceof Error ? e.message : t('gradingAgent.error.dryRun'))
    } finally {
      setDryRunning(false)
    }
  }

  const handleApply = async () => {
    if (!submissionId || !dryRunResult) return
    setSaving(true)
    try {
      const gradedByAi = true
      await putSubmissionGrade(courseCode, itemId, submissionId, {
        pointsEarned: dryRunResult.suggestedPoints,
        rubricScores: dryRunResult.rubricScores,
        instructorComment: dryRunResult.comment,
        gradedByAi,
      })
      onApplied?.()
      onClose()
    } catch (e) {
      setDryRunError(e instanceof Error ? e.message : t('gradingAgent.error.apply'))
    } finally {
      setSaving(false)
    }
  }

  const handleAccept = async () => {
    setSaving(true)
    try {
      const res = await putGraderAgentConfig(courseCode, itemId, {
        prompt,
        includeAssignmentContent: includeContent,
        includeRubric: includeContent,
        status: 'accepted',
        autoGradeNew: config?.autoGradeNew ?? false,
      })
      setConfig(res.config)
      setHadDryRun(true)
    } catch (e) {
      setDryRunError(e instanceof Error ? e.message : t('gradingAgent.error.save'))
    } finally {
      setSaving(false)
    }
  }

  const handleRun = async () => {
    if (runScope === 'all' && !confirmOverwrite) {
      setConfirmOverwrite(true)
      return
    }
    setSaving(true)
    setDryRunError(null)
    try {
      const res = await postGraderAgentRun(courseCode, itemId, {
        scope: runScope,
        submissionId: runScope === 'current' ? submissionId ?? undefined : undefined,
        overwrite: runScope === 'all',
      })
      setRunId(res.runId)
      setRunProgress({ completed: 0, failed: 0, total: res.totalCount })
      setConfirmOverwrite(false)
    } catch (e) {
      setDryRunError(e instanceof Error ? e.message : t('gradingAgent.error.run'))
    } finally {
      setSaving(false)
    }
  }

  const handleToggleAutoGrade = async (enabled: boolean) => {
    if (!config || config.status !== 'accepted') return
    const res = await putGraderAgentConfig(courseCode, itemId, {
      prompt: config.prompt,
      includeAssignmentContent: config.includeAssignmentContent,
      includeRubric: config.includeRubric,
      status: 'accepted',
      autoGradeNew: enabled,
    })
    setConfig(res.config)
  }

  if (!open) return null

  const accepted = config?.status === 'accepted'

  return (
    <div
      ref={drawerRef}
      role="dialog"
      aria-modal="true"
      aria-labelledby="grader-agent-title"
      className="fixed inset-y-0 end-0 z-[520] flex w-full max-w-md flex-col border-s border-slate-200 bg-white shadow-xl dark:border-neutral-700 dark:bg-neutral-900"
    >
      <header className="flex shrink-0 items-center justify-between border-b border-slate-200 px-4 py-3 dark:border-neutral-700">
        <h2 id="grader-agent-title" className="text-lg font-semibold text-slate-900 dark:text-neutral-50">
          {t('gradingAgent.drawer.title')}
        </h2>
        <button
          type="button"
          onClick={onClose}
          className="rounded-lg px-2 py-1 text-sm text-slate-600 hover:bg-slate-100 dark:text-neutral-300 dark:hover:bg-neutral-800"
        >
          {t('gradingAgent.close')}
        </button>
      </header>

      <div className="flex-1 space-y-4 overflow-y-auto p-4">
        <label className="block text-sm text-slate-700 dark:text-neutral-200">
          <span className="mb-1.5 block font-medium">{t('gradingAgent.prompt.label')}</span>
          <textarea
            value={prompt}
            onChange={(e) => setPrompt(e.target.value)}
            rows={5}
            disabled={accepted}
            className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-950"
            placeholder={t('gradingAgent.prompt.placeholder')}
          />
        </label>

        <label className="flex items-center gap-2 text-sm text-slate-700 dark:text-neutral-200">
          <input
            type="checkbox"
            checked={includeContent}
            onChange={(e) => setIncludeContent(e.target.checked)}
            disabled={accepted}
          />
          {t('gradingAgent.includeContentRubric')}
        </label>

        {!accepted ? (
          <button
            type="button"
            disabled={dryRunning || !prompt.trim() || !submissionId}
            onClick={() => void handleDryRun()}
            className="w-full rounded-lg bg-indigo-600 px-4 py-2 text-sm font-semibold text-white hover:bg-indigo-700 disabled:opacity-50"
          >
            {dryRunning ? t('gradingAgent.dryRun.running') : t('gradingAgent.dryRun')}
          </button>
        ) : null}

        <div id={statusId} role="status" aria-live="polite" className="min-h-[1rem] text-sm">
          {dryRunError ? (
            <p className="rounded-lg border border-rose-200 bg-rose-50 px-3 py-2 text-rose-800 dark:border-rose-900/60 dark:bg-rose-950/40 dark:text-rose-200">
              {dryRunError}
            </p>
          ) : null}
        </div>

        {dryRunResult ? (
          <div className="space-y-3 rounded-xl border border-slate-200 p-4 dark:border-neutral-600">
            <p className="text-xs font-medium uppercase tracking-wide text-slate-500">{t('gradingAgent.result.title')}</p>
            <p className="text-2xl font-semibold tabular-nums">
              {dryRunResult.suggestedPoints}
              {maxPoints != null ? <span className="text-base font-normal text-slate-500"> / {maxPoints}</span> : null}
            </p>
            {rubric && dryRunResult.rubricScores ? (
              <RubricGradePicker
                rubric={rubric}
                scores={dryRunResult.rubricScores}
                onScoresChange={(scores) =>
                  setDryRunResult((prev) =>
                    prev ? { ...prev, rubricScores: scores } : prev,
                  )
                }
                compact
              />
            ) : null}
            <textarea
              value={dryRunResult.comment}
              onChange={(e) =>
                setDryRunResult((prev) => (prev ? { ...prev, comment: e.target.value } : prev))
              }
              rows={4}
              className="w-full rounded-lg border border-slate-300 px-3 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-950"
            />
            <div className="flex flex-wrap gap-2">
              <button
                type="button"
                disabled={saving || !submissionId}
                onClick={() => void handleApply()}
                className="rounded-lg bg-emerald-600 px-3 py-2 text-sm font-semibold text-white hover:bg-emerald-700 disabled:opacity-50"
              >
                {t('gradingAgent.apply')}
              </button>
              <button
                type="button"
                disabled={dryRunning}
                onClick={() => void handleDryRun()}
                className="rounded-lg border border-slate-300 px-3 py-2 text-sm font-semibold dark:border-neutral-600"
              >
                {t('gradingAgent.rerun')}
              </button>
            </div>
          </div>
        ) : null}

        {!accepted ? (
          <button
            type="button"
            disabled={!hadDryRun || saving}
            onClick={() => void handleAccept()}
            className="w-full rounded-lg border border-indigo-300 px-4 py-2 text-sm font-semibold text-indigo-700 hover:bg-indigo-50 disabled:opacity-50 dark:border-indigo-800 dark:text-indigo-300"
          >
            {t('gradingAgent.accept')}
          </button>
        ) : (
          <div className="space-y-3 border-t border-slate-200 pt-4 dark:border-neutral-700">
            <p className="text-sm font-medium text-slate-700 dark:text-neutral-200">{t('gradingAgent.run.title')}</p>
            <div className="flex flex-col gap-1" role="group" aria-label={t('gradingAgent.run.scopeLabel')}>
              {(['current', 'ungraded', 'all'] as RunScope[]).map((scope) => (
                <button
                  key={scope}
                  type="button"
                  aria-pressed={runScope === scope}
                  onClick={() => {
                    setRunScope(scope)
                    setConfirmOverwrite(false)
                  }}
                  className={`rounded-lg px-3 py-2 text-start text-sm font-medium ${
                    runScope === scope
                      ? 'bg-indigo-100 text-indigo-800 dark:bg-indigo-950/50 dark:text-indigo-200'
                      : 'bg-slate-100 text-slate-700 dark:bg-neutral-800 dark:text-neutral-300'
                  }`}
                >
                  {t(`gradingAgent.run.scope.${scope}`)}
                </button>
              ))}
            </div>
            {confirmOverwrite ? (
              <p className="text-sm text-amber-800 dark:text-amber-200">{t('gradingAgent.run.overwriteWarning')}</p>
            ) : null}
            <button
              type="button"
              disabled={saving}
              onClick={() => void handleRun()}
              className="w-full rounded-lg bg-indigo-600 px-4 py-2 text-sm font-semibold text-white hover:bg-indigo-700 disabled:opacity-50"
            >
              {confirmOverwrite ? t('gradingAgent.run.confirm') : t('gradingAgent.run.start')}
            </button>
            {runProgress ? (
              <p className="text-sm text-slate-600 dark:text-neutral-400">
                {t('gradingAgent.run.progress', {
                  completed: runProgress.completed,
                  failed: runProgress.failed,
                  total: runProgress.total,
                })}
              </p>
            ) : null}
            <label className="flex items-center gap-2 text-sm">
              <input
                type="checkbox"
                checked={Boolean(config?.autoGradeNew)}
                onChange={(e) => void handleToggleAutoGrade(e.target.checked)}
              />
              {t('gradingAgent.autoGradeNew')}
            </label>
          </div>
        )}
      </div>
    </div>
  )
}