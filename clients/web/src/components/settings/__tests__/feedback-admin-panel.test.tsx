import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { describe, expect, it, vi, beforeEach } from 'vitest'
import { I18nProvider } from '../../../context/i18n-provider'

const { fetchFeedbackList, fetchFeedbackDetail, patchFeedback } = vi.hoisted(() => ({
  fetchFeedbackList: vi.fn(),
  fetchFeedbackDetail: vi.fn(),
  patchFeedback: vi.fn(),
}))

vi.mock('../../../lib/feedback-admin-api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../../../lib/feedback-admin-api')>()
  return {
    ...actual,
    fetchFeedbackList,
    fetchFeedbackDetail,
    patchFeedback,
  }
})

import { FeedbackAdminPanel } from '../feedback-admin-panel'

const listItem = {
  id: '11111111-1111-1111-1111-111111111111',
  message_preview: 'Something broke',
  category: 'bug' as const,
  source: 'web' as const,
  status: 'new' as const,
  submitter: { name: 'Test User', email: 'test@example.com' },
  created_at: '2026-07-10T12:00:00.000Z',
}

const detail = {
  id: listItem.id,
  message: 'Something broke <script>alert(1)</script>',
  category: 'bug' as const,
  source: 'web' as const,
  context: { route: '/courses/demo', locale: 'en' },
  status: 'new' as const,
  submitter: listItem.submitter,
  created_at: listItem.created_at,
  updated_at: listItem.created_at,
}

function renderPanel(initialPath = '/settings/feedback') {
  return render(
    <I18nProvider>
      <MemoryRouter initialEntries={[initialPath]}>
        <Routes>
          <Route path="/settings/feedback" element={<FeedbackAdminPanel />} />
          <Route path="/settings/feedback/:id" element={<FeedbackAdminPanel />} />
        </Routes>
      </MemoryRouter>
    </I18nProvider>,
  )
}

describe('FeedbackAdminPanel', () => {
  beforeEach(() => {
    fetchFeedbackList.mockReset()
    fetchFeedbackDetail.mockReset()
    patchFeedback.mockReset()
    fetchFeedbackList.mockResolvedValue({ items: [listItem], total: 1 })
    fetchFeedbackDetail.mockResolvedValue(detail)
    patchFeedback.mockResolvedValue({ ...detail, status: 'resolved' as const })
  })

  it('renders empty state when no submissions', async () => {
    fetchFeedbackList.mockResolvedValue({ items: [], total: 0 })
    renderPanel()
    await waitFor(() => {
      expect(screen.getByText('No feedback yet.')).toBeInTheDocument()
    })
  })

  it('lists submissions and opens detail with escaped message', async () => {
    const user = userEvent.setup()
    renderPanel()
    await waitFor(() => {
      expect(screen.getByRole('cell', { name: 'Test User' })).toBeInTheDocument()
    })
    await user.click(screen.getByRole('cell', { name: 'Something broke' }))
    await waitFor(() => {
      expect(fetchFeedbackDetail).toHaveBeenCalledWith(listItem.id)
    })
    expect(screen.getByText(/Something broke <script>alert\(1\)<\/script>/)).toBeInTheDocument()
  })

  it('applies status filter via query params', async () => {
    const user = userEvent.setup()
    renderPanel()
    await waitFor(() => expect(fetchFeedbackList).toHaveBeenCalled())
    fetchFeedbackList.mockClear()

    await user.selectOptions(screen.getByLabelText('Status'), 'new')
    await user.click(screen.getByRole('button', { name: 'Apply filters' }))

    await waitFor(() => {
      expect(fetchFeedbackList).toHaveBeenCalledWith(
        expect.objectContaining({ status: 'new' }),
      )
    })
  })

  it('saves status and note with optimistic update', async () => {
    const user = userEvent.setup()
    renderPanel(`/settings/feedback/${listItem.id}`)
    await waitFor(() => {
      expect(screen.getByText(/Something broke <script>/)).toBeInTheDocument()
    })

    const statusSelect = screen.getByRole('combobox', { name: 'Status' })
    await user.selectOptions(statusSelect, 'resolved')
    await user.type(screen.getByLabelText('Internal note'), 'Fixed in release')
    await user.click(screen.getByRole('button', { name: 'Save' }))

    await waitFor(() => {
      expect(patchFeedback).toHaveBeenCalledWith(listItem.id, {
        status: 'resolved',
        admin_note: 'Fixed in release',
      })
    })
  })

  it('moves focus to the back button after detail loads', async () => {
    const user = userEvent.setup()
    renderPanel()
    await waitFor(() => {
      expect(screen.getByRole('cell', { name: 'Test User' })).toBeInTheDocument()
    })

    const row = document.querySelector<HTMLTableRowElement>(
      `[data-feedback-row-id="${listItem.id}"]`,
    )!
    row.focus()
    await user.keyboard('{Enter}')

    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Back to list' })).toBeInTheDocument()
    })
    expect(document.activeElement).toBe(screen.getByRole('button', { name: 'Back to list' }))
  })

  it('moves focus to the back button after a slow detail fetch', async () => {
    fetchFeedbackDetail.mockImplementationOnce(
      () => new Promise((resolve) => setTimeout(() => resolve(detail), 50)),
    )
    renderPanel(`/settings/feedback/${listItem.id}`)

    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Back to list' })).toBeInTheDocument()
      expect(document.activeElement).toBe(screen.getByRole('button', { name: 'Back to list' }))
    })
  })

  it('returns focus to the originating row when closing detail', async () => {
    const user = userEvent.setup()
    renderPanel()
    await waitFor(() => {
      expect(screen.getByRole('cell', { name: 'Test User' })).toBeInTheDocument()
    })

    const row = document.querySelector<HTMLTableRowElement>(
      `[data-feedback-row-id="${listItem.id}"]`,
    )!
    row.focus()
    await user.keyboard('{Enter}')

    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Back to list' })).toBeInTheDocument()
    })

    await user.click(screen.getByRole('button', { name: 'Back to list' }))

    await waitFor(() => {
      expect(document.activeElement).toBe(
        document.querySelector<HTMLTableRowElement>(
          `[data-feedback-row-id="${listItem.id}"]`,
        ),
      )
    })
  })
})
