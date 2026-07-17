import { useEffect, useId, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { fetchBoardAnalytics, type BoardAnalyticsSummary } from '../../lib/boards-api'

type Props = {
  open: boolean
  onClose: () => void
  courseCode: string
  boardId: string
}

function formatBytesDay(day: string): string {
  try {
    return new Date(day).toLocaleDateString(undefined, { month: 'short', day: 'numeric' })
  } catch {
    return day
  }
}

export function BoardAnalyticsPanel({ open, onClose, courseCode, boardId }: Props) {
  const { t } = useTranslation('common')
  const titleId = useId()
  const [data, setData] = useState<BoardAnalyticsSummary | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!open) return
    setLoading(true)
    setError(null)
    void fetchBoardAnalytics(courseCode, boardId)
      .then(setData)
      .catch((err) => setError(err instanceof Error ? err.message : String(err)))
      .finally(() => setLoading(false))
  }, [open, courseCode, boardId])

  if (!open) return null

  const maxCards = Math.max(1, ...(data?.daily.map((d) => d.cardCount) ?? [1]))

  return (
    <div
      className="fixed inset-0 z-40 flex items-end justify-center bg-black/40 p-4 sm:items-center"
      role="presentation"
      onClick={onClose}
    >
      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        className="max-h-[90vh] w-full max-w-2xl overflow-auto rounded-lg border border-slate-200 bg-white p-5 shadow-xl dark:border-neutral-700 dark:bg-neutral-900"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-start justify-between gap-3">
          <div>
            <h2 id={titleId} className="text-lg font-semibold text-slate-900 dark:text-neutral-100">
              {t('boards.analytics.title')}
            </h2>
            <p className="mt-1 text-sm text-slate-600 dark:text-neutral-400">
              {t('boards.analytics.subtitle')}
            </p>
          </div>
          <button
            type="button"
            className="rounded-md px-2 py-1 text-sm text-slate-600 hover:bg-slate-100 dark:text-neutral-300 dark:hover:bg-neutral-800"
            onClick={onClose}
          >
            {t('dialogs.close')}
          </button>
        </div>

        {loading ? (
          <p className="mt-6 text-sm text-slate-500" role="status">
            {t('common.loading')}
          </p>
        ) : error ? (
          <p className="mt-6 text-sm text-red-600 dark:text-red-400" role="alert">
            {error}
          </p>
        ) : !data || (data.cardCount === 0 && data.uniqueContributors === 0) ? (
          <p className="mt-6 text-sm text-slate-600 dark:text-neutral-400">{t('boards.analytics.empty')}</p>
        ) : (
          <div className="mt-6 space-y-6">
            <dl className="grid grid-cols-2 gap-3 sm:grid-cols-4">
              <div className="rounded-md bg-slate-50 p-3 dark:bg-neutral-800">
                <dt className="text-xs text-slate-500 dark:text-neutral-400">{t('boards.analytics.cards')}</dt>
                <dd className="text-xl font-semibold text-slate-900 dark:text-neutral-100">{data.cardCount}</dd>
              </div>
              <div className="rounded-md bg-slate-50 p-3 dark:bg-neutral-800">
                <dt className="text-xs text-slate-500 dark:text-neutral-400">
                  {t('boards.analytics.contributors')}
                </dt>
                <dd className="text-xl font-semibold text-slate-900 dark:text-neutral-100">
                  {data.uniqueContributors}
                </dd>
              </div>
              <div className="rounded-md bg-slate-50 p-3 dark:bg-neutral-800">
                <dt className="text-xs text-slate-500 dark:text-neutral-400">{t('boards.analytics.reactions')}</dt>
                <dd className="text-xl font-semibold text-slate-900 dark:text-neutral-100">
                  {data.reactionCount}
                </dd>
              </div>
              <div className="rounded-md bg-slate-50 p-3 dark:bg-neutral-800">
                <dt className="text-xs text-slate-500 dark:text-neutral-400">{t('boards.analytics.comments')}</dt>
                <dd className="text-xl font-semibold text-slate-900 dark:text-neutral-100">
                  {data.commentCount}
                </dd>
              </div>
            </dl>

            {data.daily.length > 0 ? (
              <section aria-labelledby={`${titleId}-spark`}>
                <h3 id={`${titleId}-spark`} className="text-sm font-medium text-slate-800 dark:text-neutral-200">
                  {t('boards.analytics.activity')}
                </h3>
                <div
                  className="mt-2 flex h-16 items-end gap-1"
                  role="img"
                  aria-label={t('boards.analytics.activityChartAria')}
                >
                  {data.daily.map((d) => (
                    <div
                      key={d.day}
                      className="flex-1 rounded-t bg-indigo-500/80 dark:bg-indigo-400/70"
                      style={{ height: `${Math.max(8, (d.cardCount / maxCards) * 100)}%` }}
                      title={`${formatBytesDay(d.day)}: ${d.cardCount}`}
                    />
                  ))}
                </div>
                <table className="mt-3 w-full text-start text-sm">
                  <caption className="sr-only">{t('boards.analytics.activityTable')}</caption>
                  <thead>
                    <tr className="border-b border-slate-200 text-slate-500 dark:border-neutral-700 dark:text-neutral-400">
                      <th scope="col" className="py-1 font-medium">
                        {t('boards.analytics.day')}
                      </th>
                      <th scope="col" className="py-1 font-medium">
                        {t('boards.analytics.cards')}
                      </th>
                      <th scope="col" className="py-1 font-medium">
                        {t('boards.analytics.contributors')}
                      </th>
                    </tr>
                  </thead>
                  <tbody>
                    {data.daily.map((d) => (
                      <tr key={d.day} className="border-b border-slate-100 dark:border-neutral-800">
                        <td className="py-1">{formatBytesDay(d.day)}</td>
                        <td className="py-1">{d.cardCount}</td>
                        <td className="py-1">{d.contributorCount}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </section>
            ) : null}

            <section aria-labelledby={`${titleId}-contrib`}>
              <h3 id={`${titleId}-contrib`} className="text-sm font-medium text-slate-800 dark:text-neutral-200">
                {t('boards.analytics.contributorList')}
              </h3>
              {data.contributors.length === 0 ? (
                <p className="mt-2 text-sm text-slate-500">{t('boards.analytics.noContributors')}</p>
              ) : (
                <table className="mt-2 w-full text-start text-sm">
                  <caption className="sr-only">{t('boards.analytics.contributorList')}</caption>
                  <thead>
                    <tr className="border-b border-slate-200 text-slate-500 dark:border-neutral-700 dark:text-neutral-400">
                      <th scope="col" className="py-1 font-medium">
                        {t('boards.analytics.participant')}
                      </th>
                      <th scope="col" className="py-1 font-medium">
                        {t('boards.analytics.posts')}
                      </th>
                      <th scope="col" className="py-1 font-medium">
                        {t('boards.analytics.comments')}
                      </th>
                      <th scope="col" className="py-1 font-medium">
                        {t('boards.analytics.reactions')}
                      </th>
                      <th scope="col" className="py-1 font-medium">
                        {t('boards.analytics.total')}
                      </th>
                    </tr>
                  </thead>
                  <tbody>
                    {data.contributors.map((c) => (
                      <tr key={c.userId} className="border-b border-slate-100 dark:border-neutral-800">
                        <td className="py-1 font-mono text-xs">{c.userId.slice(0, 8)}…</td>
                        <td className="py-1">{c.postCount}</td>
                        <td className="py-1">{c.commentCount}</td>
                        <td className="py-1">{c.reactionCount}</td>
                        <td className="py-1">{c.contributionTotal}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              )}
            </section>
          </div>
        )}
      </div>
    </div>
  )
}
