import { describe, expect, it, vi, beforeEach } from 'vitest'
import {
  fetchIntroCourseAdminAnalytics,
  fetchIntroCourseAdminStatus,
  resyncIntroCourse,
  startIntroCourseBackfill,
} from '../intro-course-admin-api'

const authorizedFetch = vi.fn()

vi.mock('../api', () => ({
  authorizedFetch: (...args: unknown[]) => authorizedFetch(...args),
}))

describe('intro-course-admin-api', () => {
  beforeEach(() => {
    authorizedFetch.mockReset()
  })

  it('fetchIntroCourseAdminStatus calls admin status endpoint', async () => {
    authorizedFetch.mockResolvedValue({
      ok: true,
      json: async () => ({
        enabled: true,
        coursePresent: true,
        courseCode: 'C-WLCOME',
        contentVersion: 2,
        moduleCount: 7,
        availableLocales: ['en', 'es'],
        localeCoverage: { en: 1, es: 0.2 },
        backfill: { startedAt: null, completedAt: null, remaining: 0 },
      }),
    })
    const status = await fetchIntroCourseAdminStatus()
    expect(authorizedFetch).toHaveBeenCalledWith('/api/v1/admin/intro-course')
    expect(status.moduleCount).toBe(7)
    expect(status.localeCoverage.es).toBe(0.2)
  })

  it('fetchIntroCourseAdminAnalytics calls analytics endpoint', async () => {
    authorizedFetch.mockResolvedValue({
      ok: true,
      json: async () => ({
        enrolled: 10,
        completed: 4,
        completionRate: 0.4,
        perModuleFunnel: [],
      }),
    })
    const analytics = await fetchIntroCourseAdminAnalytics()
    expect(authorizedFetch).toHaveBeenCalledWith('/api/v1/admin/intro-course/analytics')
    expect(analytics.enrolled).toBe(10)
  })

  it('resyncIntroCourse posts to resync endpoint', async () => {
    authorizedFetch.mockResolvedValue({
      ok: true,
      json: async () => ({ courseId: 'id', status: 'reconciled', contentVersion: 2 }),
    })
    await resyncIntroCourse()
    expect(authorizedFetch).toHaveBeenCalledWith('/api/v1/admin/intro-course/resync', { method: 'POST' })
  })

  it('startIntroCourseBackfill posts to backfill endpoint', async () => {
    authorizedFetch.mockResolvedValue({
      ok: true,
      json: async () => ({ startedAt: null, remaining: 5 }),
    })
    await startIntroCourseBackfill()
    expect(authorizedFetch).toHaveBeenCalledWith('/api/v1/admin/intro-course/backfill', { method: 'POST' })
  })
})