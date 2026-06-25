import { act, renderHook, waitFor, type RenderHookResult } from '@testing-library/react'
import { describe, expect, it, vi, beforeEach } from 'vitest'
import {
  fetchGraderAgentConfig,
  fetchGraderAgentRun,
  fetchCourseCanvasLink,
  postGraderAgentRun,
  putGraderAgentConfig,
  type GraderAgentConfigApi,
  type GraderAgentRunStatus,
} from '../../../../lib/courses-api'
import { useGraderAgentWorkflow } from '../use-grader-agent-workflow'

// Stable t reference — the polling effect has `t` in its deps; an unstable t
// (a new arrow function on every render) would re-trigger the effect on every
// render, causing an infinite poll loop in tests.
const stableT = (key: string): string => key
vi.mock('react-i18next', () => ({
  useTranslation: () => ({ t: stableT }),
}))

vi.mock('../../../../context/platform-features-context', () => ({
  usePlatformFeatures: () => ({ graderAgentSuggestModeEnabled: false }),
}))

vi.mock('../../../../lib/courses-api', () => ({
  fetchGraderAgentConfig: vi.fn(),
  fetchGraderAgentRun: vi.fn(),
  fetchCourseCanvasLink: vi.fn(),
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

// Make the workflow graph pass validation so handleRun doesn't early-return
vi.mock('../validation', () => ({
  validateWorkflowGraph: () => [],
  isWorkflowRunnable: () => true,
}))

function makeRunStatus(status: string, completed = 1, total = 1): GraderAgentRunStatus {
  return { status, completedCount: completed, failedCount: 0, totalCount: total, results: [] }
}

const draftConfig: GraderAgentConfigApi = {
  status: 'draft',
  prompt: '',
  includeAssignmentContent: false,
  includeRubric: false,
  autoGradeNew: false,
  postPolicy: 'draft',
  workflowGraph: undefined,
}

function defaultArgs(overrides: Record<string, unknown> = {}) {
  return {
    open: true,
    courseCode: 'demo',
    itemId: 'item-1',
    submissionId: 'sub-1',
    ...overrides,
  }
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(fetchGraderAgentConfig).mockResolvedValue({ config: null })
  vi.mocked(fetchCourseCanvasLink).mockResolvedValue({ linked: false, gradeSyncEnabled: false })
  vi.mocked(putGraderAgentConfig).mockResolvedValue({ config: draftConfig })
})

type WorkflowHookResult = RenderHookResult<
  ReturnType<typeof useGraderAgentWorkflow>,
  Parameters<typeof useGraderAgentWorkflow>[0]
>

async function startRun(result: WorkflowHookResult['result'], runId = 'run-1') {
  vi.mocked(postGraderAgentRun).mockResolvedValue({ runId, totalCount: 1, mode: 'apply' })
  await act(async () => {
    await result.current.handleRun()
  })
}

describe('useGraderAgentWorkflow — polling robustness', () => {
  it('calls onApplied exactly once when run is done on first poll (AC-1, AC-2)', async () => {
    const onApplied = vi.fn()
    vi.mocked(fetchGraderAgentRun).mockResolvedValue(makeRunStatus('done'))

    const { result } = renderHook(() => useGraderAgentWorkflow(defaultArgs({ onApplied })))
    await startRun(result)

    await waitFor(() => expect(result.current.batchRunning).toBe(false))
    expect(onApplied).toHaveBeenCalledTimes(1)
  })

  it.each(['error', 'failed', 'cancelled'])(
    'treats "%s" as terminal: stops polling, calls onApplied once (AC-3, FR-4)',
    async (status) => {
      const onApplied = vi.fn()
      vi.mocked(fetchGraderAgentRun).mockResolvedValue(makeRunStatus(status))

      const { result } = renderHook(() => useGraderAgentWorkflow(defaultArgs({ onApplied })))
      await startRun(result)

      await waitFor(() => expect(result.current.batchRunning).toBe(false))
      expect(onApplied).toHaveBeenCalledTimes(1)
    },
  )

  it('calls onApplied exactly once when run finishes after several polls (AC-2)', async () => {
    const onApplied = vi.fn()
    vi.mocked(fetchGraderAgentRun)
      .mockResolvedValueOnce(makeRunStatus('running', 0, 3))
      .mockResolvedValueOnce(makeRunStatus('running', 1, 3))
      .mockResolvedValue(makeRunStatus('done', 3, 3))

    vi.useFakeTimers()
    try {
      const { result } = renderHook(() => useGraderAgentWorkflow(defaultArgs({ onApplied })))
      await startRun(result)

      // First poll resolves (running). Advance to trigger second and third interval fires.
      await act(async () => { vi.advanceTimersByTime(1500) })
      await act(async () => { vi.advanceTimersByTime(1500) })
    } finally {
      vi.useRealTimers()
    }

    expect(onApplied).toHaveBeenCalledTimes(1)
  })

  it('calls onApplied exactly once when two concurrent polls both see terminal status (FR-3, FR-5)', async () => {
    const onApplied = vi.fn()

    let resolveFirst!: (v: GraderAgentRunStatus) => void
    let resolveSecond!: (v: GraderAgentRunStatus) => void
    vi.mocked(fetchGraderAgentRun)
      .mockReturnValueOnce(new Promise((r) => { resolveFirst = r }))
      .mockReturnValueOnce(new Promise((r) => { resolveSecond = r }))

    vi.useFakeTimers()
    try {
      const { result } = renderHook(() => useGraderAgentWorkflow(defaultArgs({ onApplied })))
      await startRun(result)

      // First poll is in-flight; advance to start the second interval poll before first resolves
      await act(async () => { vi.advanceTimersByTime(1500) })

      // Both polls resolve with terminal 'done'
      await act(async () => {
        resolveFirst(makeRunStatus('done'))
        resolveSecond(makeRunStatus('done'))
        await Promise.resolve()
      })
    } finally {
      vi.useRealTimers()
    }

    expect(onApplied).toHaveBeenCalledTimes(1)
  })
})
