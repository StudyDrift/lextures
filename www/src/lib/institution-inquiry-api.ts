/** Public institution inquiry API (marketing request-information form). */

import { API_BASE } from './api-base'
import type { InstitutionInquiryForm } from './institution-inquiry-mailto'

export class InstitutionInquiryApiError extends Error {
  status: number
  constructor(status: number, message: string) {
    super(message)
    this.name = 'InstitutionInquiryApiError'
    this.status = status
  }
}

export type InstitutionInquirySubmitResult = {
  id: string
}

/** JSON body for POST /api/v1/public/institution-inquiries */
export function toInstitutionInquiryPayload(form: InstitutionInquiryForm) {
  return {
    organization_type: form.organizationType.trim(),
    organization_name: form.organizationName.trim(),
    contact_name: form.contactName.trim(),
    email: form.email.trim(),
    role: form.role.trim(),
    enrollment_size: form.enrollmentSize.trim(),
    hosting_preference: form.hostingPreference.trim(),
    message: form.message.trim(),
  }
}

export async function submitInstitutionInquiry(
  form: InstitutionInquiryForm,
  fetchImpl: typeof fetch = fetch,
): Promise<InstitutionInquirySubmitResult> {
  const res = await fetchImpl(`${API_BASE}/api/v1/public/institution-inquiries`, {
    method: 'POST',
    headers: {
      Accept: 'application/json',
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(toInstitutionInquiryPayload(form)),
  })
  const raw: unknown = await res.json().catch(() => ({}))
  if (!res.ok) {
    const msg =
      typeof raw === 'object' &&
      raw &&
      'message' in raw &&
      typeof (raw as { message: unknown }).message === 'string'
        ? (raw as { message: string }).message
        : res.status === 429
          ? 'Too many requests. Please try again later.'
          : `Request failed (${res.status})`
    throw new InstitutionInquiryApiError(res.status, msg)
  }
  const data = raw as { id?: string }
  if (!data.id?.trim()) {
    throw new InstitutionInquiryApiError(res.status, 'Unexpected response from server.')
  }
  return { id: data.id.trim() }
}
