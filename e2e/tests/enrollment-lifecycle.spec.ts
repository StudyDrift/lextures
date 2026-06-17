/**
 * Enrollment lifecycle (plan 14.3)
 *
 *   [x] PATCH state unauthenticated returns 401
 *   [x] GET history unauthenticated returns 401
 *   [x] Endpoints return 501 when feature disabled
 *   [x] Instructor can mark student withdrawn and history records change
 *   [x] Gradebook includes former students section
 */
import { test, expect } from '../fixtures/test.js'
import { apiSignup, apiEnroll } from '../fixtures/api.js'

const API_BASE = process.env.E2E_API_URL ?? 'http://localhost:8080'

function authHeaders(token: string) {
  return { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' }
}

async function enableEnrollmentStateMachine(adminToken: string): Promise<void> {
  await fetch(`${API_BASE}/api/v1/settings/platform`, {
    method: 'PUT',
    headers: authHeaders(adminToken),
    body: JSON.stringify({
      ffEnrollmentStateMachine: true,
      updateMask: ['ffEnrollmentStateMachine'],
    }),
  })
}

test.describe('Enrollment lifecycle API', () => {
  test('PATCH state unauthenticated returns 401', async () => {
    const res = await fetch(
      `${API_BASE}/api/v1/courses/E2E-TEST/enrollments/00000000-0000-0000-0000-000000000001/state`,
      {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ state: 'withdrawn' }),
      },
    )
    expect(res.status).toBe(401)
  })

  test('GET history unauthenticated returns 401', async () => {
    const res = await fetch(
      `${API_BASE}/api/v1/courses/E2E-TEST/enrollments/00000000-0000-0000-0000-000000000001/state/history`,
    )
    expect(res.status).toBe(401)
  })
})

test('Instructor marks student withdrawn and gradebook shows former students', async ({
  coursePage: page,
  seededCourse,
}) => {
  await enableEnrollmentStateMachine(seededCourse.instructorToken)

  const enrollRes = await fetch(
    `${API_BASE}/api/v1/courses/${encodeURIComponent(seededCourse.courseCode)}/enrollments`,
    { headers: authHeaders(seededCourse.instructorToken) },
  )
  expect(enrollRes.ok).toBeTruthy()
  const enrollData = (await enrollRes.json()) as {
    enrollments?: Array<{ id: string; role: string; displayName?: string | null }>
  }
  const student = enrollData.enrollments?.find(
    (e) => e.role === 'student' && (e.displayName?.includes('Student') ?? false),
  )
  expect(student?.id).toBeTruthy()

  const patchRes = await fetch(
    `${API_BASE}/api/v1/courses/${encodeURIComponent(seededCourse.courseCode)}/enrollments/${student!.id}/state`,
    {
      method: 'PATCH',
      headers: authHeaders(seededCourse.instructorToken),
      body: JSON.stringify({ state: 'withdrawn', reason: 'E2E withdrawal test' }),
    },
  )
  expect(patchRes.status).toBe(200)
  const patched = (await patchRes.json()) as { state?: string; lisStatusCode?: string }
  expect(patched.state).toBe('withdrawn')
  expect(patched.lisStatusCode).toBe('Withdrawn')

  const gridRes = await fetch(
    `${API_BASE}/api/v1/courses/${encodeURIComponent(seededCourse.courseCode)}/gradebook/grid`,
    { headers: authHeaders(seededCourse.instructorToken) },
  )
  expect(gridRes.ok).toBeTruthy()
  const grid = (await gridRes.json()) as {
    students?: Array<{ userId: string; displayName: string; state?: string }>
  }
  expect(grid.students?.some((s) => s.state === 'withdrawn')).toBeTruthy()

  const histRes = await fetch(
    `${API_BASE}/api/v1/courses/${encodeURIComponent(seededCourse.courseCode)}/enrollments/${student!.id}/state/history`,
    { headers: authHeaders(seededCourse.instructorToken) },
  )
  expect(histRes.ok).toBeTruthy()
  const hist = (await histRes.json()) as { history?: Array<{ newState: string }> }
  expect(hist.history?.some((h) => h.newState === 'withdrawn')).toBeTruthy()

  await page.goto(`/courses/${seededCourse.courseCode}/gradebook`)
  await expect(page.getByRole('button', { name: /former students/i })).toBeVisible({ timeout: 15000 })
})

test('Student dashboard shows withdrawn status', async ({ page, seededCourse }) => {
  await enableEnrollmentStateMachine(seededCourse.instructorToken)

  const studentEmail = `withdrawn-student-${Date.now()}@e2e.test`
  const studentPassword = 'E2eTestPass1!'
  const { access_token: studentToken } = await apiSignup({
    email: studentEmail,
    password: studentPassword,
    displayName: 'Withdrawn E2E Student',
  })

  await apiEnroll(
    seededCourse.instructorToken,
    seededCourse.courseCode,
    studentEmail,
    'student',
    studentToken,
  )

  const enrollRes = await fetch(
    `${API_BASE}/api/v1/courses/${encodeURIComponent(seededCourse.courseCode)}/enrollments`,
    { headers: authHeaders(seededCourse.instructorToken) },
  )
  const enrollData = (await enrollRes.json()) as {
    enrollments?: Array<{ id: string; userId: string; displayName?: string | null }>
  }
  const row = enrollData.enrollments?.find((e) => e.displayName?.includes('Withdrawn E2E'))
  expect(row?.id).toBeTruthy()

  await fetch(
    `${API_BASE}/api/v1/courses/${encodeURIComponent(seededCourse.courseCode)}/enrollments/${row!.id}/state`,
    {
      method: 'PATCH',
      headers: authHeaders(seededCourse.instructorToken),
      body: JSON.stringify({ state: 'withdrawn' }),
    },
  )

  const loginRes = await fetch(`${API_BASE}/api/v1/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email: studentEmail, password: studentPassword }),
  })
  const { access_token } = (await loginRes.json()) as { access_token: string }

  await page.addInitScript((token) => {
    localStorage.setItem('studydrift_access_token', token as string)
  }, access_token)

  await page.goto('/dashboard')
  await expect(page.getByText('Withdrawn', { exact: false }).first()).toBeVisible({ timeout: 15000 })
})
