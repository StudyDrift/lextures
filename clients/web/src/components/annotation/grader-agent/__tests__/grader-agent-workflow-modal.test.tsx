import { describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'
import { GraderAgentWorkflowModal } from '../grader-agent-workflow-modal'

vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (key: string) => key,
  }),
}))

vi.mock('../../../../lib/courses-api', () => ({
  fetchGraderAgentConfig: vi.fn().mockResolvedValue({ config: null }),
  fetchGraderAgentRun: vi.fn(),
  fetchModuleAssignmentSubmissions: vi.fn().mockResolvedValue([
    {
      id: '00000000-0000-0000-0000-000000000002',
      submittedByDisplayName: 'Ada Lovelace',
      attachmentFileId: null,
      submittedAt: '2026-01-01T00:00:00.000Z',
      updatedAt: '2026-01-01T00:00:00.000Z',
      isGraded: false,
    },
    {
      id: '00000000-0000-0000-0000-000000000003',
      submittedByDisplayName: 'Bob Builder',
      attachmentFileId: null,
      submittedAt: '2026-01-02T00:00:00.000Z',
      updatedAt: '2026-01-02T00:00:00.000Z',
      isGraded: true,
    },
  ]),
  streamGraderAgentDryRun: vi.fn(),
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

  it('does not close on Escape', () => {
    const onClose = vi.fn()
    render(
      <GraderAgentWorkflowModal
        open
        onClose={onClose}
        courseCode="demo"
        itemId="00000000-0000-0000-0000-000000000001"
        submissionId="00000000-0000-0000-0000-000000000002"
        rubric={null}
        maxPoints={100}
      />,
    )
    fireEvent.keyDown(document, { key: 'Escape' })
    expect(onClose).not.toHaveBeenCalled()
    expect(screen.getByRole('dialog')).toBeInTheDocument()
  })

  it('renders a filterable student picker in the header', async () => {
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
    expect(await screen.findByRole('button', { name: /Ada Lovelace/i })).toBeInTheDocument()
    expect(screen.getByText('1/2')).toBeInTheDocument()
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
