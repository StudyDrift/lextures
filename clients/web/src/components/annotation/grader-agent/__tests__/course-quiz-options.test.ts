import { describe, expect, it } from 'vitest'
import type { CourseStructureItem } from '../../../../lib/courses-api'
import { filterQuizOptions, quizOptionsFromStructure } from '../course-quiz-options'

function item(partial: Partial<CourseStructureItem> & Pick<CourseStructureItem, 'id' | 'kind' | 'title'>): CourseStructureItem {
  return {
    sortOrder: 0,
    parentId: null,
    published: true,
    visibleFrom: null,
    dueAt: null,
    assignmentGroupId: null,
    createdAt: '',
    updatedAt: '',
    ...partial,
  }
}

describe('quizOptionsFromStructure', () => {
  it('resolves module titles and sorts by module then quiz title', () => {
    const options = quizOptionsFromStructure([
      item({ id: 'm1', kind: 'module', title: 'Module B' }),
      item({ id: 'q1', kind: 'quiz', title: 'Sprint Planning', parentId: 'm1' }),
      item({ id: 'm2', kind: 'module', title: 'Module A' }),
      item({ id: 'q2', kind: 'quiz', title: 'Sprint Planning', parentId: 'm2' }),
      item({ id: 'q3', kind: 'quiz', title: 'Final Presentation', parentId: 'm2' }),
    ])

    expect(options).toEqual([
      { id: 'q3', title: 'Final Presentation', moduleTitle: 'Module A' },
      { id: 'q2', title: 'Sprint Planning', moduleTitle: 'Module A' },
      { id: 'q1', title: 'Sprint Planning', moduleTitle: 'Module B' },
    ])
  })

  it('filters by quiz title or module name', () => {
    const options = quizOptionsFromStructure([
      item({ id: 'm1', kind: 'module', title: 'Agile Practices' }),
      item({ id: 'q1', kind: 'quiz', title: 'Sprint Planning', parentId: 'm1' }),
    ])
    expect(filterQuizOptions(options, 'agile')).toHaveLength(1)
    expect(filterQuizOptions(options, 'planning')).toHaveLength(1)
    expect(filterQuizOptions(options, 'missing')).toHaveLength(0)
  })
})