import { authorizedFetch } from './api'
import { readApiErrorMessage } from './errors'
import type { LinkedInCertParams } from './linkedin-share'

export type IssuedCredentialSummary = {
  id: string
  title: string
  sourceType: string
  sourceId: string
  issuedAt: string
  verificationUrl: string
  revoked: boolean
}

export type CredentialsListResponse = {
  credentials: IssuedCredentialSummary[]
}

export type LinkedInParamsResponse = LinkedInCertParams & {
  url: string
  certUrl: string
  certId: string
}

export type BadgeExportResponse = {
  downloadUrl: string
  expiresAt: string
}

export type CredentialVerifyResponse = {
  valid: boolean
  status: string
  issuerName: string
  issuedAt: string
  title: string
  learnerName?: string
  verifyType?: string
  credential: Record<string, unknown>
}

export async function fetchMyCredentials(): Promise<CredentialsListResponse> {
  const res = await authorizedFetch('/api/v1/me/credentials')
  const raw = await res.json().catch(() => ({}))
  if (!res.ok) throw new Error(readApiErrorMessage(raw))
  return raw as CredentialsListResponse
}

export async function fetchLinkedInParams(credentialId: string): Promise<LinkedInParamsResponse> {
  const res = await authorizedFetch(`/api/v1/credentials/${credentialId}/linkedin-params`)
  const raw = await res.json().catch(() => ({}))
  if (!res.ok) throw new Error(readApiErrorMessage(raw))
  return raw as LinkedInParamsResponse
}

export async function fetchBadgeExportUrl(credentialId: string): Promise<BadgeExportResponse> {
  const res = await authorizedFetch(`/api/v1/credentials/${credentialId}/badge-export`)
  const raw = await res.json().catch(() => ({}))
  if (!res.ok) throw new Error(readApiErrorMessage(raw))
  return raw as BadgeExportResponse
}

export async function downloadCredentialPdf(credentialId: string): Promise<Blob> {
  const res = await authorizedFetch(`/api/v1/credentials/${credentialId}/download`)
  if (!res.ok) {
    const raw = await res.json().catch(() => ({}))
    throw new Error(readApiErrorMessage(raw))
  }
  return res.blob()
}

export async function recordCredentialShare(
  credentialId: string,
  channel: 'linkedin' | 'badge_export',
): Promise<void> {
  const res = await authorizedFetch(`/api/v1/credentials/${credentialId}/share`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ channel }),
  })
  if (!res.ok && res.status !== 204) {
    const raw = await res.json().catch(() => ({}))
    throw new Error(readApiErrorMessage(raw))
  }
}

export async function verifyCredentialId(credentialId: string): Promise<CredentialVerifyResponse> {
  const base = import.meta.env.VITE_API_URL ?? ''
  const res = await fetch(`${base}/api/v1/credentials/${encodeURIComponent(credentialId)}/verify`)
  if (res.status === 404) {
    throw new Error('Credential not found.')
  }
  if (!res.ok) {
    throw new Error('Verification failed.')
  }
  return (await res.json()) as CredentialVerifyResponse
}