import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { FeatureHelpProvider } from '../../../context/feature-help-context'
import { FeatureHelpDock } from '../feature-help-dock'

const openHelp = vi.fn()
const closeHelp = vi.fn()

let mockState = { open: false, topic: null as 'gradebook' | 'modules' | null }

vi.mock('../../../context/feature-help-context', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../../../context/feature-help-context')>()
  return {
    ...actual,
    useFeatureHelp: () => ({
      state: mockState,
      openHelp,
      closeHelp,
    }),
  }
})

function renderDock() {
  return render(
    <FeatureHelpProvider>
      <FeatureHelpDock />
    </FeatureHelpProvider>,
  )
}

describe('FeatureHelpDock (plan W06)', () => {
  beforeEach(() => {
    mockState = { open: false, topic: null }
    closeHelp.mockClear()
  })

  it('renders nothing when closed', () => {
    renderDock()
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })

  it('shows text help without placeholder copy for topics without media', () => {
    mockState = { open: true, topic: 'gradebook' }
    renderDock()

    expect(screen.getByRole('dialog', { name: /gradebook help/i })).toBeInTheDocument()
    expect(screen.getByText(/double-click to edit scores/i)).toBeInTheDocument()
    expect(screen.queryByText(/placeholder/i)).not.toBeInTheDocument()
    expect(screen.queryByText(/when ready/i)).not.toBeInTheDocument()
    expect(screen.queryByRole('figure')).not.toBeInTheDocument()
  })

  it('lazy-loads walkthrough media for topics with a configured clip', () => {
    mockState = { open: true, topic: 'modules' }
    renderDock()

    const video = screen.getByLabelText(/reorder course modules/i)
    expect(video).toHaveAttribute('src', '/feature-help/modules-walkthrough.mp4')
    expect(video).toHaveAttribute('preload', 'none')
  })

  it('omits the media region when the clip fails to load', async () => {
    mockState = { open: true, topic: 'modules' }
    renderDock()

    const video = screen.getByLabelText(/reorder course modules/i)
    fireEvent.error(video)

    await waitFor(() => {
      expect(screen.queryByRole('figure')).not.toBeInTheDocument()
    })
    expect(screen.getByText(/drag handles reorder your outline/i)).toBeInTheDocument()
  })

  it('closes on Escape', () => {
    mockState = { open: true, topic: 'gradebook' }
    renderDock()

    fireEvent.keyDown(window, { key: 'Escape' })
    expect(closeHelp).toHaveBeenCalledTimes(1)
  })
})