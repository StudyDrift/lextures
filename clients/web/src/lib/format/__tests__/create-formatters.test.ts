import { describe, expect, it } from 'vitest'
import { clearFormatterCaches, createLocaleFormatters } from '../create-formatters'

describe('createLocaleFormatters', () => {
  it('formats German dates with day.month.year', () => {
    clearFormatterCaches()
    const f = createLocaleFormatters({ locale: 'de', timeZone: 'UTC' })
    const label = f.formatDate('2026-04-15T10:00:00.000Z', { dateStyle: 'medium' })
    expect(label).toMatch(/15\.04\.2026|15\.04\.26/)
  })

  it('formats fr-CA percent with comma decimal', () => {
    clearFormatterCaches()
    const f = createLocaleFormatters({ locale: 'fr-CA', timeZone: 'UTC' })
    const label = f.formatPercent(0.925)
    expect(label).toMatch(/92,5/)
  })

  it('formats relative time in German', () => {
    clearFormatterCaches()
    const f = createLocaleFormatters({ locale: 'de', timeZone: 'UTC' })
    const now = new Date('2026-04-15T12:00:00.000Z')
    const past = new Date('2026-04-15T09:00:00.000Z')
    const label = f.formatRelativeTime(past, now)
    expect(label.length).toBeGreaterThan(0)
    expect(label.toLowerCase()).not.toBe('never')
  })
})
