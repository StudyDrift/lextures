/**
 * AI usage disclosure & opt-out (plan 10.17)
 */
import { test, expect } from '../fixtures/test.js'
import { apiSignup } from '../fixtures/api.js'

const API_BASE = process.env.E2E_API_URL ?? 'http://localhost:8080'
const PASSWORD = 'E2eTestPass1!'

function uniqueEmail(label = 'ai'): string {
  return `e2e-ai-disclosure-${label}-${Date.now()}-${Math.random().toString(36).slice(2)}@test.invalid`
}

test.describe('AI disclosure public API', () => {
  test('GET /api/v1/public/ai-disclosure returns model cards', async () => {
    const res = await fetch(`${API_BASE}/api/v1/public/ai-disclosure`)
    expect(res.status).toBe(200)
    const body = (await res.json()) as { models?: unknown[] }
    expect(Array.isArray(body.models)).toBe(true)
    expect((body.models ?? []).length).toBeGreaterThan(0)
  })
})

test.describe('AI opt-out blocks notebook query', () => {
  test('opted-out user receives 403 on notebook RAG', async () => {
    const { access_token } = await apiSignup({ email: uniqueEmail(), password: PASSWORD })

    const putRes = await fetch(`${API_BASE}/api/v1/settings/ai-opt-out`, {
      method: 'PUT',
      headers: {
        Authorization: `Bearer ${access_token}`,
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ aiProcessingOptOut: true }),
    })
    if (putRes.status === 404) {
      test.skip(true, 'AI disclosure module not enabled')
      return
    }
    expect(putRes.status).toBe(200)

    const queryRes = await fetch(`${API_BASE}/api/v1/me/notebooks/query`, {
      method: 'POST',
      headers: {
        Authorization: `Bearer ${access_token}`,
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ question: 'summarize', notebooks: [] }),
    })
    expect(queryRes.status).toBe(403)
    const err = (await queryRes.json()) as { error?: { code?: string; message?: string } }
    expect(err.error?.code).toBe('AI_PROCESSING_DISABLED')
    expect(err.error?.message).toMatch(/disabled/i)
  })
})

test.describe('AI opt-out settings', () => {
  test('authenticated user can read and toggle opt-out', async () => {
    const { access_token } = await apiSignup({ email: uniqueEmail('toggle'), password: PASSWORD })
    const headers = { Authorization: `Bearer ${access_token}` }

    const getRes = await fetch(`${API_BASE}/api/v1/settings/ai-opt-out`, { headers })
    if (getRes.status === 404) {
      test.skip(true, 'AI disclosure module not enabled')
      return
    }
    expect(getRes.status).toBe(200)
    const initial = (await getRes.json()) as { aiProcessingOptOut?: boolean }
    expect(typeof initial.aiProcessingOptOut).toBe('boolean')

    const putRes = await fetch(`${API_BASE}/api/v1/settings/ai-opt-out`, {
      method: 'PUT',
      headers: { ...headers, 'Content-Type': 'application/json' },
      body: JSON.stringify({ aiProcessingOptOut: !initial.aiProcessingOptOut }),
    })
    expect(putRes.status).toBe(200)
  })
})
