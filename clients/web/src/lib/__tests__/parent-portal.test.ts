import { describe, expect, it } from 'vitest'
import { parentChildLabel, parentGradeItemsForCourse, parentGradeScoreLabel, parentMessageTeacherHref } from '../parent-portal'

describe('parent-portal helpers', () => {
  it('prefers display name for child label', () => {
    expect(parentChildLabel('Sam Student', 'sam@school.edu')).toBe('Sam Student')
    expect(parentChildLabel(null, 'sam@school.edu')).toBe('sam@school.edu')
  })

  it('formats grade score with percentage', () => {
    expect(parentGradeScoreLabel({ itemId: '1', title: 'Quiz 1', score: '18', percentage: 90, status: 'posted' })).toBe(
      '18 (90%)',
    )
    expect(parentGradeScoreLabel({ itemId: '1', title: 'Quiz 1', score: '18', status: 'posted' })).toBe('18')
  })

  it('builds inbox compose href', () => {
    expect(
      parentMessageTeacherHref({ teacherEmail: 'teacher@school.edu', subject: 'Regarding Sam' }),
    ).toBe('/inbox?compose=1&to=teacher%40school.edu&subject=Regarding+Sam')
  })

  it('uses enriched grade items when present', () => {
    const items = parentGradeItemsForCourse({
      items: [{ itemId: 'a', title: 'Essay 1', score: '95', status: 'posted' }],
      grades: { b: '80' },
    })
    expect(items).toHaveLength(1)
    expect(items[0]?.title).toBe('Essay 1')
  })
})
