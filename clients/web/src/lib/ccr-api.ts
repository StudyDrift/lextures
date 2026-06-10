import { authorizedFetch } from './api'

export type CCRAchievement = {
  id: string
  type: string
  title: string
  description: string
  issuedAt: string
  evidenceUrl?: string
  outcomeTags?: string[]
}

export type CCRDocument = {
  id: string
  generatedAt: string
  shareable: boolean
  verificationUrl?: string
}

export type CCRSummaryResponse = {
  achievements: CCRAchievement[]
  documents: CCRDocument[]
}

export type CCRGenerateResponse = {
  document: CCRDocument
  achievements: CCRAchievement[]
  verificationUrl?: string
}

export type CCRVerifyResponse = {
  valid: boolean
  status: 'Valid' | 'Invalid'
  issuerName: string
  issuedAt: string
  credential: Record<string, unknown>
}

export async function fetchMyCCR(): Promise<CCRSummaryResponse> {
  const res = await authorizedFetch('/api/v1/me/ccr')
  if (!res.ok) {
    throw new Error('Failed to load CCR.')
  }
  return (await res.json()) as CCRSummaryResponse
}

export async function generateMyCCR(sharePublicly: boolean): Promise<CCRGenerateResponse> {
  const res = await authorizedFetch('/api/v1/me/ccr/generate', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ sharePublicly }),
  })
  if (!res.ok) {
    throw new Error('Failed to generate CCR.')
  }
  return (await res.json()) as CCRGenerateResponse
}

export async function downloadCCR(documentId: string, format: 'json' | 'pdf'): Promise<Blob> {
  const res = await authorizedFetch(`/api/v1/me/ccr/${documentId}/download?format=${format}`)
  if (!res.ok) {
    throw new Error('Failed to download CCR.')
  }
  return res.blob()
}

export async function verifyCCRShareToken(token: string): Promise<CCRVerifyResponse> {
  const base = import.meta.env.VITE_API_URL ?? ''
  const res = await fetch(`${base}/api/v1/verify/${encodeURIComponent(token)}`)
  if (res.status === 404) {
    throw new Error('Verification link not found.')
  }
  if (!res.ok) {
    throw new Error('Verification failed.')
  }
  return (await res.json()) as CCRVerifyResponse
}
