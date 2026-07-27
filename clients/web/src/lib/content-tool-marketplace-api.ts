import { authorizedFetch } from './api'

export type MarketplaceToolListing = {
  toolId: string
  displayName: string
  summary: string
  subjectTags: string[]
  gradeTags: string[]
  visibility: string
  pricingModel: string
  status: string
  version: string
  wcagLevel: string
  capabilities: string[]
  sunsetAt?: string
}

export type DeveloperTool = {
  id: string
  toolId: string
  displayName: string
  summary: string
  descriptionMd: string
  subjectTags: string[]
  gradeTags: string[]
  visibility: string
  pricingModel: string
  status: string
}

export type ConsentCapability = {
  capability: string
  plainLanguage: string
}

export type InstallPreview = {
  toolId: string
  displayName: string
  version: string
  capabilities: ConsentCapability[]
  hosts: string[]
  pricingModel: string
  bundleSri: string
}

export type ToolInstallation = {
  id: string
  orgId: string
  toolId: string
  displayName: string
  pinnedMajor: number
  currentVersion: string
  consentedCapabilities: string[]
  consentedHosts: string[]
  autoUpdateMinor: boolean
  status: string
  installedAt: string
  installedBy?: string
  revokedAt?: string
}

export type ToolReview = {
  id: string
  toolId?: string
  displayName?: string
  version: string
  reviewStatus: string
  checks: unknown
  visibility?: string
  createdAt: string
}

export async function listDeveloperTools(): Promise<DeveloperTool[]> {
  const res = await authorizedFetch('/api/v1/developer/tools')
  if (!res.ok) throw new Error(await res.text())
  const body = (await res.json()) as { tools: DeveloperTool[] }
  return body.tools ?? []
}

export async function createDeveloperTool(input: {
  toolId: string
  displayName: string
  summary: string
  descriptionMd?: string
  subjectTags?: string[]
  gradeTags?: string[]
  visibility?: string
}): Promise<DeveloperTool> {
  const res = await authorizedFetch('/api/v1/developer/tools', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(input),
  })
  if (!res.ok) throw new Error(await res.text())
  return (await res.json()) as DeveloperTool
}

export async function createDeveloperRelease(
  toolId: string,
  input: {
    version: string
    manifest: unknown
    dataSheet?: unknown
    bundleBase64: string
    axeStatus?: string
    keyboardTestStatus?: string
    i18nKeys?: Record<string, string>
  },
): Promise<{ release: { id: string; version: string; reviewStatus: string }; checks: { ok: boolean } }> {
  const res = await authorizedFetch(`/api/v1/developer/tools/${encodeURIComponent(toolId)}/releases`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(input),
  })
  if (!res.ok) throw new Error(await res.text())
  return (await res.json()) as {
    release: { id: string; version: string; reviewStatus: string }
    checks: { ok: boolean }
  }
}

export async function submitDeveloperRelease(toolId: string, version: string): Promise<void> {
  const res = await authorizedFetch(
    `/api/v1/developer/tools/${encodeURIComponent(toolId)}/releases/${encodeURIComponent(version)}/submit`,
    { method: 'POST' },
  )
  if (!res.ok) throw new Error(await res.text())
}

export async function fetchDeveloperAnalytics(toolId: string): Promise<{
  toolId: string
  activeInstalls: number
  revokedInstalls: number
  usageEvents: number
}> {
  const res = await authorizedFetch(`/api/v1/developer/tools/${encodeURIComponent(toolId)}/analytics`)
  if (!res.ok) throw new Error(await res.text())
  return (await res.json()) as {
    toolId: string
    activeInstalls: number
    revokedInstalls: number
    usageEvents: number
  }
}

export async function browseToolMarketplace(params?: {
  subject?: string
  grade?: string
  q?: string
}): Promise<MarketplaceToolListing[]> {
  const qs = new URLSearchParams()
  if (params?.subject) qs.set('subject', params.subject)
  if (params?.grade) qs.set('grade', params.grade)
  if (params?.q) qs.set('q', params.q)
  const res = await authorizedFetch(`/api/v1/tool-marketplace/tools?${qs}`)
  if (!res.ok) throw new Error(await res.text())
  const body = (await res.json()) as { tools: MarketplaceToolListing[] }
  return body.tools ?? []
}

export async function fetchInstallPreview(orgId: string, toolId: string): Promise<InstallPreview> {
  const res = await authorizedFetch(
    `/api/v1/orgs/${encodeURIComponent(orgId)}/tool-installations/preview?toolId=${encodeURIComponent(toolId)}`,
  )
  if (!res.ok) throw new Error(await res.text())
  return (await res.json()) as InstallPreview
}

export async function installTool(
  orgId: string,
  input: { toolId: string; consented: boolean; autoUpdateMinor?: boolean },
): Promise<ToolInstallation> {
  const res = await authorizedFetch(`/api/v1/orgs/${encodeURIComponent(orgId)}/tool-installations`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(input),
  })
  if (!res.ok) throw new Error(await res.text())
  return (await res.json()) as ToolInstallation
}

export async function listOrgToolInstallations(orgId: string): Promise<ToolInstallation[]> {
  const res = await authorizedFetch(`/api/v1/orgs/${encodeURIComponent(orgId)}/tool-installations`)
  if (!res.ok) throw new Error(await res.text())
  const body = (await res.json()) as { installations: ToolInstallation[] }
  return body.installations ?? []
}

export async function revokeToolInstallation(orgId: string, id: string): Promise<ToolInstallation> {
  const res = await authorizedFetch(
    `/api/v1/orgs/${encodeURIComponent(orgId)}/tool-installations/${encodeURIComponent(id)}`,
    { method: 'DELETE' },
  )
  if (!res.ok) throw new Error(await res.text())
  return (await res.json()) as ToolInstallation
}

export async function listToolReviews(status = 'pending'): Promise<ToolReview[]> {
  const res = await authorizedFetch(`/api/v1/admin/tool-reviews?status=${encodeURIComponent(status)}`)
  if (!res.ok) throw new Error(await res.text())
  const body = (await res.json()) as { reviews: ToolReview[] }
  return body.reviews ?? []
}

export async function decideToolReview(
  releaseId: string,
  input: { approve: boolean; notes: string },
): Promise<void> {
  const res = await authorizedFetch(`/api/v1/admin/tool-reviews/${encodeURIComponent(releaseId)}/decision`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(input),
  })
  if (!res.ok) throw new Error(await res.text())
}
