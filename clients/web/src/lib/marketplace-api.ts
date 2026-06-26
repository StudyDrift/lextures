import { authorizedFetch } from './api'
import { readApiErrorMessage } from './errors'

export type MarketplaceApp = {
  id: string
  name: string
  slug: string
  description: string
  logoUrl: string | null
  requestedScopes: string[]
}

export type DeveloperApp = {
  id: string
  name: string
  slug: string
  description: string
  logoUrl: string | null
  clientId: string
  clientSecretPrefix: string
  redirectUris: string[]
  requestedScopes: string[]
  published: boolean
  createdAt: string
}

export type ScopeInfo = {
  scope: string
  label: string
  isWrite: boolean
}

export type OAuthConsentInfo = {
  state: string
  appName: string
  appLogoUrl: string | null
  scopes: ScopeInfo[]
}

export type InstalledApp = {
  id: string
  appId: string
  appName: string
  appSlug: string
  appLogoUrl: string | null
  grantedScopes: string[]
  installedAt: string
  installedBy: string | null
  lastUsedAt: string | null
}

export async function fetchMarketplaceApps(): Promise<MarketplaceApp[]> {
  const res = await authorizedFetch('/api/v1/marketplace/apps')
  if (!res.ok) throw new Error(await readApiErrorMessage(res))
  const data = await res.json()
  return data.apps ?? []
}

export async function fetchMarketplaceApp(slug: string): Promise<MarketplaceApp | null> {
  const res = await authorizedFetch(`/api/v1/marketplace/apps/${slug}`)
  if (res.status === 404) return null
  if (!res.ok) throw new Error(await readApiErrorMessage(res))
  return res.json()
}

export async function fetchOAuthConsentInfo(params: {
  clientId: string
  redirectUri: string
  scope: string
  codeChallenge: string
}): Promise<OAuthConsentInfo> {
  const q = new URLSearchParams({
    client_id: params.clientId,
    redirect_uri: params.redirectUri,
    scope: params.scope,
    code_challenge: params.codeChallenge,
    code_challenge_method: 'S256',
  })
  const res = await authorizedFetch(`/oauth/authorize?${q}`)
  if (!res.ok) throw new Error(await readApiErrorMessage(res))
  return res.json()
}

export async function fetchDeveloperApps(): Promise<DeveloperApp[]> {
  const res = await authorizedFetch('/api/v1/developer/apps')
  if (!res.ok) throw new Error(await readApiErrorMessage(res))
  const data = await res.json()
  return data.apps ?? []
}

export type CreateAppInput = {
  name: string
  slug: string
  description: string
  logoUrl?: string
  redirectUris: string[]
  requestedScopes: string[]
}

export type CreateAppResult = DeveloperApp & { clientSecret: string }

export async function createDeveloperApp(input: CreateAppInput): Promise<CreateAppResult> {
  const res = await authorizedFetch('/api/v1/developer/apps', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(input),
  })
  if (!res.ok) throw new Error(await readApiErrorMessage(res))
  return res.json()
}

export async function fetchInstalledApps(): Promise<InstalledApp[]> {
  const res = await authorizedFetch('/api/v1/admin/marketplace/installed')
  if (!res.ok) throw new Error(await readApiErrorMessage(res))
  const data = await res.json()
  return data.installations ?? []
}

export async function revokeInstalledApp(id: string): Promise<void> {
  const res = await authorizedFetch(`/api/v1/admin/marketplace/installed/${id}`, {
    method: 'DELETE',
  })
  if (!res.ok && res.status !== 204) throw new Error(await readApiErrorMessage(res))
}
