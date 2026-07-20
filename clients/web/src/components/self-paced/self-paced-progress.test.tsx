import { render, screen } from '@testing-library/react'
import type { ReactNode } from 'react'
import { describe, expect, it } from 'vitest'
import { PlatformFeaturesProvider } from '../../context/platform-features-context'
import { SelfPacedProgressBar } from './self-paced-progress'

function wrap(ui: ReactNode) {
  return render(<PlatformFeaturesProvider>{ui}</PlatformFeaturesProvider>)
}

describe('SelfPacedProgressBar', () => {
  it('exposes an accessible progressbar with the percentage', () => {
    wrap(<SelfPacedProgressBar percent={30} />)
    const bar = screen.getByRole('progressbar')
    expect(bar).toHaveAttribute('aria-valuenow', '30')
    expect(bar).toHaveAttribute('aria-valuemin', '0')
    expect(bar).toHaveAttribute('aria-valuemax', '100')
    expect(screen.getByText('30% complete')).toBeInTheDocument()
  })

  it('clamps out-of-range values', () => {
    wrap(<SelfPacedProgressBar percent={150} />)
    expect(screen.getByRole('progressbar')).toHaveAttribute('aria-valuenow', '100')
  })
})
