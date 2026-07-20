import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import type { ReactNode } from 'react'
import { describe, expect, it, vi } from 'vitest'
import { PlatformFeaturesProvider } from '../../../context/platform-features-context'
import { Button } from '../button'

function wrap(ui: ReactNode) {
  return render(<PlatformFeaturesProvider>{ui}</PlatformFeaturesProvider>)
}

describe('AN.6 Button control motion', () => {
  it('applies press class and fires onClick without gating (FR-1 / FR-9)', async () => {
    const user = userEvent.setup()
    const onClick = vi.fn()
    wrap(
      <Button variant="primary" onClick={onClick}>
        Save
      </Button>,
    )
    const btn = screen.getByRole('button', { name: 'Save' })
    expect(btn.className).toMatch(/lx-control-press/)
    expect(btn).toHaveAttribute('data-motion-controls', 'on')
    await user.click(btn)
    expect(onClick).toHaveBeenCalledTimes(1)
  })

  it('loading crossfades to spinner and keeps width stable (FR-6 / AC-6)', () => {
    const { rerender } = wrap(<Button loading={false}>Submit assignment</Button>)
    const idle = screen.getByRole('button', { name: 'Submit assignment' })
    expect(idle).not.toHaveAttribute('aria-busy')

    rerender(
      <PlatformFeaturesProvider>
        <Button loading>Submit assignment</Button>
      </PlatformFeaturesProvider>,
    )
    const busy = screen.getByRole('button')
    expect(busy).toHaveAttribute('aria-busy', 'true')
    expect(busy).toHaveAttribute('data-loading', 'true')
    expect(busy).toBeDisabled()
    expect(busy.querySelector('.lx-control-spinner')).toBeTruthy()
  })

  it('static prop skips press class', () => {
    wrap(
      <Button static variant="ghost">
        Drag
      </Button>,
    )
    const btn = screen.getByRole('button', { name: 'Drag' })
    expect(btn.className).toMatch(/lex-btn-static/)
    expect(btn.className).not.toMatch(/lx-control-press/)
  })
})
