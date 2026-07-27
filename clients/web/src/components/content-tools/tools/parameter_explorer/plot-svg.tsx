import type { PlotPoint } from './types'

type Props = {
  points: PlotPoint[]
  xLabel: string
  yLabel: string
  summary: string
  reducedMotion?: boolean
}

const W = 480
const H = 280
const PAD = { l: 44, r: 16, t: 16, b: 36 }

export function PlotSvg({ points, xLabel, yLabel, summary, reducedMotion }: Props) {
  if (points.length === 0) {
    return (
      <div
        role="img"
        aria-label={summary}
        className="flex h-48 items-center justify-center rounded border border-dashed border-slate-300 text-sm text-slate-500 dark:border-neutral-600"
      >
        No plot data
      </div>
    )
  }

  const xs = points.map((p) => p.x)
  const ys = points.map((p) => p.y)
  let minX = Math.min(...xs)
  let maxX = Math.max(...xs)
  let minY = Math.min(...ys)
  let maxY = Math.max(...ys)
  if (minX === maxX) {
    minX -= 1
    maxX += 1
  }
  if (minY === maxY) {
    minY -= 1
    maxY += 1
  }
  // pad 5%
  const dx = (maxX - minX) * 0.05
  const dy = (maxY - minY) * 0.05
  minX -= dx
  maxX += dx
  minY -= dy
  maxY += dy

  const iw = W - PAD.l - PAD.r
  const ih = H - PAD.t - PAD.b
  const sx = (x: number) => PAD.l + ((x - minX) / (maxX - minX)) * iw
  const sy = (y: number) => PAD.t + ih - ((y - minY) / (maxY - minY)) * ih

  const d = points
    .map((p, i) => `${i === 0 ? 'M' : 'L'}${sx(p.x).toFixed(1)},${sy(p.y).toFixed(1)}`)
    .join(' ')

  const xTicks = [minX, (minX + maxX) / 2, maxX]
  const yTicks = [minY, (minY + maxY) / 2, maxY]

  return (
    <svg
      viewBox={`0 0 ${W} ${H}`}
      className="h-auto w-full max-w-full"
      role="img"
      aria-label={summary}
      data-testid="parameter-explorer-plot"
    >
      <title>{summary}</title>
      {/* grid */}
      {xTicks.map((t, i) => (
        <line
          key={`vx-${i}`}
          x1={sx(t)}
          x2={sx(t)}
          y1={PAD.t}
          y2={PAD.t + ih}
          stroke="currentColor"
          strokeOpacity={0.15}
        />
      ))}
      {yTicks.map((t, i) => (
        <line
          key={`hy-${i}`}
          y1={sy(t)}
          y2={sy(t)}
          x1={PAD.l}
          x2={PAD.l + iw}
          stroke="currentColor"
          strokeOpacity={0.15}
        />
      ))}
      {/* axes */}
      <line
        x1={PAD.l}
        y1={PAD.t + ih}
        x2={PAD.l + iw}
        y2={PAD.t + ih}
        stroke="currentColor"
        strokeWidth={1.5}
      />
      <line
        x1={PAD.l}
        y1={PAD.t}
        x2={PAD.l}
        y2={PAD.t + ih}
        stroke="currentColor"
        strokeWidth={1.5}
      />
      {/* series — stroke + dashed overlay for non-color distinction */}
      <path
        d={d}
        fill="none"
        stroke="#0f766e"
        strokeWidth={2.5}
        style={reducedMotion ? undefined : { transition: 'd 80ms linear' }}
      />
      <path
        d={d}
        fill="none"
        stroke="#0f766e"
        strokeWidth={2.5}
        strokeDasharray="0"
        markerStart="none"
      />
      {/* end markers */}
      <circle cx={sx(points[0]!.x)} cy={sy(points[0]!.y)} r={3.5} fill="#0f766e" />
      <circle
        cx={sx(points[points.length - 1]!.x)}
        cy={sy(points[points.length - 1]!.y)}
        r={3.5}
        fill="#0f766e"
        stroke="#fff"
        strokeWidth={1}
      />
      {/* labels */}
      {xTicks.map((t, i) => (
        <text
          key={`xt-${i}`}
          x={sx(t)}
          y={H - 8}
          textAnchor="middle"
          className="fill-current text-[10px] opacity-70"
        >
          {fmt(t)}
        </text>
      ))}
      {yTicks.map((t, i) => (
        <text
          key={`yt-${i}`}
          x={PAD.l - 6}
          y={sy(t) + 3}
          textAnchor="end"
          className="fill-current text-[10px] opacity-70"
        >
          {fmt(t)}
        </text>
      ))}
      <text
        x={PAD.l + iw / 2}
        y={H - 22}
        textAnchor="middle"
        className="fill-current text-[11px] font-medium opacity-80"
      >
        {xLabel}
      </text>
      <text
        x={14}
        y={PAD.t + ih / 2}
        textAnchor="middle"
        transform={`rotate(-90 14 ${PAD.t + ih / 2})`}
        className="fill-current text-[11px] font-medium opacity-80"
      >
        {yLabel}
      </text>
    </svg>
  )
}

function fmt(n: number): string {
  const a = Math.abs(n)
  if (a !== 0 && (a >= 1000 || a < 0.01)) return n.toExponential(1)
  return String(Math.round(n * 100) / 100)
}
