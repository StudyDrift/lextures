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
