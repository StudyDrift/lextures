import { apiUrl, authorizedFetch } from './api'
import { getImpersonationToken } from './auth'
import { endImpersonationSession, startImpersonationSession } from './impersonation'

export type ManagedDependent = {
  id: string
  displayName: string
  gradeLevel?: string | null
  relationship: string
  linkId: string
  status: string
  coppaMinor: boolean
  createdAt: string
}

export type DependentCourseEnrollment = {
  dependentId: string
  displayName: string
  courseCode: string
  courseTitle: string
  enrollmentId: string
}

export async function listDependentCourses(): Promise<DependentCourseEnrollment[]> {
  const res = await authorizedFetch('/api/v1/me/dependents/courses')
  if (!res.ok) {
    const raw = await res.json().catch(() => ({}))
    throw new Error((raw as { error?: { message?: string } })?.error?.message || 'Failed to load learner courses')
  }
  const data = (await res.json()) as { enrollments?: DependentCourseEnrollment[] }
  return data.enrollments ?? []
}

export async function listDependents(): Promise<ManagedDependent[]> {
  const res = await authorizedFetch('/api/v1/me/dependents')
  if (!res.ok) {
    const raw = await res.json().catch(() => ({}))
    throw new Error((raw as { error?: { message?: string } })?.error?.message || 'Failed to load learners')
  }
  const data = (await res.json()) as { dependents?: ManagedDependent[] }
  return data.dependents ?? []
}

export async function createDependent(input: {
  displayName: string
  gradeLevel?: string
  under13?: boolean
}): Promise<ManagedDependent> {
  const res = await authorizedFetch('/api/v1/me/dependents', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(input),
  })
  if (!res.ok) {
    const raw = await res.json().catch(() => ({}))
    throw new Error((raw as { error?: { message?: string } })?.error?.message || 'Failed to add learner')
  }
  return (await res.json()) as ManagedDependent
}

export async function patchDependent(
  id: string,
  input: { displayName?: string; gradeLevel?: string | null },
): Promise<ManagedDependent> {
  const res = await authorizedFetch(`/api/v1/me/dependents/${encodeURIComponent(id)}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(input),
  })
  if (!res.ok) {
    const raw = await res.json().catch(() => ({}))
    throw new Error((raw as { error?: { message?: string } })?.error?.message || 'Failed to update learner')
  }
  return (await res.json()) as ManagedDependent
}

export async function deleteDependent(id: string): Promise<void> {
  const res = await authorizedFetch(`/api/v1/me/dependents/${encodeURIComponent(id)}`, {
    method: 'DELETE',
  })
  if (!res.ok && res.status !== 204) {
    const raw = await res.json().catch(() => ({}))
    throw new Error((raw as { error?: { message?: string } })?.error?.message || 'Failed to remove learner')
  }
}

export async function startLearnAs(dependentId: string): Promise<void> {
  const res = await authorizedFetch(`/api/v1/me/dependents/${encodeURIComponent(dependentId)}/sessions`, {
    method: 'POST',
  })
  if (!res.ok) {
    const raw = await res.json().catch(() => ({}))
    throw new Error((raw as { error?: { message?: string } })?.error?.message || 'Failed to start Learn as')
  }
  const data = (await res.json()) as { accessToken: string }
  startImpersonationSession(data.accessToken)
}

export async function exitLearnAs(): Promise<void> {
  const token = getImpersonationToken()
  if (token) {
    await fetch(apiUrl('/api/v1/me/dependents/sessions/current'), {
      method: 'DELETE',
      headers: { Authorization: `Bearer ${token}` },
    })
  }
  endImpersonationSession()
}
