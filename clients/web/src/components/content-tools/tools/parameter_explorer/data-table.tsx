import type { PlotPoint } from './types'

type Props = {
  points: PlotPoint[]
  xLabel: string
  yLabel: string
  summary: string
  caption: string
}

export function DataTable({ points, xLabel, yLabel, summary, caption }: Props) {
  const sample =
    points.length <= 40
      ? points
      : points.filter((_, i) => i % Math.ceil(points.length / 40) === 0 || i === points.length - 1)

  return (
    <div data-testid="parameter-explorer-table">
      <p className="mb-2 text-sm text-fg-muted">{summary}</p>
      <div className="max-h-64 overflow-auto rounded border border-border-default">
        <table className="w-full border-collapse text-left text-sm">
          <caption className="sr-only">{caption}</caption>
          <thead className="sticky top-0 bg-surface-base">
            <tr>
              <th scope="col" className="border-b px-2 py-1.5 font-medium">
                {xLabel}
              </th>
              <th scope="col" className="border-b px-2 py-1.5 font-medium">
                {yLabel}
              </th>
            </tr>
          </thead>
          <tbody>
            {sample.map((p, i) => (
              <tr key={`${p.x}-${i}`} className="odd:bg-surface-raised even:bg-slate-50/60 dark:odd:bg-surface-base dark:even:bg-neutral-900/40">
                <td className="px-2 py-1 tabular-nums">{fmt(p.x)}</td>
                <td className="px-2 py-1 tabular-nums">{fmt(p.y)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}

function fmt(n: number): string {
  if (!Number.isFinite(n)) return '—'
  return String(Math.round(n * 1000) / 1000)
}
