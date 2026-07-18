/**
 * E2E.1 — course features endpoint authz + omit-preservation contracts.
 */
import { test, expect } from '../fixtures/test.js'
import {
  apiGetCourse,
  apiPatchCourseFeatures,
  apiPatchCourseFeaturesRaw,
  apiRestoreCourseFeatures,
  apiSnapshotCourseFeatures,
} from '../fixtures/api.js'
import { readCourseFeatureFlag } from '../lib/course-feature-matrix.js'

const apiBase = process.env.E2E_API_URL ?? 'http://localhost:8080'

test.describe('Course features authz & preservation', () => {
  test('learner PATCH returns 403 and does not mutate flags', async ({ seededCourse }) => {
    const snapshot = await apiSnapshotCourseFeatures(
      seededCourse.instructorToken,
      seededCourse.courseCode,
    )
    try {
      await apiPatchCourseFeatures(seededCourse.instructorToken, seededCourse.courseCode, {
        attendanceEnabled: false,
      })

      const before = await apiGetCourse(seededCourse.instructorToken, seededCourse.courseCode)
      const res = await apiPatchCourseFeaturesRaw(
        seededCourse.studentToken,
        seededCourse.courseCode,
        { attendanceEnabled: true },
      )
      expect(res.status, `${seededCourse.courseCode} learner PATCH`).toBe(403)

      const after = await apiGetCourse(seededCourse.instructorToken, seededCourse.courseCode)
      expect(readCourseFeatureFlag(after, 'attendanceEnabled')).toBe(
        readCourseFeatureFlag(before, 'attendanceEnabled'),
      )
    } finally {
      await apiRestoreCourseFeatures(
        seededCourse.instructorToken,
        seededCourse.courseCode,
        snapshot,
      ).catch(() => {})
    }
  })

  test('unauthenticated PATCH returns 401', async ({ seededCourse }) => {
    const res = await apiPatchCourseFeaturesRaw(null, seededCourse.courseCode, {
      attendanceEnabled: true,
    })
    expect(res.status, `${seededCourse.courseCode} anonymous PATCH`).toBe(401)
  })

  test('partial PATCH preserves omitted nullable flags and unrelated features', async ({
    seededCourse,
  }) => {
    const { instructorToken, courseCode } = seededCourse
    const snapshot = await apiSnapshotCourseFeatures(instructorToken, courseCode)
    try {
      await apiPatchCourseFeatures(instructorToken, courseCode, {
        notebookEnabled: true,
        feedEnabled: true,
        calendarEnabled: true,
        discussionsEnabled: true,
        questionBankEnabled: true,
        lockdownModeEnabled: false,
        attendanceEnabled: true,
        officeHoursEnabled: true,
        aiTutorEnabled: false,
        visualBoardsEnabled: false,
      })

      const before = await apiGetCourse(instructorToken, courseCode)

      // Helper path: only flip one pointer flag.
      await apiPatchCourseFeatures(instructorToken, courseCode, { aiTutorEnabled: true })
      const afterHelper = await apiGetCourse(instructorToken, courseCode)
      expect(readCourseFeatureFlag(afterHelper, 'aiTutorEnabled'), `${courseCode} aiTutor`).toBe(
        true,
      )
      expect(readCourseFeatureFlag(afterHelper, 'attendanceEnabled', false)).toBe(true)
      expect(readCourseFeatureFlag(afterHelper, 'officeHoursEnabled', false)).toBe(true)
      expect(readCourseFeatureFlag(afterHelper, 'notebookEnabled', true)).toBe(true)
      expect(readCourseFeatureFlag(afterHelper, 'feedEnabled', true)).toBe(true)
      expect(readCourseFeatureFlag(afterHelper, 'discussionsEnabled')).toBe(true)

      // Raw server path: omit nullable flags (send only required non-pointer bools + one change).
      const raw = await fetch(`${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/features`, {
        method: 'PATCH',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${instructorToken}`,
        },
        body: JSON.stringify({
          notebookEnabled: true,
          feedEnabled: true,
          calendarEnabled: true,
          questionBankEnabled: true,
          lockdownModeEnabled: false,
          discussionsEnabled: true,
          visualBoardsEnabled: true,
        }),
      })
      expect(raw.status).toBe(200)

      const afterRaw = await apiGetCourse(instructorToken, courseCode)
      expect(readCourseFeatureFlag(afterRaw, 'visualBoardsEnabled')).toBe(true)
      // Omitted nullable flags must remain (aiTutor, attendance, officeHours).
      expect(readCourseFeatureFlag(afterRaw, 'aiTutorEnabled')).toBe(true)
      expect(readCourseFeatureFlag(afterRaw, 'attendanceEnabled')).toBe(true)
      expect(readCourseFeatureFlag(afterRaw, 'officeHoursEnabled')).toBe(true)
      // Unrelated non-pointer flags stay as sent.
      expect(readCourseFeatureFlag(afterRaw, 'discussionsEnabled')).toBe(
        readCourseFeatureFlag(before, 'discussionsEnabled'),
      )
    } finally {
      await apiRestoreCourseFeatures(instructorToken, courseCode, snapshot).catch(() => {})
    }
  })

  test('settings UI surfaces an error status when PATCH fails', async ({
    coursePage: page,
    seededCourse,
  }) => {
    const { courseCode } = seededCourse
    await page.route(`**/api/v1/courses/${courseCode}/features`, async (route) => {
      if (route.request().method() === 'PATCH') {
        await route.fulfill({
          status: 500,
          contentType: 'application/json',
          body: JSON.stringify({ error: { message: 'Simulated features save failure' } }),
        })
        return
      }
      await route.continue()
    })

    await page.goto(`/courses/${courseCode}/settings/features`)
    await expect(page.getByRole('heading', { name: /^Course tools$/i })).toBeVisible({
      timeout: 15_000,
    })

    const row = page
      .locator('div')
      .filter({ has: page.locator('p').filter({ hasText: /^Attendance$/ }) })
      .filter({ has: page.getByRole('switch') })
      .last()
    const toggle = row.getByRole('switch')
    await toggle.click()

    await expect(page.getByRole('status').filter({ hasText: /./ })).toBeVisible({ timeout: 10_000 })
    // Saved success must not appear; error status should.
    await expect(page.getByRole('status').filter({ hasText: /Saved/i })).toHaveCount(0)
  })

  test('feature switch is keyboard activatable with Space', async ({
    coursePage: page,
    seededCourse,
  }) => {
    const { instructorToken, courseCode } = seededCourse
    const snapshot = await apiSnapshotCourseFeatures(instructorToken, courseCode)
    try {
      await apiPatchCourseFeatures(instructorToken, courseCode, { attendanceEnabled: false })
      await page.goto(`/courses/${courseCode}/settings/features`)
      await expect(page.getByRole('heading', { name: /^Course tools$/i })).toBeVisible({
        timeout: 15_000,
      })

      const toggle = page
        .locator('div')
        .filter({ has: page.locator('p').filter({ hasText: /^Attendance$/ }) })
        .filter({ has: page.getByRole('switch') })
        .last()
        .getByRole('switch')

      await expect(toggle).toHaveAttribute('aria-checked', 'false')
      await toggle.focus()
      await expect(toggle).toBeFocused()
      await page.keyboard.press('Space')
      await expect(page.getByRole('status').filter({ hasText: /Saved/i })).toBeVisible({
        timeout: 10_000,
      })
      await expect(toggle).toHaveAttribute('aria-checked', 'true')
      await expect(toggle).toBeFocused()
    } finally {
      await apiRestoreCourseFeatures(instructorToken, courseCode, snapshot).catch(() => {})
    }
  })
})
