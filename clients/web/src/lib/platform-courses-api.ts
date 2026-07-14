import { authorizedFetch } from './api'

export type PlatformCourseRow = {
  id: string
  courseCode: string
  title: string
  status: 'active' | 'archived' | 'draft'
  orgId: string
  orgName: string
  instructorName: string | null
  termId: string | null
  termName: string | null
  enrollmentCount: number
  createdAt: string
  updatedAt: string
}

export type PaginatedPlatformCourses = {
  items: PlatformCourseRow[]
  total: number
  page: number
  perPage: number
  totalPages: number
}

export type PlatformCourseReport = {
  id: string
  courseCode: string
  title: string
  description: string | null
  status: 'active' | 'archived' | 'draft'
  orgId: string
  orgName: string
  instructorName: string | null
  termId: string | null
  termName: string | null
  enrollmentCount: number
  published: boolean
  archived: boolean
  createdAt: string
  updatedAt: string
}

export type CoursesDashboardStats = {
  createdLast7Days: number
  activeCourses: number
  draftCourses: number
  totalCourses: number
  archivedCourses: number
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

export type PlatformCourseSearchStatus = 'open' | 'active' | 'draft' | 'archived' | 'all'

/** Dashboard segment filters matching GET /api/v1/admin/courses?filter=… */
export type CoursesListFilter =
  | 'created_7d'
  | 'active'
  | 'draft'
  | 'total'
  | 'archived'

export async function fetchCoursesStats(): Promise<CoursesDashboardStats> {
  const res = await authorizedFetch('/api/v1/admin/courses/stats')
  return parseJson(res)
}

export async function searchPlatformCourses(params: {
  q?: string
  status?: PlatformCourseSearchStatus
  filter?: CoursesListFilter
  page?: number
  perPage?: number
}): Promise<PaginatedPlatformCourses> {
  const sp = new URLSearchParams()
  const q = params.q?.trim()
  if (q) sp.set('q', q)
  if (params.status) sp.set('status', params.status)
  if (params.filter) sp.set('filter', params.filter)
  if (params.page) sp.set('page', String(params.page))
  if (params.perPage) sp.set('per_page', String(params.perPage))
  const res = await authorizedFetch(`/api/v1/admin/courses?${sp}`)
  return parseJson(res)
}

export async function fetchPlatformCourseReport(courseId: string): Promise<PlatformCourseReport> {
  const res = await authorizedFetch(`/api/v1/admin/courses/${encodeURIComponent(courseId)}/report`)
  return parseJson(res)
}

export async function ensurePlatformCourseAdminAccess(courseId: string): Promise<PlatformCourseReport> {
  const res = await authorizedFetch(`/api/v1/admin/courses/${encodeURIComponent(courseId)}/access`, {
    method: 'POST',
  })
  return parseJson(res)
}