import { authorizedFetch } from './api'
import { readApiErrorMessage } from './errors'

export type EmailTemplateSlot = {
  id: string
  description: string
  mergeFields: Record<string, string>
  defaultHtml: string
  defaultText: string
  hasCustom: boolean
  activeId?: string
  updatedAt?: string
  replyTo?: string
  senderName?: string
  unknownFields?: string[]
}

export type EmailTemplateVersion = {
  id: string
  orgId: string
  slotId: string
  htmlBody: string
  textBody?: string
  replyTo?: string
  senderName?: string
  createdBy?: string
  createdAt: string
  isActive: boolean
}

export type EmailTemplateDetail = EmailTemplateSlot & {
  active?: EmailTemplateVersion
}

export type EmailTemplatePreview = {
  html: string
  text: string
}

export type SaveEmailTemplateResult = EmailTemplateVersion & {
  unknownFields?: string[]
}

async function apiJson<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await authorizedFetch(path, init)
  if (!res.ok) throw new Error(await readApiErrorMessage(res))
  if (res.status === 204) return undefined as T
  return res.json() as Promise<T>
}

function withOrg(path: string, orgId: string): string {
  const sep = path.includes('?') ? '&' : '?'
  return `${path}${sep}orgId=${encodeURIComponent(orgId)}`
}

export function listEmailTemplateSlots(orgId: string): Promise<EmailTemplateSlot[]> {
  return apiJson(withOrg('/api/v1/admin-console/email-templates', orgId))
}

export function getEmailTemplateSlot(orgId: string, slotId: string): Promise<EmailTemplateDetail> {
  return apiJson(withOrg(`/api/v1/admin-console/email-templates/${encodeURIComponent(slotId)}`, orgId))
}

export function saveEmailTemplate(
  orgId: string,
  slotId: string,
  body: { htmlBody: string; textBody?: string; replyTo?: string; senderName?: string },
): Promise<SaveEmailTemplateResult> {
  return apiJson(withOrg(`/api/v1/admin-console/email-templates/${encodeURIComponent(slotId)}`, orgId), {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
}

export function listEmailTemplateHistory(orgId: string, slotId: string): Promise<EmailTemplateVersion[]> {
  return apiJson(withOrg(`/api/v1/admin-console/email-templates/${encodeURIComponent(slotId)}/history`, orgId))
}

export function restoreEmailTemplateVersion(
  orgId: string,
  slotId: string,
  versionId: string,
): Promise<EmailTemplateVersion> {
  return apiJson(withOrg(`/api/v1/admin-console/email-templates/${encodeURIComponent(slotId)}/restore`, orgId), {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ versionId }),
  })
}

export function resetEmailTemplate(orgId: string, slotId: string): Promise<void> {
  return apiJson(withOrg(`/api/v1/admin-console/email-templates/${encodeURIComponent(slotId)}/reset`, orgId), {
    method: 'POST',
  })
}

export function sendEmailTemplateTest(orgId: string, slotId: string): Promise<void> {
  return apiJson(withOrg(`/api/v1/admin-console/email-templates/${encodeURIComponent(slotId)}/test`, orgId), {
    method: 'POST',
  })
}

export function previewEmailTemplate(
  orgId: string,
  slotId: string,
  body: { htmlBody?: string; textBody?: string; sampleData?: Record<string, string> },
): Promise<EmailTemplatePreview> {
  return apiJson(withOrg(`/api/v1/admin-console/email-templates/${encodeURIComponent(slotId)}/preview`, orgId), {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
}
