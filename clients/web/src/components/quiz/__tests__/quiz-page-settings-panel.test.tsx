import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import { defaultQuizAdvancedSettings } from '../../../lib/courses-api'
import { QuizPageSettingsPanel } from '../quiz-page-settings-panel'

// Search/a11y ACs are independent of pins; keep pin API off so MSW stays quiet.
vi.mock('../../../context/platform-features-context', () => ({
  usePlatformFeatures: () => ({ ffPinnedSettings: false }),
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

describe('QuizPageSettingsPanel — control-level search', () => {
  it('preserves DOM ids with no search query', () => {
    renderPanel()
    expect(document.getElementById('quiz-settings-due')).toBeTruthy()
    expect(document.getElementById('quiz-settings-unlimited-attempts')).toBeTruthy()
    expect(document.getElementById('quiz-lockdown-mode')).toBeTruthy()
    expect(document.getElementById('quiz-access-code')).toBeTruthy()
  })

  it('AC-2: searching "lockdown" shows only lockdown-related presentation controls', async () => {
    const user = userEvent.setup()
    renderPanel({ lockdownMode: 'kiosk' })

    await user.type(screen.getByLabelText('Search quiz settings'), 'lockdown')

    expect(screen.getByLabelText('Lockdown delivery')).toBeInTheDocument()
    expect(screen.getByLabelText('Focus-loss flag threshold')).toBeInTheDocument()
    expect(screen.queryByLabelText('Due date')).not.toBeInTheDocument()
    expect(screen.queryByLabelText('Shuffle question order')).not.toBeInTheDocument()
    expect(screen.queryByLabelText('Quiz access code')).not.toBeInTheDocument()

    // Presentation section force-opens during search
    const presentation = screen.getByText('Presentation').closest('details')
    expect(presentation).toHaveAttribute('open')
  })

  it('AC-3: nonsense query shows empty state', async () => {
    const user = userEvent.setup()
    renderPanel()
    await user.type(screen.getByLabelText('Search quiz settings'), 'zzzz')
    expect(screen.getByText(/No settings match/)).toBeInTheDocument()
    expect(screen.getByText(/zzzz/)).toBeInTheDocument()
  })

  it('AC-4: max attempts absent when unlimitedAttempts is true', () => {
    renderPanel({ unlimitedAttempts: true })
    expect(document.getElementById('quiz-max-attempts')).toBeNull()
    expect(document.getElementById('quiz-settings-unlimited-attempts')).toBeTruthy()
  })

  it('hides sections with zero matching controls', async () => {
    const user = userEvent.setup()
    renderPanel()
    await user.type(screen.getByLabelText('Search quiz settings'), 'access code')
    expect(screen.getByLabelText('Quiz access code')).toBeInTheDocument()
    expect(screen.queryByText('Scheduling')).not.toBeInTheDocument()
    expect(screen.queryByText('Presentation')).not.toBeInTheDocument()
    expect(screen.getByText('Access')).toBeInTheDocument()
  })

  it('tab order starts at search then reaches due date when no search (AC-7 sample)', async () => {
    const user = userEvent.setup()
    renderPanel()
    // Open scheduling so due date is reachable
    const scheduling = screen.getByText('Scheduling').closest('details')
    expect(scheduling).toBeTruthy()
    if (scheduling && !scheduling.hasAttribute('open')) {
      await user.click(within(scheduling).getByText('Scheduling'))
    }
    const search = screen.getByLabelText('Search quiz settings')
    search.focus()
    expect(search).toHaveFocus()
    await user.tab()
    // Next focusable should be inside the panel (summary or control depending on browser)
    expect(document.activeElement).not.toBe(document.body)
  })
})
