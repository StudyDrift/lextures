import { authorizedFetch } from './api'
import { readApiErrorMessage } from './errors'

export type ConceptMeta = {
  id: string
  name: string
}

export type HeatmapCell = {
  conceptId: string
  masteryScore: number | null
  assessed: boolean
  updatedAt?: string | null
}

export type HeatmapRow = {
  enrollmentId: string
  userId: string
  displayName: string | null
  cells: HeatmapCell[]
}

export type ConceptSummary = {
  conceptId: string
  conceptName: string
  meanMastery: number
  pctMastered: number
  pctAtRisk: number
}

export type MasteryHeatmapResult = {
  concepts: ConceptMeta[]
  rows: HeatmapRow[]
  summary: ConceptSummary[]
  refreshedAt: string | null
}

export type DrillDownStudent = {
  enrollmentId: string
  userId: string
  displayName: string | null
  masteryScore: number | null
  assessed: boolean
}

export type StudentMasteryRow = {
  enrollmentId: string
  userId: string
  concepts: ConceptMeta[]
  cells: HeatmapCell[]
}

async function parseJson(res: Response): Promise<unknown> {
  return res.json().catch(() => ({}))
}

/** GET /api/v1/courses/:courseCode/analytics/mastery-heatmap — instructor only. */
export async function fetchMasteryHeatmap(courseCode: string): Promise<MasteryHeatmapResult> {
  const res = await authorizedFetch(
    `/api/v1/courses/${encodeURIComponent(courseCode)}/analytics/mastery-heatmap`,
  )
  const raw = await parseJson(res)
  if (!res.ok) throw new Error(readApiErrorMessage(raw))
  return raw as MasteryHeatmapResult
}

/** GET /api/v1/courses/:courseCode/analytics/mastery-heatmap/concepts/:conceptId — instructor. */
export async function fetchConceptDrillDown(
  courseCode: string,
  conceptId: string,
): Promise<{ students: DrillDownStudent[] }> {
  const res = await authorizedFetch(
    `/api/v1/courses/${encodeURIComponent(courseCode)}/analytics/mastery-heatmap/concepts/${encodeURIComponent(conceptId)}`,
  )
  const raw = await parseJson(res)
  if (!res.ok) throw new Error(readApiErrorMessage(raw))
  return raw as { students: DrillDownStudent[] }
}

/** GET /api/v1/courses/:courseCode/enrollments/:enrollmentId/mastery — instructor or self. */
export async function fetchEnrollmentMastery(
  courseCode: string,
  enrollmentId: string,
): Promise<StudentMasteryRow> {
  const res = await authorizedFetch(
    `/api/v1/courses/${encodeURIComponent(courseCode)}/enrollments/${encodeURIComponent(enrollmentId)}/mastery`,
  )
  const raw = await parseJson(res)
  if (!res.ok) throw new Error(readApiErrorMessage(raw))
  return raw as StudentMasteryRow
}

/** POST /api/v1/courses/:courseCode/analytics/mastery-heatmap/refresh — instructor only. */
export async function refreshMasteryHeatmap(courseCode: string): Promise<void> {
  const res = await authorizedFetch(
    `/api/v1/courses/${encodeURIComponent(courseCode)}/analytics/mastery-heatmap/refresh`,
    { method: 'POST' },
  )
  if (!res.ok) {
    const raw = await parseJson(res)
    throw new Error(readApiErrorMessage(raw))
  }
}

/** Returns a Tailwind background-color class based on mastery score. */
export function masteryColorClass(assessed: boolean, score: number | null): string {
  if (!assessed || score === null) return 'bg-slate-200 dark:bg-neutral-700'
  if (score >= 0.8) return 'bg-emerald-500'
  if (score >= 0.6) return 'bg-lime-400'
  if (score >= 0.4) return 'bg-amber-400'
  return 'bg-rose-500'
}

/** Returns a human-readable mastery label. */
export function masteryLabel(assessed: boolean, score: number | null): string {
  if (!assessed || score === null) return 'Not assessed'
  if (score >= 0.8) return 'Mastered'
  if (score >= 0.6) return 'Developing'
  if (score >= 0.4) return 'Beginning'
  return 'At risk'
}
