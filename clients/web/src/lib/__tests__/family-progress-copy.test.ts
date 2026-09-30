import { describe, expect, it } from 'vitest'
import { progressSurfaceCopy } from '../family-progress-copy'

describe('progressSurfaceCopy', () => {
  it('uses learner language for homeschool and K–12 progress surfaces', () => {
    const copy = progressSurfaceCopy(true)
    expect(copy.gradebookEmptyTitle).toBe('No learners in this course yet')
    expect(copy.reportsDescription('MATH')).toContain('How each learner is doing')
    expect(copy.outcomesDescription).not.toMatch(/cohort|accreditation/i)
    expect(copy.outcomesNotePlaceholder).not.toMatch(/accreditation/i)
    expect(copy.gradebookNoMatch('Alex')).toContain('No learners match')
  })

  it('keeps campus wording for higher-ed progress surfaces', () => {
    const copy = progressSurfaceCopy(false)
    expect(copy.gradebookEmptyTitle).toBe('No students in this course yet')
    expect(copy.reportsDescription('HIST')).toContain('Student progress reports')
    expect(copy.outcomesDescription).toMatch(/accreditation/)
  })
})
