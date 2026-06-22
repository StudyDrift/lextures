import { describe, expect, it } from 'vitest'
import { letterTierOptions, percentToDisplayGrade } from '../grading-display'

describe('grading-display', () => {
  const scheme = {
    type: 'letter',
    scaleJson: [
      { label: 'A', min_pct: 90 },
      { label: 'B', min_pct: 80 },
      { label: 'C', min_pct: 70 },
      { label: 'F', min_pct: 0 },
    ],
  }

  it('maps percentage to letter label', () => {
    expect(percentToDisplayGrade(92, scheme)).toBe('A')
    expect(percentToDisplayGrade(81, scheme)).toBe('B')
    expect(percentToDisplayGrade(55, scheme)).toBe('F')
  })

  it('returns null for points scheme', () => {
    expect(percentToDisplayGrade(88, { type: 'points', scaleJson: [] })).toBeNull()
  })

  it('lists letter tiers for target selector', () => {
    const opts = letterTierOptions(scheme)
    expect(opts[0]?.label).toBe('A')
    expect(opts.some((o) => o.label === 'B')).toBe(true)
  })
})
