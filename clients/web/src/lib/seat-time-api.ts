import { authorizedFetch } from './api'

export type SeatTimeProgress = {
  totalMinutes: number
  requiredHours: number
  ceuEarned: number
  progressPct: number
  awarded: boolean
}

export type CETranscriptAward = {
  courseTitle: string
  ceuCredit: number
  contactHours: number
  completedAt: string
}

export type HeartbeatResponse = {
  minutesActive: number
  counted: boolean
  anomalyFlag: boolean
}

export async function postSeatTimeHeartbeat(
  contentItemId: string,
  sessionToken: string,
): Promise<HeartbeatResponse> {
  const res = await authorizedFetch('/api/v1/seat-time/heartbeat', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ contentItemId, sessionToken }),
  })
  if (!res.ok) {
    const msg = await res.text()
    throw new Error(msg || 'Failed to send seat-time heartbeat.')
  }
  return (await res.json()) as HeartbeatResponse
}

export async function fetchMySeatTime(courseId: string): Promise<SeatTimeProgress> {
  const res = await authorizedFetch(`/api/v1/me/seat-time?courseId=${encodeURIComponent(courseId)}`)
  if (!res.ok) {
    const msg = await res.text()
    throw new Error(msg || 'Failed to load seat-time progress.')
  }
  return (await res.json()) as SeatTimeProgress
}

export async function fetchCETranscript(): Promise<{ awards: CETranscriptAward[] }> {
  const res = await authorizedFetch('/api/v1/me/ce-transcript')
  if (!res.ok) {
    const msg = await res.text()
    throw new Error(msg || 'Failed to load CE transcript.')
  }
  return (await res.json()) as { awards: CETranscriptAward[] }
}

export async function downloadCETranscriptPdf(): Promise<Blob> {
  const res = await authorizedFetch('/api/v1/me/ce-transcript?format=pdf')
  if (!res.ok) {
    const msg = await res.text()
    throw new Error(msg || 'Failed to download CE transcript.')
  }
  return res.blob()
}
