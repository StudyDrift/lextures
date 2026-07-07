/**
 * In-product feature-help dock (plan W06)
 *
 * Checklist coverage:
 *   [x] No placeholder walkthrough copy in help panels
 *   [x] Topics without media omit the clip region
 *   [x] Topics with media lazy-load a silent walkthrough video
 */
import { test, expect } from '../fixtures/test.js'

test.describe('Feature help dock', () => {
  test('gradebook help panel has no placeholder copy and no media region', async ({
    coursePage: page,
    seededCourse,
  }) => {
    await page.goto(`/courses/${seededCourse.courseCode}/gradebook`)
    await page.getByRole('button', { name: 'Open help for this area' }).click()

    const dialog = page.getByRole('dialog', { name: /gradebook help/i })
    await expect(dialog).toBeVisible()
    await expect(dialog.getByText(/placeholder/i)).toHaveCount(0)
    await expect(dialog.getByText(/when ready/i)).toHaveCount(0)
    await expect(dialog.locator('video')).toHaveCount(0)
    await expect(dialog.getByText(/double-click to edit scores/i)).toBeVisible()
  })

  test('modules help panel lazy-loads walkthrough media without placeholder copy', async ({
    coursePage: page,
    seededCourse,
  }) => {
    await page.goto(`/courses/${seededCourse.courseCode}/modules`)
    await page.getByRole('button', { name: 'Open help for this area' }).click()

    const dialog = page.getByRole('dialog', { name: /modules help/i })
    await expect(dialog).toBeVisible()
    await expect(dialog.getByText(/placeholder/i)).toHaveCount(0)
    await expect(dialog.getByText(/when ready/i)).toHaveCount(0)
    await expect(dialog.getByText(/drag handles reorder your outline/i)).toBeVisible()

    const video = dialog.locator('video')
    await expect(video).toHaveAttribute('src', /\/feature-help\/modules-walkthrough\.mp4/)
    await expect(video).toHaveAttribute('preload', 'none')
  })

  test('Escape closes the feature help panel', async ({ coursePage: page, seededCourse }) => {
    await page.goto(`/courses/${seededCourse.courseCode}/modules`)
    await page.getByRole('button', { name: 'Open help for this area' }).click()
    await expect(page.getByRole('dialog', { name: /modules help/i })).toBeVisible()

    await page.keyboard.press('Escape')
    await expect(page.getByRole('dialog', { name: /modules help/i })).toHaveCount(0)
  })
})