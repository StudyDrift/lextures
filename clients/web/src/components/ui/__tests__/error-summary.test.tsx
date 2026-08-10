import { describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { ErrorSummary } from '../error-summary'

describe('ErrorSummary (UX.6)', () => {
  it('renders links that focus the target field', async () => {
    const user = userEvent.setup()
    document.body.innerHTML = '<input id="field-email" />'
    const focusSpy = vi.spyOn(HTMLElement.prototype, 'focus')

    render(
      <ErrorSummary
        autoFocus={false}
        errors={[{ id: 'field-email', label: 'Email', message: 'Enter a valid email.' }]}
      />,
    )

    expect(screen.getByRole('alert')).toBeInTheDocument()
    await user.click(screen.getByRole('link', { name: /Email/ }))
    expect(focusSpy).toHaveBeenCalled()
    focusSpy.mockRestore()
  })

  it('returns null when there are no errors', () => {
    const { container } = render(<ErrorSummary errors={[]} />)
    expect(container).toBeEmptyDOMElement()
  })
})
