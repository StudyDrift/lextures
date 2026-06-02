import { authorizedFetch } from './api'
import { readApiErrorMessage } from './errors'

export type FileFolder = {
  id: string
  courseId: string
  parentId: string | null
  name: string
  createdBy: string | null
  createdAt: string
  updatedAt: string
}

export type FileItem = {
  id: string
  courseId: string
  folderId: string | null
  storageKey: string
  originalFilename: string
  displayName: string
  mimeType: string
  byteSize: number
  uploadedBy: string | null
  createdAt: string
  updatedAt: string
}

export type FolderContents = {
  folderId: string | null
  folders: FileFolder[]
  files: FileItem[]
}

export type UploadInitResponse = {
  objectKey: string
  presignedPutUrl?: string
  expiresAt?: string
  courseId: string
  folderId: string | null
}

async function parseJson(res: Response): Promise<unknown> {
  try {
    return await res.json()
  } catch {
    return null
  }
}

export async function listCourseFiles(courseCode: string, folderId?: string): Promise<FolderContents> {
  const url = folderId
    ? `/api/v1/courses/${encodeURIComponent(courseCode)}/files/folders/${encodeURIComponent(folderId)}`
    : `/api/v1/courses/${encodeURIComponent(courseCode)}/files`
  const res = await authorizedFetch(url)
  const raw = await parseJson(res)
  if (!res.ok) throw new Error(readApiErrorMessage(raw))
  return raw as FolderContents
}

export async function createFolder(
  courseCode: string,
  name: string,
  parentId?: string | null,
): Promise<FileFolder> {
  const res = await authorizedFetch(
    `/api/v1/courses/${encodeURIComponent(courseCode)}/files/folders`,
    {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name, parentId: parentId ?? null }),
    },
  )
  const raw = await parseJson(res)
  if (!res.ok) throw new Error(readApiErrorMessage(raw))
  return raw as FileFolder
}

export async function renameFolder(
  courseCode: string,
  folderId: string,
  name: string,
): Promise<FileFolder> {
  const res = await authorizedFetch(
    `/api/v1/courses/${encodeURIComponent(courseCode)}/files/folders/${encodeURIComponent(folderId)}`,
    {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name }),
    },
  )
  const raw = await parseJson(res)
  if (!res.ok) throw new Error(readApiErrorMessage(raw))
  return raw as FileFolder
}

export async function deleteFolder(courseCode: string, folderId: string): Promise<void> {
  const res = await authorizedFetch(
    `/api/v1/courses/${encodeURIComponent(courseCode)}/files/folders/${encodeURIComponent(folderId)}`,
    { method: 'DELETE' },
  )
  if (!res.ok) {
    const raw = await parseJson(res)
    throw new Error(readApiErrorMessage(raw))
  }
}

/**
 * Upload a file. For S3-backed servers this returns a presigned URL; the caller
 * must PUT the file to that URL and then call confirmFileUpload. For local storage
 * the server accepts the body directly and returns the registered FileItem.
 */
export async function initiateFileUpload(
  courseCode: string,
  file: File,
  folderId?: string | null,
): Promise<{ item: FileItem } | { presigned: UploadInitResponse; file: File }> {
  const params = new URLSearchParams({ filename: file.name })
  if (folderId) params.set('folderId', folderId)

  const res = await authorizedFetch(
    `/api/v1/courses/${encodeURIComponent(courseCode)}/files/items?${params.toString()}`,
    {
      method: 'POST',
      headers: { 'Content-Type': file.type || 'application/octet-stream' },
      body: file,
    },
  )
  const raw = await parseJson(res)
  if (!res.ok) throw new Error(readApiErrorMessage(raw))
  const data = raw as UploadInitResponse & Partial<FileItem>
  // If the server returned a presigned URL, the client needs to upload there
  if ('presignedPutUrl' in data && data.presignedPutUrl) {
    return { presigned: data as UploadInitResponse, file }
  }
  return { item: data as FileItem }
}

export async function uploadToPresignedUrl(presignedPutUrl: string, file: File): Promise<void> {
  const res = await fetch(presignedPutUrl, {
    method: 'PUT',
    headers: { 'Content-Type': file.type || 'application/octet-stream' },
    body: file,
  })
  if (!res.ok) throw new Error(`Upload to storage failed (HTTP ${res.status})`)
}

export async function confirmFileUpload(
  courseCode: string,
  objectKey: string,
  filename: string,
  mimeType: string,
  byteSize: number,
  folderId?: string | null,
): Promise<FileItem> {
  const res = await authorizedFetch(
    `/api/v1/courses/${encodeURIComponent(courseCode)}/files/items/confirm`,
    {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ objectKey, filename, mimeType, byteSize, folderId: folderId ?? null }),
    },
  )
  const raw = await parseJson(res)
  if (!res.ok) throw new Error(readApiErrorMessage(raw))
  return raw as FileItem
}

/** Rename a file (changes displayName). */
export async function renameFile(
  courseCode: string,
  itemId: string,
  displayName: string,
): Promise<FileItem> {
  const res = await authorizedFetch(
    `/api/v1/courses/${encodeURIComponent(courseCode)}/files/items/${encodeURIComponent(itemId)}`,
    {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ displayName }),
    },
  )
  const raw = await parseJson(res)
  if (!res.ok) throw new Error(readApiErrorMessage(raw))
  return raw as FileItem
}

/** Move a file to a different folder. Pass null/empty to move to root. */
export async function moveFile(
  courseCode: string,
  itemId: string,
  folderId: string | null,
): Promise<FileItem> {
  const res = await authorizedFetch(
    `/api/v1/courses/${encodeURIComponent(courseCode)}/files/items/${encodeURIComponent(itemId)}`,
    {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ folderId: folderId ?? '' }),
    },
  )
  const raw = await parseJson(res)
  if (!res.ok) throw new Error(readApiErrorMessage(raw))
  return raw as FileItem
}

export async function deleteFile(courseCode: string, itemId: string): Promise<void> {
  const res = await authorizedFetch(
    `/api/v1/courses/${encodeURIComponent(courseCode)}/files/items/${encodeURIComponent(itemId)}`,
    { method: 'DELETE' },
  )
  if (!res.ok) {
    const raw = await parseJson(res)
    throw new Error(readApiErrorMessage(raw))
  }
}

export function getFileContentUrl(courseCode: string, itemId: string): string {
  return `/api/v1/courses/${encodeURIComponent(courseCode)}/files/items/${encodeURIComponent(itemId)}/content`
}

export function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(1))} ${sizes[i]}`
}
