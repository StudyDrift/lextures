import { test, expect } from '@playwright/test'
import {
  apiSignup,
  apiCreateCourse,
  apiCreateModule,
  apiCreateAssignment,
  apiEnroll,
} from '../fixtures/api.js'

const API_BASE = process.env.E2E_API_URL ?? 'http://localhost:8080'
const PASSWORD = 'E2eTestPass1!'

function uniqueEmail(prefix = 'calendar-feed') {
  return `e2e-${prefix}-${Date.now()}-${Math.random().toString(36).slice(2)}@test.invalid`
}

function authHeaders(token: string) {
  return { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' }
}

test.describe('Calendar feeds (16.5)', () => {
  test('unauthenticated management routes return 401', async ({ request }) => {
    for (const method of ['GET', 'POST'] as const) {
      const res = await request.fetch(`${API_BASE}/api/v1/me/calendar-token`, { method })
      expect(res.status(), method).toBe(401)
    }
    const feedRes = await request.get(`${API_BASE}/api/v1/me/calendar.ics?token=lcf_invalid`)
    expect(feedRes.status()).toBe(401)
  })

  test('feature on: generate token, fetch iCal feed, rotate invalidates old URL', async ({
    request,
  }) => {
    const email = uniqueEmail()
    const { access_token } = await apiSignup({
      email,
      password: PASSWORD,
      displayName: 'Calendar Feed User',
      accountType: 'parent',
    })

    const platformRes = await request.get(`${API_BASE}/api/v1/platform/features`, {
      headers: { Authorization: `Bearer ${access_token}` },
    })
    expect(platformRes.ok()).toBeTruthy()
    const features = (await platformRes.json()) as { ffCalendarFeeds?: boolean }
    if (!features.ffCalendarFeeds) {
      test.skip(true, 'ffCalendarFeeds is false on the API')
    }

    const createRes = await request.post(`${API_BASE}/api/v1/me/calendar-token`, {
      headers: authHeaders(access_token),
    })
    expect(createRes.ok()).toBeTruthy()
    const created = (await createRes.json()) as { token?: string; feedUrl?: string }
    expect(created.token).toMatch(/^lcf_/)
    expect(created.feedUrl).toContain('/api/v1/me/calendar.ics?token=')

    const feedRes = await request.get(
      `${API_BASE}/api/v1/me/calendar.ics?token=${encodeURIComponent(created.token ?? '')}`,
    )
    expect(feedRes.ok()).toBeTruthy()
    expect(feedRes.headers()['content-type'] ?? '').toContain('text/calendar')
    const body = await feedRes.text()
    expect(body).toContain('BEGIN:VCALENDAR')

    const oldToken = created.token ?? ''
    const rotateRes = await request.post(`${API_BASE}/api/v1/me/calendar-token`, {
      headers: authHeaders(access_token),
    })
    expect(rotateRes.ok()).toBeTruthy()
    const rotated = (await rotateRes.json()) as { token?: string }
    expect(rotated.token).toMatch(/^lcf_/)
    expect(rotated.token).not.toBe(oldToken)

    const staleRes = await request.get(
      `${API_BASE}/api/v1/me/calendar.ics?token=${encodeURIComponent(oldToken)}`,
    )
    expect(staleRes.status()).toBe(401)
  })

  test('enrolled assignment due date appears in personal feed', async ({ request }) => {
    const instructorEmail = uniqueEmail('cal-inst')
    const { access_token: instructorToken } = await apiSignup({
      email: instructorEmail,
      password: PASSWORD,
      displayName: 'Calendar Instructor',
    })

    const studentEmail = uniqueEmail('cal-stu')
    const { access_token: studentToken } = await apiSignup({
      email: studentEmail,
      password: PASSWORD,
      displayName: 'Calendar Student',
      accountType: 'parent',
    })

    const platformRes = await request.get(`${API_BASE}/api/v1/platform/features`, {
      headers: { Authorization: `Bearer ${studentToken}` },
    })
    const features = (await platformRes.json()) as { ffCalendarFeeds?: boolean }
    if (!features.ffCalendarFeeds) {
      test.skip(true, 'ffCalendarFeeds is false on the API')
    }

    const course = await apiCreateCourse(instructorToken, { title: 'Calendar Feed Course' })
    await apiEnroll(instructorToken, course.courseCode, instructorEmail, 'teacher')
    await apiEnroll(instructorToken, course.courseCode, studentEmail, 'student', studentToken)

    const mod = await apiCreateModule(instructorToken, course.courseCode, 'Unit 1')
    const assignment = await apiCreateAssignment(
      instructorToken,
      course.courseCode,
      mod.id,
      'Feed Assignment',
    )

    const dueAt = new Date(Date.UTC(2026, 8, 15, 23, 59, 0)).toISOString()
    const dueRes = await request.patch(
      `${API_BASE}/api/v1/courses/${encodeURIComponent(course.courseCode)}/structure/items/${encodeURIComponent(assignment.id)}/due-at`,
      {
        headers: authHeaders(instructorToken),
        data: { dueAt },
      },
    )
    expect(dueRes.ok()).toBeTruthy()

    const tokenRes = await request.post(`${API_BASE}/api/v1/me/calendar-token`, {
      headers: authHeaders(studentToken),
    })
    expect(tokenRes.ok()).toBeTruthy()
    const { token } = (await tokenRes.json()) as { token?: string }
    expect(token).toMatch(/^lcf_/)

    const feedRes = await request.get(
      `${API_BASE}/api/v1/me/calendar.ics?token=${encodeURIComponent(token ?? '')}`,
    )
    expect(feedRes.ok()).toBeTruthy()
    const body = await feedRes.text()
    expect(body).toContain('Feed Assignment')
    expect(body).toContain(assignment.id)
    expect(body).toContain('BEGIN:VEVENT')
  })
})