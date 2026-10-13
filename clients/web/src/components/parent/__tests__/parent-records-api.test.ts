import { beforeEach, describe, expect, it, vi } from 'vitest'

const authorizedFetch = vi.fn()
vi.mock('../../../lib/api', () => ({ authorizedFetch: (...args: unknown[]) => authorizedFetch(...args) }))

import {
  createParentHoursEntry,
  fetchParentHoursLog,
  fetchParentRecordsSettings,
  updateParentRecordsSettings,
} from '../../../lib/parent-records-api'

describe('parent-records-api (#774)', () => {
  beforeEach(() => {
    authorizedFetch.mockReset()
  })

  it('loads settings', async () => {
    authorizedFetch.mockResolvedValue({
      ok: true,
      json: async () => ({
        schoolName: 'Willden Homeschool',
        schoolYearStartMonth: 8,
        schoolYearStartDay: 1,
        currentSchoolYear: '2025-2026',
        currentFrom: '2025-08-01',
        currentTo: '2026-07-31',
      }),
    })
    const s = await fetchParentRecordsSettings()
    expect(s.schoolName).toBe('Willden Homeschool')
    expect(authorizedFetch).toHaveBeenCalledWith('/api/v1/parent/records/settings')
  })

  it('saves school name', async () => {
    authorizedFetch.mockResolvedValue({
      ok: true,
      json: async () => ({
        schoolName: 'New School',
        schoolYearStartMonth: 8,
        schoolYearStartDay: 1,
        currentSchoolYear: '2025-2026',
        currentFrom: '2025-08-01',
        currentTo: '2026-07-31',
      }),
    })
    const s = await updateParentRecordsSettings({ schoolName: 'New School' })
    expect(s.schoolName).toBe('New School')
    expect(authorizedFetch.mock.calls[0][1].method).toBe('PUT')
  })

  it('loads hours log with school year query', async () => {
    authorizedFetch.mockResolvedValue({
      ok: true,
      json: async () => ({
        schoolYear: '2025-2026',
        from: '2025-08-01',
        to: '2026-07-31',
        entries: [{ date: '2025-09-01', subject: 'Math', hours: 1, source: 'activity', editable: false }],
        totalsBySubject: [{ subject: 'Math', hours: 1 }],
        totalHours: 1,
      }),
    })
    const log = await fetchParentHoursLog('kid-1', { schoolYear: '2025-2026' })
    expect(log.totalHours).toBe(1)
    expect(authorizedFetch.mock.calls[0][0]).toContain('schoolYear=2025-2026')
  })

  it('creates offline hours', async () => {
    authorizedFetch.mockResolvedValue({
      ok: true,
      json: async () => ({
        id: 'e1',
        date: '2025-09-02',
        subject: 'Zoo',
        hours: 2,
        source: 'offline',
        editable: true,
      }),
    })
    const row = await createParentHoursEntry('kid-1', {
      date: '2025-09-02',
      subject: 'Zoo',
      hours: 2,
      source: 'offline',
    })
    expect(row.id).toBe('e1')
    expect(authorizedFetch.mock.calls[0][1].method).toBe('POST')
  })
})
