import { authorizedFetch } from './api'

function apiBase(): string {
  return import.meta.env.VITE_API_URL ?? ''
}

export type WalletKind =
  | 'transcript'
  | 'clr'
  | 'badge'
  | 'certificate'
  | 'diploma'
  | 'ce_record'

export type WalletVerifyStatus = 'verified' | 'revoked' | 'unverified' | 'unavailable'

export type WalletDisclosure = 'validity' | 'summary' | 'full'

export type WalletItem = {
  id: string
  kind: WalletKind
  sourceId: string
  title: string
  issuer?: string
  issuedAt?: string
  revoked: boolean
  verifyStatus: WalletVerifyStatus
  verifyUrl?: string
  metadata?: Record<string, unknown>
  createdAt: string
  updatedAt: string
}

export type WalletCollection = {
  id: string
  name: string
  disclosure: WalletDisclosure
  itemIds: string[]
  shareToken?: string
  shareUrl?: string
  expiresAt?: string
  revoked: boolean
  revokedAt?: string
  createdAt: string
  updatedAt: string
}

export type WalletAccessEvent = {
  id: string
  result: string
  requesterIp?: string
  requesterUa?: string
  createdAt: string
}

export type WalletExportStatus = {
  id: string
  status: 'pending' | 'ready' | 'failed'
  createdAt: string
  completedAt?: string
  error?: string
  downloadPath?: string
}

export type PublicWalletShare = {
  name: string
  disclosure: WalletDisclosure
  items: Array<{
    kind: string
    valid: boolean
    revoked: boolean
    title?: string
    issuer?: string
    issuedAt?: string
    verifyUrl?: string
    verifyStatus?: string
  }>
}

async function readErrorMessage(res: Response, fallback: string): Promise<string> {
  try {
    const body = (await res.json()) as { error?: { message?: string } }
    return body.error?.message ?? fallback
  } catch {
    return fallback
  }
}

export async function fetchWallet(): Promise<{ items: WalletItem[]; alumniNote?: string }> {
  const res = await authorizedFetch('/api/v1/me/wallet')
  if (!res.ok) throw new Error(await readErrorMessage(res, 'Failed to load wallet.'))
  return res.json() as Promise<{ items: WalletItem[]; alumniNote?: string }>
}

export async function fetchWalletCollections(): Promise<WalletCollection[]> {
  const res = await authorizedFetch('/api/v1/me/wallet/collections')
  if (!res.ok) throw new Error(await readErrorMessage(res, 'Failed to load collections.'))
  const data = (await res.json()) as { collections: WalletCollection[] }
  return data.collections ?? []
}

export async function createWalletCollection(input: {
  name: string
  disclosure: WalletDisclosure
  itemIds: string[]
  share?: boolean
  expiresAt?: string
}): Promise<WalletCollection> {
  const res = await authorizedFetch('/api/v1/me/wallet/collections', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(input),
  })
  if (!res.ok) throw new Error(await readErrorMessage(res, 'Failed to create collection.'))
  return res.json() as Promise<WalletCollection>
}

export async function revokeWalletCollection(id: string): Promise<WalletCollection> {
  const res = await authorizedFetch(`/api/v1/me/wallet/collections/${encodeURIComponent(id)}/revoke`, {
    method: 'POST',
  })
  if (!res.ok) throw new Error(await readErrorMessage(res, 'Failed to revoke share link.'))
  return res.json() as Promise<WalletCollection>
}

export async function deleteWalletCollection(id: string): Promise<void> {
  const res = await authorizedFetch(`/api/v1/me/wallet/collections/${encodeURIComponent(id)}`, {
    method: 'DELETE',
  })
  if (!res.ok && res.status !== 204) {
    throw new Error(await readErrorMessage(res, 'Failed to delete collection.'))
  }
}

export async function fetchWalletCollectionAccess(id: string): Promise<WalletAccessEvent[]> {
  const res = await authorizedFetch(`/api/v1/me/wallet/collections/${encodeURIComponent(id)}/access`)
  if (!res.ok) throw new Error(await readErrorMessage(res, 'Failed to load access history.'))
  const data = (await res.json()) as { access: WalletAccessEvent[] }
  return data.access ?? []
}

export async function startWalletExport(): Promise<WalletExportStatus> {
  const res = await authorizedFetch('/api/v1/me/wallet/export', { method: 'POST' })
  if (!res.ok) throw new Error(await readErrorMessage(res, 'Failed to start export.'))
  return res.json() as Promise<WalletExportStatus>
}

export async function fetchWalletExport(id: string): Promise<WalletExportStatus> {
  const res = await authorizedFetch(`/api/v1/me/wallet/export/${encodeURIComponent(id)}`)
  if (!res.ok) throw new Error(await readErrorMessage(res, 'Failed to load export.'))
  return res.json() as Promise<WalletExportStatus>
}

export async function downloadWalletExport(id: string): Promise<void> {
  const res = await authorizedFetch(`/api/v1/me/wallet/export/${encodeURIComponent(id)}/download`)
  if (!res.ok) throw new Error(await readErrorMessage(res, 'Failed to download export.'))
  const blob = await res.blob()
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = 'credential-wallet.zip'
  a.click()
  URL.revokeObjectURL(url)
}

export async function fetchPublicWalletShare(token: string): Promise<PublicWalletShare> {
  const res = await fetch(`${apiBase()}/api/v1/wallet/s/${encodeURIComponent(token)}`)
  if (!res.ok) throw new Error(await readErrorMessage(res, 'Could not open this share link.'))
  return res.json() as Promise<PublicWalletShare>
}
