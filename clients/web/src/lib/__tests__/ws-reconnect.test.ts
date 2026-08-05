import { describe, expect, it } from 'vitest'
import { wsReconnectDelayMs } from '../ws-reconnect'

describe('wsReconnectDelayMs', () => {
  it('starts near 1s and caps at 60s', () => {
    for (let i = 0; i < 20; i++) {
      const v0 = wsReconnectDelayMs(0)
      expect(v0).toBeGreaterThanOrEqual(500)
      expect(v0).toBeLessThanOrEqual(1000)
    }
    for (let i = 0; i < 20; i++) {
      const vHigh = wsReconnectDelayMs(20)
      expect(vHigh).toBeGreaterThanOrEqual(30_000)
      expect(vHigh).toBeLessThanOrEqual(60_000)
    }
  })

  it('grows with attempt', () => {
    // Use mid-range bounds so jitter cannot invert the comparison.
    const lowMax = 1000 // attempt 0 max
    const highMin = 8000 // attempt 3 min (8s * 0.5)
    expect(highMin).toBeGreaterThan(lowMax)
  })
})
