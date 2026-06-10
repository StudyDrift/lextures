// ePortfolio / capstone artifact collection API client (plan 14.12).
import { apiUrl, authorizedFetch } from './api'

export type Portfolio = {
  id: string
  title: string
  introText: string
  isPublic: boolean
  publicSlug: string | null
  order: string[]
  createdAt: string
  updatedAt: string
}

export type ArtifactType = 'submission' | 'upload' | 'text_page' | 'url'

export type Artifact = {
  id: string
  portfolioId: string
  artifactType: ArtifactType
  title: string
  description: string
  sourceSubmissionId: string | null
  sourceCourseId: string | null
  fileName: string
  fileMime: string
  textContent: string
  externalUrl: string
  outcomeIds: string[]
  isPublic: boolean
  sortOrder: number
  createdAt: string
  updatedAt: string
}

export type Evaluation = {
  id: string
  artifactId: string
  reviewerId: string
  reviewer: string
  rubric: unknown
  scores: Record<string, number>
  totalScore: number | null
  feedback: string
  updatedAt: string
}

export type PortfolioDetail = {
  portfolio: Portfolio
  artifacts: Artifact[]
  evaluations: Evaluation[]
}

export type PublicPortfolio = {
  title: string
  introText: string
  ownerName: string
  artifacts: Artifact[]
  viewCount: number
}

export type OutcomeCoverage = {
  outcomeId: string
  title: string
  studentCount: number
  artifactCount: number
  submissionRate: number
}

async function jsonOrThrow<T>(res: Response): Promise<T> {
  if (!res.ok) {
    const body = (await res.json().catch(() => ({}))) as { message?: string }
    throw new Error(body.message || `Request failed (${res.status})`)
  }
  return (await res.json()) as T
}

export async function listMyPortfolios(): Promise<Portfolio[]> {
  const res = await authorizedFetch('/api/v1/me/portfolios')
  const data = await jsonOrThrow<{ portfolios: Portfolio[] }>(res)
  return data.portfolios ?? []
}

export async function createPortfolio(title: string, introText = ''): Promise<Portfolio> {
  const res = await authorizedFetch('/api/v1/me/portfolios', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ title, introText }),
  })
  return jsonOrThrow<Portfolio>(res)
}

export async function getMyPortfolio(pid: string): Promise<PortfolioDetail> {
  const res = await authorizedFetch(`/api/v1/me/portfolios/${encodeURIComponent(pid)}`)
  return jsonOrThrow<PortfolioDetail>(res)
}

export type PatchPortfolioPayload = {
  title?: string
  introText?: string
  isPublic?: boolean
  order?: string[]
}

export async function patchPortfolio(pid: string, payload: PatchPortfolioPayload): Promise<Portfolio> {
  const res = await authorizedFetch(`/api/v1/me/portfolios/${encodeURIComponent(pid)}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  })
  return jsonOrThrow<Portfolio>(res)
}

export async function deletePortfolio(pid: string): Promise<void> {
  const res = await authorizedFetch(`/api/v1/me/portfolios/${encodeURIComponent(pid)}`, {
    method: 'DELETE',
  })
  if (!res.ok && res.status !== 204) throw new Error(`Failed to delete (${res.status})`)
}

export type CreateArtifactPayload = {
  artifactType: ArtifactType
  title: string
  description?: string
  sourceSubmissionId?: string
  textContent?: string
  externalUrl?: string
  outcomeIds?: string[]
  isPublic?: boolean
}

export async function createArtifact(pid: string, payload: CreateArtifactPayload): Promise<Artifact> {
  const res = await authorizedFetch(`/api/v1/me/portfolios/${encodeURIComponent(pid)}/artifacts`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  })
  return jsonOrThrow<Artifact>(res)
}

export type PatchArtifactPayload = {
  title?: string
  description?: string
  textContent?: string
  externalUrl?: string
  outcomeIds?: string[]
  isPublic?: boolean
}

export async function patchArtifact(pid: string, aid: string, payload: PatchArtifactPayload): Promise<Artifact> {
  const res = await authorizedFetch(
    `/api/v1/me/portfolios/${encodeURIComponent(pid)}/artifacts/${encodeURIComponent(aid)}`,
    {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    },
  )
  return jsonOrThrow<Artifact>(res)
}

export async function deleteArtifact(pid: string, aid: string): Promise<void> {
  const res = await authorizedFetch(
    `/api/v1/me/portfolios/${encodeURIComponent(pid)}/artifacts/${encodeURIComponent(aid)}`,
    { method: 'DELETE' },
  )
  if (!res.ok && res.status !== 204) throw new Error(`Failed to delete (${res.status})`)
}

export type EvaluatePayload = {
  rubric?: unknown
  scores?: Record<string, number>
  totalScore?: number | null
  feedback?: string
}

export async function evaluateArtifact(pid: string, aid: string, payload: EvaluatePayload): Promise<Evaluation> {
  const res = await authorizedFetch(
    `/api/v1/portfolios/${encodeURIComponent(pid)}/artifacts/${encodeURIComponent(aid)}/evaluate`,
    {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    },
  )
  return jsonOrThrow<Evaluation>(res)
}

export async function getPortfolioOutcomesReport(
  programId: string,
): Promise<{ cohortSize: number; outcomes: OutcomeCoverage[] }> {
  const res = await authorizedFetch(
    `/api/v1/admin/programs/${encodeURIComponent(programId)}/portfolio-outcomes-report`,
  )
  return jsonOrThrow<{ cohortSize: number; outcomes: OutcomeCoverage[] }>(res)
}

/** Public, unauthenticated read of a shared portfolio. Returns null when not found. */
export async function getPublicPortfolio(slug: string): Promise<PublicPortfolio | null> {
  const res = await fetch(apiUrl(`/api/v1/portfolios/${encodeURIComponent(slug)}`))
  if (res.status === 404) return null
  if (!res.ok) throw new Error(`Failed to load portfolio (${res.status})`)
  return (await res.json()) as PublicPortfolio
}
