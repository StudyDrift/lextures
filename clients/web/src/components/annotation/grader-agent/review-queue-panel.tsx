import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import type { GraderAgentRunStatus } from '../../../lib/courses-api'

type ReviewQueuePanelProps = {
  runResults: GraderAgentRunStatus['results']
  submissionLabelById?: Record<string, string>
}

export function ReviewQueuePanel({ runResults, submissionLabelById = {} }: ReviewQueuePanelProps) {
  const { t } = useTranslation('common')
  const flagged = useMemo(
    () => runResults.filter((result) => result.status === 'flagged'),
    [runResults],
  )

  if (flagged.length === 0) return null

  return (
    <section
      className="rounded-xl border border-rose-200 bg-rose-50/50 p-4 dark:border-rose-900/50 dark:bg-rose-950/20"
      aria-label={t('gradingAgent.review.flagged.title')}
    >
      <div className="flex items-center justify-between gap-3">
        <h3 className="text-sm font-semibold text-rose-900 dark:text-rose-100">
          {t('gradingAgent.review.flagged.title')}
        </h3>
        <span
          className="rounded-full bg-rose-600 px-2 py-0.5 text-xs font-semibold text-white"
          aria-live="polite"
        >
          {flagged.length}
        </span>
      </div>
      <ul className="mt-3 space-y-2">
        {flagged.map((item) => (
          <li
            key={item.submissionId}
            className="rounded-lg border border-rose-200/80 bg-white px-3 py-2 text-sm dark:border-rose-900/40 dark:bg-neutral-950"
          >
            <p className="font-medium text-slate-900 dark:text-neutral-50">
              {submissionLabelById[item.submissionId] ?? item.submissionId.slice(0, 8)}
            </p>
            <p className="mt-1 text-xs font-semibold uppercase tracking-wide text-rose-700 dark:text-rose-300">
              {t('gradingAgent.review.flagged.badge')}
            </p>
            {item.flagReason ? (
              <p className="mt-1 text-slate-700 dark:text-neutral-300">{item.flagReason}</p>
            ) : null}
            {item.flagPriority ? (
              <p className="mt-1 text-xs text-slate-500 dark:text-neutral-400">
                {t('gradingAgent.review.flagged.priority')}: {item.flagPriority}
              </p>
            ) : null}
          </li>
        ))}
      </ul>
    </section>
  )
}