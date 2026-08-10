import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'
import { Field } from '../field'
import { Input } from '../input'

describe('Field (UX.6)', () => {
  it('wires label, description, error, and aria on the control', () => {
    render(
      <Field
        label="Email"
        description="Work address"
        error="Enter a valid email address."
        required
        htmlFor="test-email"
      >
        <Input type="email" />
      </Field>,
    )

    const input = screen.getByLabelText(/Email/)
    expect(input).toHaveAttribute('id', 'test-email')
    expect(input).toHaveAttribute('aria-invalid', 'true')
    expect(input).toHaveAttribute('aria-required', 'true')
    const describedBy = input.getAttribute('aria-describedby') ?? ''
    expect(describedBy).toContain('test-email-desc')
    expect(describedBy).toContain('test-email-err')
    expect(screen.getByText('Work address')).toBeInTheDocument()
    expect(screen.getByText('Enter a valid email address.')).toBeInTheDocument()
  })

  it('sets aria-busy when pending', () => {
    render(
      <Field label="Code" busy htmlFor="code">
        <Input />
      </Field>,
    )
    expect(screen.getByLabelText('Code')).toHaveAttribute('aria-busy', 'true')
  })
})
