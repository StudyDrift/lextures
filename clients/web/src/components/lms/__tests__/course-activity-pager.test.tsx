import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { CourseStructureItem } from '../../../lib/courses-api'
import { CourseActivityPager } from '../course-activity-pager'

vi.mock('../../../lib/courses-api', async () => {
  const actual = await vi.importActual<typeof import('../../../lib/courses-api')>(
    '../../../lib/courses-api',
  )
  return {
    ...actual,
    fetchCourseStructure: vi.fn(),
  }
})

const { fetchCourseStructure } = await import('../../../lib/courses-api')

function item(
  partial: Pick<CourseStructureItem, 'id' | 'kind' | 'title' | 'parentId' | 'sortOrder'>,
): CourseStructureItem {
  return {
    published: true,
    visibleFrom: null,
    dueAt: null,
    assignmentGroupId: null,
    createdAt: '2020-01-01T00:00:00Z',
    updatedAt: '2020-01-01T00:00:00Z',
    ...partial,
  }
}

const outline = [
  item({ id: 'm1', kind: 'module', title: 'Module', parentId: null, sortOrder: 0 }),
  item({ id: 'page-1', kind: 'content_page', title: 'Welcome', parentId: 'm1', sortOrder: 0 }),
  item({ id: 'page-2', kind: 'content_page', title: 'Cells', parentId: 'm1', sortOrder: 1 }),
  item({ id: 'quiz-1', kind: 'quiz', title: 'Cell quiz', parentId: 'm1', sortOrder: 2 }),
]

function renderPager(itemId: string) {
  return render(
    <MemoryRouter>
      <CourseActivityPager courseCode="BIO" itemId={itemId} />
    </MemoryRouter>,
  )
}

describe('CourseActivityPager', () => {
  beforeEach(() => {
    vi.mocked(fetchCourseStructure).mockResolvedValue(outline)
  })

  it('labels previous and next with the activity names', async () => {
    renderPager('page-2')
    const previous = await screen.findByRole('link', { name: 'Previous · Welcome' })
    const next = await screen.findByRole('link', { name: 'Next · Cell quiz' })
    expect(previous).toHaveAttribute('href', '/courses/BIO/modules/content/page-1')
    expect(next).toHaveAttribute('href', '/courses/BIO/modules/quiz/quiz-1')
  })

  it('shows only next on the first activity', async () => {
    renderPager('page-1')
    expect(await screen.findByRole('link', { name: 'Next · Cells' })).toBeTruthy()
    expect(screen.queryByRole('link', { name: /Previous/ })).toBeNull()
  })
})
