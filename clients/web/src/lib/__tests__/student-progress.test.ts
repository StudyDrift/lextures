import { describe, expect, it, vi } from 'vitest'
import { studentProgressFeatureEnabled } from '../student-progress'

describe('studentProgressFeatureEnabled', () => {
  it('is false when env unset', () => {
    vi.stubEnv('VITE_FEATURE_STUDENT_PROGRESS', '')
    expect(studentProgressFeatureEnabled()).toBe(false)
  })

  it('is true when env is true', () => {
    vi.stubEnv('VITE_FEATURE_STUDENT_PROGRESS', 'true')
    expect(studentProgressFeatureEnabled()).toBe(true)
  })
})
