import { describe, expect, it } from 'vitest'
import { scheduleAudienceCopy } from '../schedule-audience-copy'

describe('scheduleAudienceCopy', () => {
  it('keeps enrollment language for higher-ed', () => {
    const copy = scheduleAudienceCopy(false)
    expect(copy.sectionTitle).toBe('Fixed Schedule & Visibility')
    expect(copy.sectionIntro).toContain('each student’s enrollment')
    expect(copy.modeLabelRelative).toBe('Relative (from enrollment)')
    expect(copy.relativeBody).toContain('when the student is enrolled')
  })

  it('uses a family schedule for homeschool and K–12', () => {
    const copy = scheduleAudienceCopy(true)
    expect(copy.sectionTitle).toBe('Family schedule & visibility')
    expect(copy.sectionIntro).toContain('family schedule')
    expect(copy.sectionIntro).toContain('each learner')
    expect(copy.sectionIntro).not.toMatch(/enrollment/i)
    expect(copy.relativeSwitchLabel).toContain('each learner')
    expect(copy.modeLabelRelative).toContain('Flexible')
    expect(copy.relativeBody).toContain('when the learner starts')
    expect(copy.relativeBody).not.toMatch(/student/i)
  })
})
