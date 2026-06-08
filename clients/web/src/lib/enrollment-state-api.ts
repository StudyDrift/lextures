import { authorizedFetch } from './api'
import { readApiErrorMessage } from './errors'
import { enrollmentStateMachineFeatureEnabled } from './platform-features'

export type EnrollmentState =
  | 'active'
  | 'waitlist'
  | 'dropped'
  | 'withdrawn'
  | 'audit'
  | 'no_credit'
  | 'incomplete'

export type EnrollmentStateHistoryEntry = {
  id: string
  actorId?: string
  previousState: EnrollmentState
  newState: EnrollmentState
  reason?: string
  source: string
  createdAt: string
}

export type PatchEnrollmentStateResult = {
  id: string
  state: EnrollmentState
  stateChangedAt?: string
  stateReason?: string
  lisStatusCode: string
}

export function isFormerEnrollmentState(state: EnrollmentState | string | null | undefined): boolean {
  return state === 'dropped' || state === 'withdrawn' || state === 'no_credit'
}

export async function patchEnrollmentState(
  courseCode: string,
  enrollmentId: string,
  body: { state: EnrollmentState; reason?: string; overrideDeadline?: boolean },
): Promise<PatchEnrollmentStateResult> {
  if (!enrollmentStateMachineFeatureEnabled()) {
    throw new Error('Enrollment state machine is not enabled.')
  }
  const res = await authorizedFetch(
    `/api/v1/courses/${encodeURIComponent(courseCode)}/enrollments/${encodeURIComponent(enrollmentId)}/state`,
    {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    },
  )
  const raw: unknown = await res.json().catch(() => ({}))
  if (!res.ok) throw new Error(readApiErrorMessage(raw))
  return raw as PatchEnrollmentStateResult
}

export async function fetchEnrollmentStateHistory(
  courseCode: string,
  enrollmentId: string,
): Promise<EnrollmentStateHistoryEntry[]> {
  if (!enrollmentStateMachineFeatureEnabled()) return []
  const res = await authorizedFetch(
    `/api/v1/courses/${encodeURIComponent(courseCode)}/enrollments/${encodeURIComponent(enrollmentId)}/state/history`,
  )
  const raw: unknown = await res.json().catch(() => ({}))
  if (!res.ok) throw new Error(readApiErrorMessage(raw))
  const data = raw as { history?: EnrollmentStateHistoryEntry[] }
  return data.history ?? []
}
