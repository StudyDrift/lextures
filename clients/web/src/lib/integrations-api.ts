import { authorizedFetch } from './api'
import { readApiErrorMessage } from './errors'

export type IntegrationProvider = 'google_classroom' | 'microsoft_teams' | 'canva'

export type IntegrationConnection = {
  id?: string
  provider: IntegrationProvider
  displayName: string
  externalId?: string
  scopes: string[]
  lastSyncedAt?: string
  lastSyncError?: string
  connected: boolean
  createdAt?: string
}

export type SyncStatus = {
  connectionId: string
  provider: string
  lastSyncedAt?: string
  lastSyncError?: string
  stale: boolean
  links: Array<{
    id: string
    externalCourseId: string
    syncRoster: boolean
    syncIntervalHours: number
    lastSyncedAt?: string
  }>
}

async function readError(res: Response, fallback: string): Promise<string> {
  try {
    return readApiErrorMessage(await res.json())
  } catch {
    return fallback
  }
}

export async function fetchIntegrations(): Promise<IntegrationConnection[]> {
  const res = await authorizedFetch('/api/v1/integrations')
  if (!res.ok) {
    throw new Error(await readError(res, 'Failed to load integrations.'))
  }
  const body = (await res.json()) as { integrations: IntegrationConnection[] }
  return body.integrations ?? []
}

export async function startConnect(provider: IntegrationProvider): Promise<string> {
  const res = await authorizedFetch(`/integrations/oauth/${provider}/connect`)
  if (!res.ok) {
    throw new Error(await readError(res, 'Failed to start the connection flow.'))
  }
  const body = (await res.json()) as { authorizeUrl: string }
  return body.authorizeUrl
}

export async function disconnectIntegration(id: string): Promise<void> {
  const res = await authorizedFetch(`/api/v1/integrations/${id}`, { method: 'DELETE' })
  if (!res.ok && res.status !== 204) {
    throw new Error(await readError(res, 'Failed to disconnect integration.'))
  }
}

export async function fetchSyncStatus(id: string): Promise<SyncStatus> {
  const res = await authorizedFetch(`/api/v1/integrations/${id}/sync-status`)
  if (!res.ok) {
    throw new Error(await readError(res, 'Failed to load sync status.'))
  }
  return (await res.json()) as SyncStatus
}
