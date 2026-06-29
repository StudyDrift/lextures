import { authorizedFetch } from './api'

export type AdminConsoleCapabilities = {
  enabled: boolean
  orgId: string
  canAccess: boolean
  canManage: boolean
  isGlobalAdmin: boolean
}

export type AdminOverview = {
  orgId: string
  totalUsers: number
  activeCourses: number
  pendingEnrollments: number
  storageBytes: number
  recentAuditEvents: AuditEvent[]
}

export type AdminUser = {
  id: string
  email: string
  displayName: string | null
  role: string
  orgRole: string | null
  active: boolean
  createdAt: string
}

export type AdminCourse = {
  id: string
  courseCode: string
  title: string
  status: 'active' | 'archived' | 'draft'
  instructorName: string | null
  termId: string | null
  termName: string | null
  enrollmentCount: number
  updatedAt: string
}

export type Paginated<T> = {
  items: T[]
  total: number
  page: number
  perPage: number
  totalPages: number
}

export type AuditEvent = {
  eventId: string
  eventType: string
  actorId: string
  timestamp: string
  orgId?: string
  targetType?: string
  targetId?: string
}

export type AdminSettings = {
  orgId: string
  name: string
  slug: string
  logoUrl: string | null
  faviconUrl: string | null
  primaryColor: string
  secondaryColor: string
  customDomain: string | null
  customEmailDisplayName: string | null
  timezone: string
  locale: string
}

export type IntegrationStatus = {
  orgId: string
  sso: { saml: boolean; oidc: boolean; clever: boolean; classlink: boolean }
  oneRoster: { enabled: boolean }
  scim: { enabled: boolean }
  sis: { enabled: boolean; activeConnections: number }
  webhooks: { enabled: boolean; subscriptions: number }
}

function orgQuery(orgId?: string | null): string {
  if (!orgId) return ''
  return `?orgId=${encodeURIComponent(orgId)}`
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

export async function fetchAdminConsoleCapabilities(): Promise<AdminConsoleCapabilities> {
  const res = await authorizedFetch('/api/v1/me/admin-console-capabilities')
  return parseJson(res)
}

export async function fetchAdminOverview(orgId?: string | null): Promise<AdminOverview> {
  const res = await authorizedFetch(`/api/v1/admin-console/overview${orgQuery(orgId)}`)
  return parseJson(res)
}

export async function fetchAdminUsers(params: {
  orgId?: string | null
  q?: string
  role?: string
  page?: number
  perPage?: number
}): Promise<Paginated<AdminUser>> {
  const sp = new URLSearchParams()
  if (params.orgId) sp.set('orgId', params.orgId)
  if (params.q) sp.set('q', params.q)
  if (params.role) sp.set('role', params.role)
  if (params.page) sp.set('page', String(params.page))
  if (params.perPage) sp.set('per_page', String(params.perPage))
  const qs = sp.toString()
  const res = await authorizedFetch(`/api/v1/admin-console/users${qs ? `?${qs}` : ''}`)
  return parseJson(res)
}

export async function patchAdminUser(
  userId: string,
  body: { active?: boolean; role?: string },
): Promise<AdminUser> {
  const res = await authorizedFetch(`/api/v1/admin-console/users/${encodeURIComponent(userId)}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  return parseJson(res)
}

export async function fetchAdminCourses(params: {
  orgId?: string | null
  q?: string
  status?: string
  page?: number
  perPage?: number
}): Promise<Paginated<AdminCourse>> {
  const sp = new URLSearchParams()
  if (params.orgId) sp.set('orgId', params.orgId)
  if (params.q) sp.set('q', params.q)
  if (params.status) sp.set('status', params.status)
  if (params.page) sp.set('page', String(params.page))
  if (params.perPage) sp.set('per_page', String(params.perPage))
  const qs = sp.toString()
  const res = await authorizedFetch(`/api/v1/admin-console/courses${qs ? `?${qs}` : ''}`)
  return parseJson(res)
}

export async function patchAdminCourseStatus(
  courseId: string,
  status: 'active' | 'archived' | 'draft',
): Promise<AdminCourse> {
  const res = await authorizedFetch(
    `/api/v1/admin-console/courses/${encodeURIComponent(courseId)}/status`,
    {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ status }),
    },
  )
  return parseJson(res)
}

export async function fetchAdminAuditLog(params: {
  orgId?: string | null
  action?: string
}): Promise<{ events: AuditEvent[] }> {
  const sp = new URLSearchParams()
  if (params.orgId) sp.set('orgId', params.orgId)
  if (params.action) sp.set('action', params.action)
  const qs = sp.toString()
  const res = await authorizedFetch(`/api/v1/admin-console/audit-log${qs ? `?${qs}` : ''}`)
  return parseJson(res)
}

export async function fetchAdminSettings(orgId?: string | null): Promise<AdminSettings> {
  const res = await authorizedFetch(`/api/v1/admin-console/settings${orgQuery(orgId)}`)
  return parseJson(res)
}

export async function putAdminSettings(
  body: Partial<AdminSettings>,
  orgId?: string | null,
): Promise<AdminSettings> {
  const res = await authorizedFetch(`/api/v1/admin-console/settings${orgQuery(orgId)}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  return parseJson(res)
}

export async function fetchAdminIntegrations(orgId?: string | null): Promise<IntegrationStatus> {
  const res = await authorizedFetch(`/api/v1/admin-console/integrations${orgQuery(orgId)}`)
  return parseJson(res)
}

export function formatStorageBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
  return `${(bytes / (1024 * 1024 * 1024)).toFixed(2)} GB`
}
