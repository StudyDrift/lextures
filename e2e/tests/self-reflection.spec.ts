/**
 * Learner self-reflection & coaching (plan 9.9)
 */
import { test, expect } from '../fixtures/test.js'
import { injectToken } from '../fixtures/test.js'
import { enableEngagementTrackingForE2E } from '../fixtures/platform-features.js'

const apiBase = process.env.E2E_API_URL ?? 'http://localhost:8080'

test.describe('Self-reflection coaching', () => {
  test.beforeAll(async () => {
    await enableEngagementTrackingForE2E()
  })

  test('student opts in, sets goal, journals, and sees stats on dashboard', async ({
    page,
    seededCourse,
  }) => {
    await injectToken(page, seededCourse.studentToken)

    const featuresRes = await page.request.get(`${apiBase}/api/v1/platform/features`, {
      headers: { Authorization: `Bearer ${seededCourse.studentToken}` },
    })
    expect(featuresRes.ok()).toBeTruthy()
    const features = (await featuresRes.json()) as { selfReflectionEnabled?: boolean }
    if (!features.selfReflectionEnabled) {
      test.skip(true, 'selfReflectionEnabled is false on the API')
    }

    const goalRes = await page.request.put(`${apiBase}/api/v1/me/study-goal`, {
      headers: {
        Authorization: `Bearer ${seededCourse.studentToken}`,
        'Content-Type': 'application/json',
      },
      data: { weeklyHours: 10, optedIn: true },
    })
    expect(goalRes.ok()).toBeTruthy()

    const courseRes = await page.request.get(
      `${apiBase}/api/v1/courses/${encodeURIComponent(seededCourse.courseCode)}`,
      { headers: { Authorization: `Bearer ${seededCourse.studentToken}` } },
    )
    expect(courseRes.ok()).toBeTruthy()
    const course = (await courseRes.json()) as { id?: string }

    const events = Array.from({ length: 6 }, () => ({
      eventType: 'heartbeat',
      courseId: course.id,
      value: 1,
    }))
    const batchRes = await page.request.post(`${apiBase}/api/v1/analytics/events`, {
      headers: {
        Authorization: `Bearer ${seededCourse.studentToken}`,
        'Content-Type': 'application/json',
      },
      data: events,
    })
    if (batchRes.status() === 404) {
      test.skip(true, 'engagement tracking disabled')
    }
    expect(batchRes.ok()).toBeTruthy()

    const note = `E2E reflection ${Date.now()}`
    const journalRes = await page.request.post(`${apiBase}/api/v1/me/reflection-journal`, {
      headers: {
        Authorization: `Bearer ${seededCourse.studentToken}`,
        'Content-Type': 'application/json',
      },
      data: { entryText: note, courseId: course.id },
    })
    expect(journalRes.ok()).toBeTruthy()

    const instructorJournal = await page.request.get(`${apiBase}/api/v1/me/reflection-journal`, {
      headers: { Authorization: `Bearer ${seededCourse.instructorToken}` },
    })
    expect(instructorJournal.ok()).toBeTruthy()
    const instructorBody = (await instructorJournal.json()) as { entries: { entryText: string }[] }
    const leaked = instructorBody.entries?.some((e) => e.entryText === note)
    expect(leaked).toBeFalsy()

    await page.goto('/')
    await page.waitForResponse(
      (res) =>
        res.url().includes('/api/v1/platform/features') &&
        res.request().method() === 'GET' &&
        res.ok(),
    )
    await expect(page.getByRole('region', { name: 'Study stats' })).toBeVisible({ timeout: 15000 })
    await expect(page.getByLabel(/study streak/i).or(page.getByText(/hour/i).first())).toBeVisible()

    await page.goto('/me/study-insights')
    await expect(page.getByRole('heading', { name: 'Study insights' })).toBeVisible()
    await expect(page.getByText(note)).toBeVisible({ timeout: 8000 })

    const optOutRes = await page.request.put(`${apiBase}/api/v1/me/study-goal`, {
      headers: {
        Authorization: `Bearer ${seededCourse.studentToken}`,
        'Content-Type': 'application/json',
      },
      data: { optedIn: false },
    })
    expect(optOutRes.ok()).toBeTruthy()

    await page.goto('/')
    await expect(page.getByRole('region', { name: 'Study stats' })).toHaveCount(0, {
      timeout: 8000,
    })
  })

  test('feature disabled returns 404', async ({ page, seededCourse }) => {
    const probe = await page.request.get(`${apiBase}/api/v1/me/study-stats`, {
      headers: { Authorization: `Bearer ${seededCourse.studentToken}` },
    })
    if (probe.status() !== 404) {
      test.skip(true, 'self-reflection is enabled in this environment')
    }
    expect(probe.status()).toBe(404)
  })
})
