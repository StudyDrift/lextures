import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { ChecklistBadge } from '../checklist-badge'

describe('ChecklistBadge', () => {
  it('renders nothing when outstanding is zero', () => {
    const { container } = render(<ChecklistBadge outstandingEssential={0} />)
    expect(container).toBeEmptyDOMElement()
  })

  it('caps display at 99+ but aria-label uses the true count', () => {
    render(<ChecklistBadge outstandingEssential={137} />)
    expect(screen.getByText('99+')).toBeInTheDocument()
    expect(screen.getByLabelText('137 checklist items need attention')).toBeInTheDocument()
  })

  it('shows the exact count under 100', () => {
    render(<ChecklistBadge outstandingEssential={8} />)
    expect(screen.getByText('8')).toBeInTheDocument()
    expect(screen.getByLabelText('8 checklist items need attention')).toBeInTheDocument()
  })
})
