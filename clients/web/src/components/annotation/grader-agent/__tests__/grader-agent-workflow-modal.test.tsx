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

  it('shows a save menu while the agent is editable', async () => {
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
    const saveButton = await screen.findByRole('button', { name: /gradingAgent.save/i })
    expect(saveButton).toBeInTheDocument()
    fireEvent.click(saveButton)
    expect(screen.getByRole('menu', { name: 'gradingAgent.save.menuLabel' })).toBeInTheDocument()
    expect(screen.getByRole('menuitem', { name: 'gradingAgent.save.option' })).toBeInTheDocument()
    expect(screen.getByRole('menuitem', { name: 'gradingAgent.save.asTemplate' })).toBeInTheDocument()
  })

  it('shows save and save-as-template when the agent is accepted', async () => {
    const { fetchGraderAgentConfig } = await import('../../../../lib/courses-api')
    vi.mocked(fetchGraderAgentConfig).mockResolvedValueOnce({
      config: {
        id: '00000000-0000-0000-0000-000000000010',
        prompt: 'Grade fairly.',
        includeAssignmentContent: false,
        includeRubric: false,
        status: 'accepted',
        autoGradeNew: false,
        updatedAt: '2026-01-01T00:00:00.000Z',
      },
    })

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

    const saveButton = await screen.findByRole('button', { name: /gradingAgent.save/i })
    fireEvent.click(saveButton)
    expect(screen.getByRole('menuitem', { name: 'gradingAgent.save.option' })).toBeInTheDocument()
    expect(screen.getByRole('menuitem', { name: 'gradingAgent.save.asTemplate' })).toBeInTheDocument()
  })

  it('shows accept agent in the save menu while the agent is editable', async () => {
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
    const saveButton = await screen.findByRole('button', { name: /gradingAgent.save/i })
    fireEvent.click(saveButton)
    expect(screen.getByRole('menuitem', { name: 'gradingAgent.accept' })).toBeInTheDocument()
  })

  it('shows grouped palette nodes while the agent is editable', () => {
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
    expect(screen.getByText('gradingAgent.canvas.palette.groupInput')).toBeInTheDocument()
    expect(screen.getByText('gradingAgent.canvas.palette.groupProcessing')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'gradingAgent.canvas.palette.studentSubmission' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'gradingAgent.canvas.palette.activity' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'gradingAgent.canvas.palette.ai' })).toBeInTheDocument()
  })

  it('shows grouped palette nodes when the agent is accepted', async () => {
    const { fetchGraderAgentConfig } = await import('../../../../lib/courses-api')
    vi.mocked(fetchGraderAgentConfig).mockResolvedValueOnce({
      config: {
        id: '00000000-0000-0000-0000-000000000010',
        prompt: 'Grade fairly.',
        includeAssignmentContent: false,
        includeRubric: false,
        status: 'accepted',
        autoGradeNew: false,
        updatedAt: '2026-01-01T00:00:00.000Z',
      },
    })

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

    expect(await screen.findByText('gradingAgent.canvas.palette.groupInput')).toBeInTheDocument()
    expect(screen.getByText('gradingAgent.canvas.palette.groupProcessing')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'gradingAgent.canvas.palette.studentSubmission' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'gradingAgent.canvas.palette.activity' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'gradingAgent.canvas.palette.ai' })).toBeInTheDocument()
  })
})
