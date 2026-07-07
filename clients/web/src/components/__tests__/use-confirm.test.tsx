import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import { I18nProvider } from '../../context/i18n-provider'
import { toast } from '../../lib/lms-toast'

vi.mock('sonner', () => ({
  toast: {
    error: vi.fn(),
    success: vi.fn(),
  },
}))

describe('lms-toast error path', () => {
  it('toast.error renders message via sonner', () => {
    toast.error('Save failed.')
    expect(toast.error).toHaveBeenCalledWith('Save failed.')
  })
})

describe('useConfirm integration', () => {
  it('resolves true when confirmed and false when cancelled', async () => {
    const { useConfirm } = await import('../use-confirm')

    function TestHost() {
      const { confirm, ConfirmDialogHost } = useConfirm()
      return (
        <>
          <button
            type="button"
            onClick={() => {
              void confirm({ title: 'Proceed?' }).then((ok) => {
                document.body.dataset.result = ok ? 'yes' : 'no'
              })
            }}
          >
            Ask
          </button>
          {ConfirmDialogHost}
        </>
      )
    }

    render(
      <I18nProvider>
        <TestHost />
      </I18nProvider>,
    )
    await userEvent.click(screen.getByRole('button', { name: 'Ask' }))
    await waitFor(() => expect(screen.getByRole('dialog')).toBeInTheDocument())
    await userEvent.click(screen.getByRole('button', { name: 'Confirm' }))
    await waitFor(() => expect(document.body.dataset.result).toBe('yes'))

    await userEvent.click(screen.getByRole('button', { name: 'Ask' }))
    await waitFor(() => expect(screen.getByRole('dialog')).toBeInTheDocument())
    await userEvent.click(screen.getByRole('button', { name: 'Cancel' }))
    await waitFor(() => expect(document.body.dataset.result).toBe('no'))
  })
})
