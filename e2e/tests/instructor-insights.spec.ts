/**
 * Instructor "What's Working" signals (plan 9.10)
 *
 * Checklist coverage:
 *   [x] GET analytics/insights returns valid structure for instructor
 *   [x] POST analytics/insights/refresh returns updated insights
 *   [x] POST analytics/insights/dismiss marks a signal dismissed
 *   [x] GET analytics/cross-section returns array (empty ok)
 *   [x] Student gets 403 on analytics/insights
 *   [x] Unauthenticated gets 401
 *   [x] What's working page loads for instructor
 */
import { test, expect, injectToken } from '../fixtures/test.js'
import {
  apiGetInsights,
  apiRefreshInsights,
  apiDismissInsightSignal,
  apiGetCrossSection,
} from '../fixtures/api.js'

const apiBase = process.env.E2E_API_URL ?? 'http://localhost:8080'

test.describe('Instructor insights — API', () => {
  test('GET analytics/insights returns valid structure', async ({ seededCourse }) => {
    const insights = await apiGetInsights(seededCourse.instructorToken, seededCourse.courseCode)
    expect(insights).toHaveProperty('courseId')
    expect(insights).toHaveProperty('weekOf')
    expect(insights).toHaveProperty('generatedAt')
    expect(Array.isArray(insights.workingWell)).toBe(true)
    expect(Array.isArray(insights.needsAttention)).toBe(true)
    expect(Array.isArray(insights.scatter)).toBe(true)
  })

  test('POST analytics/insights/refresh returns updated insights', async ({ seededCourse }) => {
    const insights = await apiRefreshInsights(seededCourse.instructorToken, seededCourse.courseCode)
    expect(insights).toHaveProperty('generatedAt')
    expect(Array.isArray(insights.workingWell)).toBe(true)
    expect(Array.isArray(insights.needsAttention)).toBe(true)
  })

  test('POST analytics/insights/dismiss succeeds with any signal key', async ({ seededCourse }) => {
    const fakeKey = 'e2e-fake-signal-' + Date.now()
    await expect(
      apiDismissInsightSignal(seededCourse.instructorToken, seededCourse.courseCode, fakeKey, 'E2E test reason'),
    ).resolves.not.toThrow()
  })

  test('GET analytics/cross-section returns array', async ({ seededCourse }) => {
    const rows = await apiGetCrossSection(seededCourse.instructorToken, seededCourse.courseCode)
    expect(Array.isArray(rows)).toBe(true)
  })

  test('student gets 403 on analytics/insights', async ({ seededCourse }) => {
    const res = await fetch(
      `${apiBase}/api/v1/courses/${seededCourse.courseCode}/analytics/insights`,
      { headers: { Authorization: `Bearer ${seededCourse.studentToken}` } },
    )
    expect(res.status).toBe(403)
  })

  test('unauthenticated gets 401 on analytics/insights', async () => {
    const res = await fetch(`${apiBase}/api/v1/courses/nonexistent/analytics/insights`)
    expect(res.status).toBe(401)
  })

  test('student gets 403 on analytics/cross-section', async ({ seededCourse }) => {
    const res = await fetch(
      `${apiBase}/api/v1/courses/${seededCourse.courseCode}/analytics/cross-section`,
      { headers: { Authorization: `Bearer ${seededCourse.studentToken}` } },
    )
    expect(res.status).toBe(403)
  })

  test('POST analytics/insights/dismiss returns dismissed:true', async ({ seededCourse }) => {
    const key = 'e2e-dismiss-check-' + Date.now()
    const res = await fetch(
      `${apiBase}/api/v1/courses/${seededCourse.courseCode}/analytics/insights/dismiss`,
      {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${seededCourse.instructorToken}`,
        },
        body: JSON.stringify({ signalKey: key, reason: 'E2E check' }),
      },
    )
    expect(res.status).toBe(200)
    const body = (await res.json()) as { dismissed: boolean }
    expect(body.dismissed).toBe(true)
  })
})

test.describe('Instructor insights — UI', () => {
  test("what's working page loads for instructor", async ({ coursePage: page, seededCourse }) => {
    await page.goto(`/courses/${seededCourse.courseCode}/whats-working`)
    await expect(page.getByRole('heading', { name: /what.s working/i })).toBeVisible({
      timeout: 12000,
    })
    await expect(page.getByRole('button', { name: /refresh insights/i })).toBeVisible({
      timeout: 8000,
    })
  })

  test('working well and needs attention sections visible', async ({
    coursePage: page,
    seededCourse,
  }) => {
    await page.goto(`/courses/${seededCourse.courseCode}/whats-working`)
    await expect(page.getByRole('heading', { name: /what.s working/i })).toBeVisible({
      timeout: 12000,
    })
    await expect(page.getByRole('region', { name: /working well/i })).toBeVisible({
      timeout: 10000,
    })
    await expect(page.getByRole('region', { name: /needs attention/i })).toBeVisible({
      timeout: 10000,
    })
  })

  test('student cannot see whats-working page nav target', async ({ page, seededCourse }) => {
    await injectToken(page, seededCourse.studentToken)
    await page.goto(`/courses/${seededCourse.courseCode}`)
    await expect(page.getByRole('link', { name: /what.s working/i })).not.toBeVisible({
      timeout: 5000,
    })
  })
})
