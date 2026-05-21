import { authorizedFetch } from './api'
import { readApiErrorMessage } from './errors'

export type ScanStatus = 'pending' | 'clean' | 'quarantined' | 'scan_error'

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

export interface FileScanStatus {
  object_id: string
  status: ScanStatus
  virus_name?: string | null
  scan_completed_at?: string | null
}

export interface QuarantineItem {
  object_id: string
  object_key: string
  virus_name?: string | null
  uploader_id?: string | null
  uploader_name?: string | null
  uploader_email?: string | null
  course_code?: string | null
  course_title?: string | null
  uploaded_at: string
}

export async function fetchFileScanStatus(objectId: string): Promise<FileScanStatus> {
  return apiJson<FileScanStatus>(`/api/v1/files/${encodeURIComponent(objectId)}/scan-status`)
}

export async function fetchQuarantineList(): Promise<QuarantineItem[]> {
  const res = await apiJson<{ items: QuarantineItem[] }>('/api/v1/admin/quarantine')
  return res.items ?? []
}

export async function releaseQuarantinedFile(objectId: string): Promise<void> {
  await apiJson(`/api/v1/admin/quarantine/${encodeURIComponent(objectId)}/release`, {
    method: 'POST',
  })
}

export async function deleteQuarantinedFile(objectId: string): Promise<void> {
  await apiJson(`/api/v1/admin/quarantine/${encodeURIComponent(objectId)}`, {
    method: 'DELETE',
  })
}
