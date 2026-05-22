import { authorizedFetch } from './api'

export type LRSEndpoint = {
  id: string
  label: string
  endpointUrl: string
  authType: 'basic' | 'oauth2'
  username?: string
  enabled: boolean
  hasPassword: boolean
  hasOauthSecret: boolean
  oauthClientId?: string
  oauthTokenUrl?: string
  updatedAt: string
}

export type LRSEndpointInput = {
  label: string
  endpointUrl: string
  authType: 'basic' | 'oauth2'
  username?: string
  password?: string
  oauthClientId?: string
  oauthClientSecret?: string
  oauthTokenUrl?: string
  enabled: boolean
}

export type XAPIEventRow = {
  statementId: string
  verb: string
  objectId: string
  objectTitle?: string
  storedAt: string
  fullJson: unknown
}

export async function fetchAdminLRSEndpoints(): Promise<LRSEndpoint[]> {
  const res = await authorizedFetch('/api/v1/admin/lrs-config')
  if (!res.ok) throw new Error(`Failed to load LRS config (${res.status})`)
  return (await res.json()) as LRSEndpoint[]
}

export async function createAdminLRSEndpoint(body: LRSEndpointInput): Promise<{ id: string }> {
  const res = await authorizedFetch('/api/v1/admin/lrs-config', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  if (!res.ok) throw new Error(`Failed to create LRS endpoint (${res.status})`)
  return (await res.json()) as { id: string }
}

export async function updateAdminLRSEndpoint(id: string, body: Partial<LRSEndpointInput>): Promise<void> {
  const res = await authorizedFetch(`/api/v1/admin/lrs-config/${encodeURIComponent(id)}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  if (!res.ok) throw new Error(`Failed to update LRS endpoint (${res.status})`)
}

export async function testAdminLRSEndpoint(id: string): Promise<{ message: string }> {
  const res = await authorizedFetch(`/api/v1/admin/lrs-config/${encodeURIComponent(id)}/test`, {
    method: 'POST',
  })
  if (!res.ok) throw new Error(`LRS test failed (${res.status})`)
  return (await res.json()) as { message: string }
}

export async function fetchCourseXAPIEvents(courseCode: string): Promise<XAPIEventRow[]> {
  const res = await authorizedFetch(`/api/v1/courses/${encodeURIComponent(courseCode)}/events`)
  if (!res.ok) throw new Error(`Failed to load event log (${res.status})`)
  const data = (await res.json()) as { events: XAPIEventRow[] }
  return data.events ?? []
}
