import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import { AssignmentPageSettingsPanel } from '../assignment-page-settings-panel'

// Search ACs are independent of pins; keep pin API off so MSW stays quiet.
vi.mock('../../../context/platform-features-context', () => ({
  usePlatformFeatures: () => ({ ffPinnedSettings: false }),
}))

function renderPanel(
  overrides: Partial<Parameters<typeof AssignmentPageSettingsPanel>[0]> = {},
) {
  return render(
    <AssignmentPageSettingsPanel
      dueLocal=""
      onDueLocalChange={vi.fn()}
      availableFromLocal=""
      onAvailableFromLocalChange={vi.fn()}
      availableUntilLocal=""
      onAvailableUntilLocalChange={vi.fn()}
      pointsWorth={null}
      onPointsWorthChange={vi.fn()}
      gradingGroups={[]}
      assignmentGroupId={null}
      onAssignmentGroupChange={vi.fn()}
      submissionAllowText
      onSubmissionAllowTextChange={vi.fn()}
      submissionAllowFileUpload={false}
      onSubmissionAllowFileUploadChange={vi.fn()}
      submissionAllowUrl={false}
      onSubmissionAllowUrlChange={vi.fn()}
      assignmentAccessCode=""
      onAssignmentAccessCodeChange={vi.fn()}
      lateSubmissionPolicy="allow"
      onLateSubmissionPolicyChange={vi.fn()}
      latePenaltyPercent={null}
      onLatePenaltyPercentChange={vi.fn()}
      draftRubric={null}
      onDraftRubricChange={vi.fn()}
      blindGrading={false}
      onBlindGradingChange={vi.fn()}
      moderatedGrading={false}
      onModeratedGradingChange={vi.fn()}
      moderationThresholdPct={10}
      onModerationThresholdPctChange={vi.fn()}
      moderatorUserId={null}
      onModeratorUserIdChange={vi.fn()}
      provisionalGraderUserIds={[]}
      onProvisionalGraderUserIdsChange={vi.fn()}
      staffDirectory={[]}
      originalityDetection="disabled"
      onOriginalityDetectionChange={vi.fn()}
      originalityStudentVisibility="hide"
      onOriginalityStudentVisibilityChange={vi.fn()}
      {...overrides}
    />,
  )
}

describe('AssignmentPageSettingsPanel — control-level search', () => {
  it('preserves DOM ids with no search query', () => {
    renderPanel()
    expect(document.getElementById('assignment-settings-due')).toBeTruthy()
    expect(document.getElementById('assignment-submission-text')).toBeTruthy()
    expect(document.getElementById('assignment-access-code')).toBeTruthy()
  })

  it('AC-3: nonsense query shows empty state', async () => {
    const user = userEvent.setup()
    renderPanel()
    await user.type(screen.getByLabelText('Search assignment settings'), 'zzzz')
    expect(screen.getByText(/No settings match/)).toBeInTheDocument()
  })

  it('filters to a single matching control', async () => {
    const user = userEvent.setup()
    renderPanel()
    await user.type(screen.getByLabelText('Search assignment settings'), 'blind')
    expect(screen.getByLabelText('Blind grading')).toBeInTheDocument()
    expect(screen.queryByLabelText('Due date')).not.toBeInTheDocument()
    expect(screen.queryByLabelText('Text entry')).not.toBeInTheDocument()
  })
})
