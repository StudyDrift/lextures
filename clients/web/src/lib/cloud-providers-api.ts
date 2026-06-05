import { authorizedFetch } from './api'
import { readApiErrorMessage } from './errors'

export type CloudProviderId = 'google_drive' | 'onedrive' | 'dropbox'

export type CloudProviderSetting = {
  provider: CloudProviderId
  enabled: boolean
  clientId: string
  apiKey: string
  appKey: string
  updatedAt: string
}

export type ConfiguredCloudProvider = {
  provider: CloudProviderId
  clientId?: string
  apiKey?: string
  appKey?: string
}

async function parseJson(res: Response): Promise<unknown> {
  try {
    return await res.json()
  } catch {
    return null
  }
}

function parseProviderId(raw: unknown): CloudProviderId | null {
  if (raw === 'google_drive' || raw === 'onedrive' || raw === 'dropbox') return raw
  return null
}

function parseCloudProviderSetting(row: Record<string, unknown>): CloudProviderSetting | null {
  const provider = parseProviderId(row.provider)
  if (!provider) return null
  return {
    provider,
    enabled: Boolean(row.enabled),
    clientId: String(row.clientId ?? ''),
    apiKey: String(row.apiKey ?? ''),
    appKey: String(row.appKey ?? ''),
    updatedAt: String(row.updatedAt ?? ''),
  }
}

export async function fetchCloudProviders(): Promise<ConfiguredCloudProvider[]> {
  const res = await authorizedFetch('/api/v1/cloud-providers')
  const raw = await parseJson(res)
  if (!res.ok) throw new Error(readApiErrorMessage(raw))
  if (!Array.isArray(raw)) return []
  return raw.flatMap((row) => {
    const r = row as Record<string, unknown>
    const provider = parseProviderId(r.provider)
    if (!provider) return []
    return [{
      provider,
      clientId: r.clientId != null ? String(r.clientId) : undefined,
      apiKey: r.apiKey != null ? String(r.apiKey) : undefined,
      appKey: r.appKey != null ? String(r.appKey) : undefined,
    }]
  })
}

export async function fetchAdminCloudProviders(): Promise<CloudProviderSetting[]> {
  const res = await authorizedFetch('/api/v1/admin/cloud-providers')
  const raw = await parseJson(res)
  if (!res.ok) throw new Error(readApiErrorMessage(raw))
  if (!Array.isArray(raw)) return []
  return raw.flatMap((row) => {
    const parsed = parseCloudProviderSetting(row as Record<string, unknown>)
    return parsed ? [parsed] : []
  })
}

export type CloudProviderUpdate = {
  enabled?: boolean
  clientId?: string
  apiKey?: string
  appKey?: string
}

export async function putAdminCloudProvider(
  provider: CloudProviderId,
  update: CloudProviderUpdate,
): Promise<void> {
  const res = await authorizedFetch(
    `/api/v1/admin/cloud-providers/${encodeURIComponent(provider)}`,
    {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(update),
    },
  )
  if (!res.ok) {
    const raw = await parseJson(res)
    throw new Error(readApiErrorMessage(raw))
  }
}

export const CLOUD_PROVIDER_LABELS: Record<CloudProviderId, string> = {
  google_drive: 'Google Drive',
  onedrive: 'OneDrive',
  dropbox: 'Dropbox',
}
