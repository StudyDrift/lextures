import { test, expect } from '@playwright/test'

const API_BASE = process.env.E2E_API_URL ?? 'http://localhost:8080'

test.describe('Public REST API', () => {
  test('openapi.json is valid OpenAPI 3.1', async ({ request }) => {
    const res = await request.get(`${API_BASE}/api/v1/openapi.json`)
    expect(res.ok()).toBeTruthy()
    const doc = (await res.json()) as { openapi?: string; info?: { title?: string } }
    expect(doc.openapi).toBe('3.1.0')
    expect(doc.info?.title).toContain('Lextures')
  })

  test('unauthenticated courses returns 401 problem+json when public API enabled', async ({ request }) => {
    const featuresRes = await request.get(`${API_BASE}/api/v1/platform/features`)
    if (!featuresRes.ok()) {
      test.skip(true, 'platform features unavailable')
    }
    const features = (await featuresRes.json()) as { ffPublicApi?: boolean }
    if (!features.ffPublicApi) {
      test.skip(true, 'ffPublicApi is false on the API')
    }
    const res = await request.get(`${API_BASE}/api/v1/courses`)
    expect(res.status()).toBe(401)
    expect(res.headers()['content-type'] ?? '').toContain('application/problem+json')
  })
})
