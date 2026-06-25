/**
 * Status page incident banner (plan 17.13)
 */
import { test, expect } from '../fixtures/test.js'

test.describe('Status page incident banner', () => {
  test('shows incident banner when status summary reports an active incident', async ({ page }) => {
    await page.route('**/api/v1/status-summary', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          pageUrl: 'https://status.lextures.io',
          status: 'minor',
          configured: true,
          incidents: [
            {
              id: 'e2e-incident-1',
              name: 'Elevated API latency',
              status: 'investigating',
              impact: 'minor',
            },
          ],
        }),
      })
    })

    await page.goto('/dashboard')
    await expect(page.getByRole('alert')).toContainText(/Elevated API latency/i)
    await expect(page.getByRole('link', { name: /view system status/i })).toHaveAttribute(
      'href',
      'https://status.lextures.io',
    )
  })

  test('status summary API is publicly accessible', async () => {
    const apiBase = process.env.E2E_API_URL ?? 'http://localhost:8080'
    const res = await fetch(`${apiBase}/api/v1/status-summary`)
    expect(res.ok).toBeTruthy()
    const body = (await res.json()) as { incidents?: unknown[]; pageUrl?: string }
    expect(Array.isArray(body.incidents)).toBeTruthy()
    expect(body.pageUrl).toBeTruthy()
  })
})