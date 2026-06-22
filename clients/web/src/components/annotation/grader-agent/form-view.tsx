import { useTranslation } from 'react-i18next'
import type { GraderAgentWorkflowState } from './use-grader-agent-workflow'
import { primaryValidationMessage } from './use-grader-agent-workflow'

type FormViewProps = {
  workflow: GraderAgentWorkflowState
}

export function FormView({ workflow }: FormViewProps) {
  const { t } = useTranslation('common')
  const {
    graph,
    updateGraderNode,
    updateContextNode,
    validationIssues,
    runnable,
    dryRunning,
    saving,
    hadDryRun,
    config,
    runScope,
    setRunScope,
    confirmOverwrite,
    setConfirmOverwrite,
    runProgress,
    handleDryRun,
    handleAccept,
    handleRun,
    handleToggleAutoGrade,
  } = workflow

  const grader = graph?.nodes.find((n) => n.type === 'grader')
  const context = graph?.nodes.find((n) => n.type === 'assignmentContext')
  const gradeEdge = graph?.edges.find((e) => e.target === 'output' && e.targetHandle === 'grade')
  const commentsEdge = graph?.edges.find((e) => e.target === 'output' && e.targetHandle === 'comments')
  const accepted = config?.status === 'accepted'
  const validationMsg = primaryValidationMessage(validationIssues)

  return (
    <div className="space-y-4">
      {grader ? (
        <label className="block text-sm text-slate-700 dark:text-neutral-200">
          <span className="mb-1.5 block font-medium">{t('gradingAgent.prompt.label')}</span>
          <textarea
            value={typeof grader.data.prompt === 'string' ? grader.data.prompt : ''}
            onChange={(e) => updateGraderNode(grader.id, { prompt: e.target.value })}
            rows={5}
            disabled={accepted}
            className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-950"
            placeholder={t('gradingAgent.prompt.placeholder')}
          />
        </label>
      ) : null}

      {context ? (
        <div className="space-y-2">
          <label className="flex items-center gap-2 text-sm text-slate-700 dark:text-neutral-200">
            <input
              type="checkbox"
              checked={Boolean(context.data.includeContent)}
              onChange={(e) =>
                updateContextNode(context.id, {
                  includeContent: e.target.checked,
                  includeRubric: e.target.checked,
                })
              }
              disabled={accepted}
            />
            {t('gradingAgent.includeContentRubric')}
          </label>
        </div>
      ) : null}

      <div className="rounded-lg border border-slate-200 p-3 text-sm dark:border-neutral-600">
        <p className="font-medium text-slate-800 dark:text-neutral-100">{t('gradingAgent.canvas.form.bindings')}</p>
        <ul className="mt-2 space-y-1 text-slate-600 dark:text-neutral-300">
          <li>
            {t('gradingAgent.canvas.slots.grade')}:{' '}
            {gradeEdge ? t('gradingAgent.canvas.form.boundGrader') : t('gradingAgent.canvas.form.notSet')}
          </li>
          <li>
            {t('gradingAgent.canvas.slots.comments')}:{' '}
            {commentsEdge ? t('gradingAgent.canvas.form.boundGrader') : t('gradingAgent.canvas.form.notSet')}
          </li>
        </ul>
      </div>

      {validationMsg ? (
        <p className="rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-sm text-amber-900 dark:border-amber-900/50 dark:bg-amber-950/30 dark:text-amber-100">
          {validationMsg}
        </p>
      ) : null}

      {!accepted ? (
        <>
          <button
            type="button"
            disabled={dryRunning || !runnable}
            onClick={() => void handleDryRun()}
            className="w-full rounded-lg bg-indigo-600 px-4 py-2 text-sm font-semibold text-white hover:bg-indigo-700 disabled:opacity-50"
          >
            {dryRunning ? t('gradingAgent.dryRun.running') : t('gradingAgent.dryRun')}
          </button>
          <button
            type="button"
            disabled={!hadDryRun || saving}
            onClick={() => void handleAccept()}
            className="w-full rounded-lg border border-indigo-300 px-4 py-2 text-sm font-semibold text-indigo-700 hover:bg-indigo-50 disabled:opacity-50 dark:border-indigo-800 dark:text-indigo-300"
          >
            {t('gradingAgent.accept')}
          </button>
        </>
      ) : (
        <div className="space-y-3 border-t border-slate-200 pt-4 dark:border-neutral-700">
          <p className="text-sm font-medium">{t('gradingAgent.run.title')}</p>
          {(['current', 'ungraded', 'all'] as const).map((scope) => (
            <button
              key={scope}
              type="button"
              aria-pressed={runScope === scope}
              onClick={() => {
                setRunScope(scope)
                setConfirmOverwrite(false)
              }}
              className={`block w-full rounded-lg px-3 py-2 text-start text-sm font-medium ${
                runScope === scope
                  ? 'bg-indigo-100 text-indigo-800 dark:bg-indigo-950/50 dark:text-indigo-200'
                  : 'bg-slate-100 text-slate-700 dark:bg-neutral-800 dark:text-neutral-300'
              }`}
            >
              {t(`gradingAgent.run.scope.${scope}`)}
            </button>
          ))}
          {confirmOverwrite ? (
            <p className="text-sm text-amber-800 dark:text-amber-200">{t('gradingAgent.run.overwriteWarning')}</p>
          ) : null}
          <button
            type="button"
            disabled={saving || !runnable}
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
  )
}
