import { apiUrl, authorizedFetch } from './api'
import { readApiErrorMessage } from './errors'

export type ParentAssignStudent = {
  id: string
  email: string
  displayName?: string | null
  sid?: string | null
}

export type ParentAssignLink = {
  id: string
  parentUserId: string
  studentUserId: string
  relationship: string
  status: string
  parentEmail: string
  parentDisplayName?: string | null
  linkedAt: string
}

export type GuardianAssignInput = {
  name: string
  email: string
  relationship?: 'parent' | 'guardian' | 'other'
}

export type GuardianAssignResult = {
  email: string
  status: 'linked' | 'invited' | 'error'
  linkId?: string
  parentUserId?: string
  message?: string
}

export async function searchParentAssignStudents(
  orgId: string,
  q: string,
): Promise<ParentAssignStudent[]> {
  const res = await authorizedFetch(
    `/api/v1/orgs/${encodeURIComponent(orgId)}/parent-assign/students?q=${encodeURIComponent(q)}`,
  )
  const raw = (await res.json().catch(() => ({}))) as { students?: ParentAssignStudent[] }
  if (!res.ok) throw new Error(readApiErrorMessage(raw))
  return raw.students ?? []
}

export async function fetchParentAssignLinks(
  orgId: string,
  studentId: string,
): Promise<ParentAssignLink[]> {
  const res = await authorizedFetch(
    `/api/v1/orgs/${encodeURIComponent(orgId)}/parent-assign/students/${encodeURIComponent(studentId)}/links`,
  )
  const raw = (await res.json().catch(() => ({}))) as { links?: ParentAssignLink[] }
  if (!res.ok) throw new Error(readApiErrorMessage(raw))
  return raw.links ?? []
}

export async function assignParentGuardians(
  orgId: string,
  studentId: string,
  guardians: GuardianAssignInput[],
): Promise<GuardianAssignResult[]> {
  const res = await authorizedFetch(
    `/api/v1/orgs/${encodeURIComponent(orgId)}/parent-assign/students/${encodeURIComponent(studentId)}/guardians`,
    {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ guardians }),
    },
  )
  const raw = (await res.json().catch(() => ({}))) as { results?: GuardianAssignResult[] }
  if (!res.ok) throw new Error(readApiErrorMessage(raw))
  return raw.results ?? []
}

export async function resendParentAssignInvite(orgId: string, linkId: string): Promise<void> {
  const res = await authorizedFetch(
    `/api/v1/orgs/${encodeURIComponent(orgId)}/parent-assign/links/${encodeURIComponent(linkId)}/resend`,
    { method: 'POST' },
  )
  if (!res.ok) {
    const raw = await res.json().catch(() => ({}))
    throw new Error(readApiErrorMessage(raw))
  }
}

export async function revokeParentLink(orgId: string, linkId: string): Promise<void> {
  const res = await authorizedFetch(
    `/api/v1/orgs/${encodeURIComponent(orgId)}/parent-links/${encodeURIComponent(linkId)}`,
    { method: 'DELETE' },
  )
  if (!res.ok && res.status !== 204) {
    const raw = await res.json().catch(() => ({}))
    throw new Error(readApiErrorMessage(raw))
  }
}

export async function consumeParentInvite(
  token: string,
  password: string,
): Promise<{ message: string; redirectTo?: string }> {
  const res = await fetch(apiUrl('/api/v1/auth/parent-invite/consume'), {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ token, password }),
  })
  const raw = (await res.json().catch(() => ({}))) as {
    message?: string
    redirectTo?: string
  }
  if (!res.ok) throw new Error(readApiErrorMessage(raw))
  return {
    message: raw.message ?? 'Your account is ready.',
    redirectTo: raw.redirectTo,
  }
}
