import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import { CreateAccessKeyModal } from '../create-access-key-modal'

vi.mock('../../../lib/api', () => ({
  authorizedFetch: vi.fn(),
}))

import { authorizedFetch } from '../../../lib/api'

const scopes = [{ id: 'mcp:connect', label: 'MCP', description: 'Connect MCP', group: 'Tools' }]

describe('CreateAccessKeyModal', () => {
  it('shows course picker when selecting specific courses with partial course data', async () => {
    vi.mocked(authorizedFetch).mockResolvedValueOnce({
      ok: true,
      json: async () => ({
        courses: [{ id: '1' }, { id: '2', courseCode: 'CS101', title: 'Intro' }],
      }),
    } as Response)

    const user = userEvent.setup()
    render(<CreateAccessKeyModal open scopes={scopes} onClose={() => {}} onCreated={() => {}} />)

    await waitFor(() => expect(screen.queryByText('Loading courses…')).not.toBeInTheDocument())
    await user.click(screen.getByRole('radio', { name: /selected courses/i }))

    expect(screen.getByText('CS101')).toBeInTheDocument()
    expect(screen.getByRole('dialog')).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'Create access key' })).toBeInTheDocument()
  })

  it('keeps the dialog mounted when courses are missing codes', async () => {
    vi.mocked(authorizedFetch).mockResolvedValueOnce({
      ok: true,
      json: async () => ({
        courses: [{ id: '1' }, { id: '2' }],
      }),
    } as Response)

    const user = userEvent.setup()
    render(<CreateAccessKeyModal open scopes={scopes} onClose={() => {}} onCreated={() => {}} />)

    await waitFor(() => expect(screen.queryByText('Loading courses…')).not.toBeInTheDocument())
    await user.click(screen.getByRole('radio', { name: /selected courses/i }))

    expect(screen.getByRole('dialog')).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'Create access key' })).toBeInTheDocument()
    expect(screen.getByPlaceholderText('Search by course code or title…')).toBeInTheDocument()
  })

  it('allows selecting a course from the checklist', async () => {
    vi.mocked(authorizedFetch).mockResolvedValueOnce({
      ok: true,
      json: async () => ({
        courses: [{ id: 'course-1', courseCode: 'CS101', title: 'Intro' }],
      }),
    } as Response)

    const user = userEvent.setup()
    render(<CreateAccessKeyModal open scopes={scopes} onClose={() => {}} onCreated={() => {}} />)

    await waitFor(() => expect(screen.queryByText('Loading courses…')).not.toBeInTheDocument())
    await user.click(screen.getByRole('radio', { name: /selected courses/i }))
    await user.click(screen.getByRole('checkbox', { name: /intro/i }))

    expect(screen.getByText('1 course selected')).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'Create access key' })).toBeInTheDocument()
  })
})
