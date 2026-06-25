import { act, renderHook } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { useGraderAgentWorkflow, type GraderAgentWorkflowSeed } from '../use-grader-agent-workflow'
import { WORKFLOW_VERSION } from '../types'

const stableT = (key: string): string => key
vi.mock('react-i18next', () => ({
  useTranslation: () => ({ t: stableT }),
}))

vi.mock('../../../../context/platform-features-context', () => ({
  usePlatformFeatures: () => ({
    graderAgentSuggestModeEnabled: false,
    graderAgentRunFiltersEnabled: false,
  }),
}))

vi.mock('../../../../lib/courses-api', () => ({
  fetchGraderAgentConfig: vi.fn(),
  fetchGraderAgentRun: vi.fn(),
  fetchCourseCanvasLink: vi.fn().mockResolvedValue({ linked: false, gradeSyncEnabled: false }),
  postGraderAgentRun: vi.fn(),
  putGraderAgentConfig: vi.fn(),
  putSubmissionGrade: vi.fn(),
  streamGraderAgentDryRun: vi.fn(),
  postGraderAgentTemplate: vi.fn(),
  putGraderAgentTemplate: vi.fn(),
  fetchSubmissionGrade: vi.fn(),
}))

vi.mock('../use-rubric-library-rubrics', () => ({
  useRubricLibraryRubrics: () => ({ libraryRubrics: [], setLibraryRubricAvailability: vi.fn() }),
}))

vi.mock('../../canvas/canvas-grade-sync', () => ({
  queueCanvasGradeSync: vi.fn().mockReturnValue(null),
}))

vi.mock('../validation', () => ({
  validateWorkflowGraph: () => [],
  isWorkflowRunnable: () => true,
}))

const templateMode = { name: 'Template' } as const

const aiPatchSeedWorkflow: GraderAgentWorkflowSeed = {
  prompt: '',
  includeAssignmentContent: false,
  includeRubric: false,
  workflowGraph: {
    version: WORKFLOW_VERSION,
    nodes: [
      { id: 'ai-1', type: 'ai' as const, position: { x: 0, y: 0 }, data: { prompt: 'before' } },
      { id: 'output', type: 'output' as const, position: { x: 200, y: 0 }, data: {} },
    ],
    edges: [],
  },
}

const removeNodeSeedWorkflow: GraderAgentWorkflowSeed = {
  prompt: '',
  includeAssignmentContent: false,
  includeRubric: false,
  workflowGraph: {
    version: WORKFLOW_VERSION,
    nodes: [
      { id: 'ai-1', type: 'ai' as const, position: { x: 0, y: 0 }, data: {} },
      { id: 'output', type: 'output' as const, position: { x: 200, y: 0 }, data: {} },
    ],
    edges: [{ id: 'e-1', source: 'ai-1', target: 'output' }],
  },
}

const removeEdgeSeedWorkflow: GraderAgentWorkflowSeed = {
  prompt: '',
  includeAssignmentContent: false,
  includeRubric: false,
  workflowGraph: {
    version: WORKFLOW_VERSION,
    nodes: [
      { id: 'ai-1', type: 'ai' as const, position: { x: 0, y: 0 }, data: {} },
      { id: 'output', type: 'output' as const, position: { x: 200, y: 0 }, data: {} },
    ],
    edges: [
      { id: 'e-1', source: 'ai-1', target: 'output' },
      { id: 'e-2', source: 'ai-1', target: 'output', targetHandle: 'comments' },
    ],
  },
}

function templateWorkflowArgs(seedWorkflow: GraderAgentWorkflowSeed) {
  return {
    open: true,
    courseCode: 'demo',
    itemId: 'item-1',
    submissionId: null,
    templateMode,
    seedWorkflow,
  }
}

describe('useGraderAgentWorkflow graph mutations', () => {
  it('updateNodeData merges a patch into the selected node', async () => {
    const { result } = renderHook(() =>
      useGraderAgentWorkflow(templateWorkflowArgs(aiPatchSeedWorkflow)),
    )

    await act(async () => {
      await Promise.resolve()
    })

    act(() => {
      result.current.updateNodeData('ai-1', { prompt: 'after' })
    })

    const aiNode = result.current.graph?.nodes.find((node) => node.id === 'ai-1')
    expect(aiNode?.data.prompt).toBe('after')
  })

  it('removeNode drops the node, connected edges, and clears selection', async () => {
    const { result } = renderHook(() =>
      useGraderAgentWorkflow(templateWorkflowArgs(removeNodeSeedWorkflow)),
    )

    await act(async () => {
      await Promise.resolve()
    })

    act(() => {
      result.current.setSelectedNodeId('ai-1')
    })
    act(() => {
      result.current.removeNode('ai-1')
    })

    expect(result.current.graph?.nodes.some((node) => node.id === 'ai-1')).toBe(false)
    expect(result.current.graph?.edges).toEqual([])
    expect(result.current.selectedNodeId).toBeNull()
  })

  it('removeEdge drops only the matching edge', async () => {
    const { result } = renderHook(() =>
      useGraderAgentWorkflow(templateWorkflowArgs(removeEdgeSeedWorkflow)),
    )

    await act(async () => {
      await Promise.resolve()
    })

    act(() => {
      result.current.removeEdge('e-1')
    })

    expect(result.current.graph?.edges).toEqual([
      { id: 'e-2', source: 'ai-1', target: 'output', targetHandle: 'comments' },
    ])
  })
})
