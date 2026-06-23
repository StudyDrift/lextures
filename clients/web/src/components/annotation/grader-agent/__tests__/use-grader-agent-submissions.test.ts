import { renderHook, waitFor } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { fetchModuleAssignmentSubmissions } from '../../../../lib/courses-api'
import { useGraderAgentSubmissions } from '../use-grader-agent-submissions'

vi.mock('../../../../lib/courses-api', () => ({
  fetchModuleAssignmentSubmissions: vi.fn(),
}))

describe('useGraderAgentSubmissions', () => {
  it('selects the initial submission when provided', async () => {
    vi.mocked(fetchModuleAssignmentSubmissions).mockResolvedValue([
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

    expect(result.current.index).toBe(1)
    expect(result.current.selectedSubmissionId).toBe('sub-b')
  })
})