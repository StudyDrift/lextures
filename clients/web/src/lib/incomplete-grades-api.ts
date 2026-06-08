import { authorizedFetch } from './api'
import { readApiErrorMessage } from './errors'
import { incompleteGradeWorkflowFeatureEnabled } from './platform-features'

export type IncompleteGradeRecord = {
  id: string
  enrollmentId: string
  grantedBy: string
  extensionDeadline: string
  outstandingItemIds: string[]
  status: 'open' | 'resolved' | 'lapsed'
  notes?: string
  resolvedGrade?: string
  resolvedAt?: string
  resolvedBy?: string
  createdAt: string
}

export type AdminIncompleteRow = {
  id: string
  enrollmentId: string
  studentUserId: string
  studentName: string
  courseCode: string
  courseTitle: string
  extensionDeadline: string
  outstandingItemIds: string[]
  outstandingTitles: string[]
  status: string
  notes?: string
}

function requireFeature(): void {
  if (!incompleteGradeWorkflowFeatureEnabled()) {
    throw new Error('Incomplete grade workflow is not enabled.')
  }
}

export async function fetchIncompleteGrade(
  courseCode: string,
  enrollmentId: string,
): Promise<IncompleteGradeRecord | null> {
  requireFeature()
  const res = await authorizedFetch(
    `/api/v1/courses/${encodeURIComponent(courseCode)}/enrollments/${encodeURIComponent(enrollmentId)}/incomplete`,
  )
  const raw: unknown = await res.json().catch(() => ({}))
  if (!res.ok) throw new Error(readApiErrorMessage(raw))
  const data = raw as { record?: IncompleteGradeRecord | null }
  return data.record ?? null
}

export async function grantIncompleteGrade(
  courseCode: string,
  enrollmentId: string,
  body: { extensionDeadline: string; outstandingItemIds: string[]; notes?: string },
): Promise<IncompleteGradeRecord> {
  requireFeature()
  const res = await authorizedFetch(
    `/api/v1/courses/${encodeURIComponent(courseCode)}/enrollments/${encodeURIComponent(enrollmentId)}/incomplete`,
    {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    },
  )
  const raw: unknown = await res.json().catch(() => ({}))
  if (!res.ok) throw new Error(readApiErrorMessage(raw))
  const data = raw as { record: IncompleteGradeRecord }
  return data.record
}

export async function resolveIncompleteGrade(
  courseCode: string,
  enrollmentId: string,
  resolvedGrade: string,
): Promise<IncompleteGradeRecord> {
  requireFeature()
  const res = await authorizedFetch(
    `/api/v1/courses/${encodeURIComponent(courseCode)}/enrollments/${encodeURIComponent(enrollmentId)}/incomplete`,
    {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ resolvedGrade }),
    },
  )
  const raw: unknown = await res.json().catch(() => ({}))
  if (!res.ok) throw new Error(readApiErrorMessage(raw))
  const data = raw as { record: IncompleteGradeRecord }
  return data.record
}

export async function fetchAdminIncompletes(params?: {
  termId?: string
  status?: string
}): Promise<AdminIncompleteRow[]> {
  requireFeature()
  const q = new URLSearchParams()
  if (params?.termId) q.set('term_id', params.termId)
  if (params?.status) q.set('status', params.status)
  const qs = q.toString()
  const res = await authorizedFetch(`/api/v1/admin/incompletes${qs ? `?${qs}` : ''}`)
  const raw: unknown = await res.json().catch(() => ({}))
  if (!res.ok) throw new Error(readApiErrorMessage(raw))
  const data = raw as { incompletes?: AdminIncompleteRow[] }
  return data.incompletes ?? []
}
