/**
 * LH.1 — Lighthouse harness prerequisites and smoke checks.
 */
import { test, expect } from '@playwright/test'
import { chromium } from '@playwright/test'

import {
  assertLocalhostOrigin,
  registerAuthThemeInitScript,
  seedLighthouseDashboardUser,
  waitForDashboardReady,
  assertDocumentTheme,
} from '../lib/lighthouse-harness.js'
import { injectToken } from '../fixtures/test.js'

test.describe('Lighthouse harness helpers', () => {
  test('assertLocalhostOrigin rejects non-localhost URLs', () => {
    expect(() => assertLocalhostOrigin('https://app.example.com/')).toThrow(/localhost/)
    expect(() => assertLocalhostOrigin('http://localhost:5173/')).not.toThrow()
    expect(() => assertLocalhostOrigin('http://127.0.0.1:5173/')).not.toThrow()
  })

  test('assertLocalhostOrigin allows override', () => {
    expect(() =>
      assertLocalhostOrigin('https://staging.example.com/', true),
    ).not.toThrow()
  })
})

test.describe('Lighthouse harness — authenticated dashboard', () => {
  let token: string

  test.beforeAll(async () => {
    ;({ token } = await seedLighthouseDashboardUser('dark'))
  })

  test('dashboard ready signal: main nav visible, loading skeleton gone', async ({ page }) => {
    await injectToken(page, token)
    await page.goto('/')
    await waitForDashboardReady(page)
    await expect(page.getByRole('navigation', { name: 'Main' })).toBeVisible()
  })

  test('dark theme applied via init script before paint (AC-2)', async () => {
    const browser = await chromium.launch({ channel: process.env.LH_BROWSER_CHANNEL ?? undefined })
    try {
      const context = await browser.newContext()
      await registerAuthThemeInitScript(context, token, 'dark')
      const page = await context.newPage()
      await page.goto('/')
      await waitForDashboardReady(page)
      await assertDocumentTheme(page, 'dark')
      await context.close()
    } finally {
      await browser.close()
    }
  })
})

test.describe('Lighthouse harness — auth guard', () => {
  test('runLighthouseDashboard fails fast without auth when LH_REQUIRE_AUTH=1', async () => {
    const { runLighthouseDashboard } = await import('../lib/lighthouse-harness.js')
    await expect(
      runLighthouseDashboard({
        pageUrl: process.env.E2E_BASE_URL ?? 'http://localhost:5173/',
        theme: 'dark',
        outputPath: '/tmp/lh-auth-test.json',
        requireAuth: true,
      }),
    ).rejects.toThrow(/auth required/)
  })
})
