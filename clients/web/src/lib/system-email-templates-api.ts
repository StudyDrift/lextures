import { authorizedFetch } from './api'
import { readApiErrorMessage } from './errors'

export type SystemEmailTemplateSlot = {
  id: string
  description: string
  mergeFields: Record<string, string>
  defaultHtml: string
  defaultText: string
  defaultMarkdown: string
  hasCustom: boolean
  activeId?: string
  updatedAt?: string
  replyTo?: string
  senderName?: string
  unknownFields?: string[]
}

export type SystemEmailTemplateVersion = {
  id: string
  slotId: string
  sourceMarkdown: string
  htmlBody: string
  textBody?: string
  replyTo?: string
  senderName?: string
  createdBy?: string
  createdAt: string
  isActive: boolean
}

export type SystemEmailTemplateDetail = SystemEmailTemplateSlot & {
  active?: SystemEmailTemplateVersion
}

export type SystemEmailTemplatePreview = {
  html: string
  text: string
}

export type SaveSystemEmailTemplateResult = SystemEmailTemplateVersion & {
  unknownFields?: string[]
}

async function apiJson<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await authorizedFetch(path, init)
  if (!res.ok) throw new Error(await readApiErrorMessage(res))
  if (res.status === 204) return undefined as T
  return res.json() as Promise<T>
}

const base = '/api/v1/settings/platform/email-templates'

export function listSystemEmailTemplateSlots(): Promise<SystemEmailTemplateSlot[]> {
  return apiJson(base)
}

export function getSystemEmailTemplateSlot(slotId: string): Promise<SystemEmailTemplateDetail> {
  return apiJson(`${base}/${encodeURIComponent(slotId)}`)
}

export function saveSystemEmailTemplate(
  slotId: string,
  body: { sourceMarkdown: string; textBody?: string; replyTo?: string; senderName?: string },
): Promise<SaveSystemEmailTemplateResult> {
  return apiJson(`${base}/${encodeURIComponent(slotId)}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
}

export function listSystemEmailTemplateHistory(slotId: string): Promise<SystemEmailTemplateVersion[]> {
  return apiJson(`${base}/${encodeURIComponent(slotId)}/history`)
}

export function restoreSystemEmailTemplateVersion(
  slotId: string,
  versionId: string,
): Promise<SystemEmailTemplateVersion> {
  return apiJson(`${base}/${encodeURIComponent(slotId)}/restore`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ versionId }),
  })
}

export function resetSystemEmailTemplate(slotId: string): Promise<void> {
  return apiJson(`${base}/${encodeURIComponent(slotId)}/reset`, {
    method: 'POST',
  })
}

export function sendSystemEmailTemplateTest(slotId: string): Promise<void> {
  return apiJson(`${base}/${encodeURIComponent(slotId)}/test`, {
    method: 'POST',
  })
}

export function previewSystemEmailTemplate(
  slotId: string,
  body: { sourceMarkdown?: string; textBody?: string; sampleData?: Record<string, string> },
): Promise<SystemEmailTemplatePreview> {
  return apiJson(`${base}/${encodeURIComponent(slotId)}/preview`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
}
