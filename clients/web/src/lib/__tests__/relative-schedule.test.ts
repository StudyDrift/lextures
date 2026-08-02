import { describe, expect, it } from 'vitest'
import {
  addRelativeDuration,
  dateToDatetimeLocalValue,
  datetimeLocalFromRelativeParts,
  dayDeltaToParts,
  isRelativeScheduleMode,
  partsFromDatetimeLocal,
} from '../relative-schedule'

describe('relative-schedule', () => {
  it('detects relative schedule mode', () => {
    expect(isRelativeScheduleMode('relative')).toBe(true)
    expect(isRelativeScheduleMode('fixed')).toBe(false)
    expect(isRelativeScheduleMode(undefined)).toBe(false)
  })

  it('maps day deltas to weeks when divisible by 7', () => {
    expect(dayDeltaToParts(14)).toEqual({ amount: '2', unit: 'W' })
    expect(dayDeltaToParts(3)).toEqual({ amount: '3', unit: 'D' })
    expect(dayDeltaToParts(0)).toEqual({ amount: '', unit: 'D' })
    expect(dayDeltaToParts(-2)).toEqual({ amount: '', unit: 'D' })
  })

  it('adds calendar durations from an anchor', () => {
    const anchor = new Date(2026, 0, 1, 0, 0, 0, 0) // local Jan 1
    const week = addRelativeDuration(anchor, 1, 'W')
    expect(week.getDate()).toBe(8)
    const month = addRelativeDuration(anchor, 1, 'M')
    expect(month.getMonth()).toBe(1)
  })

  it('round-trips datetime-local offsets from an anchor', () => {
    const anchor = new Date(2026, 5, 1, 9, 0, 0, 0) // local June 1 09:00
    const anchorIso = anchor.toISOString()
    const local = datetimeLocalFromRelativeParts(anchorIso, '7', 'D', {
      defaultTime: '23:59',
    })
    expect(local).toMatch(/T23:59$/)
    const parts = partsFromDatetimeLocal(local, anchorIso)
    expect(parts).toEqual({ amount: '1', unit: 'W' })
  })

  it('preserves time when updating amount', () => {
    const anchor = new Date(2026, 5, 1, 0, 0, 0, 0)
    const anchorIso = anchor.toISOString()
    const first = datetimeLocalFromRelativeParts(anchorIso, '3', 'D', {
      defaultTime: '17:30',
    })
    const second = datetimeLocalFromRelativeParts(anchorIso, '10', 'D', {
      previousDatetimeLocal: first,
    })
    expect(second).toMatch(/T17:30$/)
    expect(partsFromDatetimeLocal(second, anchorIso)).toEqual({ amount: '10', unit: 'D' })
  })

  it('formats local datetime values', () => {
    const d = new Date(2026, 0, 5, 8, 7)
    expect(dateToDatetimeLocalValue(d)).toBe('2026-01-05T08:07')
  })
})
