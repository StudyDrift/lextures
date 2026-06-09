import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { describe, expect, it } from 'vitest'
import { UiDensityProvider } from '../../../../context/ui-density-context'
import { GradebookGrid, type GradebookColumn, type GradebookStudent } from '../gradebook-grid'

const columns: GradebookColumn[] = [
  { id: 'a1', title: 'Homework 1', maxPoints: 100 },
  { id: 'a2', title: 'Quiz 1', maxPoints: 50 },
]

const students: GradebookStudent[] = [
  { id: 's1', name: 'Alice Smith' },
  { id: 's2', name: 'Bob Jones' },
]

const grades = {
  s1: { a1: '90', a2: '45' },
  s2: { a1: '80', a2: '40' },
}

function renderGrid(props: Partial<React.ComponentProps<typeof GradebookGrid>> = {}) {
  return render(
    <MemoryRouter>
      <UiDensityProvider>
        <GradebookGrid
          columns={columns}
          students={students}
          initialGrades={grades}
          {...props}
        />
      </UiDensityProvider>
    </MemoryRouter>,
  )
}

describe('GradebookGrid — accessibility', () => {
  it('renders a grid element with accessible label', () => {
    renderGrid()
    const grid = screen.getByRole('grid', { name: /grades by student and assignment/i })
    expect(grid).toBeInTheDocument()
  })

  it('has aria-rowcount on the grid', () => {
    renderGrid()
    const grid = screen.getByRole('grid', { name: /grades by student and assignment/i })
    // rowcount = students.length + 2 (header + stats rows)
    expect(grid).toHaveAttribute('aria-rowcount', String(students.length + 2))
  })

  it('has aria-colcount on the grid', () => {
    renderGrid()
    const grid = screen.getByRole('grid', { name: /grades by student and assignment/i })
    // 2 (student + final) + column count
    expect(grid).toHaveAttribute('aria-colcount', String(2 + columns.length))
  })

  it('column headers have aria-sort="none" when no sort is active', () => {
    renderGrid()
    // The Student column header button is inside a <th scope="col"> with aria-sort
    const grid = screen.getByRole('grid', { name: /grades by student and assignment/i })
    const allColHeaders = within(grid).getAllByRole('columnheader')
    // Student + Final + each assignment column all have scope="col" or are columnheaders
    const withSort = allColHeaders.filter((h) => h.hasAttribute('aria-sort'))
    // At minimum the Student column and each assignment column should carry aria-sort
    expect(withSort.length).toBeGreaterThan(0)
    withSort.forEach((h) => {
      expect(h.getAttribute('aria-sort')).toBe('none')
    })
  })

  it('student data rows have aria-rowindex starting at 3', () => {
    renderGrid()
    const grid = screen.getByRole('grid', { name: /grades by student and assignment/i })
    const rows = within(grid).getAllByRole('row')
    // rows[0] = header (rowindex 1), rows[1] = stats (rowindex 2), rows[2..] = students
    const dataRows = rows.filter((r) => {
      const idx = r.getAttribute('aria-rowindex')
      return idx !== null && Number(idx) >= 3
    })
    expect(dataRows).toHaveLength(students.length)
    expect(dataRows[0]).toHaveAttribute('aria-rowindex', '3')
    expect(dataRows[1]).toHaveAttribute('aria-rowindex', '4')
  })

  it('header row has aria-rowindex=1', () => {
    renderGrid()
    const grid = screen.getByRole('grid', { name: /grades by student and assignment/i })
    const rows = within(grid).getAllByRole('row')
    const headerRow = rows.find((r) => r.getAttribute('aria-rowindex') === '1')
    expect(headerRow).toBeDefined()
  })

  it('stats row has aria-rowindex=2', () => {
    renderGrid()
    const grid = screen.getByRole('grid', { name: /grades by student and assignment/i })
    const rows = within(grid).getAllByRole('row')
    const statsRow = rows.find((r) => r.getAttribute('aria-rowindex') === '2')
    expect(statsRow).toBeDefined()
  })

  it('renders gridcells for each student/column combination', () => {
    renderGrid()
    // Each student has a final cell + one cell per assignment
    const cells = screen.getAllByRole('gridcell')
    // (1 final + 2 assignment) × 2 students = 6 cells
    expect(cells.length).toBeGreaterThanOrEqual(students.length * (1 + columns.length))
  })

  it('shows former students panel when all roster students are withdrawn', () => {
    render(
      <MemoryRouter>
        <UiDensityProvider>
          <GradebookGrid
            columns={columns}
            students={[{ id: 's1', name: 'Alice Smith', state: 'withdrawn' }]}
            initialGrades={{ s1: { a1: '90', a2: '45' } }}
          />
        </UiDensityProvider>
      </MemoryRouter>,
    )
    expect(screen.getByRole('button', { name: /former students \(1\)/i })).toBeInTheDocument()
    expect(screen.queryByText(/no students in this course yet/i)).not.toBeInTheDocument()
  })

  it('toggles transposed layout when Transpose is clicked', async () => {
    const user = userEvent.setup()
    renderGrid()
    expect(screen.getByRole('grid', { name: /grades by student and assignment/i })).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: /^transpose$/i }))

    expect(screen.getByRole('grid', { name: /grades by assignment and student/i })).toBeInTheDocument()
    expect(screen.queryByRole('grid', { name: /grades by student and assignment/i })).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: /^transpose$/i })).toHaveAttribute('aria-pressed', 'true')
    expect(screen.getByRole('button', { name: /resize assignment column/i })).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: /^transpose$/i }))

    expect(screen.getByRole('grid', { name: /grades by student and assignment/i })).toBeInTheDocument()
    expect(screen.queryByRole('grid', { name: /grades by assignment and student/i })).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: /^transpose$/i })).toHaveAttribute('aria-pressed', 'false')
  })
})
