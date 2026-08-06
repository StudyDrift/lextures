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
      <section className="mt-4 rounded-xl border border-border-default p-4 text-sm text-fg-muted dark:border-border-default dark:text-fg-muted">
        {t('gradingAgent.review.history.loading')}
      </section>
    )
  }

  if (runs.length === 0) return null

  return (
    <section
      className="mt-4 rounded-xl border border-border-default bg-surface-raised p-4 dark:border-border-default dark:bg-surface-base"
      aria-label={t('gradingAgent.review.history.title')}
    >
      <h3 className="text-sm font-semibold text-fg-default">
        {t('gradingAgent.review.history.title')}
      </h3>
      <ul className="mt-3 space-y-2">
        {runs.map((run) => (
          <li
            key={run.id}
            className="rounded-lg border border-border-default px-3 py-2 text-sm dark:border-border-default"
          >
            <div className="flex flex-wrap items-center justify-between gap-2">
              <span className="font-medium text-fg-default">
                {t(`gradingAgent.review.history.scope.${run.scope}`, { defaultValue: run.scope })}
              </span>
              <span className="text-xs uppercase tracking-wide text-fg-muted">
                {t(`gradingAgent.review.history.status.${run.status}`, { defaultValue: run.status })}
              </span>
            </div>
            <p className="mt-1 text-xs text-fg-muted">
              {t('gradingAgent.review.history.counts', {
                completed: run.completedCount,
                failed: run.failedCount,
                total: run.totalCount,
              })}
            </p>
            {run.model ? (
              <p className="mt-1 text-xs text-fg-muted">
                {t('gradingAgent.review.history.model', { model: run.model })}
              </p>
            ) : null}
            {run.costUsd != null ? (
              <p className="mt-1 text-xs text-fg-muted">
                {t('gradingAgent.review.history.cost', { cost: run.costUsd.toFixed(4) })}
              </p>
            ) : null}
            {run.promptTokens != null || run.completionTokens != null ? (
              <p className="mt-1 text-xs text-fg-muted">
                {t('gradingAgent.review.history.tokens', {
                  prompt: run.promptTokens ?? 0,
                  completion: run.completionTokens ?? 0,
                })}
              </p>
            ) : null}
            <p className="mt-1 text-xs text-fg-muted">
              {formatAbsolute(run.createdAt)}
            </p>
          </li>
        ))}
      </ul>
    </section>
  )
}