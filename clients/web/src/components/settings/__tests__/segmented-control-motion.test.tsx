import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import { PlatformFeaturesProvider } from '../../../context/platform-features-context'
import { SegmentedControl } from '../segmented-control'

describe('AN.6 SegmentedControl indicator', () => {
  it('renders sliding indicator and changes selection (FR-3 / AC-3)', async () => {
    const user = userEvent.setup()
    const onChange = vi.fn()
    // Stub layout measurements so the indicator mounts.
    Object.defineProperty(HTMLElement.prototype, 'offsetWidth', {
      configurable: true,
      get() {
        return 80
      },
    })

    render(
      <PlatformFeaturesProvider>
        <SegmentedControl
          aria-label="Theme"
          value="light"
          options={[
            { value: 'light', label: 'Light' },
            { value: 'dark', label: 'Dark' },
          ]}
          onChange={onChange}
        />
      </PlatformFeaturesProvider>,
    )

    expect(screen.getByTestId('segmented-indicator')).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Dark' }))
    expect(onChange).toHaveBeenCalledWith('dark')
  })
})
