import { authorizedFetch } from './api'
import { readApiErrorMessage } from './errors'

async function parseJson(res: Response): Promise<unknown> {
  const text = await res.text()
  if (!text) return null
  try {
    return JSON.parse(text) as unknown
  } catch {
    return text
  }
}

export type OERProviderId = 'oer_commons' | 'merlot' | 'openstax'

export type OERSearchResult = {
  id: string
  title: string
  description: string
  url: string
  previewUrl?: string
  provider: OERProviderId
  licenseSpdx: string
  licenseLabel: string
  gradeLevel?: string
  subject?: string
  attribution?: string
}

export type OERSearchResponse = {
  results: OERSearchResult[]
  provider?: string
  fromCache: boolean
  cacheAsOf?: string
  staleCache?: boolean
}

export function oerLibraryEnabled(): boolean {
  const v = import.meta.env.VITE_FEATURE_OER_LIBRARY
  return v === 'true' || v === '1'
}

export async function fetchOERProviders(): Promise<OERProviderId[]> {
  const res = await authorizedFetch('/api/v1/oer/providers')
  const raw = await parseJson(res)
  if (!res.ok) throw new Error(readApiErrorMessage(raw))
  if (!Array.isArray(raw)) return []
  return raw
    .map((row) => (row as { provider?: string }).provider)
    .filter((p): p is OERProviderId => p === 'oer_commons' || p === 'merlot' || p === 'openstax')
}

export async function searchOER(
  provider: OERProviderId,
  params: { q?: string; subject?: string; level?: string; license?: string },
): Promise<OERSearchResponse> {
  const qs = new URLSearchParams({ provider })
  if (params.q) qs.set('q', params.q)
  if (params.subject) qs.set('subject', params.subject)
  if (params.level) qs.set('level', params.level)
  if (params.license) qs.set('license', params.license)
  const res = await authorizedFetch(`/api/v1/oer/search?${qs.toString()}`)
  const raw = await parseJson(res)
  if (!res.ok) throw new Error(readApiErrorMessage(raw))
  const r = raw as Record<string, unknown>
  const results = Array.isArray(r.results) ? (r.results as OERSearchResult[]) : []
  return {
    results,
    provider: String(r.provider ?? provider),
    fromCache: Boolean(r.fromCache),
    cacheAsOf: r.cacheAsOf != null ? String(r.cacheAsOf) : undefined,
    staleCache: Boolean(r.staleCache),
  }
}

export type OERImportBody = {
  title: string
  url: string
  provider: OERProviderId
  externalId?: string
  licenseSpdx?: string
  attributionText?: string
}

export async function importOERToModule(
  courseCode: string,
  moduleId: string,
  body: OERImportBody,
): Promise<{ id: string }> {
  const res = await authorizedFetch(
    `/api/v1/courses/${encodeURIComponent(courseCode)}/structure/modules/${encodeURIComponent(moduleId)}/oer-import`,
    {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    },
  )
  const raw = await parseJson(res)
  if (!res.ok) throw new Error(readApiErrorMessage(raw))
  const r = raw as Record<string, unknown>
  return { id: String(r.id ?? '') }
}

export type OERProviderSetting = {
  provider: OERProviderId
  enabled: boolean
  updatedAt: string
}

export async function fetchAdminOERProviders(): Promise<OERProviderSetting[]> {
  const res = await authorizedFetch('/api/v1/admin/oer-providers')
  const raw = await parseJson(res)
  if (!res.ok) throw new Error(readApiErrorMessage(raw))
  if (!Array.isArray(raw)) return []
  return raw.map((row) => {
    const r = row as Record<string, unknown>
    return {
      provider: String(r.provider ?? '') as OERProviderId,
      enabled: Boolean(r.enabled),
      updatedAt: String(r.updatedAt ?? ''),
    }
  })
}

export async function putAdminOERProvider(provider: OERProviderId, enabled: boolean): Promise<void> {
  const res = await authorizedFetch(`/api/v1/admin/oer-providers/${encodeURIComponent(provider)}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ enabled }),
  })
  if (!res.ok) {
    const raw = await parseJson(res)
    throw new Error(readApiErrorMessage(raw))
  }
}
