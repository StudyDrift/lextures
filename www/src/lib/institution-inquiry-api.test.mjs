/**
 * Unit tests for institution inquiry API payload helpers.
 */
import assert from 'node:assert/strict'
import { describe, it, mock } from 'node:test'

function toInstitutionInquiryPayload(form) {
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

class InstitutionInquiryApiError extends Error {
  constructor(status, message) {
    super(message)
    this.name = 'InstitutionInquiryApiError'
    this.status = status
  }
}

async function submitInstitutionInquiry(form, fetchImpl, apiBase = 'https://self.lextures.com') {
  const res = await fetchImpl(`${apiBase}/api/v1/public/institution-inquiries`, {
    method: 'POST',
    headers: {
      Accept: 'application/json',
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(toInstitutionInquiryPayload(form)),
  })
  const raw = await res.json().catch(() => ({}))
  if (!res.ok) {
    const msg =
      typeof raw === 'object' && raw && typeof raw.message === 'string'
        ? raw.message
        : res.status === 429
          ? 'Too many requests. Please try again later.'
          : `Request failed (${res.status})`
    throw new InstitutionInquiryApiError(res.status, msg)
  }
  if (!raw.id?.trim()) {
    throw new InstitutionInquiryApiError(res.status, 'Unexpected response from server.')
  }
  return { id: raw.id.trim() }
}

const sampleForm = {
  organizationType: ' University ',
  organizationName: ' State U ',
  contactName: ' Ada ',
  email: ' ada@example.edu ',
  role: ' CIO ',
  enrollmentSize: '1,000 – 10,000',
  hostingPreference: 'Not sure yet',
  message: ' Pilot interest ',
}

describe('institution-inquiry-api', () => {
  it('maps form fields to snake_case API payload', () => {
    assert.deepEqual(toInstitutionInquiryPayload(sampleForm), {
      organization_type: 'University',
      organization_name: 'State U',
      contact_name: 'Ada',
      email: 'ada@example.edu',
      role: 'CIO',
      enrollment_size: '1,000 – 10,000',
      hosting_preference: 'Not sure yet',
      message: 'Pilot interest',
    })
  })

  it('returns id on 201', async () => {
    const fetchImpl = mock.fn(async () => ({
      ok: true,
      status: 201,
      json: async () => ({ id: '11111111-1111-1111-1111-111111111111' }),
    }))
    const result = await submitInstitutionInquiry(sampleForm, fetchImpl)
    assert.equal(result.id, '11111111-1111-1111-1111-111111111111')
    assert.equal(fetchImpl.mock.callCount(), 1)
    const [url, init] = fetchImpl.mock.calls[0].arguments
    assert.equal(url, 'https://self.lextures.com/api/v1/public/institution-inquiries')
    assert.equal(init.method, 'POST')
  })

  it('throws InstitutionInquiryApiError on failure', async () => {
    const fetchImpl = mock.fn(async () => ({
      ok: false,
      status: 400,
      json: async () => ({ message: 'email is required.' }),
    }))
    await assert.rejects(
      () => submitInstitutionInquiry(sampleForm, fetchImpl),
      err => err instanceof InstitutionInquiryApiError && err.status === 400 && /email/.test(err.message),
    )
  })
})
