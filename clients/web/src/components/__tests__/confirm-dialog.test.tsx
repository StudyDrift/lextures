import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import { ConfirmDialog } from '../confirm-dialog'

describe('ConfirmDialog — accessibility', () => {
  it('is not rendered when closed', () => {
    render(
      <ConfirmDialog open={false} title="Delete?" onConfirm={() => {}} onClose={() => {}} />,
    )
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })

  it('renders dialog with title and focus-trapped cancel on open', async () => {
    render(
      <ConfirmDialog
        open
        title="Delete item?"
        description="This cannot be undone."
        confirmLabel="Delete"
        cancelLabel="Cancel"
        variant="danger"
        onConfirm={() => {}}
        onClose={() => {}}
      />,
    )
    expect(screen.getByRole('dialog')).toHaveAttribute('aria-modal', 'true')
    expect(screen.getByText('Delete item?')).toBeInTheDocument()
    await waitFor(() => expect(screen.getByRole('button', { name: 'Cancel' })).toHaveFocus())
  })

  it('closes on Escape and calls onClose', async () => {
    const onClose = vi.fn()
    render(
      <ConfirmDialog open title="Confirm?" onConfirm={() => {}} onClose={onClose} />,
    )
    await userEvent.keyboard('{Escape}')
    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it('does not close on Escape while busy', async () => {
    const onClose = vi.fn()
    render(
      <ConfirmDialog open busy title="Confirm?" onConfirm={() => {}} onClose={onClose} />,
    )
    await userEvent.keyboard('{Escape}')
    expect(onClose).not.toHaveBeenCalled()
  })

  it('calls onConfirm when confirm is clicked', async () => {
    const onConfirm = vi.fn()
    render(
      <ConfirmDialog
        open
        title="Delete?"
        confirmLabel="Delete"
        variant="danger"
        onConfirm={onConfirm}
        onClose={() => {}}
      />,
    )
    await userEvent.click(screen.getByRole('button', { name: 'Delete' }))
    expect(onConfirm).toHaveBeenCalledTimes(1)
  })

  it('requires typed phrase before enabling confirm', async () => {
    const onConfirm = vi.fn()
    render(
      <ConfirmDialog
        open
        title="Revoke token?"
        requireTypedPhrase="REVOKE"
        typedPhrase=""
        onTypedPhraseChange={() => {}}
        confirmLabel="Revoke"
        onConfirm={onConfirm}
        onClose={() => {}}
      />,
    )
    const confirmBtn = screen.getByRole('button', { name: 'Revoke' })
    expect(confirmBtn).toBeDisabled()
    await userEvent.click(confirmBtn)
    expect(onConfirm).not.toHaveBeenCalled()
  })
})
