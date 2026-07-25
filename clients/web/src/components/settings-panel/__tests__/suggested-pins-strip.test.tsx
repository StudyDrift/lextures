import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { onSettingsTelemetry } from '../../../lib/settings-telemetry'
import {
  usePinnedSettings,
  __resetPinnedHintDismissedForTests,
  __resetSuggestionsDismissedForTests,
} from '../use-pinned-settings'
import { SuggestedPinsStrip } from '../suggested-pins-strip'
import { PinnedSettingsGroup } from '../pinned-settings-group'
import { SettingsPanelProvider } from '../settings-panel-context'

const mockFetchDetailed = vi.fn()
const mockSave = vi.fn()
let ffPinnedSettings = true

vi.mock('../../../lib/pinned-settings-api', () => ({
  fetchPinnedSettingsDetailed: (...args: unknown[]) => mockFetchDetailed(...args),
  savePinnedSettings: (...args: unknown[]) => mockSave(...args),
}))

vi.mock('../../../lib/lms-toast', () => ({
  toastMutationError: vi.fn(),
}))

vi.mock('../../../context/platform-features-context', () => ({
  usePlatformFeatures: () => ({ ffPinnedSettings }),
}))

function Harness({ surface = 'quiz' as const }: { surface?: 'quiz' | 'assignment' }) {
  const pins = usePinnedSettings(surface)
  return (
    <SettingsPanelProvider surface={surface} query="" pins={pins}>
      <PinnedSettingsGroup pins={pins} visiblePinned={pins.resolved} />
    </SettingsPanelProvider>
  )
}

function StripOnly() {
  const pins = usePinnedSettings('quiz')
  return <SuggestedPinsStrip pins={pins} />
}

describe('Suggested pins strip (PS.4)', () => {
  beforeEach(() => {
    ffPinnedSettings = true
    __resetPinnedHintDismissedForTests(false)
    __resetSuggestionsDismissedForTests('quiz', false)
    __resetSuggestionsDismissedForTests('assignment', false)
    mockFetchDetailed.mockReset()
    mockSave.mockReset()
    mockFetchDetailed.mockResolvedValue({
      ok: true,
      surfaces: { assignment: [], quiz: [] },
    })
    mockSave.mockImplementation(async (_s: string, keys: string[]) => ({
      assignment: [],
      quiz: keys,
    }))
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('AC-1: zero pins, not dismissed → strip renders; generic hint does not', async () => {
    render(<Harness />)
    await waitFor(() => {
      expect(screen.getByTestId('suggested-pins-strip')).toBeInTheDocument()
    })
    const strip = screen.getByTestId('suggested-pins-strip')
    expect(strip).toHaveTextContent(/Suggested pins/i)
    expect(screen.queryByText(/Pin the settings you use most/i)).not.toBeInTheDocument()
    // Curated quiz suggestion present
    expect(screen.getByRole('button', { name: /Pin Due date to top/i })).toBeInTheDocument()
  })

  it('AC-2: accepting a suggestion pins via normal path and hides strip', async () => {
    const user = userEvent.setup()
    const events: string[] = []
    const unsub = onSettingsTelemetry((e) => events.push(e.event))
    render(<Harness />)
    await waitFor(() => screen.getByTestId('suggested-pins-strip'))

    await user.click(screen.getByRole('button', { name: /Pin Due date to top/i }))

    await waitFor(() => {
      expect(screen.queryByTestId('suggested-pins-strip')).not.toBeInTheDocument()
    })
    expect(screen.getByRole('heading', { name: /^Pinned/ })).toBeInTheDocument()
    expect(events).toContain('settings_suggestion_accepted')
    expect(events).toContain('settings_pin_added')
    unsub()
  })

  it('AC-3: Not now dismisses permanently for the surface', async () => {
    const user = userEvent.setup()
    const events: string[] = []
    const unsub = onSettingsTelemetry((e) => events.push(e.event))
    const { unmount } = render(<Harness />)
    await waitFor(() => screen.getByTestId('suggested-pins-strip'))
    await user.click(screen.getByRole('button', { name: /Dismiss pin suggestions/i }))
    expect(screen.queryByTestId('suggested-pins-strip')).not.toBeInTheDocument()
    expect(events).toContain('settings_suggestion_dismissed')
    unsub()
    unmount()

    // Re-open: still dismissed
    render(<Harness />)
    await waitFor(() => expect(mockFetchDetailed).toHaveBeenCalled())
    expect(screen.queryByTestId('suggested-pins-strip')).not.toBeInTheDocument()
  })

  it('AC-4: existing pins → no strip', async () => {
    mockFetchDetailed.mockResolvedValue({
      ok: true,
      surfaces: {
        assignment: [],
        quiz: ['quiz.scheduling.due-date'],
      },
    })
    render(<Harness />)
    await waitFor(() => expect(mockFetchDetailed).toHaveBeenCalled())
    expect(screen.queryByTestId('suggested-pins-strip')).not.toBeInTheDocument()
  })

  it('AC-9: flag off → no strip and no suggestion telemetry', async () => {
    ffPinnedSettings = false
    const events: string[] = []
    const unsub = onSettingsTelemetry((e) => events.push(e.event))
    render(<StripOnly />)
    await waitFor(() => {
      expect(screen.queryByTestId('suggested-pins-strip')).not.toBeInTheDocument()
    })
    expect(events).toEqual([])
    unsub()
  })
})
