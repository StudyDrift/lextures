import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import StarterLibraryPage from '../starter-library-page'

const { listStarterCourses, getStarterCourse, copyStarterCourse, listDependents } = vi.hoisted(() => ({
  listStarterCourses: vi.fn(),
  getStarterCourse: vi.fn(),
  copyStarterCourse: vi.fn(),
  listDependents: vi.fn(),
}))

vi.mock('../../../lib/starter-library-api', async () => {
  const actual = await vi.importActual<typeof import('../../../lib/starter-library-api')>(
    '../../../lib/starter-library-api',
  )
  return {
    ...actual,
    listStarterCourses,
    getStarterCourse,
    copyStarterCourse,
    generateStarterCourse: vi.fn(),
  }
})

vi.mock('../../../lib/managed-learners-api', () => ({
  listDependents,
}))

const math = {
  slug: 'math-3-5',
  title: 'Multiplication and fractions',
  summary: 'Equal groups and parts of a whole.',
  gradeBand: '3-5',
  subject: 'math',
  subjectLabel: 'Math',
  estimatedMinutes: 50,
  standards: ['CCSS.MATH.CONTENT.3.OA.A.1'],
  modules: [
    { title: 'Equal groups', pageCount: 2 },
    { title: 'Fractions of a whole', pageCount: 2 },
  ],
}

describe('Starter library page', () => {
  beforeEach(() => {
    listStarterCourses.mockResolvedValue({
      gradeBands: ['K-2', '3-5', '6-8', '9-12'],
      subjects: [
        { id: 'math', label: 'Math' },
        { id: 'ela', label: 'Reading and writing' },
      ],
      courses: [math],
    })
    getStarterCourse.mockResolvedValue({
      ...math,
      modules: [
        { title: 'Equal groups', pages: [{ title: 'Start here', markdown: 'Count groups.' }] },
        { title: 'Fractions of a whole', pages: [{ title: 'Try it', markdown: 'Fold paper.' }] },
      ],
    })
    listDependents.mockResolvedValue([{ id: 'learner-1', displayName: 'Ada', relationship: 'child' }])
    copyStarterCourse.mockResolvedValue({ courseCode: 'C-ABC123', title: math.title, enrolled: true })
  })

  it('filters by subject and copies a course for a learner', async () => {
    const user = userEvent.setup()
    render(
      <MemoryRouter>
        <StarterLibraryPage />
      </MemoryRouter>,
    )

    expect(await screen.findByRole('heading', { name: 'Multiplication and fractions' })).toBeInTheDocument()
    expect(screen.getByText(/2 modules: Equal groups, Fractions of a whole/)).toBeInTheDocument()
    expect(screen.getByText(/CCSS.MATH.CONTENT.3.OA.A.1/)).toBeInTheDocument()
    expect(screen.getByText(/About 50 minutes/)).toBeInTheDocument()

    await user.selectOptions(screen.getByLabelText('Filter by subject'), 'math')
    expect(listStarterCourses).toHaveBeenCalledWith({ gradeBand: '', subject: 'math' })

    await user.click(screen.getByRole('button', { name: 'View Multiplication and fractions' }))
    expect(await screen.findByRole('button', { name: 'Use this course' })).toBeInTheDocument()
    expect(screen.getByText('Start here')).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Use this course' }))
    expect(await screen.findByRole('radio', { name: 'Ada' })).toBeInTheDocument()
    fireEvent.click(screen.getByRole('radio', { name: 'Ada' }))
    fireEvent.click(screen.getByRole('button', { name: 'Copy course' }))

    await waitFor(() => {
      expect(copyStarterCourse).toHaveBeenCalledWith('math-3-5', 'learner-1')
    })
  })
})
