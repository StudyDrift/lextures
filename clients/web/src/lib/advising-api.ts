import { authorizedFetch } from './api'
import { readApiErrorMessage } from './errors'

export type RequirementGroup = {
  group: string
  coursesRemaining: number
}

export type DegreeProgress = {
  configured: boolean
  completionPercent?: number
  remainingRequiredCount?: number
  remainingRequirements?: RequirementGroup[]
  atRisk?: boolean
  lastUpdated?: string
  stale?: boolean
  appointmentUrl?: string
  recentNotesCount?: number
}

export type AdvisingNote = {
  id: string
  studentId: string
  advisorId: string
  content: string
  visibleToStudent: boolean
  createdAt: string
  advisorEmail?: string
  advisorDisplayName?: string
}

export type AdvisingConfig = {
  appointmentUrl: string
  degreeAuditProvider: 'none' | 'degreeworks' | 'stellic'
  degreeAuditBaseUrl: string
  apiCredentialsRef: string
  atRiskBannerEnabled: boolean
}

export async function fetchDegreeProgress(): Promise<DegreeProgress> {
  const res = await authorizedFetch('/api/v1/me/degree-progress')
  if (!res.ok) {
    throw new Error('Could not load degree progress.')
  }
  return (await res.json()) as DegreeProgress
}

export async function fetchAdvisingNotes(): Promise<AdvisingNote[]> {
  const res = await authorizedFetch('/api/v1/me/advising-notes')
  if (!res.ok) {
    throw new Error('Could not load advising notes.')
  }
  const data = (await res.json()) as { notes?: AdvisingNote[] }
  return data.notes ?? []
}

export async function fetchAdminAdvisingConfig(): Promise<AdvisingConfig> {
  const res = await authorizedFetch('/api/v1/admin/advising/config')
  if (!res.ok) {
    throw new Error('Could not load advising settings.')
  }
  return (await res.json()) as AdvisingConfig
}

export async function saveAdminAdvisingConfig(payload: Partial<AdvisingConfig>): Promise<AdvisingConfig> {
  const res = await authorizedFetch('/api/v1/admin/advising/config', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  })
  if (!res.ok) {
    const raw = (await res.json().catch(() => ({}))) as Record<string, unknown>
    throw new Error(readApiErrorMessage(raw) || 'Could not save advising settings.')
  }
  return (await res.json()) as AdvisingConfig
}
