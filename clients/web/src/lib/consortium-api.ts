import { authorizedFetch } from './api'

export type ConsortiumAgreement = {
  id: string
  hostOrgId: string
  guestOrgId: string
  hostOrgName?: string
  guestOrgName?: string
  status: 'pending' | 'active' | 'terminated'
  signedAt?: string
  expiresAt?: string
  createdAt: string
}

export type ConsortiumSharedCourse = {
  id: string
  courseCode: string
  title: string
  description: string
  hostOrgId: string
  hostOrgName: string
}

export type ConsortiumSettings = {
  consortiumShareable: boolean
}

export type ConsortiumHomeBranding = {
  active: boolean
  orgName?: string
  primaryColor?: string
  secondaryColor?: string
  logoUrl?: string | null
}

export async function listConsortiumAgreements(orgId: string): Promise<ConsortiumAgreement[]> {
  const res = await authorizedFetch(`/api/v1/admin/consortium/agreements?orgId=${encodeURIComponent(orgId)}`)
  if (!res.ok) throw new Error('Failed to load consortium agreements.')
  const data = (await res.json()) as { agreements?: ConsortiumAgreement[] }
  return data.agreements ?? []
}

export async function createConsortiumAgreement(body: {
  hostOrgId: string
  guestOrgId: string
  status?: string
}): Promise<ConsortiumAgreement> {
  const res = await authorizedFetch('/api/v1/admin/consortium/agreements', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  if (!res.ok) throw new Error('Failed to create agreement.')
  const data = (await res.json()) as { agreement: ConsortiumAgreement }
  return data.agreement
}

export async function activateConsortiumAgreement(id: string): Promise<ConsortiumAgreement> {
  const res = await authorizedFetch(`/api/v1/admin/consortium/agreements/${encodeURIComponent(id)}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ status: 'active' }),
  })
  if (!res.ok) throw new Error('Failed to activate agreement.')
  const data = (await res.json()) as { agreement: ConsortiumAgreement }
  return data.agreement
}

export async function listConsortiumCourses(): Promise<ConsortiumSharedCourse[]> {
  const res = await authorizedFetch('/api/v1/consortium/courses')
  if (!res.ok) throw new Error('Failed to load partner courses.')
  const data = (await res.json()) as { courses?: ConsortiumSharedCourse[] }
  return data.courses ?? []
}

export async function enrollConsortiumCourse(courseId: string): Promise<void> {
  const res = await authorizedFetch(`/api/v1/consortium/courses/${encodeURIComponent(courseId)}/enroll`, {
    method: 'POST',
  })
  if (!res.ok) throw new Error('Enrollment failed.')
}

export async function fetchCourseConsortiumSettings(courseCode: string): Promise<ConsortiumSettings | null> {
  const res = await authorizedFetch(
    `/api/v1/courses/${encodeURIComponent(courseCode)}/consortium-settings`,
  )
  if (res.status === 404) return null
  if (!res.ok) throw new Error('Failed to load consortium settings.')
  return (await res.json()) as ConsortiumSettings
}

export async function patchCourseConsortiumSettings(
  courseCode: string,
  consortiumShareable: boolean,
): Promise<ConsortiumSettings> {
  const res = await authorizedFetch(
    `/api/v1/courses/${encodeURIComponent(courseCode)}/consortium-settings`,
    {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ consortiumShareable }),
    },
  )
  if (!res.ok) throw new Error('Failed to save consortium settings.')
  return (await res.json()) as ConsortiumSettings
}

export async function fetchConsortiumHomeBranding(courseCode: string): Promise<ConsortiumHomeBranding> {
  const res = await authorizedFetch(
    `/api/v1/me/consortium-branding?courseCode=${encodeURIComponent(courseCode)}`,
  )
  if (!res.ok) return { active: false }
  return (await res.json()) as ConsortiumHomeBranding
}
