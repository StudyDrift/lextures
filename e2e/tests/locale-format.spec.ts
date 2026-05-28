/**
 * Locale-aware date/number formatting (plan 11.3).
 */

import { test, expect } from '../fixtures/test.js'

test.describe('Locale-aware formatting', () => {
  test('French locale formats sample due date with day/month/year', async ({ authedPage: page }) => {
    await page.goto('/settings/account')
    await expect(page.getByTestId('locale-switcher')).toBeVisible()

    await page.getByTestId('locale-switcher').selectOption('fr')
    await expect(page.getByTestId('settings-locale-sample-date')).toBeVisible({ timeout: 15_000 })

    const sample = page.getByTestId('settings-locale-sample-date')
    await expect(sample).toHaveAttribute('datetime', /2026-04-15T10:00:00/)
    const text = await sample.innerText()
    expect(text).toMatch(/\d{1,2}[./]\d{1,2}[./]\d{2,4}/)
  })

  test('timezone sample keeps ISO datetime attribute', async ({ authedPage: page }) => {
    await page.goto('/settings/account')
    await page.getByTestId('settings-timezone-select').selectOption('Europe/Berlin')
    await expect(page.getByTestId('settings-locale-sample-date')).toBeVisible({ timeout: 15_000 })
    const sample = page.getByTestId('settings-locale-sample-date')
    await expect(sample).toHaveAttribute('datetime', /2026-04-15T10:00:00/)
  })
})
