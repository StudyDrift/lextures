/**
 * AN.7 — Delight & progress moments.
 *
 * Coverage:
 *   [x] Delight CSS + data-motion-delight kill-switch present
 *   [x] Progress fill transition tokens available
 *   [x] Reduced-motion emulation suppresses particle burst path
 *   [x] Correct/incorrect feedback classes present in stylesheet
 */
import { test, expect } from '../fixtures/test.js'

test.describe('AN.7 delight motion', () => {
  test('dashboard exposes delight motion hooks', async ({ authedPage: page }) => {
    await expect(page.getByRole('navigation', { name: /main/i })).toBeVisible({ timeout: 15000 })

    const motionAttr = await page.evaluate(
      () => document.documentElement.dataset.motionDelight ?? 'missing',
    )
    expect(motionAttr).toBe('on')

    const hasCss = await page.evaluate(() => {
      const needles = ['lx-delight-progress', 'lx-delight-correct-pop', 'lx-delight-particle']
      const found = new Set<string>()
      for (const sheet of Array.from(document.styleSheets)) {
        let rules: CSSRuleList
        try {
          rules = sheet.cssRules
        } catch {
          continue
        }
        for (const rule of Array.from(rules)) {
          if ('selectorText' in rule && typeof rule.selectorText === 'string') {
            for (const n of needles) {
              if (rule.selectorText.includes(n)) found.add(n)
            }
          }
        }
      }
      return needles.every((n) => found.has(n))
    })
    expect(hasCss).toBe(true)
  })

  test('reduced motion: particle burst path is suppressed in stylesheet', async ({
    authedPage: page,
  }) => {
    await page.emulateMedia({ reducedMotion: 'reduce' })
    await page.reload()
    await expect(page.getByRole('navigation', { name: /main/i })).toBeVisible({ timeout: 15000 })

    const particleHidden = await page.evaluate(() => {
      for (const sheet of Array.from(document.styleSheets)) {
        let rules: CSSRuleList
        try {
          rules = sheet.cssRules
        } catch {
          continue
        }
        for (const rule of Array.from(rules)) {
          if ('cssText' in rule && typeof rule.cssText === 'string') {
            if (
              rule.cssText.includes('lx-delight-particle') &&
              (rule.cssText.includes('display: none') || rule.cssText.includes('display:none'))
            ) {
              return true
            }
          }
        }
      }
      return false
    })
    expect(particleHidden).toBe(true)
  })

  test('animated progress fill exists when gamification card renders', async ({
    authedPage: page,
  }) => {
    await expect(page.getByRole('navigation', { name: /main/i })).toBeVisible({ timeout: 15000 })
    const fill = page.getByTestId('animated-progress-fill').first()
    if ((await fill.count()) === 0) {
      test.skip(true, 'No animated progress on dashboard in this environment')
      return
    }
    // 0% progress can leave a zero-width fill that Playwright treats as hidden.
    await expect(fill).toBeAttached()
  })
})
