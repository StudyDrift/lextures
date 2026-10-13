import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const sendMessage = vi.fn()
const fetchMessagingPolicy = vi.fn()
vi.mock('../../../lib/communication-api', async (orig) => {
  const actual = await orig<typeof import('../../../lib/communication-api')>()
  return {
    ...actual,
    fetchMailboxMessages: vi.fn(async () => []),
    patchMailbox: vi.fn(async () => undefined),
    fetchMessagingPolicy: () => fetchMessagingPolicy(),
    sendMessage: (b: unknown) => sendMessage(b),
  }
})
vi.mock('../../../context/use-inbox-unread', () => ({
  useMailboxRevision: () => 0,
  useCoursesRevision: () => 0,
  useRefreshUnread: () => async () => undefined,
}))

import Inbox from '../inbox'

function renderInbox() {
  return render(
    <MemoryRouter>
      <Inbox />
    </MemoryRouter>,
  )
}

describe('Inbox kid-safe Compose (#775)', () => {
  beforeEach(() => {
    sendMessage.mockReset()
    fetchMessagingPolicy.mockReset()
  })

  it('offers only allowed recipients to a managed learner and sends by user id', async () => {
    fetchMessagingPolicy.mockResolvedValue({
      restricted: true,
      recipients: [{ id: 'parent-1', displayName: 'Mom', kind: 'parent' }],
    })
    sendMessage.mockResolvedValue('new-id')
    renderInbox()
    const compose = await screen.findAllByRole('button', { name: 'Message your parent' })
    expect(screen.queryByText(/campus/i)).not.toBeInTheDocument()
    fireEvent.click(compose[0])
    expect(await screen.findByRole('dialog', { name: 'Message your parent' })).toBeInTheDocument()
    expect(await screen.findByLabelText('Mom (parent)')).toBeChecked()
    expect(screen.queryByPlaceholderText('name@example.com')).not.toBeInTheDocument()
    fireEvent.change(screen.getByPlaceholderText('Write something…'), { target: { value: 'Can I play outside?' } })
    fireEvent.click(screen.getByRole('button', { name: 'Send' }))
    await waitFor(() => expect(sendMessage).toHaveBeenCalled())
    expect(sendMessage.mock.calls[0][0]).toMatchObject({ toUserId: 'parent-1', body: 'Can I play outside?' })
    expect(sendMessage.mock.calls[0][0]).not.toHaveProperty('to_email')
  })

  it('keeps the free-text To field for adults', async () => {
    fetchMessagingPolicy.mockResolvedValue({ restricted: false, recipients: [] })
    renderInbox()
    await waitFor(() => expect(fetchMessagingPolicy).toHaveBeenCalled())
    fireEvent.click((await screen.findAllByRole('button', { name: 'Compose' }))[0])
    expect(await screen.findByPlaceholderText('name@example.com')).toBeInTheDocument()
    expect(screen.queryByRole('radio')).not.toBeInTheDocument()
  })
})
