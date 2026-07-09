import { describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { CoursePurchasedBadge } from '../course-purchased-badge'
import { shouldShowPurchasedBadge } from '../../../lib/course-purchased-badge'

vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (key: string) => (key === 'courses.badge.purchased' ? 'Purchased' : key),
  }),
}))

describe('shouldShowPurchasedBadge', () => {
  it('requires marketplace flag and acquisition', () => {
    expect(shouldShowPurchasedBadge(false, { acquiredViaMarketplace: true })).toBe(false)
    expect(shouldShowPurchasedBadge(true, { acquiredViaMarketplace: false })).toBe(false)
    expect(shouldShowPurchasedBadge(true, {})).toBe(false)
    expect(shouldShowPurchasedBadge(true, { acquiredViaMarketplace: true })).toBe(true)
  })
})

describe('CoursePurchasedBadge', () => {
  it('renders an accessible Purchased label', () => {
    render(<CoursePurchasedBadge />)
    const badge = screen.getByTestId('course-purchased-badge')
    expect(badge).toHaveAttribute('aria-label', 'Purchased')
    expect(badge).toHaveTextContent('Purchased')
  })
})
