import { lazy, Suspense, useEffect, useId, useRef } from 'react'
import { useTranslation } from 'react-i18next'
import type { RubricDefinition } from '../../../lib/courses-api'
import { FormView } from './form-view'
import { InspectorPanel } from './inspector-panel'
import { PreviewDock } from './preview-dock'
import { primaryValidationMessage, useGraderAgentWorkflow } from './use-grader-agent-workflow'

const CanvasView = lazy(() =>
  import('./canvas-view').then((m) => ({ default: m.CanvasView })),
)

type GraderAgentWorkflowModalProps = {
  open: boolean
  onClose: () => void
  courseCode: string
  itemId: string
  assignmentTitle?: string
  submissionId: string | null
  rubric: RubricDefinition | null
  maxPoints: number | null
  onApplied?: () => void
}

export function GraderAgentWorkflowModal({
  open,
  onClose,
  courseCode,
  itemId,
  assignmentTitle,
  submissionId,
  rubric,
  maxPoints,
  onApplied,
}: GraderAgentWorkflowModalProps) {
  const { t } = useTranslation('common')
  const titleId = useId()
  const statusId = useId()
  const modalRef = useRef<HTMLDivElement>(null)
  const closeRef = useRef<HTMLButtonElement>(null)
  const workflow = useGraderAgentWorkflow({ open, courseCode, itemId, submissionId, onApplied })

  const {
    config,
    viewMode,
    setViewMode,
    dryRunning,
    dryRunError,
    hadDryRun,
    saving,
    runnable,
    validationIssues,
    runScope,
    setRunScope,
    confirmOverwrite,
    setConfirmOverwrite,
    runProgress,
    statusMessage,
    handleDryRun,
    handleAccept,
    handleRun,
    handleToggleAutoGrade,
    addGraderNode,
    addContextNode,
  } = workflow

  const accepted = config?.status === 'accepted'
  const validationMsg = primaryValidationMessage(validationIssues)

  useEffect(() => {
    if (!open) return
    closeRef.current?.focus()
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose()
    }
    document.addEventListener('keydown', onKey)
    return () => document.removeEventListener('keydown', onKey)
  }, [open, onClose])

  if (!open) return null

  return (
    <div className="fixed inset-0 z-[520] flex flex-col bg-white dark:bg-neutral-950" role="presentation">
      <div
        ref={modalRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        className="flex h-full flex-col"
      >
        <header className="flex shrink-0 flex-wrap items-center gap-3 border-b border-slate-200 px-4 py-3 dark:border-neutral-700">
          <div className="min-w-0 flex-1">
            <h2 id={titleId} className="text-lg font-semibold text-slate-900 dark:text-neutral-50">
              {t('gradingAgent.canvas.modal.title')}
            </h2>
            {assignmentTitle ? (
              <p className="truncate text-sm text-slate-500 dark:text-neutral-400">{assignmentTitle}</p>
            ) : null}
          </div>
          <div
            className="flex rounded-lg border border-slate-200 p-0.5 dark:border-neutral-600"
            role="group"
            aria-label={t('gradingAgent.canvas.viewToggle.label')}
          >
            {(['canvas', 'form'] as const).map((mode) => (
              <button
                key={mode}
                type="button"
                aria-pressed={viewMode === mode}
                onClick={() => setViewMode(mode)}
                className={`rounded-md px-3 py-1.5 text-sm font-medium ${
                  viewMode === mode
                    ? 'bg-indigo-600 text-white'
                    : 'text-slate-700 hover:bg-slate-100 dark:text-neutral-200 dark:hover:bg-neutral-800'
                }`}
              >
                {t(`gradingAgent.canvas.viewToggle.${mode}`)}
              </button>
            ))}
          </div>
          {!accepted && viewMode === 'canvas' ? (
            <button
              type="button"
              disabled={dryRunning || !runnable || !submissionId}
              onClick={() => void handleDryRun()}
              className="rounded-lg bg-indigo-600 px-3 py-2 text-sm font-semibold text-white hover:bg-indigo-700 disabled:opacity-50"
            >
              {dryRunning ? t('gradingAgent.dryRun.running') : t('gradingAgent.dryRun')}
            </button>
          ) : null}
          {accepted ? (
            <button
              type="button"
              disabled={saving || !runnable}
              onClick={() => void handleRun()}
              className="rounded-lg bg-indigo-600 px-3 py-2 text-sm font-semibold text-white hover:bg-indigo-700 disabled:opacity-50"
            >
              {confirmOverwrite ? t('gradingAgent.run.confirm') : t('gradingAgent.run.start')}
            </button>
          ) : (
            <button
              type="button"
              disabled={!hadDryRun || saving}
              onClick={() => void handleAccept()}
              className="rounded-lg border border-indigo-300 px-3 py-2 text-sm font-semibold text-indigo-700 dark:border-indigo-800 dark:text-indigo-300 disabled:opacity-50"
            >
              {t('gradingAgent.accept')}
            </button>
          )}
          <button
            ref={closeRef}
            type="button"
            onClick={onClose}
            className="rounded-lg px-2 py-1 text-sm text-slate-600 hover:bg-slate-100 dark:text-neutral-300 dark:hover:bg-neutral-800"
          >
            {t('gradingAgent.close')}
          </button>
        </header>

        <div id={statusId} role="status" aria-live="polite" className="sr-only">
          {statusMessage}
        </div>

        {dryRunError || validationMsg ? (
          <div className="border-b border-rose-200 bg-rose-50 px-4 py-2 text-sm text-rose-800 dark:border-rose-900/60 dark:bg-rose-950/40 dark:text-rose-200">
            {dryRunError ?? validationMsg}
          </div>
        ) : null}

        <div className="flex min-h-0 flex-1 flex-col lg:flex-row">
          {viewMode === 'canvas' ? (
            <>
              <aside className="w-full shrink-0 border-b border-slate-200 p-3 lg:w-48 lg:border-b-0 lg:border-e dark:border-neutral-700">
                <p className="mb-2 text-xs font-semibold uppercase tracking-wide text-slate-500">
                  {t('gradingAgent.canvas.palette.title')}
                </p>
                {!accepted ? (
                  <div className="flex flex-col gap-2">
                    <button
                      type="button"
                      onClick={addGraderNode}
                      className="rounded-lg border border-indigo-200 px-3 py-2 text-start text-sm font-medium text-indigo-800 hover:bg-indigo-50 dark:border-indigo-900 dark:text-indigo-200"
                    >
                      {t('gradingAgent.canvas.palette.grader')}
                    </button>
                    <button
                      type="button"
                      onClick={addContextNode}
                      className="rounded-lg border border-amber-200 px-3 py-2 text-start text-sm font-medium text-amber-900 hover:bg-amber-50 dark:border-amber-900 dark:text-amber-200"
                    >
                      {t('gradingAgent.canvas.palette.context')}
                    </button>
                  </div>
                ) : null}
                <p className="mt-4 text-xs text-slate-400 dark:text-neutral-500">
                  {t('gradingAgent.canvas.palette.comingSoon')}
                </p>
              </aside>
              <main className="min-h-0 flex-1 p-3">
                <Suspense fallback={<div className="h-full motion-safe:animate-pulse rounded-xl bg-slate-100 dark:bg-neutral-800" />}>
                  <CanvasView workflow={workflow} readOnly={accepted} />
                </Suspense>
              </main>
              <aside className="w-full shrink-0 border-t border-slate-200 p-3 lg:w-72 lg:border-t-0 lg:border-s dark:border-neutral-700">
                <p className="mb-2 text-xs font-semibold uppercase tracking-wide text-slate-500">
                  {t('gradingAgent.canvas.inspector.title')}
                </p>
                <InspectorPanel workflow={workflow} accepted={accepted} />
              </aside>
            </>
          ) : (
            <main className="flex-1 overflow-y-auto p-4">
              <FormView workflow={workflow} />
            </main>
          )}
        </div>

        <footer className="shrink-0 border-t border-slate-200 p-4 dark:border-neutral-700">
          <PreviewDock
            workflow={workflow}
            rubric={rubric}
            maxPoints={maxPoints}
            submissionId={submissionId}
          />
          {accepted && viewMode === 'canvas' ? (
            <div className="mt-4 space-y-2">
              <p className="text-sm font-medium">{t('gradingAgent.run.title')}</p>
              <div className="flex flex-wrap gap-2">
                {(['current', 'ungraded', 'all'] as const).map((scope) => (
                  <button
                    key={scope}
                    type="button"
                    aria-pressed={runScope === scope}
                    onClick={() => {
                      setRunScope(scope)
                      setConfirmOverwrite(false)
                    }}
                    className={`rounded-lg px-3 py-1.5 text-sm font-medium ${
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
          ) : null}
        </footer>
      </div>
    </div>
  )
}
