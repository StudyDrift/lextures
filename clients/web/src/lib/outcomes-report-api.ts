import { authorizedFetch } from './api'
import { readApiErrorMessage } from './errors'

export type OutcomesReportOutcome = {
  outcomeId: string
  title: string
  sortOrder: number
  nStudents: number
  nAssessed: number
  meanScore: number | null
  pctMet: number
  pctNotMet: number
  threshold: number
  alignmentCount: number
  improvementNote: string
  noAlignments: boolean
}

export type OutcomesReport = {
  courseId: string
  masteryThreshold: number
  dataAsOf: string
  staleMinutes: number
  outcomes: OutcomesReportOutcome[]
}

export type OutcomesReportFilters = {
  sectionId?: string
  groupId?: string
}

export async function fetchOutcomesReport(
  courseCode: string,
  filters?: OutcomesReportFilters,
): Promise<OutcomesReport> {
  const params = new URLSearchParams()
  if (filters?.sectionId) params.set('sectionId', filters.sectionId)
  if (filters?.groupId) params.set('groupId', filters.groupId)
  const qs = params.toString()
  const path = `/api/v1/courses/${encodeURIComponent(courseCode)}/analytics/outcomes${qs ? `?${qs}` : ''}`
  const res = await authorizedFetch(path)
  const raw: unknown = await res.json().catch(() => ({}))
  if (!res.ok) {
    throw new Error(readApiErrorMessage(raw))
  }
  return raw as OutcomesReport
}

export async function refreshOutcomesReport(courseCode: string): Promise<{ refreshedAt: string }> {
  const res = await authorizedFetch(
    `/api/v1/courses/${encodeURIComponent(courseCode)}/analytics/outcomes/refresh`,
    { method: 'POST' },
  )
  const raw: unknown = await res.json().catch(() => ({}))
  if (!res.ok) {
    throw new Error(readApiErrorMessage(raw))
  }
  return raw as { refreshedAt: string }
}

export async function updateOutcomesReportThreshold(
  courseCode: string,
  masteryThreshold: number,
): Promise<void> {
  const res = await authorizedFetch(
    `/api/v1/courses/${encodeURIComponent(courseCode)}/analytics/outcomes/settings`,
    {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ masteryThreshold }),
    },
  )
  if (!res.ok) {
    const raw: unknown = await res.json().catch(() => ({}))
    throw new Error(readApiErrorMessage(raw))
  }
}

export async function saveOutcomeImprovementNote(
  courseCode: string,
  outcomeId: string,
  noteText: string,
): Promise<void> {
  const res = await authorizedFetch(
    `/api/v1/courses/${encodeURIComponent(courseCode)}/outcomes/${encodeURIComponent(outcomeId)}/notes`,
    {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ noteText }),
    },
  )
  const raw: unknown = await res.json().catch(() => ({}))
  if (!res.ok) {
    throw new Error(readApiErrorMessage(raw))
  }
}
