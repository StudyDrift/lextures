import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import { PinToggle } from '../pin-toggle'

describe('PinToggle', () => {
  it('exposes aria-pressed and pin label', () => {
    render(<PinToggle label="Due date" pinned={false} onToggle={() => {}} />)
    const btn = screen.getByRole('button', { name: 'Pin Due date to top' })
    expect(btn).toHaveAttribute('aria-pressed', 'false')
  })

  it('shows unpin label when pinned', () => {
    render(<PinToggle label="Due date" pinned onToggle={() => {}} />)
    const btn = screen.getByRole('button', { name: 'Unpin Due date' })
    expect(btn).toHaveAttribute('aria-pressed', 'true')
  })

  it('disables when at cap and unpinned', async () => {
    const onToggle = vi.fn()
    const user = userEvent.setup()
    render(
      <PinToggle label="Due date" pinned={false} disabledAtCap onToggle={onToggle} />,
    )
    const btn = screen.getByRole('button', { name: 'Pin Due date to top' })
    expect(btn).toBeDisabled()
    expect(btn).toHaveAttribute('aria-describedby', 'pinned-settings-cap-help')
    await user.click(btn)
    expect(onToggle).not.toHaveBeenCalled()
  })

  it('calls onToggle when activated', async () => {
    const onToggle = vi.fn()
    const user = userEvent.setup()
    render(<PinToggle label="Due date" pinned={false} onToggle={onToggle} alwaysVisible />)
    await user.click(screen.getByRole('button', { name: 'Pin Due date to top' }))
    expect(onToggle).toHaveBeenCalledTimes(1)
  })
})
