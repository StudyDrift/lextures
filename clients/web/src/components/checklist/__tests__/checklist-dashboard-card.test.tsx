import type { ReactNode } from 'react'
import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { describe, expect, it } from 'vitest'
import type { ChecklistItem, ChecklistSummary } from '../../../lib/course-checklist-api-schemas'
import { ChecklistDashboardCard } from '../checklist-dashboard-card'

function wrap(ui: ReactNode) {
  return render(<MemoryRouter>{ui}</MemoryRouter>)
}

const baseSummary = (over: Partial<ChecklistSummary>): ChecklistSummary => ({
  outstandingEssential: 0,
  outstandingTotal: 0,
  done: 0,
  total: 0,
  dismissed: 0,
  computedAt: '2026-01-01T00:00:00Z',
  stale: false,
  ...over,
})

const todoItem = (id: string, title: string): ChecklistItem =>
  ({
    id,
    titleKey: id,
    title,
    whyKey: 'why',
    why: 'why',
    tier: 'recommended',
    status: 'todo',
    sources: [],
  }) as ChecklistItem

describe('ChecklistDashboardCard', () => {
  it('shows complete only when outstandingTotal is 0', () => {
    wrap(
      <ChecklistDashboardCard
        courseCode="C1"
        summary={baseSummary({
          outstandingEssential: 0,
          outstandingTotal: 0,
          done: 10,
          total: 10,
        })}
        topItems={[]}
      />,
    )
    expect(screen.getByText('Your course checklist is complete')).toBeInTheDocument()
  })

  it('does not claim complete when essentials are done but recommended remain (#641)', () => {
    wrap(
      <ChecklistDashboardCard
        courseCode="C1"
        summary={baseSummary({
          outstandingEssential: 0,
          outstandingTotal: 32,
          done: 26,
          total: 58,
        })}
        topItems={[todoItem('a', 'Add outcomes')]}
      />,
    )
    expect(screen.queryByText('Your course checklist is complete')).not.toBeInTheDocument()
    expect(screen.getByText(/26 of 58 done/)).toBeInTheDocument()
    expect(screen.getByText(/32 need attention/)).toBeInTheDocument()
    expect(screen.getByText(/Add outcomes/)).toBeInTheDocument()
  })
})
