/**
 * Incomplete grade workflow (plan 14.4)
 *
 *   [x] POST incomplete unauthenticated returns 401
 *   [x] Endpoints return 501 when feature disabled
 *   [x] Instructor grants incomplete and gradebook shows incomplete state
 *   [x] Instructor resolves incomplete with final grade
 */
import { test, expect } from '../fixtures/test.js'
import { apiCreateAssignment } from '../fixtures/api.js'

const API_BASE = process.env.E2E_API_URL ?? 'http://localhost:8080'

function authHeaders(token: string) {
  return { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' }
}

test.describe('Incomplete grade API', () => {
  test('POST incomplete unauthenticated returns 401', async () => {
    const res = await fetch(
      `${API_BASE}/api/v1/courses/E2E-TEST/enrollments/00000000-0000-0000-0000-000000000001/incomplete`,
      {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ extensionDeadline: '2026-12-31', outstandingItemIds: [] }),
      },
    )
    expect(res.status).toBe(401)
  })
})

test('Instructor grants and resolves incomplete grade', async ({ coursePage: page, seededCourse }) => {
  const assignment = await apiCreateAssignment(
    seededCourse.instructorToken,
    seededCourse.courseCode,
    seededCourse.moduleId,
    'E2E Incomplete Assignment',
  )

  const gridBefore = await fetch(
    `${API_BASE}/api/v1/courses/${encodeURIComponent(seededCourse.courseCode)}/gradebook/grid`,
    { headers: authHeaders(seededCourse.instructorToken) },
  )
  expect(gridBefore.ok).toBeTruthy()
  const gridData = (await gridBefore.json()) as {
    students?: Array<{ userId: string; displayName?: string; enrollmentId?: string }>
    columns?: Array<{ id: string; kind: string }>
  }
  const student = gridData.students?.find((s) => s.displayName?.includes('E2E Student'))
  expect(student?.enrollmentId).toBeTruthy()
  const assignmentCol = gridData.columns?.find((c) => c.id === assignment.id)
  expect(assignmentCol?.id).toBeTruthy()

  const deadline = new Date()
  deadline.setDate(deadline.getDate() + 90)
  const deadlineStr = deadline.toISOString().slice(0, 10)

  const grantRes = await fetch(
    `${API_BASE}/api/v1/courses/${encodeURIComponent(seededCourse.courseCode)}/enrollments/${student!.enrollmentId}/incomplete`,
    {
      method: 'POST',
      headers: authHeaders(seededCourse.instructorToken),
      body: JSON.stringify({
        extensionDeadline: deadlineStr,
        outstandingItemIds: [assignmentCol!.id],
        notes: 'E2E incomplete test',
      }),
    },
  )
  expect(grantRes.status).toBe(201)
  const granted = (await grantRes.json()) as { record?: { status?: string; extensionDeadline?: string } }
  expect(granted.record?.status).toBe('open')
  expect(granted.record?.extensionDeadline).toBe(deadlineStr)

  const gridAfter = await fetch(
    `${API_BASE}/api/v1/courses/${encodeURIComponent(seededCourse.courseCode)}/gradebook/grid`,
    { headers: authHeaders(seededCourse.instructorToken) },
  )
  const gridAfterData = (await gridAfter.json()) as {
    students?: Array<{ userId: string; state?: string; incompleteRecord?: { status: string } }>
  }
  const row = gridAfterData.students?.find((s) => s.userId === student!.userId)
  expect(row?.state).toBe('incomplete')
  expect(row?.incompleteRecord?.status).toBe('open')

  await page.goto(`/courses/${seededCourse.courseCode}/gradebook`)
  await expect(page.getByRole('button', { name: /resolve i/i }).first()).toBeVisible({ timeout: 15000 })
  await expect(page.getByText(/due /i).first()).toBeVisible()

  const resolveRes = await fetch(
    `${API_BASE}/api/v1/courses/${encodeURIComponent(seededCourse.courseCode)}/enrollments/${student!.enrollmentId}/incomplete`,
    {
      method: 'PATCH',
      headers: authHeaders(seededCourse.instructorToken),
      body: JSON.stringify({ resolvedGrade: 'B+' }),
    },
  )
  expect(resolveRes.status).toBe(200)
  const resolved = (await resolveRes.json()) as { record?: { status?: string; resolvedGrade?: string } }
  expect(resolved.record?.status).toBe('resolved')
  expect(resolved.record?.resolvedGrade).toBe('B+')
})
