/**
 * Course Attendance (docs/plan/attendance.md)
 *
 *   [x] Unauthenticated requests return 401
 *   [x] Attendance disabled returns 404 on API
 *   [x] Instructor enables feature and creates roll-call session
 *   [x] Batch save marks students present
 *   [x] Close session with gradebook creates gradebook column
 *   [x] Self-report: student checks in; duplicate rejected
 *   [x] UI: nav link when enabled; page loads
 */
import { test, expect } from '@playwright/test'
import { apiSignup, apiCreateCourse, apiEnroll } from '../fixtures/api.js'
import { injectToken } from '../fixtures/test.js'

const API_BASE = process.env.E2E_API_URL ?? 'http://localhost:8080'
const PASSWORD = 'E2eTestPass1!'

function uniqueEmail(prefix = 'catt') {
  return `e2e-${prefix}-${Date.now()}-${Math.random().toString(36).slice(2)}@test.invalid`
}

function authHeaders(token: string) {
  return { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' }
}

async function enableAttendance(token: string, courseCode: string): Promise<void> {
  const res = await fetch(`${API_BASE}/api/v1/courses/${courseCode}/features`, {
    method: 'PATCH',
    headers: authHeaders(token),
    body: JSON.stringify({
      notebookEnabled: true,
      feedEnabled: false,
      calendarEnabled: true,
      questionBankEnabled: false,
      lockdownModeEnabled: false,
      standardsAlignmentEnabled: false,
      discussionsEnabled: false,
      attendanceEnabled: true,
    }),
  })
  if (!res.ok) {
    throw new Error(`enable attendance failed: ${res.status} ${await res.text()}`)
  }
}

test.describe('Course Attendance API', () => {
  test('unauthenticated access returns 401', async ({ request }) => {
    const sid = '00000000-0000-0000-0000-000000000001'
    const paths = [
      { method: 'GET', path: `/api/v1/courses/C-FAKE/attendance/sessions` },
      { method: 'POST', path: `/api/v1/courses/C-FAKE/attendance/sessions` },
      { method: 'GET', path: `/api/v1/courses/C-FAKE/attendance/sessions/${sid}` },
      { method: 'PUT', path: `/api/v1/courses/C-FAKE/attendance/sessions/${sid}/records` },
      { method: 'POST', path: `/api/v1/courses/C-FAKE/attendance/sessions/${sid}/self-report` },
      { method: 'POST', path: `/api/v1/courses/C-FAKE/attendance/sessions/${sid}/close` },
    ]
    for (const { method, path } of paths) {
      const res = await request.fetch(`${API_BASE}${path}`, { method })
      expect(res.status(), `${method} ${path}`).toBe(401)
    }
  })

  test('roll call with gradebook end-to-end', async ({ request }) => {
    const instrEmail = uniqueEmail('instr')
    const stuEmail = uniqueEmail('stu')
    const { access_token: instrToken } = await apiSignup({ email: instrEmail, password: PASSWORD })
    await apiSignup({ email: stuEmail, password: PASSWORD })
    const course = await apiCreateCourse(instrToken, { title: 'Course Attendance E2E' })
    const cc = course.courseCode
    await apiEnroll(instrToken, cc, stuEmail, 'student')
    await enableAttendance(instrToken, cc)

    const rosterRes = await request.get(`${API_BASE}/api/v1/courses/${cc}/enrollments`, {
      headers: { Authorization: `Bearer ${instrToken}` },
    })
    expect(rosterRes.ok()).toBeTruthy()
    const roster = (await rosterRes.json()) as { enrollments: Array<{ userId: string; role: string }> }
    const stuId = roster.enrollments.find((e) => e.role === 'student')?.userId
    expect(stuId).toBeTruthy()

    const createRes = await request.post(`${API_BASE}/api/v1/courses/${cc}/attendance/sessions`, {
      headers: authHeaders(instrToken),
      data: {
        collectionMethod: 'roll_call',
        title: 'E2E Roll Call',
        gradebookEnabled: true,
        pointsPossible: 10,
      },
    })
    expect(createRes.status()).toBe(201)
    const session = (await createRes.json()) as { id: string }
    expect(session.id).toBeTruthy()

    const saveRes = await request.put(
      `${API_BASE}/api/v1/courses/${cc}/attendance/sessions/${session.id}/records`,
      {
        headers: authHeaders(instrToken),
        data: {
          records: [{ studentUserId: stuId, status: 'present' }],
        },
      },
    )
    expect(saveRes.ok()).toBeTruthy()
    const saved = (await saveRes.json()) as { message: string }
    expect(saved.message).toContain('saved')

    const closeRes = await request.post(
      `${API_BASE}/api/v1/courses/${cc}/attendance/sessions/${session.id}/close`,
      {
        headers: authHeaders(instrToken),
        data: { finalizeMissingAsAbsent: true },
      },
    )
    expect(closeRes.ok()).toBeTruthy()
    const closed = (await closeRes.json()) as { structureItemId?: string }
    expect(closed.structureItemId).toBeTruthy()

    const gridRes = await request.get(`${API_BASE}/api/v1/courses/${cc}/gradebook/grid`, {
      headers: { Authorization: `Bearer ${instrToken}` },
    })
    expect(gridRes.ok()).toBeTruthy()
    const grid = (await gridRes.json()) as {
      columns: Array<{ id: string; kind: string }>
      grades: Record<string, Record<string, string>>
    }
    const col = grid.columns.find((c) => c.id === closed.structureItemId)
    expect(col?.kind).toBe('attendance')
    const pts = grid.grades[stuId!]?.[closed.structureItemId!]
    expect(Number(pts)).toBe(10)
  })

  test('self-report check-in and duplicate rejection', async ({ request }) => {
    const instrEmail = uniqueEmail('sr-i')
    const stuEmail = uniqueEmail('sr-s')
    const { access_token: instrToken } = await apiSignup({ email: instrEmail, password: PASSWORD })
    const { access_token: stuToken } = await apiSignup({ email: stuEmail, password: PASSWORD })
    const course = await apiCreateCourse(instrToken, { title: 'Self Report E2E' })
    const cc = course.courseCode
    await apiEnroll(instrToken, cc, stuEmail, 'student')
    await enableAttendance(instrToken, cc)

    const now = new Date()
    const createRes = await request.post(`${API_BASE}/api/v1/courses/${cc}/attendance/sessions`, {
      headers: authHeaders(instrToken),
      data: {
        collectionMethod: 'self_report',
        opensAt: now.toISOString(),
        closesAt: new Date(now.getTime() + 15 * 60_000).toISOString(),
      },
    })
    expect(createRes.status()).toBe(201)
    const session = (await createRes.json()) as { id: string }

    const checkIn = await request.post(
      `${API_BASE}/api/v1/courses/${cc}/attendance/sessions/${session.id}/self-report`,
      { headers: authHeaders(stuToken), data: { status: 'present' } },
    )
    expect(checkIn.ok()).toBeTruthy()

    const dup = await request.post(
      `${API_BASE}/api/v1/courses/${cc}/attendance/sessions/${session.id}/self-report`,
      { headers: authHeaders(stuToken), data: { status: 'present' } },
    )
    expect(dup.status()).toBe(409)
  })

  test('attendance disabled returns 404', async ({ request }) => {
    const email = uniqueEmail('off')
    const { access_token: token } = await apiSignup({ email, password: PASSWORD })
    const course = await apiCreateCourse(token, { title: 'Attendance Off' })
    const cc = course.courseCode
    const res = await request.get(`${API_BASE}/api/v1/courses/${cc}/attendance/sessions`, {
      headers: { Authorization: `Bearer ${token}` },
    })
    expect(res.status()).toBe(404)
  })
})

test.describe('Course Attendance UI', () => {
  test('nav link appears when feature enabled', async ({ page }) => {
    const email = uniqueEmail('ui-nav')
    const { access_token: token } = await apiSignup({ email, password: PASSWORD })
    const course = await apiCreateCourse(token, { title: 'Attendance Nav' })
    await enableAttendance(token, course.courseCode)
    await injectToken(page, token)
    await page.goto(`/courses/${course.courseCode}`)
    await expect(page.getByRole('link', { name: /^Attendance$/i })).toBeVisible({ timeout: 10000 })
  })

  test('attendance page loads for instructor', async ({ page }) => {
    const email = uniqueEmail('ui-page')
    const { access_token: token } = await apiSignup({ email, password: PASSWORD })
    const course = await apiCreateCourse(token, { title: 'Attendance Page' })
    await enableAttendance(token, course.courseCode)
    await injectToken(page, token)
    await page.goto(`/courses/${course.courseCode}/attendance`)
    await expect(page.getByRole('heading', { name: /^Attendance$/i })).toBeVisible({ timeout: 10000 })
    await expect(page.getByRole('button', { name: /start session/i })).toBeVisible()
  })

  test('nav hidden when feature disabled', async ({ page }) => {
    const email = uniqueEmail('ui-off')
    const { access_token: token } = await apiSignup({ email, password: PASSWORD })
    const course = await apiCreateCourse(token, { title: 'No Attendance' })
    await injectToken(page, token)
    await page.goto(`/courses/${course.courseCode}`)
    await expect(page.getByRole('link', { name: /^Attendance$/i })).not.toBeVisible()
  })
})
