import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { StructureItemStudentWorkError } from '../../../lib/course-trash-api'
import type { CourseStructureItem } from '../../../lib/courses-api'
import { CourseArchivedContentSection } from '../course-archived-content-section'

const { fetchCourseArchivedStructure, unarchiveCourseStructureItem, permanentlyDeleteCourseStructureItem } =
  vi.hoisted(() => ({
    fetchCourseArchivedStructure: vi.fn(),
    unarchiveCourseStructureItem: vi.fn(),
    permanentlyDeleteCourseStructureItem: vi.fn(),
  }))

vi.mock('../../../context/use-permissions', () => ({
  usePermissions: () => ({ allows: () => true, loading: false }),
}))

vi.mock('../../../lib/courses-api', async () => {
  const actual = await vi.importActual<typeof import('../../../lib/courses-api')>('../../../lib/courses-api')
  return {
    ...actual,
    fetchCourseArchivedStructure,
    unarchiveCourseStructureItem,
    courseItemCreatePermission: () => 'course:C-TEST01:item:create',
  }
})

vi.mock('../../../lib/course-trash-api', async () => {
  const actual = await vi.importActual<typeof import('../../../lib/course-trash-api')>(
    '../../../lib/course-trash-api',
  )
  return {
    ...actual,
    permanentlyDeleteCourseStructureItem,
  }
})

const archivedPage: CourseStructureItem = {
  id: 'page-1',
  sortOrder: 0,
  kind: 'content_page',
  title: 'Scratch page',
  parentId: 'mod-1',
  published: false,
  visibleFrom: null,
  archived: true,
  dueAt: null,
  assignmentGroupId: null,
  createdAt: '2026-01-01T00:00:00Z',
  updatedAt: '2026-01-01T00:00:00Z',
}

const moduleRow: CourseStructureItem = {
  id: 'mod-1',
  sortOrder: 0,
  kind: 'module',
  title: 'Module 1',
  parentId: null,
  published: true,
  visibleFrom: null,
  archived: false,
  dueAt: null,
  assignmentGroupId: null,
  createdAt: '2026-01-01T00:00:00Z',
  updatedAt: '2026-01-01T00:00:00Z',
}

function renderTrash() {
  return render(
    <MemoryRouter>
      <CourseArchivedContentSection courseCode="C-TEST01" />
    </MemoryRouter>,
  )
}

describe('Course trash', () => {
  beforeEach(() => {
    fetchCourseArchivedStructure.mockReset()
    unarchiveCourseStructureItem.mockReset()
    permanentlyDeleteCourseStructureItem.mockReset()
    fetchCourseArchivedStructure.mockResolvedValue([moduleRow, archivedPage])
    unarchiveCourseStructureItem.mockResolvedValue(archivedPage)
    permanentlyDeleteCourseStructureItem.mockResolvedValue(undefined)
  })

  it('restores an archived item', async () => {
    const user = userEvent.setup()
    renderTrash()
    expect(await screen.findByText('Scratch page')).toBeInTheDocument()
    expect(screen.getByText(/Trash holds items removed from the outline/)).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Restore' }))
    expect(unarchiveCourseStructureItem).toHaveBeenCalledWith('C-TEST01', 'page-1')
  })

  it('asks again before deleting an item that has student work', async () => {
    const user = userEvent.setup()
    permanentlyDeleteCourseStructureItem.mockRejectedValueOnce(
      new StructureItemStudentWorkError(
        'This item has student submissions or grades. Confirm to delete that work permanently.',
      ),
    )
    renderTrash()
    expect(await screen.findByText('Scratch page')).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Delete permanently' }))
    const first = screen.getByRole('dialog', { name: 'Delete permanently?' })
    await user.click(within(first).getByRole('button', { name: 'Delete permanently' }))

    const second = await screen.findByRole('dialog', { name: 'Delete student work too?' })
    await user.click(within(second).getByRole('button', { name: 'Delete item and student work' }))

    expect(permanentlyDeleteCourseStructureItem).toHaveBeenNthCalledWith(1, 'C-TEST01', 'page-1', {
      confirmStudentWork: false,
    })
    expect(permanentlyDeleteCourseStructureItem).toHaveBeenNthCalledWith(2, 'C-TEST01', 'page-1', {
      confirmStudentWork: true,
    })
  })
})
