import { useTranslation } from 'react-i18next'
import type { GraderAgentRunHistoryEntry } from '../../../lib/courses-api'
import { formatAbsolute } from '../../../lib/format-datetime'

type RunHistoryPanelProps = {
  runs: GraderAgentRunHistoryEntry[]
  loading?: boolean
}

export function RunHistoryPanel({ runs, loading = false }: RunHistoryPanelProps) {
  const { t } = useTranslation('common')

  if (loading && runs.length === 0) {
    return (
      <section className="mt-4 rounded-xl border border-slate-200 p-4 text-sm text-slate-600 dark:border-neutral-700 dark:text-neutral-300">
        {t('gradingAgent.review.history.loading')}
      </section>
    )
  }

  if (runs.length === 0) return null

  return (
    <section
      className="mt-4 rounded-xl border border-slate-200 bg-white p-4 dark:border-neutral-700 dark:bg-neutral-950"
      aria-label={t('gradingAgent.review.history.title')}
    >
      <h3 className="text-sm font-semibold text-slate-900 dark:text-neutral-100">
        {t('gradingAgent.review.history.title')}
      </h3>
      <ul className="mt-3 space-y-2">
        {runs.map((run) => (
          <li
            key={run.id}
            className="rounded-lg border border-slate-200 px-3 py-2 text-sm dark:border-neutral-700"
          >
            <div className="flex flex-wrap items-center justify-between gap-2">
              <span className="font-medium text-slate-900 dark:text-neutral-50">
                {t(`gradingAgent.review.history.scope.${run.scope}`, { defaultValue: run.scope })}
              </span>
              <span className="text-xs uppercase tracking-wide text-slate-500 dark:text-neutral-400">
                {t(`gradingAgent.review.history.status.${run.status}`, { defaultValue: run.status })}
              </span>
            </div>
            <p className="mt-1 text-xs text-slate-600 dark:text-neutral-400">
              {t('gradingAgent.review.history.counts', {
                completed: run.completedCount,
                failed: run.failedCount,
                total: run.totalCount,
              })}
            </p>
            {run.model ? (
              <p className="mt-1 text-xs text-slate-500 dark:text-neutral-400">
                {t('gradingAgent.review.history.model', { model: run.model })}
              </p>
            ) : null}
            {run.costUsd != null ? (
              <p className="mt-1 text-xs text-slate-500 dark:text-neutral-400">
                {t('gradingAgent.review.history.cost', { cost: run.costUsd.toFixed(4) })}
              </p>
            ) : null}
            {run.promptTokens != null || run.completionTokens != null ? (
              <p className="mt-1 text-xs text-slate-500 dark:text-neutral-400">
                {t('gradingAgent.review.history.tokens', {
                  prompt: run.promptTokens ?? 0,
                  completion: run.completionTokens ?? 0,
                })}
              </p>
            ) : null}
            <p className="mt-1 text-xs text-slate-500 dark:text-neutral-400">
              {formatAbsolute(run.createdAt)}
            </p>
          </li>
        ))}
      </ul>
    </section>
  )
}