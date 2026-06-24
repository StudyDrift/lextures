import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import type { ModuleAssignmentSubmissionApi } from '../../../lib/courses-api'
import { SubmissionNavigator, SubmissionStudentPicker } from '../submission-navigator'

function submission(
  id: string,
  label: string,
  isGraded?: boolean,
): ModuleAssignmentSubmissionApi {
  return {
    id,
    submittedByDisplayName: label,
    attachmentFileId: null,
    submittedAt: '2026-01-01T00:00:00.000Z',
    updatedAt: '2026-01-01T00:00:00.000Z',
    isGraded,
  }
}

describe('SubmissionStudentPicker status badge', () => {
  it('shows a spinner while syncing to Canvas', async () => {
    const user = userEvent.setup()
    const submissions = [submission('a', 'Alice', false)]

    render(
      <SubmissionStudentPicker
        submissions={submissions}
        index={0}
        syncingSubmissionIds={new Set(['a'])}
        onIndexChange={vi.fn()}
      />,
    )

    await user.click(screen.getByRole('button', { name: /Alice/i }))
    expect(screen.getAllByTitle('Syncing to Canvas')).toHaveLength(2)
  })
})

describe('SubmissionNavigator student filter', () => {
  it('navigates filtered students with arrow keys and selects with Enter', async () => {
    const user = userEvent.setup()
    const onIndexChange = vi.fn()
    const submissions = [
      submission('a', 'Alice'),
      submission('b', 'Bob'),
      submission('c', 'Carol'),
    ]

    render(
      <SubmissionNavigator
        submissions={submissions}
        index={0}
        onIndexChange={onIndexChange}
        gradedFilter="all"
        onGradedFilterChange={vi.fn()}
      />,
    )

    await user.click(screen.getByRole('button', { name: /Alice/i }))
    const filter = screen.getByRole('searchbox', { name: /filter students/i })
    await user.type(filter, 'b')

    expect(screen.getByRole('menuitemradio', { name: /Bob/i })).toBeInTheDocument()
    expect(screen.queryByRole('menuitemradio', { name: /Alice/i })).not.toBeInTheDocument()

    await user.keyboard('{ArrowDown}')
    await user.keyboard('{Enter}')

    expect(onIndexChange).toHaveBeenCalledWith(1)
  })

  it('highlights the current student when the picker opens', async () => {
    const user = userEvent.setup()
    const onIndexChange = vi.fn()
    const submissions = [
      submission('a', 'Alice'),
      submission('b', 'Bob'),
      submission('c', 'Carol'),
    ]

    render(
      <SubmissionNavigator
        submissions={submissions}
        index={1}
        onIndexChange={onIndexChange}
        gradedFilter="all"
        onGradedFilterChange={vi.fn()}
      />,
    )

    await user.click(screen.getByRole('button', { name: /Bob/i }))
    const filter = screen.getByRole('searchbox', { name: /filter students/i })
    await waitFor(() => expect(filter).toHaveFocus())

    await user.keyboard('{Enter}')

    expect(onIndexChange).toHaveBeenCalledWith(1)
  })
})