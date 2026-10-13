/**
 * Locale-aware date/number formatting (plan 11.3).
 */

import { test, expect } from '../fixtures/test.js'

test.describe('Locale-aware formatting', () => {
  test('French locale formats the date preview day-first', async ({ authedPage: page }) => {
    await page.goto('/settings/account')
    await expect(page.getByTestId('locale-switcher')).toBeVisible()

    const sample = page.getByTestId('settings-locale-sample-date')
    await expect(sample).toBeVisible()

    await page.getByTestId('locale-switcher').selectOption('fr')
    // The preview shows the current time (#791). fr-FR medium dateStyle puts the day first
    // with an abbreviated month name (e.g. "13 oct. 2026"), unlike en-US ("Oct 13, 2026").
    await expect(sample).toHaveAttribute('datetime', /^\d{4}-\d{2}-\d{2}T/, { timeout: 15_000 })
    await expect(sample).toHaveText(/^\d{1,2}\s\p{L}+\.?\s\d{4}/u, { timeout: 15_000 })
  })

  test('timezone sample keeps ISO datetime attribute', async ({ authedPage: page }) => {
    await page.goto('/settings/account')
    await page.getByTestId('settings-timezone-select').selectOption('Europe/Berlin')
    await expect(page.getByTestId('settings-locale-sample-date')).toBeVisible({ timeout: 15_000 })
    const sample = page.getByTestId('settings-locale-sample-date')
    await expect(sample).toHaveAttribute('datetime', /^\d{4}-\d{2}-\d{2}T/)
  })
})
