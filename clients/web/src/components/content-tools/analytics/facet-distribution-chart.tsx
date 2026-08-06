import { useState } from 'react'
import { useTranslation } from 'react-i18next'

export type FacetDistributionChartProps = {
  facetKey: string
  label: string
  values: Array<{ value: string; count: number; correct?: boolean | null }>
}

export function FacetDistributionChart({ facetKey, label, values }: FacetDistributionChartProps) {
  const { t } = useTranslation('contentTools')
  const [showTable, setShowTable] = useState(false)
  const max = Math.max(1, ...values.map((v) => v.count))
  const summary = values.map((v) => `${v.value}: ${v.count}`).join(', ')

  return (
    <div
      className="space-y-2 rounded-xl border border-border-subtle bg-slate-50/60 p-3 dark:border-border-subtle/50"
      data-testid={`facet-chart-${facetKey}`}
    >
      <div className="flex items-center justify-between gap-2">
        <h4 className="text-sm font-semibold text-fg-default">
          {t(label, { defaultValue: label })}
        </h4>
        <button
          type="button"
          className="rounded-md px-2 py-1 text-xs font-medium text-accent-fg hover:bg-indigo-50 dark:text-indigo-300 dark:hover:bg-indigo-950/40"
          onClick={() => setShowTable((v) => !v)}
        >
          {showTable
            ? t('contentTools.analytics.showChart')
            : t('contentTools.analytics.showTable')}
        </button>
      </div>
      {showTable ? (
        <table className="w-full text-start text-sm" data-testid={`facet-table-${facetKey}`}>
          <caption className="sr-only">{t(label, { defaultValue: label })}</caption>
          <thead>
            <tr className="border-b border-border-default">
              <th className="py-1.5 pe-2 text-xs font-semibold uppercase tracking-wide text-fg-muted">
                {t('contentTools.analytics.colValue')}
              </th>
              <th className="py-1.5 text-xs font-semibold uppercase tracking-wide text-fg-muted">
                {t('contentTools.analytics.colCount')}
              </th>
            </tr>
          </thead>
          <tbody>
            {values.map((v) => (
              <tr key={v.value} className="border-b border-border-subtle">
                <td className="py-1.5 pe-2 text-fg-default">{v.value}</td>
                <td className="py-1.5 tabular-nums text-fg-muted">{v.count}</td>
              </tr>
            ))}
          </tbody>
        </table>
      ) : (
        <div role="img" aria-label={summary} className="space-y-1.5">
          {values.map((v) => (
            <div key={v.value} className="flex items-center gap-2 text-sm">
              <span className="w-20 shrink-0 truncate text-fg-muted">
                {v.value}
              </span>
              <div className="h-2 flex-1 rounded-full bg-slate-200/80 dark:bg-surface-overlay">
                <div
                  className={`h-2 rounded-full ${ v.correct === false ? 'bg-amber-500' : 'bg-indigo-500 dark:bg-indigo-400' }`}
                  style={{ width: `${(v.count / max) * 100}%` }}
                />
              </div>
              <span className="w-8 text-end tabular-nums text-fg-default">
                {v.count}
              </span>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
