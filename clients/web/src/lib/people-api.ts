import { authorizedFetch } from './api'

export type PersonRow = {
  id: string
  email: string
  firstName: string | null
  lastName: string | null
  displayName: string | null
  orgId: string
  orgName: string
  role: string
  active: boolean
  createdAt: string
}

export type PaginatedPeople = {
  items: PersonRow[]
  total: number
  page: number
  perPage: number
  totalPages: number
}

export type PersonEnrollment = {
  courseId: string
  courseCode: string
  courseTitle: string
  role: string
  active: boolean
  state: string
  enrolledAt: string
  orgName?: string | null
}

export type PersonActivity = {
  eventKind: string
  courseCode: string
  courseTitle: string
  occurredAt: string
}

export type PersonReport = {
  id: string
  email: string
  firstName: string | null
  lastName: string | null
  displayName: string | null
  orgId: string
  orgName: string
  role: string
  active: boolean
  createdAt: string
  lastActivityAt: string | null
  enrollmentCount: number
  enrollments: PersonEnrollment[]
  recentActivity: PersonActivity[]
}

async function parseJson<T>(res: Response): Promise<T> {
  const raw: unknown = await res.json().catch(() => ({}))
  if (!res.ok) {
    const msg =
      typeof raw === 'object' && raw !== null && 'message' in raw
        ? String((raw as { message: string }).message)
        : res.statusText
    throw new Error(msg || 'Request failed')
  }
  return raw as T
}

export function personDisplayName(row: {
  displayName?: string | null
  firstName?: string | null
  lastName?: string | null
  email: string
}): string {
  const dn = row.displayName?.trim()
  if (dn) return dn
  const full = [row.firstName?.trim(), row.lastName?.trim()].filter(Boolean).join(' ')
  if (full) return full
  return row.email
}

export async function searchPeople(params: {
  q: string
  page?: number
  perPage?: number
}): Promise<PaginatedPeople> {
  const sp = new URLSearchParams()
  sp.set('q', params.q.trim())
  if (params.page) sp.set('page', String(params.page))
  if (params.perPage) sp.set('per_page', String(params.perPage))
  const res = await authorizedFetch(`/api/v1/admin/people?${sp}`)
  return parseJson(res)
}

export async function invitePerson(body: {
  email: string
  firstName?: string
  lastName?: string
  orgId?: string
  role?: string
}): Promise<PersonRow> {
  const res = await authorizedFetch('/api/v1/admin/people/invite', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  return parseJson(res)
}

export async function fetchPersonReport(userId: string): Promise<PersonReport> {
  const res = await authorizedFetch(`/api/v1/admin/people/${encodeURIComponent(userId)}/report`)
  return parseJson(res)
}

export async function patchPerson(userId: string, body: { active: boolean }): Promise<PersonRow> {
  const res = await authorizedFetch(`/api/v1/admin/people/${encodeURIComponent(userId)}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  return parseJson(res)
}

export async function deletePerson(userId: string): Promise<void> {
  const res = await authorizedFetch(`/api/v1/admin/people/${encodeURIComponent(userId)}`, {
    method: 'DELETE',
  })
  await parseJson(res)
}