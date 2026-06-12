import { authorizedFetch } from './api'

export type TranscriptRequest = {
  id: string
  status: 'queued' | 'submitted' | 'failed'
  requestedAt: string
  submittedAt?: string
  errorMessage?: string
  webhookResponseCode?: number
}

export type TranscriptsConfig = {
  webhookUrl: string
  webhookSecret: string
  hasWebhookSecret: boolean
}

export async function fetchTranscriptRequests(): Promise<TranscriptRequest[]> {
  const res = await authorizedFetch('/api/v1/transcripts/requests')
  if (!res.ok) {
    throw new Error('Could not load transcript requests.')
  }
  const data = (await res.json()) as { requests?: TranscriptRequest[] }
  return data.requests ?? []
}

export async function submitTranscriptRequest(): Promise<TranscriptRequest> {
  const res = await authorizedFetch('/api/v1/transcripts/requests', { method: 'POST' })
  if (!res.ok) {
    const msg =
      res.status === 503
        ? 'Transcript requests are not configured yet. Contact your institution.'
        : 'Could not submit transcript request.'
    throw new Error(msg)
  }
  const data = (await res.json()) as { request?: TranscriptRequest }
  if (!data.request) {
    throw new Error('Unexpected response from server.')
  }
  return data.request
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
