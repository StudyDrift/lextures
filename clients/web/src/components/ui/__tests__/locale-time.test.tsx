import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { LocaleFormatProvider } from '../../../context/locale-format-context'
import { LocaleTime } from '../locale-time'

describe('LocaleTime', () => {
  it('renders datetime attribute in ISO 8601', () => {
    render(
      <LocaleFormatProvider>
        <LocaleTime date="2026-04-15T10:00:00.000Z" data-testid="t" />
      </LocaleFormatProvider>,
    )
    const el = screen.getByTestId('t')
    expect(el.tagName).toBe('TIME')
    expect(el).toHaveAttribute('datetime', '2026-04-15T10:00:00.000Z')
  })
})
