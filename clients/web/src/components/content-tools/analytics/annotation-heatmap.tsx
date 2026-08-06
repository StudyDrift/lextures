import { useTranslation } from 'react-i18next'

export type HeatmapUnit = {
  unitIndex: number
  label: string
  count: number
  byTag?: Array<{ tagId: string; label: string; count: number }>
}

export type AnnotationHeatmapProps = {
  units: HeatmapUnit[]
  maxCount?: number
}

/** Instructor heat map with accessible table alternative (CT.13 FR-9). */
export function AnnotationHeatmap({ units, maxCount }: AnnotationHeatmapProps) {
  const { t } = useTranslation('contentTools')
  if (!units.length) return null
  const peak = maxCount ?? Math.max(1, ...units.map((u) => u.count))

  return (
    <div className="space-y-3" data-testid="annotation-heatmap">
      <h3 className="text-sm font-medium text-fg-default">
        {t('contentTools.tools.highlight_annotate.heatmap.title')}
      </h3>
      <div className="space-y-1" aria-hidden="true">
        {units.map((u) => {
          const intensity = u.count / peak
          return (
            <div
              key={u.unitIndex}
              className="flex items-start gap-2 text-xs"
              title={t('contentTools.tools.highlight_annotate.heatmap.unitCount', {
                n: u.unitIndex + 1,
                count: u.count,
              })}
            >
              <span className="w-8 shrink-0 text-fg-muted">{u.unitIndex + 1}</span>
              <div
                className="min-h-[1.5rem] flex-1 rounded px-2 py-1 text-fg-default"
                style={{
                  backgroundColor: `rgba(15, 118, 110, ${0.08 + intensity * 0.55})`,
                }}
              >
                <span className="line-clamp-2">{u.label}</span>
              </div>
              <span className="w-6 shrink-0 text-end tabular-nums text-fg-muted">{u.count}</span>
            </div>
          )
        })}
      </div>

      <div className="overflow-x-auto">
        <table className="min-w-full border-collapse text-sm" role="table">
          <caption className="mb-1 text-start text-xs text-fg-muted">
            {t('contentTools.tools.highlight_annotate.heatmap.tableCaption')}
          </caption>
          <thead>
            <tr className="border-b border-border-default text-start dark:border-border-default">
              <th scope="col" className="px-2 py-1 font-medium">
                {t('contentTools.tools.highlight_annotate.heatmap.colUnit')}
              </th>
              <th scope="col" className="px-2 py-1 font-medium">
                {t('contentTools.tools.highlight_annotate.heatmap.colText')}
              </th>
              <th scope="col" className="px-2 py-1 font-medium">
                {t('contentTools.tools.highlight_annotate.heatmap.colCount')}
              </th>
            </tr>
          </thead>
          <tbody>
            {units.map((u) => (
              <tr key={u.unitIndex} className="border-b border-border-subtle">
                <td className="px-2 py-1 tabular-nums">{u.unitIndex + 1}</td>
                <td className="px-2 py-1">{u.label}</td>
                <td className="px-2 py-1 tabular-nums">{u.count}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}
