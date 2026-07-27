import { authorizedFetch } from './api'

export type ContentToolVersionRow = {
  toolId: string
  version: string
  configSchemaVersion: number
  stateSchemaVersion: number
  sandboxMode: string
  status: string
  breakerOpen: boolean
  breakerOpenAt?: string | null
  sunsetAt?: string | null
  firstSeenAt: string
}

export type ContentToolsVersionsResponse = {
  versions: ContentToolVersionRow[]
  sandboxMode: string
  contractSupported: number[]
}

export type ContentToolMigrationJob = {
  id: string
  toolId: string
  fromVersion: number
  toVersion: number
  dryRun: boolean
  status: string
  totalDocs: number
  migratedDocs: number
  failedDocs: number
  error?: string | null
  createdAt: string
  finishedAt?: string | null
}

export type ContentToolQuarantineItem = {
  id: string
  stateId: string
  toolId: string
  fromVersion: number
  toVersion: number
  error: string
  createdAt: string
}

async function parseJSON<T>(res: Response): Promise<T> {
  if (!res.ok) {
    const text = await res.text()
    throw new Error(text || `HTTP ${res.status}`)
  }
  return (await res.json()) as T
}

export async function fetchContentToolVersions(): Promise<ContentToolsVersionsResponse> {
  const res = await authorizedFetch('/api/v1/admin/content-tools/versions')
  return parseJSON(res)
}

export async function patchContentToolVersion(
  toolId: string,
  version: string,
  body: { status?: string; resetBreaker?: boolean; openBreaker?: boolean },
): Promise<unknown> {
  const res = await authorizedFetch(
    `/api/v1/admin/content-tools/versions/${encodeURIComponent(toolId)}/${encodeURIComponent(version)}`,
    {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    },
  )
  return parseJSON(res)
}

/** CT.7 — Platform admin cross-course telemetry (counts only). */
export type ContentToolTelemetryRow = {
  toolId: string
  instances: number
  learners: number
  engagements: number
  completions: number
  meanScorePct?: number | null
  aiTokens: number
  aiCostUsd: number
  renderErrors: number
}

export async function fetchContentToolTelemetry(opts?: {
  from?: string
  to?: string
}): Promise<{ from: string; to: string; tools: ContentToolTelemetryRow[] }> {
  const qs = new URLSearchParams()
  if (opts?.from) qs.set('from', opts.from)
  if (opts?.to) qs.set('to', opts.to)
  const suffix = qs.toString() ? `?${qs}` : ''
  const res = await authorizedFetch(`/api/v1/admin/content-tools/telemetry${suffix}`)
  return parseJSON(res)
}

export async function createContentToolMigration(body: {
  toolId: string
  fromVersion?: number
  toVersion?: number
  dryRun?: boolean
}): Promise<ContentToolMigrationJob> {
  const res = await authorizedFetch('/api/v1/admin/content-tools/migrations', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  return parseJSON(res)
}

export async function fetchContentToolMigration(jobId: string): Promise<ContentToolMigrationJob> {
  const res = await authorizedFetch(
    `/api/v1/admin/content-tools/migrations/${encodeURIComponent(jobId)}`,
  )
  return parseJSON(res)
}

export async function fetchContentToolQuarantine(toolId?: string): Promise<{
  items: ContentToolQuarantineItem[]
}> {
  const q = toolId ? `?toolId=${encodeURIComponent(toolId)}` : ''
  const res = await authorizedFetch(`/api/v1/admin/content-tools/quarantine${q}`)
  return parseJSON(res)
}
