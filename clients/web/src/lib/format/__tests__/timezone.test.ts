import { describe, expect, it } from 'vitest'
import {
  detectBrowserTimezone,
  formatDeadlineDisplay,
  isValidTimezoneId,
  resolveDisplayTimezone,
} from '../timezone'

describe('timezone format', () => {
  it('validates known IANA ids', () => {
    expect(isValidTimezoneId('Asia/Kolkata')).toBe(true)
    expect(isValidTimezoneId('Not/Real')).toBe(false)
  })

  it('resolves user then course then UTC', () => {
    expect(resolveDisplayTimezone('Asia/Kolkata', 'America/New_York')).toBe('Asia/Kolkata')
    expect(resolveDisplayTimezone(null, 'America/Los_Angeles')).toBe('America/Los_Angeles')
    expect(resolveDisplayTimezone(null, null)).toBe('UTC')
    // LOCAL is not a display zone — fall through to UTC when no user zone
    expect(resolveDisplayTimezone(null, 'LOCAL')).toBe('UTC')
  })

  it('formats India deadline for AC-1', () => {
    const d = formatDeadlineDisplay('2026-04-15T23:59:00Z', {
      displayTimeZone: 'Asia/Kolkata',
      locale: 'en-US',
    })
    expect(d.primary).toMatch(/April 16, 2026/)
    expect(d.abbrev.length).toBeGreaterThan(0)
  })

  it('formats learner-local floating wall clock without converting', () => {
    const d = formatDeadlineDisplay('2026-04-15T23:59:00Z', {
      displayTimeZone: 'Asia/Kolkata',
      locale: 'en-US',
      instructorTimeZone: 'LOCAL',
    })
    expect(d.primary).toMatch(/April 15, 2026/)
    expect(d.primary).toMatch(/11:59/)
    expect(d.abbrev).toBe('local')
    expect(d.instructorHint).toMatch(/each learner/i)
  })

  it('detectBrowserTimezone returns a non-empty string', () => {
    expect(detectBrowserTimezone().length).toBeGreaterThan(0)
  })
})
