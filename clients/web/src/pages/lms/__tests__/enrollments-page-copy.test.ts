import { describe, expect, it } from 'vitest'
import { enrollmentsPageSubtitle } from '../enrollments-page-copy'

describe('enrollmentsPageSubtitle', () => {
  it('uses the course title when available', () => {
    expect(enrollmentsPageSubtitle('Intro Biology', 'C-8Q421K')).toBe(
      'People and roles for Intro Biology.',
    )
  })

  it('falls back to the course code when title is missing', () => {
    expect(enrollmentsPageSubtitle(null, 'C-8Q421K')).toBe(
      'People and roles for course C-8Q421K.',
    )
    expect(enrollmentsPageSubtitle('   ', 'C-8Q421K')).toBe(
      'People and roles for course C-8Q421K.',
    )
  })

  it('falls back to a generic label when both are missing', () => {
    expect(enrollmentsPageSubtitle(null, null)).toBe('Course enrollments')
  })
})
