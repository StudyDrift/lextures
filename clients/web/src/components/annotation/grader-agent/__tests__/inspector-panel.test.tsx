import { describe, expect, it, vi } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { InspectorPanel } from '../inspector-panel'
import type { GraderAgentWorkflowState } from '../use-grader-agent-workflow'
import { WORKFLOW_VERSION } from '../types'

vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (key: string) => key,
  }),
}))

vi.mock('../../../../hooks/use-text-models', () => ({
  useTextModels: () => ({ models: [] }),
}))

vi.mock('../../../../hooks/use-course-assignments', () => ({
  useCourseAssignments: () => ({ assignments: [], loading: false, error: null }),
}))

vi.mock('../workflow-prompt-editor', () => ({
  WorkflowPromptEditor: () => <div data-testid="workflow-prompt-editor" />,
}))

function workflowStub(overrides: Partial<GraderAgentWorkflowState> = {}): GraderAgentWorkflowState {
  const graph = {
    version: WORKFLOW_VERSION,
    nodes: [
      { id: 'sub-1', type: 'studentSubmission' as const, position: { x: 0, y: 0 }, data: {} },
      { id: 'output', type: 'output' as const, position: { x: 200, y: 0 }, data: {} },
    ],
    edges: [],
  }
  return {
    graph,
    selectedNodeId: null,
    updateNodeData: vi.fn(),
    removeNode: vi.fn(),
    nodeDryRunDetails: {},
    nodeExecutionStates: {},
    ...overrides,
  } as GraderAgentWorkflowState
}

describe('InspectorPanel', () => {
  it('prompts to select a node when nothing is selected', () => {
    render(
      <InspectorPanel
        workflow={workflowStub()}
        courseCode="demo"
        itemId="item-1"
      />,
    )
    expect(screen.getByText('gradingAgent.canvas.inspector.empty')).toBeInTheDocument()
  })

  it('renders quiz question previews for a selected quiz responses node', async () => {
    render(
      <InspectorPanel
        workflow={workflowStub({
          selectedNodeId: 'quiz-responses',
          graph: {
            version: WORKFLOW_VERSION,
            nodes: [
              { id: 'quiz-responses', type: 'quizResponses', position: { x: 0, y: 0 }, data: {} },
              { id: 'output', type: 'output', position: { x: 200, y: 0 }, data: {} },
            ],
            edges: [],
          },
        })}
        courseCode="demo"
        itemId="quiz-1"
        quizQuestionSlots={[
          {
            index: 0,
            label: 'Question 1',
            questionType: 'multiple_choice',
            needsManualGrading: false,
            isPoolSlot: false,
            isShuffled: false,
            maxPoints: 10,
            promptPreview: 'What is 2+2?',
          },
        ]}
        quizQuestions={[
          {
            id: 'q1',
            prompt: 'What is 2+2?',
            questionType: 'multiple_choice',
            choices: ['3', '4'],
            correctChoiceIndex: 1,
            multipleAnswer: false,
            answerWithImage: false,
            required: true,
            points: 10,
            estimatedMinutes: 0,
          },
        ]}
      />,
    )
    expect(screen.getByText('gradingAgent.canvas.inspector.quizResponses.help')).toBeInTheDocument()
    await waitFor(() => {
      expect(screen.getByText('What is 2+2?')).toBeInTheDocument()
    })
    expect(screen.getByText('3')).toBeInTheDocument()
    expect(screen.getByText('4')).toBeInTheDocument()
  })

  it('renders submission inspector content for a selected student submission node', () => {
    render(
      <InspectorPanel
        workflow={workflowStub({ selectedNodeId: 'sub-1' })}
        courseCode="demo"
        itemId="item-1"
        selectedSubmission={{
          id: 'submission-1',
          attachmentFileId: 'file-a',
          attachments: [
            {
              fileId: 'file-a',
              filename: 'essay.pdf',
              mimeType: 'application/pdf',
              contentPath: '/api/v1/files/a',
            },
          ],
        }}
      />,
    )
    expect(screen.getByText('gradingAgent.canvas.inspector.submissionHelp')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'essay.pdf' })).toBeInTheDocument()
  })
})