import { render } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { AriaAnnouncer } from '../aria-announcer'

describe('AriaAnnouncer', () => {
  it('renders a polite live region', () => {
    const { container } = render(<AriaAnnouncer />)
    const polite = container.querySelector('#a11y-polite-announcer')
    expect(polite).toBeTruthy()
    expect(polite).toHaveAttribute('aria-live', 'polite')
    expect(polite).toHaveAttribute('aria-atomic', 'true')
    expect(polite).toHaveAttribute('role', 'status')
  })

  it('renders an assertive live region', () => {
    const { container } = render(<AriaAnnouncer />)
    const assertive = container.querySelector('#a11y-assertive-announcer')
    expect(assertive).toBeTruthy()
    expect(assertive).toHaveAttribute('aria-live', 'assertive')
    expect(assertive).toHaveAttribute('aria-atomic', 'true')
    expect(assertive).toHaveAttribute('role', 'alert')
  })
})
