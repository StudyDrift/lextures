import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { LiveRegion } from '../live-region'

describe('LiveRegion', () => {
  it('renders with aria-live="polite" by default', () => {
    render(<LiveRegion>5 results</LiveRegion>)
    const el = screen.getByRole('status')
    expect(el).toHaveAttribute('aria-live', 'polite')
    expect(el).toHaveAttribute('aria-atomic', 'true')
  })

  it('renders with aria-live="assertive" when requested', () => {
    render(<LiveRegion politeness="assertive">Error occurred</LiveRegion>)
    const el = screen.getByRole('status')
    expect(el).toHaveAttribute('aria-live', 'assertive')
  })

  it('applies sr-only class when visuallyHidden (default)', () => {
    render(<LiveRegion>Hidden text</LiveRegion>)
    const el = screen.getByRole('status')
    expect(el.className).toContain('sr-only')
  })

  it('does not apply sr-only when visuallyHidden=false', () => {
    render(<LiveRegion visuallyHidden={false}>Visible text</LiveRegion>)
    const el = screen.getByRole('status')
    expect(el.className).not.toContain('sr-only')
  })

  it('renders children content', () => {
    render(<LiveRegion>Announcement text</LiveRegion>)
    expect(screen.getByText('Announcement text')).toBeInTheDocument()
  })
})
