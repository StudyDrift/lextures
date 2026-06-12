import { authorizedFetch } from './api'
import { readApiErrorMessage } from './errors'

export type TranscriptDeliveryType = 'email' | 'mail' | 'pickup'
export type TranscriptUrgencyUnit = 'days' | 'business_days'
export type MailUrgency = 'standard' | 'rush'

export type TranscriptRequest = {
  id: string
  status: 'queued' | 'submitted' | 'failed'
  deliveryType: TranscriptDeliveryType
  deliveryEmail?: string
  deliveryAddress?: string
  urgencyDays?: number
  urgencyDaysMin?: number
  urgencyUnit?: TranscriptUrgencyUnit
  requestedAt: string
  submittedAt?: string
  errorMessage?: string
  webhookResponseCode?: number
}

export type TranscriptsConfig = {
  webhookUrl: string
  webhookSecret: string
  hasWebhookSecret: boolean
  pickupInstructions?: string
}

export type TranscriptsStudentConfig = {
  pickupInstructions?: string
  pickupAvailable: boolean
}

export type SubmitTranscriptRequestPayload = {
  deliveryType: TranscriptDeliveryType
  deliveryEmail?: string
  deliveryAddress?: string
  mailUrgency?: MailUrgency
  urgencyDays?: number
}

export async function fetchTranscriptRequests(): Promise<TranscriptRequest[]> {
  const res = await authorizedFetch('/api/v1/transcripts/requests')
  if (!res.ok) {
    throw new Error('Could not load transcript requests.')
  }
  const data = (await res.json()) as { requests?: TranscriptRequest[] }
  return data.requests ?? []
}

export async function fetchTranscriptsConfig(): Promise<TranscriptsStudentConfig> {
  const res = await authorizedFetch('/api/v1/transcripts/config')
  if (!res.ok) {
    throw new Error('Could not load transcript options.')
  }
  return (await res.json()) as TranscriptsStudentConfig
}

export async function submitTranscriptRequest(
  payload: SubmitTranscriptRequestPayload,
): Promise<TranscriptRequest> {
  const res = await authorizedFetch('/api/v1/transcripts/requests', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  })
  if (!res.ok) {
    const raw = (await res.json().catch(() => ({}))) as Record<string, unknown>
    const msg =
      res.status === 503
        ? 'Transcript requests are not configured yet. Contact your institution.'
        : readApiErrorMessage(raw) || 'Could not submit transcript request.'
    throw new Error(msg)
  }
  const data = (await res.json()) as { request?: TranscriptRequest }
  if (!data.request) {
    throw new Error('Unexpected response from server.')
  }
  return data.request
}

export async function fetchAdminTranscriptRequests(): Promise<TranscriptRequest[]> {
  const res = await authorizedFetch('/api/v1/admin/transcripts/requests')
  if (!res.ok) {
    throw new Error('Could not load transcript delivery failures.')
  }
  const data = (await res.json()) as { requests?: TranscriptRequest[] }
  return data.requests ?? []
}

export async function fetchAdminTranscriptsConfig(): Promise<TranscriptsConfig> {
  const res = await authorizedFetch('/api/v1/admin/transcripts/config')
  if (!res.ok) {
    throw new Error('Could not load transcripts configuration.')
  }
  return (await res.json()) as TranscriptsConfig
}

export async function saveAdminTranscriptsConfig(payload: {
  webhookUrl: string
  webhookSecret?: string
  pickupInstructions?: string
}): Promise<TranscriptsConfig> {
  const res = await authorizedFetch('/api/v1/admin/transcripts/config', {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  })
  if (!res.ok) {
    throw new Error('Could not save transcripts configuration.')
  }
  return (await res.json()) as TranscriptsConfig
}