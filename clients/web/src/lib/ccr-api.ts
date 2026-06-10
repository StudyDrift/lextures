import { authorizedFetch } from './api'

export type CCRAchievement = {
  id: string
  achievementType: string
  title: string
  description?: string
  issuedAt: string
  evidenceUrl?: string
  outcomeTags?: string[]
}

export type CCRDocumentSummary = {
  id: string
  generatedAt: string
  hasShareLink: boolean
  shareToken?: string
  consentedAt?: string
}

export type MyCCRResponse = {
  achievements: CCRAchievement[]
  documents: CCRDocumentSummary[]
}

export type GenerateCCRResponse = {
  document: CCRDocumentSummary
}

export type VerifyCCRResponse = {
  valid: boolean
  status: 'Valid' | 'Invalid'
  issuerName?: string
  issuedAt?: string
  achievements: CCRAchievement[]
}

export async function fetchMyCCR(): Promise<MyCCRResponse> {
  const res = await authorizedFetch('/api/v1/me/ccr')
  if (!res.ok) {
    throw new Error('Failed to load CCR dashboard')
  }
  return res.json() as Promise<MyCCRResponse>
}

export async function generateCCR(consentToShare: boolean): Promise<GenerateCCRResponse> {
  const res = await authorizedFetch('/api/v1/me/ccr/generate', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ consentToShare }),
  })
  if (!res.ok) {
    throw new Error('Failed to generate CCR')
  }
  return res.json() as Promise<GenerateCCRResponse>
}

export async function downloadCCR(documentId: string, format: 'json' | 'pdf'): Promise<Blob> {
  const res = await authorizedFetch(`/api/v1/me/ccr/${encodeURIComponent(documentId)}/download?format=${format}`)
  if (!res.ok) {
    throw new Error(`Failed to download CCR ${format}`)
  }
  return res.blob()
}

export async function verifyCCR(token: string): Promise<VerifyCCRResponse> {
  const res = await fetch(`/api/v1/verify/${encodeURIComponent(token)}`)
  if (res.status === 404) {
    throw new Error('Verification link not found')
  }
  if (!res.ok) {
    throw new Error('Failed to verify CCR')
  }
  return res.json() as Promise<VerifyCCRResponse>
}

export async function addExtracurricularAchievement(
  studentId: string,
  body: {
    title: string
    description?: string
    issuedAt?: string
    evidenceUrl?: string
    outcomeTags?: string[]
  },
): Promise<CCRAchievement> {
  const res = await authorizedFetch(`/api/v1/admin/students/${encodeURIComponent(studentId)}/ccr/achievements`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  if (!res.ok) {
    throw new Error('Failed to add extracurricular achievement')
  }
  const data = (await res.json()) as { achievement: CCRAchievement }
  return data.achievement
}
