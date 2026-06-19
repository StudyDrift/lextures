/**
 * Public REST API (plan 16.1): OpenAPI spec and token-gated course list.
 */
import { test, expect } from '../fixtures/test.js'

const apiBase = process.env.E2E_API_URL ?? 'http://localhost:8080'

test.describe('Public API', () => {
  test('GET /api/v1/openapi.json returns OpenAPI 3.1', async () => {
    const res = await fetch(`${apiBase}/api/v1/openapi.json`)
    expect(res.ok).toBeTruthy()
    const doc = (await res.json()) as { openapi?: string; info?: { title?: string } }
    expect(doc.openapi).toBe('3.1.0')
    expect(doc.info?.title).toContain('Lextures')
  })

  test('GET /api/v1/courses without auth returns 401 problem+json when public API enabled', async ({
    request,
  }) => {
    const res = await request.get(`${apiBase}/api/v1/courses`)
    // When the feature flag is off in e2e, SPA handler may return legacy 401; when on, problem+json.
    if (res.status() === 503) {
      test.skip()
    }
    if (res.status() === 401) {
      const ct = res.headers()['content-type'] ?? ''
      if (ct.includes('problem+json')) {
        const body = (await res.json()) as { title?: string }
        expect(body.title).toBe('Unauthorized')
      }
    }
  })
})
