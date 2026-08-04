import { render, screen, fireEvent } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { describe, expect, it, vi } from 'vitest'
import type { ChecklistItem } from '../../../lib/course-checklist-api-schemas'
import { ChecklistItemRow } from '../../../pages/lms/course-checklist/checklist-item-row'

vi.mock('../../../context/platform-features-context', () => ({
  usePlatformFeatures: () => ({ ffMotionOverlays: false }),
}))

function baseItem(overrides: Partial<ChecklistItem> = {}): ChecklistItem {
  return {
    id: 'orientation.welcome-message',
    titleKey: 'k',
    title: 'Post a welcome announcement',
    whyKey: 'w',
    why: 'Students need a welcome',
    tier: 'essential',
    status: 'todo',
    detail: 'Missing announcement',
    progress: null,
    sources: ['QM 1.2'],
    helpRef: null,
    target: { route: '/courses/C1/feed' },
    evidence: null,
    dismissal: null,
    ...overrides,
  }
}

describe('ChecklistItemRow', () => {
  it('shows completed label and strikethrough for done items', () => {
    render(
      <MemoryRouter>
        <ul>
          <ChecklistItemRow
            item={baseItem({ status: 'done' })}
            onDismiss={vi.fn()}
            onRecheck={vi.fn()}
          />
        </ul>
      </MemoryRouter>,
    )
    const completed = screen.getAllByText('Completed')
    expect(completed.length).toBeGreaterThanOrEqual(1)
    expect(completed.every((el) => el.classList.contains('sr-only'))).toBe(true)
    expect(screen.getByText('Post a welcome announcement').className).toMatch(/line-through/)
  })

  it('expands evidence table when evidence rows exist', () => {
    render(
      <MemoryRouter>
        <ul>
          <ChecklistItemRow
            item={baseItem({
              target: null,
              evidence: {
                columns: ['Item', 'Type'],
                rows: [
                  {
                    label: 'Essay 1',
                    sublabel: 'Assign.',
                    status: 'unmapped',
                    target: { route: '/courses/C1/modules/assignment/a1' },
                  },
                ],
                truncatedAt: null,
              },
            })}
            onDismiss={vi.fn()}
            onRecheck={vi.fn()}
          />
        </ul>
      </MemoryRouter>,
    )
    const toggle = screen.getByRole('button', { name: /show the 1/i })
    expect(toggle).toHaveAttribute('aria-expanded', 'false')
    fireEvent.click(toggle)
    expect(toggle).toHaveAttribute('aria-expanded', 'true')
    expect(screen.getByRole('columnheader', { name: 'Item' })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Essay 1' })).toHaveAttribute(
      'href',
      '/courses/C1/modules/assignment/a1',
    )
  })

  it('links directly to target when there is no evidence', () => {
    render(
      <MemoryRouter>
        <ul>
          <ChecklistItemRow item={baseItem()} onDismiss={vi.fn()} onRecheck={vi.fn()} />
        </ul>
      </MemoryRouter>,
    )
    expect(screen.getByRole('link', { name: /post a welcome announcement/i })).toHaveAttribute(
      'href',
      '/courses/C1/feed',
    )
  })

  it('offers re-check for unknown items', () => {
    const onRecheck = vi.fn()
    render(
      <MemoryRouter>
        <ul>
          <ChecklistItemRow
            item={baseItem({ status: 'unknown', detail: null })}
            onDismiss={vi.fn()}
            onRecheck={onRecheck}
          />
        </ul>
      </MemoryRouter>,
    )
    expect(screen.getByText(/couldn't check this right now/i)).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: /^re-check$/i }))
    expect(onRecheck).toHaveBeenCalled()
  })
})
