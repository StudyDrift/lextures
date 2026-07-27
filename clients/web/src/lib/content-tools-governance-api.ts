import { authorizedFetch } from './api'

async function parseJSON<T>(res: Response): Promise<T> {
  if (!res.ok) {
    const text = await res.text()
    throw new Error(text || `HTTP ${res.status}`)
  }
  return (await res.json()) as T
}

export type ContentToolOrgPolicy = {
  deniedCapabilities: string[]
  deniedToolIds: string[]
  allowedToolIds: string[]
  aiDisclosureMode: 'none' | 'banner' | 'acknowledge'
  freeTextFilterAction: 'allow' | 'flag' | 'block'
  crisisEscalationEnabled?: boolean
  aiLogRetentionDays: number
  updatedAt?: string
}

export type ContentToolDataSheet = {
  toolId: string
  version: string
  name: string
  collects: Record<string, { purpose: string; retention: string }>
  leavesPlatform: boolean
  processors: string[]
  visibility: string
  wcagLevel: string
  a11yLimitations?: string
  capabilities: string[]
  aiTransparency?: {
    purpose: string
    modelFamily: string
    humanOversight: string
    limitations?: string
  }
}

export type ContentToolModerationAction = {
  id: string
  instanceId: string
  action: string
  category?: string
  reason?: string
  contentPath?: string
  createdAt: string
}

export async function fetchContentToolOrgPolicy(orgId: string): Promise<ContentToolOrgPolicy> {
  const res = await authorizedFetch(`/api/v1/orgs/${encodeURIComponent(orgId)}/content-tool-policy`)
  return parseJSON(res)
}

export async function putContentToolOrgPolicy(
  orgId: string,
  policy: ContentToolOrgPolicy,
): Promise<ContentToolOrgPolicy> {
  const res = await authorizedFetch(`/api/v1/orgs/${encodeURIComponent(orgId)}/content-tool-policy`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(policy),
  })
  return parseJSON(res)
}

export async function fetchContentToolDataSheets(): Promise<ContentToolDataSheet[]> {
  const res = await authorizedFetch('/api/v1/content-tools/data-sheets')
  const body = await parseJSON<{ dataSheets: ContentToolDataSheet[] }>(res)
  return body.dataSheets ?? []
}

export async function fetchContentToolAIConsent(
  courseCode: string,
  toolId: string,
): Promise<{ aiDisclosureMode: string; decision: string | null }> {
  const qs = new URLSearchParams({ toolId })
  const res = await authorizedFetch(
    `/api/v1/courses/${encodeURIComponent(courseCode)}/content-tools/ai-consent?${qs}`,
  )
  return parseJSON(res)
}

export async function postContentToolAIConsent(
  courseCode: string,
  body: { toolId?: string; decision: 'acknowledged' | 'opted_out' },
): Promise<unknown> {
  const res = await authorizedFetch(
    `/api/v1/courses/${encodeURIComponent(courseCode)}/content-tools/ai-consent`,
    {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    },
  )
  return parseJSON(res)
}

export async function reportContentToolContent(
  courseCode: string,
  instanceId: string,
  body: { category?: string; reason?: string; contentPath?: string },
): Promise<ContentToolModerationAction> {
  const res = await authorizedFetch(
    `/api/v1/courses/${encodeURIComponent(courseCode)}/content-tools/instances/${encodeURIComponent(instanceId)}/report`,
    {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    },
  )
  return parseJSON(res)
}

export async function moderateContentToolContent(
  courseCode: string,
  instanceId: string,
  body: { action: string; category?: string; reason?: string; contentPath?: string },
): Promise<ContentToolModerationAction> {
  const res = await authorizedFetch(
    `/api/v1/courses/${encodeURIComponent(courseCode)}/content-tools/instances/${encodeURIComponent(instanceId)}/moderate`,
    {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    },
  )
  return parseJSON(res)
}

export async function fetchContentToolModeration(
  courseCode: string,
  instanceId: string,
): Promise<ContentToolModerationAction[]> {
  const res = await authorizedFetch(
    `/api/v1/courses/${encodeURIComponent(courseCode)}/content-tools/instances/${encodeURIComponent(instanceId)}/moderation`,
  )
  const body = await parseJSON<{ items: ContentToolModerationAction[] }>(res)
  return body.items ?? []
}

export async function postContentToolKill(body: {
  scope: 'tool' | 'capability' | 'all_ai' | 'instance'
  target: string
  engaged?: boolean
  reason?: string
}): Promise<unknown> {
  const res = await authorizedFetch('/api/v1/admin/content-tools/kill', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  return parseJSON(res)
}

export async function fetchContentToolConformance(): Promise<{
  ok: boolean
  tools: Array<{ toolId: string; ok: boolean; wcagLevel: string; errors?: string[] }>
}> {
  const res = await authorizedFetch('/api/v1/content-tools/conformance')
  return parseJSON(res)
}
