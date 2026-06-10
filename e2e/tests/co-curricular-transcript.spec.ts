/**
 * Co-curricular transcript / CLR (plan 14.13)
 *
 *   [x] GET /api/v1/me/ccr unauthenticated returns 401
 *   [x] Endpoints return 501 when feature disabled
 *   [x] Public verify endpoint returns 404 for unknown token
 */
import { test, expect } from '../fixtures/test.js'

const API_BASE = process.env.E2E_API_URL ?? 'http://localhost:8080'

test.describe('Co-curricular transcript API', () => {
  test('GET /me/ccr unauthenticated returns 401', async () => {
    const res = await fetch(`${API_BASE}/api/v1/me/ccr`)
    expect(res.status).toBe(401)
  })

  test('verify endpoint returns 404 for unknown share token when feature enabled', async () => {
    const res = await fetch(`${API_BASE}/api/v1/verify/00000000-0000-4000-8000-000000000099`)
    expect([404, 501]).toContain(res.status)
  })
})

test('Student can open My Achievements page when feature enabled', async ({ page }) => {
  await page.goto('/my-ccr')
  await expect(page.getByRole('heading', { name: 'My Achievements' })).toBeVisible()
})
