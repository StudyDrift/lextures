/**
 * AN.5 — Overlay & surface motion.
 *
 * Coverage:
 *   [x] Command palette (menu overlay) opens with overlay phase + Esc dismisses
 *   [x] Reduced-motion emulation uses fade-only overlay classes
 *   [x] Global toaster mounts with AN.1-tuned motion class
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

  test('toaster uses motion-tuned class', async ({ authedPage: page }) => {
    const toaster = page.locator('[data-sonner-toaster]')
    await expect(toaster).toHaveCount(1)
    await expect(toaster).toHaveClass(/lx-toaster-motion/)
  })
})
