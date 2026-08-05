import { describe, expect, it } from 'vitest'
import { datetimeLocalValueToIso, isoToDatetimeLocalValue } from '../datetime-local'

describe('datetime-local floating helpers', () => {
  it('stores LOCAL wall clock as UTC components', () => {
    const iso = datetimeLocalValueToIso('2026-04-15T23:59', 'LOCAL')
    expect(iso).toBe('2026-04-15T23:59:00.000Z')
  })

  it('loads LOCAL wall clock from UTC components', () => {
    expect(isoToDatetimeLocalValue('2026-04-15T23:59:00.000Z', 'LOCAL')).toBe(
      '2026-04-15T23:59',
    )
  })

  it('round-trips LOCAL without browser offset', () => {
    const wall = '2026-12-01T11:30'
    const iso = datetimeLocalValueToIso(wall, 'LOCAL')
    expect(isoToDatetimeLocalValue(iso, 'LOCAL')).toBe(wall)
  })
})
