/**
 * Per-student progress dashboard (plan 9.1)
 */
import { test, expect } from '../fixtures/test.js'
import { injectToken } from '../fixtures/test.js'
import { apiListEnrollments } from '../fixtures/api.js'

const apiBase = process.env.E2E_API_URL ?? 'http://localhost:8080'

async function studentEnrollmentId(
  instructorToken: string,
  courseCode: string,
): Promise<string> {
  const rows = await apiListEnrollments(instructorToken, courseCode)
  const student = rows.find(
    (e) =>
      e.role === 'student' ||
      (e.displayName ?? '').includes('E2E Student'),
  )
  if (!student?.id) {
    throw new Error('E2E student enrollment not found')
  }
  return student.id
}

test.describe('Student progress', () => {
  test('instructor opens student progress and adds a private note', async ({
    coursePage: page,
    seededCourse,
  }) => {
    const enrollmentId = await studentEnrollmentId(
      seededCourse.instructorToken,
      seededCourse.courseCode,
    )
    const apiProgress = `${apiBase}/api/v1/courses/${encodeURIComponent(seededCourse.courseCode)}/enrollments/${encodeURIComponent(enrollmentId)}/progress`
    const probe = await page.request.get(apiProgress, {
      headers: { Authorization: `Bearer ${seededCourse.instructorToken}` },
    })
    if (probe.status() === 404) {
      test.skip(true, 'FEATURE_STUDENT_PROGRESS is disabled on the API')
    }
    expect(probe.ok()).toBeTruthy()

    await page.goto(
      `/courses/${seededCourse.courseCode}/students/${enrollmentId}/progress`,
    )
    await page.waitForResponse(
      (res) =>
        res.url().includes('/api/v1/platform/features') &&
        res.request().method() === 'GET' &&
        res.ok(),
    )

    await expect(page.getByRole('heading', { name: /E2E Student/i })).toBeVisible({
      timeout: 15000,
    })
    await expect(page.getByText(/assignments submitted|modules viewed/i).first()).toBeVisible()

    await page.getByRole('tab', { name: /notes/i }).click()
    const noteText = `E2E note ${Date.now()}`
    await page.locator('#progress-note').fill(noteText)
    await page.getByRole('button', { name: /^save$/i }).click()
    await expect(page.getByText(noteText)).toBeVisible({ timeout: 8000 })

    await page.reload()
    await page.getByRole('tab', { name: /notes/i }).click()
    await expect(page.getByText(noteText)).toBeVisible({ timeout: 8000 })
  })

  test('instructor opens student report from course reports page', async ({
    coursePage: page,
    seededCourse,
  }) => {
    const enrollmentId = await studentEnrollmentId(
      seededCourse.instructorToken,
      seededCourse.courseCode,
    )
    const apiProgress = `${apiBase}/api/v1/courses/${encodeURIComponent(seededCourse.courseCode)}/enrollments/${encodeURIComponent(enrollmentId)}/progress`
    const probe = await page.request.get(apiProgress, {
      headers: { Authorization: `Bearer ${seededCourse.instructorToken}` },
    })
    if (probe.status() === 404) {
      test.skip(true, 'FEATURE_STUDENT_PROGRESS is disabled on the API')
    }
    expect(probe.ok()).toBeTruthy()

    await page.goto(`/courses/${seededCourse.courseCode}/reports`)
    await expect(page.getByRole('heading', { name: /^reports$/i })).toBeVisible()
    await page.getByRole('link', { name: /e2e student/i }).first().click()

    await expect(page).toHaveURL(
      new RegExp(
        `/courses/${seededCourse.courseCode}/students/${enrollmentId}/progress`,
      ),
    )
    await expect(page.getByText(/assignments submitted|modules viewed/i).first()).toBeVisible({
      timeout: 15000,
    })
  })

  test('instructor opens student report from enrollments roster', async ({
    coursePage: page,
    seededCourse,
  }) => {
    const enrollmentId = await studentEnrollmentId(
      seededCourse.instructorToken,
      seededCourse.courseCode,
    )
    const apiProgress = `${apiBase}/api/v1/courses/${encodeURIComponent(seededCourse.courseCode)}/enrollments/${encodeURIComponent(enrollmentId)}/progress`
    await page.goto(`/courses/${seededCourse.courseCode}/enrollments`)
    const studentRow = page.locator('tr').filter({ hasText: 'E2E Student' })
    await studentRow.hover()
    await studentRow.getByRole('link', { name: /view report for e2e student/i }).click()

    await expect(page).toHaveURL(
      new RegExp(
        `/courses/${seededCourse.courseCode}/students/${enrollmentId}/progress`,
      ),
    )
    await expect(page.getByText(/assignments submitted|modules viewed/i).first()).toBeVisible({
      timeout: 15000,
    })
  })

  test('student views own progress without instructor notes tab', async ({
    page,
    seededCourse,
  }) => {
    await injectToken(page, seededCourse.studentToken)
    const courseRes = await page.request.get(
      `${apiBase}/api/v1/courses/${encodeURIComponent(seededCourse.courseCode)}`,
      { headers: { Authorization: `Bearer ${seededCourse.studentToken}` } },
    )
    expect(courseRes.ok()).toBeTruthy()
    const course = (await courseRes.json()) as { viewerStudentEnrollmentId?: string }
    const eid = course.viewerStudentEnrollmentId
    expect(eid).toBeTruthy()

    const progressPath = `/api/v1/courses/${encodeURIComponent(seededCourse.courseCode)}/enrollments/${eid}/progress`
    const probe = await page.request.get(`${apiBase}${progressPath}`, {
      headers: { Authorization: `Bearer ${seededCourse.studentToken}` },
    })
    if (probe.status() === 404) {
      test.skip(true, 'FEATURE_STUDENT_PROGRESS is disabled on the API')
    }
    expect(probe.ok()).toBeTruthy()

    await page.goto(`/courses/${seededCourse.courseCode}/students/${eid}/progress`)
    await page.waitForResponse(
      (res) =>
        res.url().includes('/api/v1/platform/features') &&
        res.request().method() === 'GET' &&
        res.ok(),
    )

    await expect(page.getByRole('heading', { name: /E2E Student/i })).toBeVisible({
      timeout: 15000,
    })
    await expect(page.getByRole('tab', { name: /notes/i })).toHaveCount(0)
  })
})
