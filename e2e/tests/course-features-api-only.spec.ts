/**
 * E2E.1 — API-only course flags (currently groupSpacesEnabled) until a settings row exists.
 */
import { test, expect, injectToken } from '../fixtures/test.js'
import {
  apiGetCourse,
  apiPatchCourseFeatures,
  apiWaitForCourseFeature,
} from '../fixtures/api.js'
import { readCourseFeatureFlag } from '../lib/course-feature-matrix.js'
import { withCourseFeatureRestore } from '../lib/course-feature-matrix-helpers.js'

test.describe('Course features API-only flags', () => {
  test('groupSpacesEnabled persists via API and gates Groups nav/route', async ({
    page,
    seededCourse,
  }) => {
    const { instructorToken, studentToken, courseCode } = seededCourse

    await withCourseFeatureRestore(instructorToken, courseCode, async () => {
      await apiPatchCourseFeatures(instructorToken, courseCode, { groupSpacesEnabled: false })
      await apiWaitForCourseFeature(instructorToken, courseCode, 'groupSpacesEnabled', false)

      let course = await apiGetCourse(instructorToken, courseCode)
      expect(
        readCourseFeatureFlag(course, 'groupSpacesEnabled'),
        `${courseCode} groupSpacesEnabled off`,
      ).toBe(false)

      await apiPatchCourseFeatures(instructorToken, courseCode, { groupSpacesEnabled: true })
      await apiWaitForCourseFeature(instructorToken, courseCode, 'groupSpacesEnabled', true)
      course = await apiGetCourse(instructorToken, courseCode)
      expect(
        readCourseFeatureFlag(course, 'groupSpacesEnabled'),
        `${courseCode} groupSpacesEnabled on`,
      ).toBe(true)

      await injectToken(page, studentToken)
      await page.goto(`/courses/${courseCode}`)
      await expect(page.getByRole('link', { name: /^Groups$/ })).toBeVisible({ timeout: 12_000 })

      await apiPatchCourseFeatures(instructorToken, courseCode, { groupSpacesEnabled: false })
      await apiWaitForCourseFeature(instructorToken, courseCode, 'groupSpacesEnabled', false)

      await page.goto(`/courses/${courseCode}`)
      await expect(page.getByRole('link', { name: /^Groups$/ })).toHaveCount(0)

      await page.goto(`/courses/${courseCode}/groups`)
      await expect(page).not.toHaveURL(/\/groups/, { timeout: 10_000 })
    })
  })

  test('groupSpacesEnabled is absent from Course tools settings rows', async ({
    coursePage: page,
    seededCourse,
  }) => {
    await page.goto(`/courses/${seededCourse.courseCode}/settings/features`)
    await expect(page.getByRole('heading', { name: /^Course tools$/i })).toBeVisible({
      timeout: 15_000,
    })
    await expect(page.getByText(/^Group spaces$/i)).toHaveCount(0)
  })
})
