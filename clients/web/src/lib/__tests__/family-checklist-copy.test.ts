import { describe, expect, it } from 'vitest'
import {
  FAMILY_CHECKLIST_CATEGORY,
  familyChecklistHelp,
  isFamilyChecklist,
} from '../family-checklist-copy'

describe('family checklist copy', () => {
  it('recognizes the family category', () => {
    expect(isFamilyChecklist([{ id: FAMILY_CHECKLIST_CATEGORY }])).toBe(true)
    expect(isFamilyChecklist([{ id: 'foundations' }])).toBe(false)
    expect(isFamilyChecklist(undefined)).toBe(false)
  })

  it('has plain help for each family step', () => {
    for (const id of [
      'structure.modules-exist',
      'people.students-enrolled',
      'assessment.gradable-items',
      'launch.student-preview',
      'structure.pacing-signal',
    ]) {
      const entry = familyChecklistHelp(id)
      expect(entry?.title.length).toBeGreaterThan(0)
      expect(entry?.what).not.toMatch(/QM|OSCQR|co-teacher/i)
      expect(entry?.why.length).toBeGreaterThan(0)
    }
    expect(familyChecklistHelp('orientation.welcome-message')).toBeNull()
  })
})
