import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { SkipLink } from '../skip-link'

describe('SkipLink', () => {
  it('renders a link to #main-content by default', () => {
    render(<SkipLink />)
    const link = screen.getByRole('link', { name: /skip to main content/i })
    expect(link).toHaveAttribute('href', '#main-content')
  })

  it('accepts a custom target', () => {
    render(<SkipLink target="#custom-target" />)
    const link = screen.getByRole('link', { name: /skip to main content/i })
    expect(link).toHaveAttribute('href', '#custom-target')
  })

  it('accepts a custom label', () => {
    render(<SkipLink label="Skip to navigation" />)
    expect(screen.getByRole('link', { name: /skip to navigation/i })).toBeInTheDocument()
  })

  it('is an anchor element (not a button)', () => {
    render(<SkipLink />)
    const link = screen.getByRole('link')
    expect(link.tagName).toBe('A')
  })
})
