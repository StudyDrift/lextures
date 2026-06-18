/**
 * Daily study goal reminders (plan 15.10)
 */
import { test, expect } from '../fixtures/test.js'
import { injectToken } from '../fixtures/test.js'

const apiBase = process.env.E2E_API_URL ?? 'http://localhost:8080'

test.describe('Study reminders', () => {
  test('student configures daily reminder goal via API', async ({ page, seededCourse }) => {
    await injectToken(page, seededCourse.studentToken)

    const featuresRes = await page.request.get(`${apiBase}/api/v1/platform/features`, {
      headers: { Authorization: `Bearer ${seededCourse.studentToken}` },
    })
    expect(featuresRes.ok()).toBeTruthy()
    const features = (await featuresRes.json()) as { ffStudyReminders?: boolean }
    if (!features.ffStudyReminders) {
      test.skip(true, 'ffStudyReminders is false on the API')
    }

    const patchRes = await page.request.patch(`${apiBase}/api/v1/me/reminder-config`, {
      headers: {
        Authorization: `Bearer ${seededCourse.studentToken}`,
        'Content-Type': 'application/json',
      },
      data: {
        enabled: true,
        dailyGoalMinutes: 20,
        reminderTime: '19:00',
        reminderChannels: ['email'],
        weeklySummary: true,
      },
    })
    expect(patchRes.ok()).toBeTruthy()
    const patched = (await patchRes.json()) as { enabled?: boolean; dailyGoalMinutes?: number }
    expect(patched.enabled).toBe(true)
    expect(patched.dailyGoalMinutes).toBe(20)

    const getRes = await page.request.get(`${apiBase}/api/v1/me/reminder-config`, {
      headers: { Authorization: `Bearer ${seededCourse.studentToken}` },
    })
    expect(getRes.ok()).toBeTruthy()
    const cfg = (await getRes.json()) as { enabled?: boolean; reminderTime?: string }
    expect(cfg.enabled).toBe(true)
    expect(cfg.reminderTime).toBe('19:00')
  })
})
