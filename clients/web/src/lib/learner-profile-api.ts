import { authorizedFetch } from './api'
import { readApiErrorMessage } from './errors'

export type LearnerProfileStatus = 'active' | 'insufficient_data' | 'paused'

export type FacetState = 'ok' | 'insufficient_data' | 'error'

export type FacetKey =
  | 'study_rhythm'
  | 'content_modality'
  | 'strengths_growth'
  | 'interests'
  | 'learning_approach'

export const FACET_PRIORITY: FacetKey[] = [
  'study_rhythm',
  'content_modality',
  'strengths_growth',
  'interests',
  'learning_approach',
]

export type FacetSummary = {
  facetKey: FacetKey
  state: FacetState
  summary: Record<string, unknown>
  confidence: number
  computedVersion: number
  updatedAt: string
}

export type LearnerProfile = {
  status: LearnerProfileStatus
  lastComputedAt?: string
  facets: FacetSummary[]
}

export type EvidenceRow = {
  sourceKind: string
  sourceTable: string
  observationCount: number
  courseId?: string
  windowStart?: string
  windowEnd?: string
  contribution?: number
}

export type Insight = {
  insightKey: string
  label: string
  value: Record<string, unknown>
  confidence: number
  salience: number
  evidence?: EvidenceRow[]
}

export type FacetDetail = {
  facet: FacetSummary
  insights: Insight[]
}

export type FacetEvidenceMap = Record<string, EvidenceRow[]>

export async function fetchLearnerProfile(): Promise<LearnerProfile> {
  const res = await authorizedFetch('/api/v1/me/learner-profile')
  const raw: unknown = await res.json().catch(() => ({}))
  if (!res.ok) {
    throw new Error(readApiErrorMessage(raw))
  }
  const data = raw as { profile?: LearnerProfile }
  return data.profile ?? { status: 'insufficient_data', facets: [] }
}

export async function fetchLearnerProfileFacet(facetKey: FacetKey): Promise<FacetDetail | null> {
  const res = await authorizedFetch(
    `/api/v1/me/learner-profile/facets/${encodeURIComponent(facetKey)}`,
  )
  const raw: unknown = await res.json().catch(() => ({}))
  if (res.status === 404) return null
  if (!res.ok) {
    throw new Error(readApiErrorMessage(raw))
  }
  const data = raw as FacetDetail
  return data
}

export async function fetchLearnerProfileFacetEvidence(
  facetKey: FacetKey,
): Promise<FacetEvidenceMap> {
  const res = await authorizedFetch(
    `/api/v1/me/learner-profile/facets/${encodeURIComponent(facetKey)}/evidence`,
  )
  const raw: unknown = await res.json().catch(() => ({}))
  if (!res.ok) {
    throw new Error(readApiErrorMessage(raw))
  }
  return (raw as FacetEvidenceMap) ?? {}
}

export function sortFacetsByPriority(facets: FacetSummary[]): FacetSummary[] {
  const order = new Map(FACET_PRIORITY.map((key, index) => [key, index]))
  return [...facets].sort((a, b) => {
    const ai = order.get(a.facetKey) ?? 999
    const bi = order.get(b.facetKey) ?? 999
    return ai - bi
  })
}

export function uniqueCourseCount(evidence: EvidenceRow[]): number {
  const ids = new Set<string>()
  for (const row of evidence) {
    if (row.courseId) ids.add(row.courseId)
  }
  return ids.size
}

export function totalObservationCount(evidence: EvidenceRow[]): number {
  let total = 0
  for (const row of evidence) {
    total += row.observationCount
  }
  return total
}

export type LearnerProfileControlStatus = 'paused' | 'active' | 'reset'

export async function pauseLearnerProfile(): Promise<LearnerProfileControlStatus> {
  const res = await authorizedFetch('/api/v1/me/learner-profile/pause', { method: 'POST' })
  const raw: unknown = await res.json().catch(() => ({}))
  if (!res.ok) {
    throw new Error(readApiErrorMessage(raw))
  }
  return (raw as { status?: LearnerProfileControlStatus }).status ?? 'paused'
}

export async function resumeLearnerProfile(): Promise<LearnerProfileControlStatus> {
  const res = await authorizedFetch('/api/v1/me/learner-profile/resume', { method: 'POST' })
  const raw: unknown = await res.json().catch(() => ({}))
  if (!res.ok) {
    throw new Error(readApiErrorMessage(raw))
  }
  return (raw as { status?: LearnerProfileControlStatus }).status ?? 'active'
}

export async function resetLearnerProfile(): Promise<LearnerProfileControlStatus> {
  const res = await authorizedFetch('/api/v1/me/learner-profile/reset', { method: 'POST' })
  const raw: unknown = await res.json().catch(() => ({}))
  if (!res.ok) {
    throw new Error(readApiErrorMessage(raw))
  }
  return (raw as { status?: LearnerProfileControlStatus }).status ?? 'reset'
}

export async function downloadLearnerProfileExport(): Promise<void> {
  const res = await authorizedFetch('/api/v1/me/learner-profile/export')
  const raw: unknown = await res.json().catch(() => ({}))
  if (!res.ok) {
    throw new Error(readApiErrorMessage(raw))
  }
  const blob = new Blob([JSON.stringify(raw, null, 2)], { type: 'application/json' })
  const url = URL.createObjectURL(blob)
  const anchor = document.createElement('a')
  anchor.href = url
  anchor.download = 'learner-profile-export.json'
  anchor.click()
  URL.revokeObjectURL(url)
}

export function dominantSourceKind(evidence: EvidenceRow[]): string | null {
  if (evidence.length === 0) return null
  const counts = new Map<string, number>()
  for (const row of evidence) {
    counts.set(row.sourceKind, (counts.get(row.sourceKind) ?? 0) + row.observationCount)
  }
  let best: string | null = null
  let bestCount = -1
  for (const [kind, count] of counts) {
    if (count > bestCount) {
      best = kind
      bestCount = count
    }
  }
  return best
}