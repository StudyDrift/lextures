import { evalExpression } from '../../../../lib/safe-expression'
import type { ModelConfig, Parameter, PlotPoint } from './types'

export type PresetId =
  | 'linear'
  | 'quadratic'
  | 'exponential'
  | 'logistic'
  | 'projectile'
  | 'supply_demand'
  | 'normal'
  | 'compound_interest'

export type PresetMeta = {
  id: PresetId
  expression: string
  sweepFrom: number
  sweepTo: number
  sweepPoints: number
  slots: string[]
  xLabel: string
  yLabel: string
}

export const PRESET_LIBRARY: PresetMeta[] = [
  {
    id: 'linear',
    expression: 'm * x + b',
    sweepFrom: -10,
    sweepTo: 10,
    sweepPoints: 101,
    slots: ['m', 'b'],
    xLabel: 'x',
    yLabel: 'y',
  },
  {
    id: 'quadratic',
    expression: 'a * x^2 + b * x + c',
    sweepFrom: -10,
    sweepTo: 10,
    sweepPoints: 101,
    slots: ['a', 'b', 'c'],
    xLabel: 'x',
    yLabel: 'y',
  },
  {
    id: 'exponential',
    expression: 'a * exp(k * x)',
    sweepFrom: -2,
    sweepTo: 4,
    sweepPoints: 101,
    slots: ['a', 'k'],
    xLabel: 'x',
    yLabel: 'y',
  },
  {
    id: 'logistic',
    expression: 'K / (1 + ((K - P0) / P0) * exp(-r * x))',
    sweepFrom: 0,
    sweepTo: 40,
    sweepPoints: 121,
    slots: ['K', 'P0', 'r'],
    xLabel: 't',
    yLabel: 'P',
  },
  {
    id: 'projectile',
    expression: 'x * tan(theta) - (g * x^2) / (2 * v0^2 * cos(theta)^2)',
    sweepFrom: 0,
    sweepTo: 80,
    sweepPoints: 101,
    slots: ['v0', 'theta', 'g'],
    xLabel: 'x',
    yLabel: 'y',
  },
  {
    id: 'supply_demand',
    expression: '(a - b * x) - (c + d * x)',
    sweepFrom: 0,
    sweepTo: 20,
    sweepPoints: 101,
    slots: ['a', 'b', 'c', 'd'],
    xLabel: 'Q',
    yLabel: 'surplus',
  },
  {
    id: 'normal',
    expression: '(1 / (sigma * sqrt(2 * pi))) * exp(-0.5 * ((x - mu) / sigma)^2)',
    sweepFrom: -5,
    sweepTo: 5,
    sweepPoints: 121,
    slots: ['mu', 'sigma'],
    xLabel: 'x',
    yLabel: 'pdf',
  },
  {
    id: 'compound_interest',
    expression: 'P * (1 + r / n)^(n * x)',
    sweepFrom: 0,
    sweepTo: 20,
    sweepPoints: 101,
    slots: ['P', 'r', 'n'],
    xLabel: 't',
    yLabel: 'A',
  },
]

export function lookupPreset(id: string): PresetMeta | undefined {
  return PRESET_LIBRARY.find((p) => p.id === id)
}

export function defaultParams(parameters: Parameter[]): Record<string, number | boolean | string> {
  const out: Record<string, number | boolean | string> = {}
  for (const p of parameters) {
    out[p.id] = p.default
  }
  return out
}

export function paramsAsFloats(
  params: Record<string, number | boolean | string>,
): Record<string, number> {
  const out: Record<string, number> = {}
  for (const [k, v] of Object.entries(params)) {
    if (typeof v === 'number' && Number.isFinite(v)) out[k] = v
    else if (typeof v === 'boolean') out[k] = v ? 1 : 0
  }
  return out
}

