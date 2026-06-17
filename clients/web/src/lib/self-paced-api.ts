// Client API for self-paced enrollment, progress, and completion (plan 15.2).
import { authorizedFetch } from './api'

/** Per-module progress and gating state. Mirrors server `selfpaced.ModuleView`. */
export type SelfPacedModule = {
  moduleId: string
  title: string
  sortOrder: number
  totalItems: number
  completedItems: number
  progressPercent: number
  completed: boolean
  locked: boolean
}

/** Learner progress snapshot for a course. Mirrors server `progressResponse`. */
export type SelfPacedProgress = {
  totalItems: number
  completedItems: number
  progressPercent: number
  completed: boolean
  gatingEnabled: boolean
  modules: SelfPacedModule[]
  lastVisitedItemId?: string
  enrollmentId: string
  resumeItemId?: string
  justCompleted?: boolean
  credentialId?: string
}

/** One self-paced enrollment with progress, for the Dashboard. */
export type SelfPacedEnrollment = {
  courseCode: string
  title: string
  enrollmentId: string
  progressPercent: number
  totalItems: number
  completedItems: number
  completed: boolean
  resumeItemId?: string
}

export type SelfEnrollResult = {
  enrolled: boolean
  enrollmentId: string
  firstItemId?: string
}

/** Clamp and round a fraction to an integer percentage (0–100). */
export function progressPercent(completed: number, total: number): number {
  if (total <= 0 || completed <= 0) return 0
  if (completed >= total) return 100
  return Math.floor((completed / total) * 100)
}

/** Locale-formatted "X% complete" label. */
export function formatProgressLabel(percent: number): string {
  return `${percent.toLocaleString()}% complete`
}

export async function selfEnroll(courseCode: string): Promise<SelfEnrollResult> {
  const res = await authorizedFetch(
    `/api/v1/courses/${encodeURIComponent(courseCode)}/self-enroll`,
    { method: 'POST' },
  )
  if (!res.ok) throw new Error(`self-enroll failed: ${res.status}`)
  return (await res.json()) as SelfEnrollResult
}

export async function fetchMyProgress(courseCode: string): Promise<SelfPacedProgress> {
  const res = await authorizedFetch(
    `/api/v1/courses/${encodeURIComponent(courseCode)}/my-progress`,
  )
  if (!res.ok) throw new Error(`progress failed: ${res.status}`)
  return (await res.json()) as SelfPacedProgress
}

export async function completeItem(
  courseCode: string,
  itemId: string,
): Promise<SelfPacedProgress> {
  const res = await authorizedFetch(
    `/api/v1/courses/${encodeURIComponent(courseCode)}/items/${encodeURIComponent(itemId)}/complete`,
    { method: 'POST' },
  )
  if (!res.ok) throw new Error(`complete failed: ${res.status}`)
  return (await res.json()) as SelfPacedProgress
}

export async function fetchSelfPacedEnrollments(): Promise<SelfPacedEnrollment[]> {
  const res = await authorizedFetch('/api/v1/me/enrollments?mode=self_paced')
  if (!res.ok) throw new Error(`enrollments failed: ${res.status}`)
  const body = (await res.json()) as { enrollments?: SelfPacedEnrollment[] }
  return body.enrollments ?? []
}
