import { describe, expect, it } from 'vitest'
import { estimateClockOffsetMs, secondsUntilDeadline } from '../live-quiz-countdown'

describe('live-quiz-countdown', () => {
  it('returns remaining seconds from deadline', () => {
    const deadline = new Date(Date.now() + 5500).toISOString()
    const left = secondsUntilDeadline(deadline, 0)
    expect(left).toBeGreaterThanOrEqual(5)
    expect(left).toBeLessThanOrEqual(6)
  })

  it('applies clock offset', () => {
    const deadline = new Date(Date.now() + 10_000).toISOString()
    expect(secondsUntilDeadline(deadline, 2000)).toBeLessThan(secondsUntilDeadline(deadline, 0)!)
  })

  it('estimates offset from RTT', () => {
    expect(estimateClockOffsetMs({ serverDeadlineIso: new Date().toISOString(), timeLimitSeconds: 20, rttMs: 80 })).toBe(40)
  })
})
