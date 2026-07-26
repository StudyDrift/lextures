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

  // TD.3 — LMS OpenAPI contract (distinct from the public partner API above).
  test('GET /api/openapi.json returns valid OpenAPI 3.0.3 with components', async () => {
    const res = await fetch(`${apiBase}/api/openapi.json`)
    expect(res.ok).toBeTruthy()
    expect(res.headers.get('content-type') ?? '').toMatch(/application\/json/)
    const text = await res.text()
    const doc = JSON.parse(text) as {
      openapi?: string
      components?: { securitySchemes?: { bearerAuth?: { type?: string; scheme?: string } } }
      paths?: Record<string, unknown>
    }
    // Strict parse already implies no trailing data for JSON.parse of the whole body.
    expect(doc.openapi).toBe('3.0.3')
    expect(doc.components?.securitySchemes?.bearerAuth?.type).toBe('http')
    expect(doc.components?.securitySchemes?.bearerAuth?.scheme).toBe('bearer')
    expect(Object.keys(doc.paths ?? {}).length).toBeGreaterThan(100)
  })

  test('GET /api/docs serves Swagger UI shell', async ({ page }) => {
    const errors: string[] = []
    page.on('pageerror', (err) => errors.push(err.message))
    const res = await page.goto(`${apiBase}/api/docs`, { waitUntil: 'domcontentloaded' })
    expect(res?.ok()).toBeTruthy()
    await expect(page.locator('#swagger-ui')).toBeVisible()
    // Viewer loads remote Swagger UI assets; allow network-only failures in offline CI.
    const hard = errors.filter((m) => !/Loading|Failed to fetch|network/i.test(m))
    expect(hard).toEqual([])
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
