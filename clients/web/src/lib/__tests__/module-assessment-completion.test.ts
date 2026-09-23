import { describe, expect, it } from 'vitest'
import { moduleAssessmentsComplete } from '../module-assessment-completion'

const assignment = (id: string, extra?: { published?: boolean; archived?: boolean }) => ({
  id,
  kind: 'assignment' as const,
  published: true,
  ...extra,
})

const quiz = (id: string) => ({ id, kind: 'quiz' as const, published: true })

describe('moduleAssessmentsComplete', () => {
  it('is false when the module has no assignments or quizzes', () => {
    expect(
      moduleAssessmentsComplete(
        [{ id: 'page', kind: 'content_page', published: true }],
        new Set(['page']),
      ),
    ).toBe(false)
  })

  it('is false until every assignment and quiz is finished', () => {
    const children = [assignment('a1'), quiz('q1'), { id: 'page', kind: 'content_page', published: true }]
    expect(moduleAssessmentsComplete(children, new Set(['a1']))).toBe(false)
    expect(moduleAssessmentsComplete(children, new Set(['a1', 'q1']))).toBe(true)
  })

  it('ignores unpublished and archived assessments', () => {
    const children = [
      assignment('a1'),
      assignment('draft', { published: false }),
      assignment('old', { archived: true }),
    ]
    expect(moduleAssessmentsComplete(children, new Set(['a1']))).toBe(true)
  })

  it('is true when the only assessments are quizzes and each is finished', () => {
    expect(moduleAssessmentsComplete([quiz('q1'), quiz('q2')], new Set(['q1', 'q2']))).toBe(true)
  })
})
