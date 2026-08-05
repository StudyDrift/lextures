import { afterEach, describe, expect, it, vi } from 'vitest'
import {
  CourseChecklistApiError,
  dismissChecklistItem,
  fetchCourseChecklist,
  fetchCourseChecklistSummary,
} from '../course-checklist-api'

const authorizedFetch = vi.fn()

vi.mock('../api', () => ({
  authorizedFetch: (...args: unknown[]) => authorizedFetch(...args),
}))

afterEach(() => {
  authorizedFetch.mockReset()
})

describe('course-checklist-api', () => {
  it('fetches and validates summary', async () => {
    authorizedFetch.mockResolvedValue({
      ok: true,
      json: async () => ({
        outstandingEssential: 1,
        outstandingTotal: 2,
        done: 3,
        total: 5,
        dismissed: 0,
        computedAt: '2026-08-04T12:00:00Z',
        stale: false,
      }),
    })
    const summary = await fetchCourseChecklistSummary('C1')
    expect(summary.outstandingEssential).toBe(1)
    expect(authorizedFetch).toHaveBeenCalledWith('/api/v1/courses/C1/checklist/summary')
  })

  it('maps 403 to CourseChecklistApiError', async () => {
    authorizedFetch.mockResolvedValue({
      ok: false,
      status: 403,
      json: async () => ({ error: 'forbidden' }),
    })
    await expect(fetchCourseChecklist('C1')).rejects.toBeInstanceOf(CourseChecklistApiError)
    await expect(fetchCourseChecklist('C1')).rejects.toMatchObject({ status: 403 })
  })

  it('posts dismiss body', async () => {
    authorizedFetch.mockResolvedValue({
      ok: true,
      json: async () => ({
        id: 'orientation.welcome-message',
        titleKey: 't',
        title: 'Welcome',
        whyKey: 'w',
        why: 'Why',
        tier: 'essential',
        status: 'todo',
        sources: [],
        dismissal: {
          dismissedAt: '2026-08-04T12:00:00Z',
          byUserId: 'u1',
          byDisplayName: 'Teacher',
          reason: 'not_applicable',
          note: 'n/a',
        },
      }),
    })
    const item = await dismissChecklistItem('C1', 'orientation.welcome-message', {
      reason: 'not_applicable',
      note: 'n/a',
    })
    expect(item.dismissal?.reason).toBe('not_applicable')
    expect(authorizedFetch).toHaveBeenCalledWith(
      '/api/v1/courses/C1/checklist/items/orientation.welcome-message/dismiss',
      expect.objectContaining({ method: 'POST' }),
    )
  })
})
