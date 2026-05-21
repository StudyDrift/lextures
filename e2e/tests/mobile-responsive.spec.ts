/**
 * Mobile viewport regressions (plan 7.2 — mobile-responsive review).
 */
import { test, expect } from '../fixtures/test.js'
import { apiCreateModule } from '../fixtures/api.js'

test.describe('Mobile responsive (390×844)', () => {
  test.use({ viewport: { width: 390, height: 844 } })

  test('gradebook page does not introduce horizontal page overflow', async ({ coursePage: page, seededCourse }) => {
    await page.goto(`/courses/${seededCourse.courseCode}/gradebook`)
    await expect(page.getByRole('heading', { name: /gradebook/i })).toBeVisible({ timeout: 15000 })

    const metrics = await page.evaluate(() => {
      const doc = document.documentElement
      const main = document.querySelector('main.lms-scope')
      const gb = document.querySelector('[data-testid="gradebook-scroll"]')
      return {
        docScrollWidth: doc.scrollWidth,
        docClientWidth: doc.clientWidth,
        mainScrollWidth: main?.scrollWidth ?? 0,
        mainClientWidth: main?.clientWidth ?? 0,
        hasGradebookScroll: gb instanceof HTMLElement,
        gbScrollWidth: gb instanceof HTMLElement ? gb.scrollWidth : 0,
        gbClientWidth: gb instanceof HTMLElement ? gb.clientWidth : 0,
      }
    })

    expect(metrics.docScrollWidth).toBeLessThanOrEqual(metrics.docClientWidth + 1)
    expect(metrics.mainScrollWidth).toBeLessThanOrEqual(metrics.mainClientWidth + 1)

    if (metrics.hasGradebookScroll && metrics.gbScrollWidth > metrics.gbClientWidth) {
      const scroll = page.getByTestId('gradebook-scroll')
      await expect(scroll).toBeVisible()
    }
  })

  test('modules page shows touch reorder controls on narrow viewports', async ({
    coursePage: page,
    seededCourse,
  }) => {
    await apiCreateModule(seededCourse.instructorToken, seededCourse.courseCode, 'Unit 2 — mobile')
    await page.goto(`/courses/${seededCourse.courseCode}/modules`)
    await expect(page.getByRole('heading', { name: /modules/i })).toBeVisible({ timeout: 15000 })

    await expect(page.getByRole('button', { name: /move module up/i }).first()).toBeVisible({ timeout: 8000 })
    await expect(page.getByRole('button', { name: /move module down/i }).first()).toBeVisible()

    if (process.env.SAVE_PR_SCREENSHOT === '1') {
      await page.screenshot({
        path: '/opt/cursor/artifacts/7.2-mobile-modules.png',
        fullPage: false,
      })
    }
  })
})
