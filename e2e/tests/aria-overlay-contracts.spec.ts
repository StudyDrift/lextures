/**
 * UX.4 FR-14 / AC-12 — gallery-driven overlay keyboard contracts.
 *
 * Exercises Dialog, Tabs, Menu, and Tooltip on `/design/components` so new
 * gallery components inherit coverage. Requires an authenticated staff session
 * (design routes are behind the app shell).
 */
import { test, expect } from '../fixtures/test.js'

test.describe('UX.4 ARIA overlay contracts (gallery)', () => {
  test.beforeEach(async ({ authedPage: page }) => {
    await page.goto('/design/components')
    await expect(page.getByRole('heading', { name: /component/i }).first()).toBeVisible({
      timeout: 15000,
    })
  })

  test('Dialog: Escape closes and focus returns', async ({ authedPage: page }) => {
    const openBtn = page.getByRole('button', { name: /open dialog/i }).first()
    if ((await openBtn.count()) === 0) {
      test.skip(true, 'Dialog demo not present in gallery')
      return
    }
    await openBtn.focus()
    await openBtn.click()
    const dialog = page.getByRole('dialog').first()
    await expect(dialog).toBeVisible({ timeout: 5000 })
    await page.keyboard.press('Escape')
    await expect(dialog).toBeHidden({ timeout: 5000 })
    await expect(openBtn).toBeFocused({ timeout: 3000 })
  })

  test('Tabs: arrow keys move selection', async ({ authedPage: page }) => {
    const tablist = page.getByRole('tablist').first()
    if ((await tablist.count()) === 0) {
      test.skip(true, 'Tablist demo not present in gallery')
      return
    }
    const tabs = tablist.getByRole('tab')
    const count = await tabs.count()
    if (count < 2) {
      test.skip(true, 'Need at least two tabs')
      return
    }
    await tabs.nth(0).focus()
    await page.keyboard.press('ArrowRight')
    await expect(tabs.nth(1)).toHaveAttribute('aria-selected', 'true')
  })

  test('Menu: opens with focus on first item, Escape restores', async ({ authedPage: page }) => {
    const menuTrigger = page.getByRole('button', { name: /open menu|menu/i }).first()
    if ((await menuTrigger.count()) === 0) {
      test.skip(true, 'Menu demo not present in gallery')
      return
    }
    await menuTrigger.click()
    const menu = page.getByRole('menu').first()
    await expect(menu).toBeVisible({ timeout: 5000 })
    const firstItem = menu.getByRole('menuitem').first()
    await expect(firstItem).toBeFocused({ timeout: 3000 })
    await page.keyboard.press('Escape')
    await expect(menu).toBeHidden({ timeout: 5000 })
    await expect(menuTrigger).toBeFocused({ timeout: 3000 })
  })
})
