import { useTranslation } from 'react-i18next'
import { GRID_SIZE } from '../shared/region-geometry'

export type DiagramHeatCell = { cell: string; count: number }
export type DiagramRegionStat = {
  regionId: string
  label: string
  count: number
  incorrectCount?: number
}

export type DiagramHeatmapProps = {
  cells: DiagramHeatCell[]
  regions: DiagramRegionStat[]
  swaps?: Array<{ pair: string; count: number }>
}

function parseCell(cell: string): { row: number; col: number } | null {
  const m = /^r(\d+)c(\d+)$/.exec(cell)
  if (!m) return null
  return { row: Number(m[1]), col: Number(m[2]) }
}

/** Instructor placement heat map + per-region table (CT.15 FR-11). */
export function DiagramHeatmap({ cells, regions, swaps }: DiagramHeatmapProps) {
  const { t } = useTranslation('contentTools')
  const peak = Math.max(1, ...cells.map((c) => c.count), ...regions.map((r) => r.count))
  const grid: number[][] = Array.from({ length: GRID_SIZE }, () =>
    Array.from({ length: GRID_SIZE }, () => 0),
  )
  for (const c of cells) {
    const parsed = parseCell(c.cell)
    if (!parsed) continue
    if (parsed.row >= 0 && parsed.row < GRID_SIZE && parsed.col >= 0 && parsed.col < GRID_SIZE) {
      grid[parsed.row]![parsed.col] = c.count
    }
  }

  return (
    <div className="space-y-3" data-testid="diagram-heatmap">
      <h3 className="text-sm font-medium text-fg-default">
        {t('contentTools.tools.diagram_hotspot.heatmap.title')}
      </h3>
      <div
        className="mx-auto grid max-w-xs gap-0.5"
        style={{ gridTemplateColumns: `repeat(${GRID_SIZE}, minmax(0, 1fr))` }}
        aria-hidden="true"
      >
        {grid.flatMap((row, r) =>
          row.map((count, c) => (
            <div
              key={`${r}-${c}`}
              className="aspect-square rounded-sm"
              style={{
                backgroundColor: `rgba(15, 118, 110, ${0.05 + (count / peak) * 0.7})`,
              }}
              title={`r${r}c${c}: ${count}`}
            />
          )),
        )}
      </div>

      <div className="overflow-x-auto">
        <table className="min-w-full border-collapse text-sm" role="table">
          <caption className="mb-1 text-start text-xs text-fg-muted">
            {t('contentTools.tools.diagram_hotspot.heatmap.tableCaption')}
          </caption>
          <thead>
            <tr className="border-b border-border-default text-start dark:border-border-default">
              <th scope="col" className="px-2 py-1 font-medium">
                {t('contentTools.tools.diagram_hotspot.heatmap.colRegion')}
              </th>
              <th scope="col" className="px-2 py-1 font-medium">
                {t('contentTools.tools.diagram_hotspot.heatmap.colCount')}
              </th>
            </tr>
          </thead>
          <tbody>
            {regions.map((r) => (
              <tr key={r.regionId} className="border-b border-border-subtle">
                <td className="px-2 py-1">{r.label || r.regionId}</td>
                <td className="px-2 py-1 tabular-nums">{r.count}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {swaps && swaps.length > 0 ? (
        <div>
          <h4 className="mb-1 text-xs font-medium text-fg-default">
            {t('contentTools.tools.diagram_hotspot.heatmap.swaps')}
          </h4>
          <ul className="space-y-1 text-xs text-fg-muted">
            {swaps.slice(0, 5).map((s) => (
              <li key={s.pair}>
                {t('contentTools.tools.diagram_hotspot.heatmap.swapRow', {
                  pair: s.pair,
                  count: s.count,
                })}
              </li>
            ))}
          </ul>
        </div>
      ) : null}
    </div>
  )
}
