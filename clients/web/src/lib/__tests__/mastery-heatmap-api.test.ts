import { http, HttpResponse } from 'msw'
import { beforeEach, describe, expect, it } from 'vitest'
import { setAccessToken } from '../auth'
import {
  fetchMasteryHeatmap,
  fetchConceptDrillDown,
  fetchEnrollmentMastery,
  refreshMasteryHeatmap,
  masteryColorClass,
  masteryLabel,
  type MasteryHeatmapResult,
} from '../mastery-heatmap-api'
import { server } from '../../test/mocks/server'

const sampleHeatmap: MasteryHeatmapResult = {
  concepts: [{ id: 'c1', name: 'Quadratic Equations' }],
  rows: [
    {
      enrollmentId: 'e1',
      userId: 'u1',
      displayName: 'Alice Johnson',
      cells: [{ conceptId: 'c1', masteryScore: 0.75, assessed: true, updatedAt: null }],
    },
  ],
  summary: [
    { conceptId: 'c1', conceptName: 'Quadratic Equations', meanMastery: 0.75, pctMastered: 0.5, pctAtRisk: 0.1 },
  ],
  refreshedAt: '2026-05-01T00:00:00Z',
}

describe('fetchMasteryHeatmap', () => {
  beforeEach(() => {
    setAccessToken('test-token')
    server.use(
      http.get('http://localhost:8080/api/v1/courses/CS101/analytics/mastery-heatmap', () => {
        return HttpResponse.json(sampleHeatmap)
      }),
    )
  })

  it('returns parsed heatmap data', async () => {
    const r = await fetchMasteryHeatmap('CS101')
    expect(r.concepts).toHaveLength(1)
    expect(r.concepts[0]!.name).toBe('Quadratic Equations')
    expect(r.rows[0]!.displayName).toBe('Alice Johnson')
    expect(r.rows[0]!.cells[0]!.masteryScore).toBe(0.75)
  })
})

describe('fetchConceptDrillDown', () => {
  beforeEach(() => {
    setAccessToken('test-token')
    server.use(
      http.get(
        'http://localhost:8080/api/v1/courses/CS101/analytics/mastery-heatmap/concepts/c1',
        () => {
          return HttpResponse.json({
            students: [
              { enrollmentId: 'e1', userId: 'u1', displayName: 'Alice', masteryScore: 0.3, assessed: true },
            ],
          })
        },
      ),
    )
  })

  it('returns drill-down students', async () => {
    const r = await fetchConceptDrillDown('CS101', 'c1')
    expect(r.students).toHaveLength(1)
    expect(r.students[0]!.displayName).toBe('Alice')
    expect(r.students[0]!.masteryScore).toBe(0.3)
  })
})

describe('fetchEnrollmentMastery', () => {
  beforeEach(() => {
    setAccessToken('test-token')
    server.use(
      http.get(
        'http://localhost:8080/api/v1/courses/CS101/enrollments/e1/mastery',
        () => {
          return HttpResponse.json({
            enrollmentId: 'e1',
            userId: 'u1',
            concepts: [{ id: 'c1', name: 'Quadratic Equations' }],
            cells: [{ conceptId: 'c1', masteryScore: 0.75, assessed: true }],
          })
        },
      ),
    )
  })

  it('returns student mastery row', async () => {
    const r = await fetchEnrollmentMastery('CS101', 'e1')
    expect(r.enrollmentId).toBe('e1')
    expect(r.cells[0]!.masteryScore).toBe(0.75)
  })
})

describe('refreshMasteryHeatmap', () => {
  beforeEach(() => {
    setAccessToken('test-token')
    server.use(
      http.post(
        'http://localhost:8080/api/v1/courses/CS101/analytics/mastery-heatmap/refresh',
        () => new HttpResponse(null, { status: 204 }),
      ),
    )
  })

  it('resolves without error on 204', async () => {
    await expect(refreshMasteryHeatmap('CS101')).resolves.toBeUndefined()
  })
})

describe('masteryColorClass', () => {
  it('returns grey for unassessed', () => {
    expect(masteryColorClass(false, null)).toContain('slate-200')
  })
  it('returns emerald for mastered (>=0.8)', () => {
    expect(masteryColorClass(true, 0.85)).toBe('bg-emerald-500')
  })
  it('returns lime for developing (0.6-0.79)', () => {
    expect(masteryColorClass(true, 0.65)).toBe('bg-lime-400')
  })
  it('returns amber for beginning (0.4-0.59)', () => {
    expect(masteryColorClass(true, 0.5)).toBe('bg-amber-400')
  })
  it('returns rose for at-risk (<0.4)', () => {
    expect(masteryColorClass(true, 0.3)).toBe('bg-rose-500')
  })
})

describe('masteryLabel', () => {
  it('returns Not assessed when unassessed', () => {
    expect(masteryLabel(false, null)).toBe('Not assessed')
  })
  it('returns Mastered for >=0.8', () => {
    expect(masteryLabel(true, 0.9)).toBe('Mastered')
  })
  it('returns At risk for <0.4', () => {
    expect(masteryLabel(true, 0.2)).toBe('At risk')
  })
})
