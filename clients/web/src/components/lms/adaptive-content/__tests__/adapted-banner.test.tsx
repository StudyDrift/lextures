import { describe, expect, it, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { AdaptedBanner } from '../adapted-banner'

describe('AdaptedBanner', () => {
  it('renders adapted label and reason', () => {
    render(
      <AdaptedBanner
        adaptationReason="extra practice on key ideas"
        showingOriginal={false}
        canViewOriginal
        onToggleOriginal={() => {}}
      />,
    )
    expect(screen.getByRole('region', { name: /Adapted for you/i })).toBeTruthy()
    expect(screen.getAllByText(/extra practice on key ideas/i).length).toBeGreaterThan(0)
  })

  it('toggles View original with aria-pressed', () => {
    const onToggle = vi.fn()
    render(
      <AdaptedBanner
        showingOriginal={false}
        canViewOriginal
        onToggleOriginal={onToggle}
      />,
    )
    const btn = screen.getByRole('button', { name: /View original/i })
    expect(btn.getAttribute('aria-pressed')).toBe('false')
    fireEvent.click(btn)
    expect(onToggle).toHaveBeenCalledTimes(1)
  })

  it('shows View adapted when viewing original', () => {
    render(
      <AdaptedBanner
        showingOriginal
        canViewOriginal
        onToggleOriginal={() => {}}
      />,
    )
    expect(screen.getByRole('region', { name: /Showing the original/i })).toBeTruthy()
    expect(screen.getByRole('button', { name: /View adapted/i })).toBeTruthy()
  })

  it('offers prefer standard when opt-out allowed', () => {
    const onPrefer = vi.fn()
    render(
      <AdaptedBanner
        showingOriginal={false}
        canViewOriginal
        optoutAllowed
        onToggleOriginal={() => {}}
        onPreferStandard={onPrefer}
      />,
    )
    fireEvent.click(screen.getByRole('button', { name: /Prefer standard content/i }))
    expect(onPrefer).toHaveBeenCalled()
  })

  it('announces via live region', () => {
    render(
      <AdaptedBanner
        showingOriginal={false}
        canViewOriginal
        onToggleOriginal={() => {}}
      />,
    )
    expect(screen.getByRole('status')).toBeTruthy()
  })
})
