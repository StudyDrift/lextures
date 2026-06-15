/**
 * LH.3 — Global dashboard dark mode accessibility regression checks.
 *
 * Validates the committed Lighthouse baseline and dashboard markup regressions
 * for weighted audits: color-contrast, button-name, link-name, heading-order.
 */
import { test, expect } from '@playwright/test'

import {
  assertAccessibilityScore,
  loadCommittedDashboardReport,
  WEIGHTED_A11Y_AUDIT_IDS,
} from '../lib/lighthouse-a11y.js'
import { seedLighthouseDashboardUser, waitForDashboardReady } from '../lib/lighthouse-harness.js'
import { injectToken } from '../fixtures/test.js'

test.describe('Lighthouse dashboard accessibility — committed baseline', () => {
  test('accessibility score >= 0.95 (AC-1)', () => {
    const report = loadCommittedDashboardReport()
    const summary = assertAccessibilityScore(report, 0.95)
    expect(summary.score).toBeGreaterThanOrEqual(0.95)
  })

  test('weighted audits pass in committed baseline', () => {
    const report = loadCommittedDashboardReport()
    const summary = assertAccessibilityScore(report)

    for (const auditId of WEIGHTED_A11Y_AUDIT_IDS) {
      const audit = report.audits[auditId]
      expect(audit, `audit ${auditId} should exist`).toBeDefined()
      expect(
        audit?.score,
        `${auditId} should pass in docs/lighthouse/global-dashboard-darkmode.json`,
      ).toBeGreaterThanOrEqual(1)
    }

    expect(summary.failureCount, 'no weighted accessibility audit failures').toBe(0)
  })

  test('color-contrast audit has no failing elements (FR-7)', () => {
    const report = loadCommittedDashboardReport()
    const audit = report.audits['color-contrast']
    expect(audit?.score).toBe(1)

    const items =
      audit?.details && 'items' in audit.details ? audit.details.items : undefined
    expect(Array.isArray(items) ? items.length : 0).toBe(0)
  })
})

test.describe('Dashboard accessibility — markup regression', () => {
  let token: string

  test.beforeAll(async () => {
    ;({ token } = await seedLighthouseDashboardUser('dark'))
  })

  test.beforeEach(async ({ page }) => {
    await injectToken(page, token)
    await page.goto('/')
    await waitForDashboardReady(page)
    await page.evaluate(() => document.documentElement.classList.add('dark'))
  })

  test('student overview collapse toggle exposes Expand/Collapse Learning (AC-3)', async ({
    page,
  }) => {
    const toggle = page.getByRole('button', { name: /^(Expand|Collapse) Learning$/ })
    await expect(toggle).toBeVisible({ timeout: 15_000 })

    const label = await toggle.getAttribute('aria-label')
    expect(label === 'Collapse Learning' || label === 'Expand Learning').toBe(true)

    await toggle.click()
    await expect(page.getByRole('button', { name: /^(Expand|Collapse) Learning$/ })).toHaveAttribute(
      'aria-label',
      label === 'Collapse Learning' ? 'Expand Learning' : 'Collapse Learning',
    )
  })

  test('heading outline does not skip levels between h1 and section headings (AC-6)', async ({
    page,
  }) => {
    const levels = await page.evaluate(() => {
      const headings = Array.from(document.querySelectorAll('h1, h2, h3, h4, h5, h6'))
      return headings
        .filter((el) => {
          const style = window.getComputedStyle(el)
          return style.display !== 'none' && style.visibility !== 'hidden'
        })
        .map((el) => Number.parseInt(el.tagName.slice(1), 10))
    })

    expect(levels.length, 'dashboard should expose heading structure').toBeGreaterThan(0)
    expect(levels[0], 'first visible heading should be h1').toBe(1)

    for (let i = 1; i < levels.length; i++) {
      const jump = levels[i] - levels[i - 1]
      expect(
        jump,
        `heading level skip: h${levels[i - 1]} followed by h${levels[i]}`,
      ).toBeLessThanOrEqual(1)
    }
  })
})
