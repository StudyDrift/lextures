import { describe, expect, it } from 'vitest'
import { appendTrace, computeSeries, trendSummary } from '../model'
import type { ModelConfig } from '../types'

describe('parameter_explorer model', () => {
  it('computes quadratic series matching independent calc', () => {
    const model: ModelConfig = {
      kind: 'preset',
      preset: 'quadratic',
      bind: { a: 'a', b: 'b', c: 'c' },
    }
    const series = computeSeries(model, { a: 1, b: 0, c: 0 })
    expect(series.length).toBeGreaterThan(10)
    const at3 = series.find((p) => Math.abs(p.x - 3) < 0.05)
    expect(at3).toBeTruthy()
    expect(Math.abs(at3!.y - 9)).toBeLessThan(0.2)
  })

  it('downsamples trace', () => {
    let trace: Array<{ at: string; params: Record<string, number | boolean | string> }> = []
    for (let i = 0; i < 300; i++) {
      trace = appendTrace(trace, { a: i }, 200)
    }
    expect(trace.length).toBeLessThanOrEqual(200)
  })

  it('produces a trend summary', () => {
    const s = trendSummary([
      { x: 0, y: 0 },
      { x: 1, y: 1 },
      { x: 2, y: 2 },
    ])
    expect(s.toLowerCase()).toContain('rise')
  })
})
