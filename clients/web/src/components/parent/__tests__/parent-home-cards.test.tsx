import { render, screen, within } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { describe, expect, it } from 'vitest'
import type { ParentHomeChild } from '../../../lib/parent-api'
import { parentProgressHref } from '../../../lib/dashboard-view-preference'
import { ParentHomeCards } from '../parent-home-cards'

const kid: ParentHomeChild = {
  studentUserId: 'kid-1',
  displayName: 'Ava',
  courses: [
    { courseCode: 'C-AAAAAA', title: 'Math', enrollmentId: 'enr-1', gradePercent: 92.25 },
    { courseCode: 'C-BBBBBB', title: 'Art', enrollmentId: null, gradePercent: null },
  ],
  dueThisWeek: [
    {
      courseCode: 'C-AAAAAA',
      courseTitle: 'Math',
      itemId: 'item-1',
      kind: 'assignment',
      title: 'Fractions worksheet',
      dueAt: '2026-10-13T18:00:00Z',
      overdue: true,
    },
  ],
  completedThisWeek: [
    {
      courseCode: 'C-AAAAAA',
      courseTitle: 'Math',
      itemId: 'quiz-1',
      kind: 'quiz',
      title: 'Decimals quiz',
      completedAt: '2026-10-12T18:00:00Z',
      overdue: false,
    },
  ],
  timeSpentSeconds: 3900,
}

const emptyKid: ParentHomeChild = {
  studentUserId: 'kid-2',
  displayName: 'Ben',
  courses: [],
  dueThisWeek: [],
  completedThisWeek: [],
  timeSpentSeconds: 0,
}

function renderCards(items: ParentHomeChild[]) {
  return render(
    <MemoryRouter>
      <ParentHomeCards items={items} />
    </MemoryRouter>,
  )
}

describe('ParentHomeCards', () => {
  it('renders one card per child with the four sections', () => {
    renderCards([kid, emptyKid])
    const cards = screen.getAllByTestId('parent-home-card')
    expect(cards).toHaveLength(2)
    const ava = within(cards[0])
    expect(ava.getByRole('heading', { name: 'Ava' })).toBeInTheDocument()
    expect(ava.getByText(/Due this week/)).toBeInTheDocument()
    expect(ava.getByText(/Completed this week/)).toBeInTheDocument()
    expect(ava.getByText('Current grades')).toBeInTheDocument()
    expect(ava.getByText(/1h 05m/)).toBeInTheDocument()
    expect(ava.getByText('Overdue')).toBeInTheDocument()
    expect(ava.getByRole('link', { name: /Fractions worksheet/ })).toHaveAttribute(
      'href',
      '/courses/C-AAAAAA/modules/assignment/item-1',
    )
    expect(ava.getByRole('link', { name: /Decimals quiz/ })).toHaveAttribute(
      'href',
      '/courses/C-AAAAAA/modules/quiz/quiz-1',
    )
    expect(ava.getByRole('link', { name: /Math\s*92\.3%/ })).toHaveAttribute(
      'href',
      parentProgressHref('C-AAAAAA', 'enr-1'),
    )
    expect(ava.getByText('No grade yet')).toBeInTheDocument()
  })

  it('shows an Add a course empty state for a child with no courses', () => {
    renderCards([emptyKid])
    const card = screen.getByTestId('parent-home-card')
    expect(within(card).getByRole('link', { name: 'Add a course' })).toHaveAttribute('href', '/courses')
    expect(within(card).queryByText('Current grades')).not.toBeInTheDocument()
  })

  it('keeps work turned in early under Due this week, marked Turned in (#787)', () => {
    renderCards([
      {
        ...kid,
        dueThisWeek: [
          {
            courseCode: 'C-AAAAAA',
            courseTitle: 'Math',
            itemId: 'item-2',
            kind: 'assignment',
            title: 'QA Assignment',
            dueAt: '2026-10-17T05:59:00Z',
            completedAt: '2026-10-09T15:09:00Z',
            overdue: false,
            completed: true,
          },
        ],
      },
    ])
    const card = within(screen.getByTestId('parent-home-card'))
    expect(card.getByText('Due this week (1)')).toBeInTheDocument()
    expect(card.queryByText('Nothing due this week.')).not.toBeInTheDocument()
    expect(card.getByRole('link', { name: /QA Assignment/ })).toBeInTheDocument()
    expect(card.getByText('Turned in')).toBeInTheDocument()
    expect(card.queryByText('Overdue')).not.toBeInTheDocument()
  })

  it('stacks cards in a single column on mobile', () => {
    renderCards([kid, emptyKid])
    const grid = screen.getByTestId('parent-home-cards')
    expect(grid.className).toContain('grid-cols-1')
    expect(grid.className).toContain('lg:grid-cols-2')
  })
})
