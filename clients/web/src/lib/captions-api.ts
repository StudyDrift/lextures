import { authorizedFetch } from './api'
import { readApiErrorMessage } from './errors'

async function apiJson<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await authorizedFetch(path, init)
  if (!res.ok) {
    throw new Error(await readApiErrorMessage(res))
  }
  if (res.status === 204) {
    return undefined as T
  }
  return res.json() as Promise<T>
}

export type CaptionRecord = {
  id: string
  storage_object_id: string
  lang: string
  status: string
  has_low_confidence: boolean
  confidence_avg?: number
  backend: string
  created_at: string
  reviewed_at?: string
}

export type CaptionCoverageRow = {
  object_id: string
  object_key: string
  mime_type: string
  caption_id?: string
  caption_status?: string
  reviewed_at?: string
}

export async function listCaptions(objectId: string): Promise<CaptionRecord[]> {
  return apiJson<CaptionRecord[]>(`/api/v1/files/${encodeURIComponent(objectId)}/captions`)
}

export function captionVttUrl(objectId: string, captionId: string): string {
  return `/api/v1/files/${encodeURIComponent(objectId)}/captions/${encodeURIComponent(captionId)}/vtt`
}

export async function patchCaptionVtt(
  objectId: string,
  captionId: string,
  vttContent: string,
): Promise<CaptionRecord> {
  return apiJson<CaptionRecord>(
    `/api/v1/files/${encodeURIComponent(objectId)}/captions/${encodeURIComponent(captionId)}`,
    {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ vtt_content: vttContent }),
    },
  )
}

export async function updateCaptionTranscript(
  objectId: string,
  captionId: string,
  transcriptText: string,
): Promise<CaptionRecord> {
  return apiJson<CaptionRecord>(
    `/api/v1/files/${encodeURIComponent(objectId)}/captions/${encodeURIComponent(captionId)}`,
    {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ transcript_text: transcriptText }),
    },
  )
}

export async function importCaptionFile(objectId: string, file: File): Promise<CaptionRecord> {
  const form = new FormData()
  form.append('file', file)
  const res = await authorizedFetch(
    `/api/v1/files/${encodeURIComponent(objectId)}/captions/import`,
    { method: 'POST', body: form },
  )
  if (!res.ok) {
    const text = await res.text()
    throw new Error(text || `Import failed (${res.status})`)
  }
  return (await res.json()) as CaptionRecord
}

export async function fetchCaptionCompliance(): Promise<{ rows: CaptionCoverageRow[] }> {
  return apiJson<{ rows: CaptionCoverageRow[] }>('/api/v1/admin/captions/compliance')
}

export async function patchCourseCaptionPolicy(
  courseCode: string,
  requireCaptions: boolean,
): Promise<void> {
  await apiJson(`/api/v1/courses/${encodeURIComponent(courseCode)}/caption-policy`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ requireCaptions }),
  })
}

export { videoCaptionsFeatureEnabled as isVideoCaptionsEnabled } from './platform-features.js'