export function resolveSweep(model: ModelConfig): {
  expression: string
  from: number
  to: number
  points: number
  xLabel: string
  yLabel: string
  bind: Record<string, string>
} {
  if (model.kind === 'expression') {
    return {
      expression: model.expression,
      from: model.sweep.from,
      to: model.sweep.to,
      points: Math.min(500, Math.max(2, model.sweep.points)),
      xLabel: model.sweep.paramId || 'x',
      yLabel: 'y',
      bind: {},
    }
  }
  const preset = lookupPreset(model.preset)
  if (!preset) {
    return {
      expression: '0',
      from: -1,
      to: 1,
      points: 11,
      xLabel: 'x',
      yLabel: 'y',
      bind: model.bind,
    }
  }
  return {
    expression: preset.expression,
    from: preset.sweepFrom,
    to: preset.sweepTo,
    points: preset.sweepPoints,
    xLabel: preset.xLabel,
    yLabel: preset.yLabel,
    bind: model.bind,
  }
}

export function computeSeries(
  model: ModelConfig,
  params: Record<string, number | boolean | string>,
): PlotPoint[] {
  const sweep = resolveSweep(model)
  const base = paramsAsFloats(params)
  const points: PlotPoint[] = []
  const n = sweep.points
  const span = sweep.to - sweep.from
  for (let i = 0; i < n; i++) {
    const x = n === 1 ? sweep.from : sweep.from + (span * i) / (n - 1)
    const env: Record<string, number> = { ...base, x }
    for (const [slot, paramId] of Object.entries(sweep.bind)) {
      if (typeof base[paramId] === 'number') env[slot] = base[paramId]!
    }
    try {
      const y = evalExpression(sweep.expression, env)
      if (Number.isFinite(y)) points.push({ x, y })
    } catch {
      // skip invalid points
    }
  }
  return points
}

export function scalarReadouts(
  model: ModelConfig,
  params: Record<string, number | boolean | string>,
): Array<{ label: string; value: number }> {
  const series = computeSeries(model, params)
  if (series.length === 0) return []
  const ys = series.map((p) => p.y)
  const mid = series[Math.floor(series.length / 2)]!
  return [
    { label: 'y(mid)', value: mid.y },
    { label: 'y_min', value: Math.min(...ys) },
    { label: 'y_max', value: Math.max(...ys) },
  ]
}

export function trendSummary(points: PlotPoint[]): string {
  if (points.length < 2) return 'Not enough data to describe a trend.'
  const first = points[0]!
  const last = points[points.length - 1]!
  const ys = points.map((p) => p.y)
  const minY = Math.min(...ys)
  const maxY = Math.max(...ys)
  const delta = last.y - first.y
  const direction =
    Math.abs(delta) < (maxY - minY) * 0.05
      ? 'stays roughly level'
      : delta > 0
        ? 'generally rises'
        : 'generally falls'
  return `From x=${fmt(first.x)} to x=${fmt(last.x)}, the value ${direction} (range ${fmt(minY)} to ${fmt(maxY)}).`
}

function fmt(n: number): string {
  if (!Number.isFinite(n)) return '—'
  const a = Math.abs(n)
  if (a !== 0 && (a >= 1000 || a < 0.001)) return n.toExponential(2)
  return String(Math.round(n * 1000) / 1000)
}

export function appendTrace(
  trace: Array<{ at: string; params: Record<string, number | boolean | string> }>,
  params: Record<string, number | boolean | string>,
  max = 200,
): Array<{ at: string; params: Record<string, number | boolean | string> }> {
  const last = trace[trace.length - 1]
  if (last && JSON.stringify(last.params) === JSON.stringify(params)) return trace
  const next = [...trace, { at: new Date().toISOString(), params: { ...params } }]
  if (next.length <= max) return next
  // keep first, evenly spaced middle, last
  const out = [next[0]!]
  const inner = max - 2
  const step = (next.length - 2) / inner
  for (let i = 0; i < inner; i++) {
    const idx = Math.min(next.length - 2, 1 + Math.floor(i * step))
    out.push(next[idx]!)
  }
  out.push(next[next.length - 1]!)
  return out
}
