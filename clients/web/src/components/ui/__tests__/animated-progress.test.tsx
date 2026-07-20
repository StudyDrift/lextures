import { render, screen } from '@testing-library/react'
import type { ReactNode } from 'react'
import { describe, expect, it } from 'vitest'
import { PlatformFeaturesProvider } from '../../../context/platform-features-context'
import { AnimatedProgress } from '../animated-progress'

function wrap(ui: ReactNode) {
  return render(<PlatformFeaturesProvider>{ui}</PlatformFeaturesProvider>)
}

describe('AN.7 AnimatedProgress', () => {
  it('exposes accessible progressbar and fill (AC-1)', () => {
    wrap(<AnimatedProgress value={40} label="Module progress" />)
    const bar = screen.getByRole('progressbar', { name: 'Module progress' })
    expect(bar).toHaveAttribute('aria-valuenow', '40')
    expect(screen.getByTestId('animated-progress-fill')).toBeTruthy()
  })

  it('renders ring variant', () => {
    wrap(<AnimatedProgress value={75} variant="ring" label="Mastery" />)
    expect(screen.getByRole('progressbar', { name: 'Mastery' })).toBeTruthy()
  })
})
