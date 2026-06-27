import { renderHook, waitFor } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { fetchGraderAgentSubmissions } from '../../../../lib/courses-api'
import { useGraderAgentSubmissions } from '../use-grader-agent-submissions'

vi.mock('../../../../lib/courses-api', () => ({
  fetchGraderAgentSubmissions: vi.fn(),
}))

describe('useGraderAgentSubmissions', () => {
  it('selects the initial submission when provided', async () => {
    vi.mocked(fetchGraderAgentSubmissions).mockResolvedValue([
      {
        id: 'sub-a',
        submittedByDisplayName: 'Alice',
        attachmentFileId: null,
        submittedAt: '2026-01-01T00:00:00.000Z',
        updatedAt: '2026-01-01T00:00:00.000Z',
        isGraded: false,
      },
      {
        id: 'sub-b',
        submittedByDisplayName: 'Bob',
        attachmentFileId: null,
        submittedAt: '2026-01-02T00:00:00.000Z',
        updatedAt: '2026-01-02T00:00:00.000Z',
        isGraded: true,
      },
    ])

    const { result } = renderHook(() =>
      useGraderAgentSubmissions({
        open: true,
        courseCode: 'demo',
        itemId: 'item-1',
        initialSubmissionId: 'sub-b',
      }),
    )

    await waitFor(() => expect(result.current.loading).toBe(false))

    expect(fetchGraderAgentSubmissions).toHaveBeenCalledWith('demo', 'item-1', 'assignment', {
      graded: 'all',
    })
    expect(result.current.index).toBe(1)
    expect(result.current.selectedSubmissionId).toBe('sub-b')
  })

  it('loads quiz attempts when itemKind is quiz', async () => {
    vi.mocked(fetchGraderAgentSubmissions).mockResolvedValue([
      {
        id: 'attempt-1',
        submittedByDisplayName: 'Ada',
        attachmentFileId: null,
        isGraded: false,
      },
    ])

    const { result } = renderHook(() =>
      useGraderAgentSubmissions({
        open: true,
        courseCode: 'demo',
        itemId: 'quiz-1',
        itemKind: 'quiz',
        initialSubmissionId: null,
      }),
    )

    await waitFor(() => expect(result.current.loading).toBe(false))

    expect(fetchGraderAgentSubmissions).toHaveBeenCalledWith('demo', 'quiz-1', 'quiz', {
      graded: 'all',
    })
    expect(result.current.submissions).toHaveLength(1)
    expect(result.current.selectedSubmissionId).toBe('attempt-1')
  })
})