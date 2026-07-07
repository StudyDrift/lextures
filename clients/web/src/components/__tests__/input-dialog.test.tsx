import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import { InputDialog } from '../input-dialog'

describe('InputDialog — accessibility', () => {
  it('is not rendered when closed', () => {
    render(
      <InputDialog
        open={false}
        title="Link URL"
        value=""
        onValueChange={() => {}}
        onConfirm={() => {}}
        onClose={() => {}}
      />,
    )
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })

  it('focuses input on open and submits value', async () => {
    const onConfirm = vi.fn()
    render(
      <InputDialog
        open
        title="Link URL"
        label="URL"
        value="https://example.com"
        onValueChange={() => {}}
        confirmLabel="Insert"
        onConfirm={onConfirm}
        onClose={() => {}}
      />,
    )
    const input = screen.getByRole('textbox', { name: 'URL' })
    await waitFor(() => expect(input).toHaveFocus())
    await userEvent.click(screen.getByRole('button', { name: 'Insert' }))
    expect(onConfirm).toHaveBeenCalledWith('https://example.com')
  })

  it('closes on Escape', async () => {
    const onClose = vi.fn()
    render(
      <InputDialog
        open
        title="Thread title"
        value=""
        onValueChange={() => {}}
        onConfirm={() => {}}
        onClose={onClose}
      />,
    )
    await userEvent.keyboard('{Escape}')
    expect(onClose).toHaveBeenCalledTimes(1)
  })
})
