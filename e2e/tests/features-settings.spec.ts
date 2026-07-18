/**
 * Course Settings → Features smoke coverage.
 * Full toggle matrix lives in course-features-ui-matrix-{a,b,c}.spec.ts (E2E.1).
 */
import { test, expect } from '../fixtures/test.js'

test.describe('Course Settings - Features', () => {
  test('features tab loads with Course tools heading', async ({ coursePage: page, seededCourse }) => {
    await page.goto(`/courses/${seededCourse.courseCode}/settings/features`)
    await expect(page.getByRole('heading', { name: /^Course tools$/i })).toBeVisible({
      timeout: 12_000,
    })
    await expect(page.getByText(/Turn tools on or off/i)).toBeVisible()
  })

  test('search filters the tools list', async ({ coursePage: page, seededCourse }) => {
    await page.goto(`/courses/${seededCourse.courseCode}/settings/features`)
    await expect(page.getByRole('heading', { name: /^Course tools$/i })).toBeVisible({
      timeout: 12_000,
    })
    await page.getByPlaceholder('Search tools…').fill('Discussion forums')
    await expect(page.getByText('Discussion forums', { exact: true })).toBeVisible()
    await expect(page.getByText('Course sections', { exact: true })).toHaveCount(0)
  })
})
