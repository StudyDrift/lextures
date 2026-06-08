import { authorizedFetch } from './api'
import { readApiErrorMessage } from './errors'

export type SISVendor =
  | 'powerschool'
  | 'infinite_campus'
  | 'skyward'
  | 'aeries'
  | 'banner'
  | 'workday'
  | 'colleague'
  | 'jenzabar'
  | 'peoplesoft'

export type SISConnection = {
  id: string
  orgId: string
  vendor: SISVendor
  market: 'k12' | 'he'
  baseUrl: string
  clientIdRef: string
  clientSecretRef: string
  syncSchedule: string
  syncMode: string
  active: boolean
  lastSyncAt: string | null
  createdAt: string
}

export type SISSyncLog = {
  id: string
  connectionId: string
  startedAt: string
  finishedAt: string | null
  status: string
  summary: Record<string, number> | null
  errors: Array<{ record_id?: string; message: string }> | null
}

export const HE_SIS_VENDORS: Array<{ value: SISVendor; label: string }> = [
  { value: 'banner', label: 'Ellucian Banner' },
  { value: 'workday', label: 'Workday Student' },
  { value: 'colleague', label: 'Ellucian Colleague' },
  { value: 'jenzabar', label: 'Jenzabar' },
  { value: 'peoplesoft', label: 'Oracle PeopleSoft' },
]

export function vendorLabel(vendor: string): string {
  const found = HE_SIS_VENDORS.find((v) => v.value === vendor)
  if (found) return found.label
  return vendor.replace(/_/g, ' ')
}

export async function fetchSISConnections(orgId: string): Promise<SISConnection[]> {
  const res = await authorizedFetch(`/api/v1/admin/orgs/${encodeURIComponent(orgId)}/sis/connections`)
  const raw = await res.json().catch(() => ({}))
  if (!res.ok) throw new Error(readApiErrorMessage(raw))
  const data = raw as { connections?: SISConnection[] }
  return data.connections ?? []
}

export async function createSISConnection(
  orgId: string,
  body: {
    vendor: SISVendor
    baseUrl: string
    clientIdRef: string
    clientSecretRef: string
    syncSchedule?: string
    syncMode?: string
  },
): Promise<SISConnection> {
  const res = await authorizedFetch(`/api/v1/admin/orgs/${encodeURIComponent(orgId)}/sis/connections`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  const raw = await res.json().catch(() => ({}))
  if (!res.ok) throw new Error(readApiErrorMessage(raw))
  const data = raw as { connection: SISConnection }
  return data.connection
}

export async function triggerSISSync(
  orgId: string,
  connectionId: string,
): Promise<{ logId: string; status: string }> {
  const res = await authorizedFetch(
    `/api/v1/admin/orgs/${encodeURIComponent(orgId)}/sis/connections/${encodeURIComponent(connectionId)}/sync`,
    { method: 'POST' },
  )
  const raw = await res.json().catch(() => ({}))
  if (!res.ok) throw new Error(readApiErrorMessage(raw))
  return raw as { logId: string; status: string }
}

export async function testSISConnection(
  orgId: string,
  connectionId: string,
): Promise<{ ok: boolean; message: string }> {
  const res = await authorizedFetch(
    `/api/v1/admin/orgs/${encodeURIComponent(orgId)}/sis/connections/${encodeURIComponent(connectionId)}/test`,
    { method: 'POST' },
  )
  const raw = await res.json().catch(() => ({}))
  if (!res.ok) throw new Error(readApiErrorMessage(raw))
  return raw as { ok: boolean; message: string }
}

export async function fetchSISSyncLogs(orgId: string): Promise<SISSyncLog[]> {
  const res = await authorizedFetch(`/api/v1/admin/orgs/${encodeURIComponent(orgId)}/sis/sync-logs`)
  const raw = await res.json().catch(() => ({}))
  if (!res.ok) throw new Error(readApiErrorMessage(raw))
  const data = raw as { logs?: SISSyncLog[] }
  return data.logs ?? []
}
