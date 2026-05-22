/**
 * Per-student progress dashboard (plan 9.1)
 */
import { test, expect } from '../fixtures/test.js'
import { injectToken } from '../fixtures/test.js'

test.describe('Student progress', () => {
  test('instructor opens student progress and adds a private note', async ({
    coursePage: page,
    seededCourse,
  }) => {
    await page.goto(`/courses/${seededCourse.courseCode}/enrollments`)
    const table = page.locator('table, [role="table"]').first()
    await expect(table.getByRole('link', { name: 'E2E Student' })).toBeVisible({ timeout: 8000 })
    await table.getByRole('link', { name: 'E2E Student' }).click()

    await expect(page.getByRole('heading', { name: /E2E Student|progress/i })).toBeVisible({
      timeout: 8000,
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

  test('student views own progress without instructor notes tab', async ({
    page,
    seededCourse,
  }) => {
    await injectToken(page, seededCourse.studentToken)
    const courseRes = await page.request.get(
      `${process.env.E2E_API_URL ?? 'http://localhost:8080'}/api/v1/courses/${encodeURIComponent(seededCourse.courseCode)}`,
      { headers: { Authorization: `Bearer ${seededCourse.studentToken}` } },
    )
    expect(courseRes.ok()).toBeTruthy()
    const course = (await courseRes.json()) as { viewerStudentEnrollmentId?: string }
    const eid = course.viewerStudentEnrollmentId
    expect(eid).toBeTruthy()
    await page.goto(`/courses/${seededCourse.courseCode}/students/${eid}/progress`)

    await expect(page.getByRole('heading', { name: /my progress|E2E Student/i })).toBeVisible({
      timeout: 8000,
    })
    await expect(page.getByRole('tab', { name: /notes/i })).toHaveCount(0)
  })
})
