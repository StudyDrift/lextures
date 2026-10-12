import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { describe, expect, it, vi } from 'vitest'
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

describe('GradebookGrid — cell editing', () => {
  const emptyGrades = {
    s1: { a1: '', a2: '' },
    s2: { a1: '', a2: '' },
  }

  function lastGrades(fn: ReturnType<typeof vi.fn>) {
    const calls = fn.mock.calls
    return calls[calls.length - 1]?.[0] as Record<string, Record<string, string>>
  }

  function scoreCells() {
    return screen.getAllByRole('gridcell').filter((c) => c.hasAttribute('data-gradebook-cell'))
  }

  it('stages a score typed into a focused cell when Enter is pressed (#763)', async () => {
    const user = userEvent.setup()
    const onGradesChange = vi.fn()
    renderGrid({ initialGrades: emptyGrades, onGradesChange })
    const cell = scoreCells()[3]!
    await user.click(cell)
    await user.keyboard('9')
    expect(screen.getByRole('textbox', { name: /grade for bob jones, quiz 1/i })).toHaveValue('9')
    await user.keyboard('{Enter}')
    expect(screen.queryByRole('textbox', { name: /grade for/i })).not.toBeInTheDocument()
    expect(lastGrades(onGradesChange).s2!.a2).toBe('9')
    expect(cell).toHaveTextContent('9')
  })

  it('stages a score typed into a focused cell when Tab is pressed', async () => {
    const user = userEvent.setup()
    const onGradesChange = vi.fn()
    renderGrid({ initialGrades: emptyGrades, onGradesChange })
    await user.click(scoreCells()[1]!)
    await user.keyboard('95')
    await user.keyboard('{Tab}')
    expect(lastGrades(onGradesChange).s1!.a2).toBe('95')
  })

  it('typing replaces an existing score, Enter commits it', async () => {
    const user = userEvent.setup()
    const onGradesChange = vi.fn()
    renderGrid({ onGradesChange })
    await user.click(scoreCells()[0]!)
    await user.keyboard('7{Enter}')
    expect(lastGrades(onGradesChange).s1!.a1).toBe('7')
  })

  it('commits an edited score on Enter after double-click', async () => {
    const user = userEvent.setup()
    const onGradesChange = vi.fn()
    renderGrid({ initialGrades: emptyGrades, onGradesChange })
    await user.dblClick(scoreCells()[3]!)
    const input = screen.getByRole('textbox', { name: /grade for bob jones, quiz 1/i })
    await user.type(input, '9')
    await user.keyboard('{Enter}')
    expect(lastGrades(onGradesChange).s2!.a2).toBe('9')
  })

  it('Escape after typing discards the typed score', async () => {
    const user = userEvent.setup()
    const onGradesChange = vi.fn()
    renderGrid({ initialGrades: emptyGrades, onGradesChange })
    await user.click(scoreCells()[0]!)
    await user.keyboard('9{Escape}')
    expect(lastGrades(onGradesChange).s1!.a1).toBe('')
  })

  it('does not start editing on typing when read-only', async () => {
    const user = userEvent.setup()
    renderGrid({ initialGrades: emptyGrades, readOnly: true })
    await user.click(scoreCells()[0]!)
    await user.keyboard('9')
    expect(screen.queryByRole('textbox', { name: /grade for/i })).not.toBeInTheDocument()
  })
})
