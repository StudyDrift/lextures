import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { SelfPacedProgressBar } from './self-paced-progress'

describe('SelfPacedProgressBar', () => {
  it('exposes an accessible progressbar with the percentage', () => {
    render(<SelfPacedProgressBar percent={30} />)
    const bar = screen.getByRole('progressbar')
    expect(bar).toHaveAttribute('aria-valuenow', '30')
    expect(bar).toHaveAttribute('aria-valuemin', '0')
    expect(bar).toHaveAttribute('aria-valuemax', '100')
    expect(screen.getByText('30% complete')).toBeInTheDocument()
  })

  it('clamps out-of-range values', () => {
    render(<SelfPacedProgressBar percent={150} />)
    expect(screen.getByRole('progressbar')).toHaveAttribute('aria-valuenow', '100')
  })
})
