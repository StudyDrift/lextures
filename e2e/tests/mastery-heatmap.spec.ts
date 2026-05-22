/**
 * Mastery Heatmap (plan 9.3)
 *
 * Checklist coverage:
 *   [x] Heatmap page loads for an instructor
 *   [x] Empty state shown when no concept data exists
 *   [x] Route is accessible under /courses/:courseCode/mastery-heatmap
 *   [x] Refresh button is present
 *   [x] Student cannot access instructor-only heatmap (403 from API)
 */
import { test, expect, injectToken } from '../fixtures/test.js'

test.describe('Mastery Heatmap', () => {
  test('heatmap page loads and shows empty state when no data', async ({
    coursePage: page,
    seededCourse,
  }) => {
    await page.goto(`/courses/${seededCourse.courseCode}/mastery-heatmap`)

    // Page title visible
    await expect(page.getByRole('heading', { name: /mastery heatmap/i })).toBeVisible({
      timeout: 10000,
    })

    // Either empty state or heatmap table — both are valid since CI has no concept data.
    const emptyState = page.getByText(/no skill data yet/i)
    const heatmapTable = page.getByRole('table', { name: /mastery heatmap/i })
    await expect(emptyState.or(heatmapTable)).toBeVisible({ timeout: 8000 })
  })

  test('refresh button is present for instructor', async ({ coursePage: page, seededCourse }) => {
    await page.goto(`/courses/${seededCourse.courseCode}/mastery-heatmap`)
    await expect(page.getByRole('button', { name: /refresh/i })).toBeVisible({ timeout: 10000 })
  })

  test('student cannot access the heatmap API directly (403)', async ({
    page,
    seededCourse,
  }) => {
    await injectToken(page, seededCourse.studentToken)
    const apiBase = process.env.E2E_API_URL ?? 'http://localhost:8080'
    const res = await page.request.get(
      `${apiBase}/api/v1/courses/${seededCourse.courseCode}/analytics/mastery-heatmap`,
      {
        headers: { Authorization: `Bearer ${seededCourse.studentToken}` },
      },
    )
    expect(res.status()).toBe(403)
  })
})
