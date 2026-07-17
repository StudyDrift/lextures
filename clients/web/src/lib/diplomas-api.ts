import { authorizedFetch } from './api'

export type DiplomaKind = 'diploma' | 'certificate'

export type DiplomaTemplate = {
  id: string
  orgId: string
  kind: DiplomaKind
  name: string
  title: string
  program?: string
  conferralText?: string
  layout: Record<string, unknown>
  active: boolean
  createdAt: string
  updatedAt: string
}

export type IssuedDiploma = {
  id: string
  userId: string
  orgId: string
  templateId?: string
  kind: DiplomaKind
  credentialTitle: string
  program?: string
  honors?: string
  conferredAt: string
  version: number
  replacesId?: string
  contentHash: string
  verifyToken?: string
  revokedAt?: string
  revokeReason?: string
  issuedAt: string
  programRef?: string
  hasPdf: boolean
  hasVc: boolean
}

export type DiplomaBatch = {
  id: string
  orgId: string
  templateId: string
  programRef?: string
  program?: string
  honors?: string
  conferredAt: string
  status: string
  totalCount: number
  successCount: number
  failCount: number
  skipCount: number
  errorSummary?: string
  createdAt: string
  startedAt?: string
  finishedAt?: string
}

export async function fetchDiplomaTemplates(activeOnly = false): Promise<DiplomaTemplate[]> {
  const q = activeOnly ? '?active=true' : ''
  const res = await authorizedFetch(`/api/v1/admin/credentials/templates${q}`)
  if (!res.ok) throw new Error(`Failed to load templates (${res.status})`)
  const data = (await res.json()) as { templates?: DiplomaTemplate[] }
  return data.templates ?? []
}

export async function createDiplomaTemplate(body: {
  kind: DiplomaKind
  name: string
  title?: string
  program?: string
  conferralText?: string
  layout?: Record<string, unknown>
}): Promise<DiplomaTemplate> {
  const res = await authorizedFetch('/api/v1/admin/credentials/templates', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  if (!res.ok) throw new Error(`Failed to create template (${res.status})`)
  const data = (await res.json()) as { template: DiplomaTemplate }
  return data.template
}

export async function updateDiplomaTemplate(
  id: string,
  body: {
    name?: string
    title?: string
    program?: string
    conferralText?: string
    layout?: Record<string, unknown>
    active?: boolean
  },
): Promise<DiplomaTemplate> {
  const res = await authorizedFetch(`/api/v1/admin/credentials/templates/${encodeURIComponent(id)}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  if (!res.ok) throw new Error(`Failed to update template (${res.status})`)
  const data = (await res.json()) as { template: DiplomaTemplate }
  return data.template
}

export async function issueDiploma(body: {
  userId: string
  templateId: string
  learnerName?: string
  program?: string
  honors?: string
  conferredAt?: string
  programRef?: string
  correctPrior?: boolean
}): Promise<{ diploma: IssuedDiploma; skipped: boolean; reason?: string }> {
  const res = await authorizedFetch('/api/v1/admin/credentials/issue', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  if (!res.ok) throw new Error(`Failed to issue credential (${res.status})`)
  return (await res.json()) as { diploma: IssuedDiploma; skipped: boolean; reason?: string }
}

export async function issueDiplomaBatch(body: {
  templateId: string
  userIds: string[]
  program?: string
  honors?: string
  conferredAt?: string
  programRef?: string
}): Promise<DiplomaBatch> {
  const res = await authorizedFetch('/api/v1/admin/credentials/issue/batch', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  if (!res.ok) throw new Error(`Failed to start batch (${res.status})`)
  const data = (await res.json()) as { batch: DiplomaBatch }
  return data.batch
}

export async function fetchDiplomaBatch(id: string): Promise<DiplomaBatch> {
  const res = await authorizedFetch(`/api/v1/admin/credentials/batches/${encodeURIComponent(id)}`)
  if (!res.ok) throw new Error(`Failed to load batch (${res.status})`)
  const data = (await res.json()) as { batch: DiplomaBatch }
  return data.batch
}

export async function revokeDiploma(id: string, reason?: string): Promise<IssuedDiploma> {
  const res = await authorizedFetch(`/api/v1/admin/credentials/${encodeURIComponent(id)}/revoke`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ reason: reason ?? '' }),
  })
  if (!res.ok) throw new Error(`Failed to revoke (${res.status})`)
  const data = (await res.json()) as { diploma: IssuedDiploma }
  return data.diploma
}

export async function unrevokeDiploma(id: string): Promise<IssuedDiploma> {
  const res = await authorizedFetch(`/api/v1/admin/credentials/${encodeURIComponent(id)}/unrevoke`, {
    method: 'POST',
  })
  if (!res.ok) throw new Error(`Failed to unrevoke (${res.status})`)
  const data = (await res.json()) as { diploma: IssuedDiploma }
  return data.diploma
}
