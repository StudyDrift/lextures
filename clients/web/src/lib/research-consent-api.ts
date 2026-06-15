import { authorizedFetch } from './api'
import { readApiErrorMessage } from './errors'

export type ConsentDecision = 'granted' | 'declined' | 'withdrawn'
export type StudyStatus = 'draft' | 'active' | 'closed'

export type ConsentStudy = {
  id: string
  researcherId: string
  title: string
  irbProtocol: string
  consentText: string
  dataUseDescription: string
  targetCriteria: { courseIds?: string[] } | Record<string, unknown>
  status: StudyStatus
  createdAt: string
}

export type ConsentRate = {
  granted: number
  declined: number
  withdrawn: number
}

export type ConsentStudyWithRate = {
  study: ConsentStudy
  consentRate: ConsentRate
}

export type ConsentRecord = {
  id: string
  studyId: string
  userId: string
  decision: ConsentDecision
  ipAddress?: string
  userAgent?: string
  hmac?: string
  createdAt: string
}

export type ConsentHistoryEntry = {
  id: string
  studyId: string
  studyTitle: string
  decision: ConsentDecision
  createdAt: string
}

export type ConsentParticipant = {
  userId: string
  email: string
  displayName?: string
  consentedAt: string
}

// --- student ---

export async function fetchPendingConsentStudies(): Promise<ConsentStudy[]> {
  const res = await authorizedFetch('/api/v1/me/consent-studies')
  if (!res.ok) {
    throw new Error('Could not load consent studies.')
  }
  const data = (await res.json()) as { studies?: ConsentStudy[] }
  return data.studies ?? []
}

export async function fetchConsentHistory(): Promise<ConsentHistoryEntry[]> {
  const res = await authorizedFetch('/api/v1/me/consent-studies/history')
  if (!res.ok) {
    throw new Error('Could not load consent history.')
  }
  const data = (await res.json()) as { history?: ConsentHistoryEntry[] }
  return data.history ?? []
}

export async function respondToConsentStudy(
  studyId: string,
  decision: ConsentDecision,
): Promise<ConsentRecord> {
  const res = await authorizedFetch(`/api/v1/me/consent-studies/${studyId}/respond`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ decision }),
  })
  if (!res.ok) {
    const raw = (await res.json().catch(() => ({}))) as Record<string, unknown>
    throw new Error(readApiErrorMessage(raw) || 'Could not record your decision.')
  }
  const data = (await res.json()) as { record?: ConsentRecord }
  if (!data.record) {
    throw new Error('Unexpected response from server.')
  }
  return data.record
}

// --- researcher / admin ---

export async function fetchConsentStudies(): Promise<ConsentStudyWithRate[]> {
  const res = await authorizedFetch('/api/v1/admin/consent-studies')
  if (!res.ok) {
    throw new Error('Could not load studies.')
  }
  const data = (await res.json()) as { studies?: ConsentStudyWithRate[] }
  return data.studies ?? []
}

export type CreateStudyInput = {
  title: string
  irbProtocol: string
  consentText: string
  dataUseDescription: string
  targetCriteria?: { courseIds?: string[] }
  status?: StudyStatus
}

export async function createConsentStudy(input: CreateStudyInput): Promise<ConsentStudy> {
  const res = await authorizedFetch('/api/v1/admin/consent-studies', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(input),
  })
  if (!res.ok) {
    const raw = (await res.json().catch(() => ({}))) as Record<string, unknown>
    throw new Error(readApiErrorMessage(raw) || 'Could not create study.')
  }
  const data = (await res.json()) as { study?: ConsentStudy }
  if (!data.study) {
    throw new Error('Unexpected response from server.')
  }
  return data.study
}

export async function updateConsentStudy(
  studyId: string,
  patch: Partial<CreateStudyInput> & { status?: StudyStatus },
): Promise<ConsentStudy> {
  const res = await authorizedFetch(`/api/v1/admin/consent-studies/${studyId}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(patch),
  })
  if (!res.ok) {
    const raw = (await res.json().catch(() => ({}))) as Record<string, unknown>
    throw new Error(readApiErrorMessage(raw) || 'Could not update study.')
  }
  const data = (await res.json()) as { study?: ConsentStudy }
  if (!data.study) {
    throw new Error('Unexpected response from server.')
  }
  return data.study
}

export async function fetchConsentRecords(studyId: string): Promise<ConsentRecord[]> {
  const res = await authorizedFetch(`/api/v1/admin/consent-studies/${studyId}/records`)
  if (!res.ok) {
    throw new Error('Could not load consent records.')
  }
  const data = (await res.json()) as { records?: ConsentRecord[] }
  return data.records ?? []
}

export async function exportConsentingParticipants(
  studyId: string,
): Promise<{ participants: ConsentParticipant[]; count: number }> {
  const res = await authorizedFetch(`/api/v1/admin/consent-studies/${studyId}/export`)
  if (!res.ok) {
    throw new Error('Could not export participants.')
  }
  return (await res.json()) as { participants: ConsentParticipant[]; count: number }
}
