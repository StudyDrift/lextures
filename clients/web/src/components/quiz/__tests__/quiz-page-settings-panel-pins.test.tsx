import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { defaultQuizAdvancedSettings } from '../../../lib/courses-api'
import { QuizPageSettingsPanel } from '../quiz-page-settings-panel'
import {
  __resetPinnedHintDismissedForTests,
  __resetSuggestionsDismissedForTests,
} from '../../settings-panel/use-pinned-settings'

const mockFetchDetailed = vi.fn()
const mockSave = vi.fn()
let ffPinnedSettings = true

vi.mock('../../../lib/pinned-settings-api', () => ({
  fetchPinnedSettingsDetailed: (...args: unknown[]) => mockFetchDetailed(...args),
  fetchPinnedSettings: async () => {
    const r = await mockFetchDetailed()
    return r.surfaces
  },
  savePinnedSettings: (...args: unknown[]) => mockSave(...args),
}))

vi.mock('../../../lib/lms-toast', () => ({
  toastMutationError: vi.fn(),
}))

vi.mock('../../../context/platform-features-context', () => ({
  usePlatformFeatures: () => ({ ffPinnedSettings }),
}))

function renderPanel(
  overrides: Partial<Parameters<typeof QuizPageSettingsPanel>[0]> = {},
) {
  const advanced = overrides.advanced ?? defaultQuizAdvancedSettings()
  return render(
    <QuizPageSettingsPanel
      dueLocal=""
      onDueLocalChange={vi.fn()}
      availableFromLocal=""
      onAvailableFromLocalChange={vi.fn()}
      availableUntilLocal=""
      onAvailableUntilLocalChange={vi.fn()}
      unlimitedAttempts={false}
      onUnlimitedAttemptsChange={vi.fn()}
      oneQuestionAtATime={false}
      onOneQuestionAtATimeChange={vi.fn()}
      pointsWorth={null}
      onPointsWorthChange={vi.fn()}
      gradingGroups={[]}
      assignmentGroupId={null}
      onAssignmentGroupChange={vi.fn()}
      advanced={advanced}
      onAdvancedChange={vi.fn()}
      showAdaptiveSection={false}
      lockdownDeliveryEnabled
      lockdownMode="standard"
      onLockdownModeChange={vi.fn()}
      focusLossThreshold={null}
      onFocusLossThresholdChange={vi.fn()}
      {...overrides}
    />,
  )
}

describe('QuizPageSettingsPanel — pinned settings (PS.3)', () => {
  beforeEach(() => {
    ffPinnedSettings = true
    // Dismiss first-run hint + suggestions so AC-1/AC-2 assert the Pinned group, not the coach.
    __resetPinnedHintDismissedForTests(true)
    __resetSuggestionsDismissedForTests('quiz', true)
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

  it('AC-12: flag off → no pin toggles and no pin API traffic', async () => {
    ffPinnedSettings = false
    renderPanel()
    expect(screen.queryByRole('button', { name: /Pin .* to top/i })).not.toBeInTheDocument()
    expect(screen.queryByRole('heading', { name: /^Pinned/ })).not.toBeInTheDocument()
    await waitFor(() => {
      expect(mockFetchDetailed).not.toHaveBeenCalled()
    })
  })

  it('AC-1: zero pins → no Pinned group (hint dismissed)', async () => {
    renderPanel()
    await waitFor(() => expect(mockFetchDetailed).toHaveBeenCalled())
    expect(screen.queryByRole('heading', { name: /^Pinned/ })).not.toBeInTheDocument()
  })

  it('shows suggested pins strip when zero pins and not dismissed (PS.4)', async () => {
    __resetPinnedHintDismissedForTests(false)
    __resetSuggestionsDismissedForTests('quiz', false)
    renderPanel()
    await waitFor(() => {
      expect(screen.getByTestId('suggested-pins-strip')).toBeInTheDocument()
    })
    // Generic PS.3 first-run hint must not stack with suggestions (FR-2).
    expect(screen.queryByText(/Pin the settings you use most/i)).not.toBeInTheDocument()
  })

  it('AC-2: pin lockdown → appears in Pinned group once + section hint', async () => {
    const user = userEvent.setup()
    renderPanel()
    await waitFor(() => expect(mockFetchDetailed).toHaveBeenCalled())

    // Expand Presentation
    const presentation = screen.getByText('Presentation').closest('details')
    expect(presentation).toBeTruthy()
    if (presentation && !presentation.hasAttribute('open')) {
      await user.click(within(presentation).getByText('Presentation'))
    }

    const pinBtn = await screen.findByRole('button', {
      name: 'Pin Lockdown delivery to top',
    })
    await user.click(pinBtn)

    await waitFor(() => {
      expect(screen.getByRole('heading', { name: /^Pinned/ })).toBeInTheDocument()
    })
    expect(screen.getByText('1 pinned to top')).toBeInTheDocument()

    // Control appears exactly once
    expect(screen.getAllByLabelText('Lockdown delivery')).toHaveLength(1)
    // And lives under the Pinned group
    const pinnedSection = screen.getByRole('heading', { name: /^Pinned/ }).closest('section')
    expect(pinnedSection).toBeTruthy()
    expect(within(pinnedSection!).getByLabelText('Lockdown delivery')).toBeInTheDocument()
  })

  it('AC-8: pinned max attempts hidden while unlimited is on', async () => {
    mockFetchDetailed.mockResolvedValue({
      ok: true,
      surfaces: {
        assignment: [],
        quiz: ['quiz.attempts-grading.max-attempts'],
      },
    })
    const { rerender } = renderPanel({ unlimitedAttempts: true })
    await waitFor(() => expect(mockFetchDetailed).toHaveBeenCalled())

    // Max attempts control not mounted → absent from Pinned group
    expect(document.getElementById('quiz-max-attempts')).toBeNull()
    expect(screen.queryByRole('heading', { name: /^Pinned/ })).not.toBeInTheDocument()

    rerender(
      <QuizPageSettingsPanel
        dueLocal=""
        onDueLocalChange={vi.fn()}
        availableFromLocal=""
        onAvailableFromLocalChange={vi.fn()}
        availableUntilLocal=""
        onAvailableUntilLocalChange={vi.fn()}
        unlimitedAttempts={false}
        onUnlimitedAttemptsChange={vi.fn()}
        oneQuestionAtATime={false}
        onOneQuestionAtATimeChange={vi.fn()}
        pointsWorth={null}
        onPointsWorthChange={vi.fn()}
        gradingGroups={[]}
        assignmentGroupId={null}
        onAssignmentGroupChange={vi.fn()}
        advanced={defaultQuizAdvancedSettings()}
        onAdvancedChange={vi.fn()}
        showAdaptiveSection={false}
        lockdownDeliveryEnabled
        lockdownMode="standard"
        onLockdownModeChange={vi.fn()}
        focusLossThreshold={null}
        onFocusLossThresholdChange={vi.fn()}
      />,
    )

    await waitFor(() => {
      expect(document.getElementById('quiz-max-attempts')).toBeTruthy()
    })
    expect(screen.getByRole('heading', { name: /^Pinned/ })).toBeInTheDocument()
  })

  it('loads existing pins into Pinned group', async () => {
    mockFetchDetailed.mockResolvedValue({
      ok: true,
      surfaces: {
        assignment: [],
        quiz: ['quiz.presentation.lockdown-mode'],
      },
    })
    renderPanel()
    await waitFor(() => {
      expect(screen.getByRole('heading', { name: /^Pinned/ })).toBeInTheDocument()
    })
    expect(screen.getByLabelText('Lockdown delivery')).toBeInTheDocument()
  })
})
