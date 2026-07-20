/**
 * AN.5 — Overlay & surface motion.
 *
 * Coverage:
 *   [x] Command palette (menu overlay) opens with overlay phase + Esc dismisses
 *   [x] Reduced-motion emulation uses fade-only overlay classes
 *   [x] Toaster motion CSS (`.lx-toaster-motion`) is present in the stylesheet
 */
import type { Page } from '@playwright/test'
import { test, expect } from '../fixtures/test.js'

async function openCommandPalette(page: Page) {
  await expect(page.getByRole('navigation', { name: /main/i })).toBeVisible({ timeout: 15000 })
  const trigger = page.locator('[data-command-palette-anchor="sidebar"]')
  await expect(trigger).toBeVisible({ timeout: 5000 })
  await trigger.click()
  await expect(page.getByRole('dialog', { name: /command palette/i })).toBeVisible({
    timeout: 5000,
  })
}

test.describe('AN.5 overlay motion', () => {
  test('command palette opens with overlay motion and Esc dismisses', async ({
    authedPage: page,
  }) => {
    await openCommandPalette(page)

    const root = page.locator('[data-overlay-phase]').first()
    await expect(root).toHaveAttribute('data-overlay-phase', /opening|open/)

    await page.keyboard.press('Escape')
    await expect(page.getByRole('dialog', { name: /command palette/i })).toBeHidden({
      timeout: 5000,
    })
  })

  test('reduced motion: overlays fade only', async ({ authedPage: page }) => {
    await page.emulateMedia({ reducedMotion: 'reduce' })
    await page.reload()
    await openCommandPalette(page)

    const panelClass =
      (await page.locator('[data-overlay-phase] .relative.z-10').first().getAttribute('class')) ??
      ''
    expect(panelClass).toMatch(/lx-overlay-fade-in/)

    await page.keyboard.press('Escape')
    await expect(page.getByRole('dialog', { name: /command palette/i })).toBeHidden({
      timeout: 5000,
    })
  })

  test('toaster motion CSS is present', async ({ authedPage: page }) => {
    await expect(page.getByRole('navigation', { name: /main/i })).toBeVisible({ timeout: 15000 })

    // Sonner only mounts `[data-sonner-toaster]` while toasts are visible, so assert the
    // AN.5 stylesheet hook instead of requiring a live toast node.
    const hasToasterMotion = await page.evaluate(() => {
      for (const sheet of Array.from(document.styleSheets)) {
        let rules: CSSRuleList
        try {
          rules = sheet.cssRules
        } catch {
          continue
        }
        for (const rule of Array.from(rules)) {
          if ('selectorText' in rule && typeof rule.selectorText === 'string') {
            if (rule.selectorText.includes('lx-toaster-motion')) return true
          }
        }
      }
      return false
    })
    expect(hasToasterMotion).toBe(true)
  })
})
