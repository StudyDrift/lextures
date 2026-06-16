import { describe, expect, it } from 'vitest'
import { formatProgressLabel, progressPercent } from './self-paced-api'

describe('progressPercent', () => {
  it('returns 0 for empty or zero-progress courses', () => {
    expect(progressPercent(0, 0)).toBe(0)
    expect(progressPercent(0, 10)).toBe(0)
    expect(progressPercent(5, 0)).toBe(0)
  })
  it('floors the fractional percentage', () => {
    expect(progressPercent(3, 10)).toBe(30)
    expect(progressPercent(1, 3)).toBe(33)
  })
  it('caps at 100', () => {
    expect(progressPercent(10, 10)).toBe(100)
    expect(progressPercent(11, 10)).toBe(100)
  })
})

describe('formatProgressLabel', () => {
  it('renders an X% complete label', () => {
    expect(formatProgressLabel(30)).toBe('30% complete')
    expect(formatProgressLabel(0)).toBe('0% complete')
  })
})
