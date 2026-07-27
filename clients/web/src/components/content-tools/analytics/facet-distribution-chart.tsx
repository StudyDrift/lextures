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
    <div className="space-y-2" data-testid={`facet-chart-${facetKey}`}>
      <div className="flex items-center justify-between gap-2">
        <h4 className="text-sm font-medium text-slate-800 dark:text-neutral-100">
          {t(label, { defaultValue: label })}
        </h4>
        <button
          type="button"
          className="text-xs text-sky-700 underline dark:text-sky-300"
          onClick={() => setShowTable((v) => !v)}
        >
          {showTable
            ? t('contentTools.analytics.showChart')
            : t('contentTools.analytics.showTable')}
        </button>
      </div>
      {showTable ? (
        <table className="w-full text-left text-sm" data-testid={`facet-table-${facetKey}`}>
          <caption className="sr-only">{t(label, { defaultValue: label })}</caption>
          <thead>
            <tr className="border-b border-slate-200 dark:border-neutral-700">
              <th className="py-1 pr-2 font-medium">{t('contentTools.analytics.colValue')}</th>
              <th className="py-1 font-medium">{t('contentTools.analytics.colCount')}</th>
            </tr>
          </thead>
          <tbody>
            {values.map((v) => (
              <tr key={v.value} className="border-b border-slate-100 dark:border-neutral-800">
                <td className="py-1 pr-2">{v.value}</td>
                <td className="py-1">{v.count}</td>
              </tr>
            ))}
          </tbody>
        </table>
      ) : (
        <div
          role="img"
          aria-label={summary}
          className="space-y-1.5"
        >
          {values.map((v) => (
            <div key={v.value} className="flex items-center gap-2 text-sm">
              <span className="w-20 shrink-0 truncate text-slate-600 dark:text-neutral-300">
                {v.value}
              </span>
              <div className="h-2 flex-1 rounded bg-slate-100 dark:bg-neutral-800">
                <div
                  className={`h-2 rounded ${v.correct === false ? 'bg-amber-600' : 'bg-sky-600'}`}
                  style={{ width: `${(v.count / max) * 100}%` }}
                />
              </div>
              <span className="w-8 text-right tabular-nums text-slate-700 dark:text-neutral-200">
                {v.count}
              </span>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
