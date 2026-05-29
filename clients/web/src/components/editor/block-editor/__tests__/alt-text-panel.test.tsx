import { render, screen, fireEvent } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { AltTextPanel } from '../alt-text-panel'
import { AltTextEnforcementProvider } from '../alt-text-enforcement-context'

describe('AltTextPanel', () => {
  it('renders required prompt and applies alt text', () => {
    const onApply = vi.fn()
    const onClose = vi.fn()
    render(
      <AltTextEnforcementProvider value={{ enabled: true, hardBlock: true, courseCode: 'DEMO' }}>
        <AltTextPanel
          alt=""
          decorative={false}
          imageSrc="/api/v1/courses/DEMO/files/x.png"
          onApply={onApply}
          onClose={onClose}
        />
      </AltTextEnforcementProvider>,
    )
    expect(screen.getByText(/Add alt text \(required for accessibility\)/i)).toBeInTheDocument()
    const input = screen.getByLabelText(/Alternative text for image/i)
    fireEvent.change(input, { target: { value: 'A chart showing growth' } })
    fireEvent.click(screen.getByRole('button', { name: /Save alt text/i }))
    expect(onApply).toHaveBeenCalledWith({
      alt: 'A chart showing growth',
      decorative: false,
    })
  })

  it('marks image decorative with empty alt', () => {
    const onApply = vi.fn()
    render(
      <AltTextEnforcementProvider value={{ enabled: true, hardBlock: false }}>
        <AltTextPanel
          alt="placeholder"
          decorative={false}
          imageSrc="/img.png"
          onApply={onApply}
          onClose={() => {}}
        />
      </AltTextEnforcementProvider>,
    )
    fireEvent.click(screen.getByRole('checkbox', { name: /Mark as decorative/i }))
    fireEvent.click(screen.getByRole('button', { name: /Save alt text/i }))
    expect(onApply).toHaveBeenCalledWith({ alt: '', decorative: true })
  })
})
