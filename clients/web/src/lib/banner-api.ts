import { apiUrl, authorizedFetch } from './api'
import { getAccessToken } from './auth'

export type MaintenanceBanner = {
  id: string
  scope: 'global' | 'org'
  orgId?: string
  message: string
  severity: 'info' | 'warning' | 'error'
  ctaText?: string
  ctaUrl?: string
  startsAt?: string
  expiresAt?: string
  isActive: boolean
  updatedAt: string
}

const DISMISS_STORAGE_KEY = 'lextures.maintenanceBanner.dismissed'
export const BANNER_POLL_INTERVAL_MS = 30_000

type DismissMap = Record<string, string>

function readDismissMap(): DismissMap {
  try {
    const raw = localStorage.getItem(DISMISS_STORAGE_KEY)
    if (!raw) return {}
    const parsed = JSON.parse(raw) as unknown
    if (!parsed || typeof parsed !== 'object') return {}
    return parsed as DismissMap
  } catch {
    return {}
  }
}

export function isBannerDismissed(banner: MaintenanceBanner): boolean {
  const map = readDismissMap()
  return map[banner.id] === banner.updatedAt
}

export function dismissBanner(banner: MaintenanceBanner): void {
  try {
    const map = readDismissMap()
    map[banner.id] = banner.updatedAt
    localStorage.setItem(DISMISS_STORAGE_KEY, JSON.stringify(map))
  } catch {
    /* ignore when storage unavailable (SSR/tests) */
  }
}

export async function fetchActiveBanner(orgSlug?: string | null): Promise<MaintenanceBanner | null> {
  const params = new URLSearchParams()
  if (orgSlug) params.set('orgSlug', orgSlug)
  const suffix = params.toString() ? `?${params.toString()}` : ''
  const headers: HeadersInit = {}
  const token = getAccessToken()
  if (token) headers.Authorization = `Bearer ${token}`
  const res = await fetch(apiUrl(`/api/v1/status/banner${suffix}`), { headers })
  if (!res.ok) {
    return null
  }
  const data = (await res.json()) as MaintenanceBanner | null
  if (!data || !data.id) return null
  return data
}

function orgQuery(orgId: string | null | undefined): string {
  if (!orgId) return ''
  return `?orgId=${encodeURIComponent(orgId)}`
}

export async function fetchAdminBanners(orgId?: string | null, scope?: 'global'): Promise<MaintenanceBanner[]> {
  const params = new URLSearchParams()
  if (orgId) params.set('orgId', orgId)
  if (scope === 'global') params.set('scope', 'global')
  const suffix = params.toString() ? `?${params.toString()}` : ''
  const res = await authorizedFetch(`/api/v1/admin/banners${suffix}`)
  if (!res.ok) throw new Error(`Failed to load banners (${res.status})`)
  return res.json() as Promise<MaintenanceBanner[]>
}

export async function createAdminBanner(
  body: {
    scope: 'global' | 'org'
    message: string
    severity: 'info' | 'warning' | 'error'
    ctaText?: string
    ctaUrl?: string
    startsAt?: string
    expiresAt?: string
  },
  orgId?: string | null,
): Promise<MaintenanceBanner> {
  const res = await authorizedFetch(`/api/v1/admin/banners${orgQuery(orgId)}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  if (!res.ok) {
    const text = await res.text()
    throw new Error(text || `Failed to create banner (${res.status})`)
  }
  return res.json() as Promise<MaintenanceBanner>
}

export async function updateAdminBanner(
  id: string,
  body: {
    message: string
    severity: 'info' | 'warning' | 'error'
    ctaText?: string
    ctaUrl?: string
    startsAt?: string
    expiresAt?: string
    isActive?: boolean
  },
  orgId?: string | null,
): Promise<MaintenanceBanner> {
  const res = await authorizedFetch(`/api/v1/admin/banners/${id}${orgQuery(orgId)}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  if (!res.ok) throw new Error(`Failed to update banner (${res.status})`)
  return res.json() as Promise<MaintenanceBanner>
}

export async function deleteAdminBanner(id: string, orgId?: string | null): Promise<void> {
  const res = await authorizedFetch(`/api/v1/admin/banners/${id}${orgQuery(orgId)}`, {
    method: 'DELETE',
  })
  if (!res.ok) throw new Error(`Failed to delete banner (${res.status})`)
}
