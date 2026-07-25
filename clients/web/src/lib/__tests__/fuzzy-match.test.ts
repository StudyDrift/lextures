import { describe, expect, it } from 'vitest'
import { fuzzyMatchScore, fuzzyMatches } from '../fuzzy-match'

describe('fuzzyMatchScore', () => {
  it('returns 0 for empty query', () => {
    expect(fuzzyMatchScore('', 'Scheduling')).toBe(0)
    expect(fuzzyMatchScore('   ', 'Scheduling')).toBe(0)
  })

  it('matches substrings', () => {
    expect(fuzzyMatchScore('sched', 'Scheduling')).not.toBeNull()
    expect(fuzzyMatchScore('late', 'Late submission (after due)')).not.toBeNull()
  })

  it('matches subsequences (typos / skipped letters)', () => {
    expect(fuzzyMatchScore('schd', 'Scheduling')).not.toBeNull()
    expect(fuzzyMatchScore('grdng', 'Grading')).not.toBeNull()
  })

  it('rejects unrelated text', () => {
    expect(fuzzyMatchScore('zzz', 'Scheduling')).toBeNull()
    expect(fuzzyMatchScore('xyz', 'Access')).toBeNull()
  })

  it('scores prefix matches higher than mid-string', () => {
    const prefix = fuzzyMatchScore('grade', 'Grade posting')
    const mid = fuzzyMatchScore('grade', 'Blind grading notes')
    expect(prefix).not.toBeNull()
    expect(mid).not.toBeNull()
    expect(prefix!).toBeGreaterThan(mid!)
  })
})

describe('fuzzyMatches', () => {
  it('requires every token to match', () => {
    expect(fuzzyMatches('late pen', 'Late submission penalty after due')).toBe(true)
    expect(fuzzyMatches('late missing', 'Late submission penalty after due')).toBe(false)
  })

  it('matches across title + keywords haystack', () => {
    const haystack = 'Grading blind moderated points worth assignment group'
    expect(fuzzyMatches('blind', haystack)).toBe(true)
    expect(fuzzyMatches('points', haystack)).toBe(true)
    expect(fuzzyMatches('rubric', haystack)).toBe(false)
  })
})
