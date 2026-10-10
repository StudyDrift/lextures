import { describe, expect, it } from 'vitest'
import { gradeSubmissionsLabel } from '../grade-submissions-label'

describe('gradeSubmissionsLabel', () => {
  it('falls back to the plain label while counts are unknown', () => {
    expect(gradeSubmissionsLabel(null, null)).toBe('Grade submissions')
    expect(gradeSubmissionsLabel(undefined, 3)).toBe('Grade submissions')
  })

  it('says all graded when nothing needs grading', () => {
    expect(gradeSubmissionsLabel(0, 1)).toBe('Grade submissions · all graded')
    expect(gradeSubmissionsLabel(0, null)).toBe('Grade submissions · all graded')
  })

  it('reports how many students still need grading', () => {
    expect(gradeSubmissionsLabel(1, 1)).toBe('Grade submissions · 1 of 1 to grade')
    expect(gradeSubmissionsLabel(2, 5)).toBe('Grade submissions · 2 of 5 to grade')
    expect(gradeSubmissionsLabel(2, null)).toBe('Grade submissions · 2 to grade')
  })
})
