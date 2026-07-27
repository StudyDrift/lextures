import { useTranslation } from 'react-i18next'

export type ConfusionRow = {
  itemId: string
  itemLabel: string
  distributions: Array<{ placedIn: string; count: number; isCorrect?: boolean }>
  mostCommonError?: { placedIn: string; count: number } | null
}

export type SortConfusionViewProps = {
  rows: ConfusionRow[]
}

/** Instructor confusion map: per-item placement distribution (CT.14 FR-13 / AC-7). */
export function SortConfusionView({ rows }: SortConfusionViewProps) {
  const { t } = useTranslation('contentTools')
  if (rows.length === 0) return null

  return (
    <div className="space-y-3" data-testid="sort-confusion-view">
      <h3 className="text-sm font-medium text-slate-800 dark:text-neutral-100">
        {t('contentTools.tools.sort_sequence.confusion.title')}
      </h3>
      <ul className="space-y-3">
        {rows.map((row) => {
          const max = Math.max(1, ...row.distributions.map((d) => d.count))
          return (
            <li
              key={row.itemId}
              className="rounded border border-slate-200 p-3 dark:border-neutral-700"
              data-testid={`confusion-item-${row.itemId}`}
            >
              <div className="mb-2 text-sm font-medium text-slate-900 dark:text-neutral-100">
                {row.itemLabel}
              </div>
              {row.mostCommonError ? (
                <p className="mb-2 text-xs text-amber-800 dark:text-amber-200">
                  {t('contentTools.tools.sort_sequence.confusion.mostCommonError', {
                    placedIn: row.mostCommonError.placedIn,
                    count: row.mostCommonError.count,
                  })}
                </p>
              ) : null}
              <div className="space-y-1.5" role="img" aria-label={row.itemLabel}>
                {row.distributions.map((d) => (
                  <div key={d.placedIn} className="flex items-center gap-2 text-sm">
                    <span className="w-28 shrink-0 truncate text-slate-600 dark:text-neutral-300">
                      {d.placedIn}
                    </span>
                    <div className="h-2 flex-1 rounded bg-slate-100 dark:bg-neutral-800">
                      <div
                        className={`h-2 rounded ${
                          d.isCorrect === false
                            ? 'bg-amber-600'
                            : d.isCorrect
                              ? 'bg-emerald-600'
                              : 'bg-sky-600'
                        }`}
                        style={{ width: `${(d.count / max) * 100}%` }}
                      />
                    </div>
                    <span className="w-8 text-end tabular-nums">{d.count}</span>
                  </div>
                ))}
              </div>
            </li>
          )
        })}
      </ul>
    </div>
  )
}
