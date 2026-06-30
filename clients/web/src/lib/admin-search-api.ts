import { authorizedFetch } from './api'

export type AdminSearchResult = {
  id: string
  type: 'users' | 'courses' | 'content'
  title: string
  subtitle: string
  snippet?: string
  path: string
  score?: number
}

export type AdminOmnisearchResponse = {
  users: AdminSearchResult[]
  courses: AdminSearchResult[]
  content: AdminSearchResult[]
  tookMs: number
}

export type AdminSearchPaginated<T = AdminSearchResult> = {
  items: T[]
  total: number
  page: number
  perPage: number
  totalPages: number
  tookMs: number
}

function orgQuery(orgId?: string | null): string {
  if (!orgId) return ''
  return `orgId=${encodeURIComponent(orgId)}`
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

export async function fetchAdminOmnisearch(params: {
  q: string
  types?: string
  orgId?: string | null
}): Promise<AdminOmnisearchResponse> {
  const sp = new URLSearchParams()
  sp.set('q', params.q)
  if (params.types) sp.set('types', params.types)
  const org = orgQuery(params.orgId)
  const qs = sp.toString()
  const res = await authorizedFetch(
    `/api/v1/admin/search${qs ? `?${qs}${org ? `&${org}` : ''}` : org ? `?${org}` : ''}`,
  )
  const data = await parseJson<AdminOmnisearchResponse>(res)
  return {
    users: data.users ?? [],
    courses: data.courses ?? [],
    content: data.content ?? [],
    tookMs: data.tookMs ?? 0,
  }
}

export async function fetchAdminSearchUsers(params: {
  q: string
  page?: number
  perPage?: number
  orgId?: string | null
}): Promise<AdminSearchPaginated> {
  const sp = new URLSearchParams()
  sp.set('q', params.q)
  if (params.page) sp.set('page', String(params.page))
  if (params.perPage) sp.set('per_page', String(params.perPage))
  const org = orgQuery(params.orgId)
  const qs = sp.toString()
  const res = await authorizedFetch(
    `/api/v1/admin/search/users?${qs}${org ? `&${org}` : ''}`,
  )
  return parseJson(res)
}

export async function fetchAdminSearchCourses(params: {
  q: string
  page?: number
  perPage?: number
  orgId?: string | null
}): Promise<AdminSearchPaginated> {
  const sp = new URLSearchParams()
  sp.set('q', params.q)
  if (params.page) sp.set('page', String(params.page))
  if (params.perPage) sp.set('per_page', String(params.perPage))
  const org = orgQuery(params.orgId)
  const qs = sp.toString()
  const res = await authorizedFetch(
    `/api/v1/admin/search/courses?${qs}${org ? `&${org}` : ''}`,
  )
  return parseJson(res)
}

export async function fetchAdminSearchContent(params: {
  q: string
  page?: number
  perPage?: number
  orgId?: string | null
}): Promise<AdminSearchPaginated> {
  const sp = new URLSearchParams()
  sp.set('q', params.q)
  if (params.page) sp.set('page', String(params.page))
  if (params.perPage) sp.set('per_page', String(params.perPage))
  const org = orgQuery(params.orgId)
  const qs = sp.toString()
  const res = await authorizedFetch(
    `/api/v1/admin/search/content?${qs}${org ? `&${org}` : ''}`,
  )
  return parseJson(res)
}

export function adminSearchResultsPath(q: string, type: string, orgId?: string | null): string {
  const sp = new URLSearchParams()
  sp.set('q', q)
  sp.set('type', type)
  if (orgId) sp.set('orgId', orgId)
  return `/org-admin/search?${sp.toString()}`
}
