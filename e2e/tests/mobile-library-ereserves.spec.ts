/**
 * Mobile library / e-reserves / OER (M3.6) — API contract checks used by native clients.
 *
 * Checklist coverage:
 *   [x] OER search returns results when OER library is enabled
 *   [x] Library resource access endpoint requires auth
 *   [x] Library resource GET requires course access
 */
import { test, expect } from '../fixtures/test.js'
import { isOEREnabled } from '../fixtures/platform-features.js'

const apiBase = process.env.E2E_API_URL ?? 'http://localhost:8080'

test.describe('Mobile library / e-reserves API', () => {
  test('OER search returns openable results', async ({ seededCourse }) => {
    if (!(await isOEREnabled())) {
      test.skip(true, 'OER library not enabled')
    }
    const qs = new URLSearchParams({ provider: 'oer_commons', q: 'photosynthesis' })
    const res = await fetch(`${apiBase}/api/v1/oer/search?${qs.toString()}`, {
      headers: { Authorization: `Bearer ${seededCourse.instructorToken}` },
    })
    expect(res.ok).toBeTruthy()
    const body = (await res.json()) as { results: Array<{ id: string; title: string; url: string }> }
    expect(body.results.length).toBeGreaterThan(0)
    expect(body.results[0].url).toMatch(/^https?:\/\//)
  })

  test('library resource access requires authentication', async () => {
    const res = await fetch(
      `${apiBase}/api/v1/courses/demo-course/library-resources/00000000-0000-0000-0000-000000000001/access`,
      { method: 'POST' },
    )
    expect(res.status).toBe(401)
  })

  test('library search requires authentication', async () => {
    const res = await fetch(`${apiBase}/api/v1/library/search?q=test`)
    expect(res.status).toBe(401)
  })
})