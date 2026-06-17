import { authorizedFetch } from './api'

export type IssuedCredential = {
  id: string
  sourceType: 'course' | 'path' | 'ceu'
  sourceId: string
  title: string
  issuedAt: string
  revoked: boolean
  hasPdf: boolean
  verificationUrl: string
}

export type CredentialsListResponse = {
  credentials: IssuedCredential[]
}

export type CredentialVerifyResponse = {
  valid: boolean
  status: string
  revoked?: boolean
  issuerName: string
  learnerName?: string
  achievement?: string
  issuedAt: string
  credential: Record<string, unknown>
}

export async function fetchMyCredentials(): Promise<CredentialsListResponse> {
  const res = await authorizedFetch('/api/v1/me/credentials')
  if (!res.ok) {
    throw new Error('Failed to load credentials.')
  }
  return (await res.json()) as CredentialsListResponse
}

export async function downloadCredentialPDF(id: string): Promise<Blob> {
  const res = await authorizedFetch(`/api/v1/credentials/${encodeURIComponent(id)}/download`)
  if (!res.ok) {
    throw new Error('Failed to download certificate PDF.')
  }
  return res.blob()
}

export async function verifyCredential(id: string): Promise<CredentialVerifyResponse> {
  const base = import.meta.env.VITE_API_URL ?? ''
  const res = await fetch(`${base}/api/v1/credentials/${encodeURIComponent(id)}/verify`)
  if (res.status === 404) {
    throw new Error('Credential not found.')
  }
  if (!res.ok) {
    throw new Error('Verification failed.')
  }
  return (await res.json()) as CredentialVerifyResponse
}