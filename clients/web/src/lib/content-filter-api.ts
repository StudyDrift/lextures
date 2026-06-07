import { authorizedFetch } from './api'
import { readApiErrorMessage } from './errors'

export type ContentFilterSettings = {
  orgId: string
  goGuardianEnabled: boolean
  goGuardianApiKey: string
  hasGoGuardianApiKey: boolean
  securlyEnabled: boolean
  updatedAt: string
  allowlistUrl: string
}

export const CONTENT_FILTER_SECRET_PLACEHOLDER = '••••••••••••'

async function parseJson(res: Response): Promise<unknown> {
  return res.json().catch(() => ({}))
}

export async function fetchContentFilterSettings(orgId: string): Promise<ContentFilterSettings> {
  const res = await authorizedFetch(
    `/api/v1/orgs/${encodeURIComponent(orgId)}/settings/content-filter`,
  )
  const raw = await parseJson(res)
  if (!res.ok) throw new Error(readApiErrorMessage(raw))
  const data = raw as Partial<ContentFilterSettings>
  return {
    orgId: data.orgId ?? orgId,
    goGuardianEnabled: data.goGuardianEnabled === true,
    goGuardianApiKey: data.goGuardianApiKey ?? '',
    hasGoGuardianApiKey: data.hasGoGuardianApiKey === true,
    securlyEnabled: data.securlyEnabled === true,
    updatedAt: data.updatedAt ?? '',
    allowlistUrl: data.allowlistUrl ?? '/.well-known/content-filter-allowlist.json',
  }
}

export async function patchContentFilterSettings(
  orgId: string,
  body: {
    goGuardianEnabled?: boolean
    goGuardianApiKey?: string
    clearGoGuardianApiKey?: boolean
    securlyEnabled?: boolean
  },
): Promise<ContentFilterSettings> {
  const res = await authorizedFetch(
    `/api/v1/orgs/${encodeURIComponent(orgId)}/settings/content-filter`,
    {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    },
  )
  const raw = await parseJson(res)
  if (!res.ok) throw new Error(readApiErrorMessage(raw))
  const data = raw as Partial<ContentFilterSettings>
  return {
    orgId: data.orgId ?? orgId,
    goGuardianEnabled: data.goGuardianEnabled === true,
    goGuardianApiKey: data.goGuardianApiKey ?? '',
    hasGoGuardianApiKey: data.hasGoGuardianApiKey === true,
    securlyEnabled: data.securlyEnabled === true,
    updatedAt: data.updatedAt ?? '',
    allowlistUrl: data.allowlistUrl ?? '/.well-known/content-filter-allowlist.json',
  }
}

export async function emitContentFilterActivity(url: string, title: string): Promise<void> {
  try {
    await authorizedFetch('/api/v1/content-filter/activity', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ url, title }),
    })
  } catch {
    /* best-effort; filter integration must not block navigation */
  }
}
