import { describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { GraderAgentWorkflowModal } from '../grader-agent-workflow-modal'

vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (key: string) => key,
  }),
}))

vi.mock('../../../../lib/courses-api', () => ({
  fetchGraderAgentConfig: vi.fn().mockResolvedValue({ config: null }),
  fetchGraderAgentRun: vi.fn(),
  postGraderAgentDryRun: vi.fn(),
  postGraderAgentRun: vi.fn(),
  putGraderAgentConfig: vi.fn(),
  putSubmissionGrade: vi.fn(),
}))

describe('GraderAgentWorkflowModal', () => {
  it('renders nothing when closed', () => {
    const { container } = render(
      <GraderAgentWorkflowModal
        open={false}
        onClose={() => undefined}
        courseCode="demo"
        itemId="00000000-0000-0000-0000-000000000001"
        submissionId={null}
        rubric={null}
        maxPoints={100}
      />,
    )
    expect(container.firstChild).toBeNull()
  })

  it('renders dialog when open', () => {
    render(
      <GraderAgentWorkflowModal
        open
        onClose={() => undefined}
        courseCode="demo"
        itemId="00000000-0000-0000-0000-000000000001"
        submissionId="00000000-0000-0000-0000-000000000002"
        rubric={null}
        maxPoints={100}
      />,
    )
    expect(screen.getByRole('dialog')).toBeInTheDocument()
  })

  it('shows a save button while the agent is editable', () => {
    render(
      <GraderAgentWorkflowModal
        open
        onClose={() => undefined}
        courseCode="demo"
        itemId="00000000-0000-0000-0000-000000000001"
        submissionId="00000000-0000-0000-0000-000000000002"
        rubric={null}
        maxPoints={100}
      />,
    )
    expect(screen.getByRole('button', { name: 'gradingAgent.save' })).toBeInTheDocument()
  })
})
