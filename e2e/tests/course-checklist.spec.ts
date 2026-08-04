/**
 * Course Checklist state API (CC.2) — API-level Playwright suite.
 *
 * Checklist coverage:
 *   [x] Unauthenticated GET /checklist returns 401
 *   [x] Student GET /checklist returns 403
 *   [x] Teacher GET /checklist returns 200 with summary
 *   [x] Teacher dismisses item → badge decrements → restore → badge increments
 */

import { test, expect } from '@playwright/test'
import { apiSignup, apiCreateCourse, apiEnroll } from '../fixtures/api.js'

const API_BASE = process.env.E2E_API_URL ?? 'http://localhost:8080'
const PASSWORD = 'E2eTestPass1!'

let _seq = 0
function uniqueEmail(prefix = 'cc2') {
  return `e2e-${prefix}-${Date.now()}-${++_seq}@test.invalid`
}

test('Course checklist: unauthenticated GET returns 401', async () => {
  const res = await fetch(`${API_BASE}/api/v1/courses/C-FAKE/checklist`)
  expect(res.status).toBe(401)
})

test('Course checklist: student receives 403; teacher dismiss/restore updates badge', async () => {
  const teacherEmail = uniqueEmail('cc2-teacher')
  const studentEmail = uniqueEmail('cc2-student')
  const { access_token: teacherTok } = await apiSignup({ email: teacherEmail, password: PASSWORD })
  const { access_token: studentTok } = await apiSignup({ email: studentEmail, password: PASSWORD })
  const course = await apiCreateCourse(teacherTok, { title: 'CC2 Checklist Course' })
  await apiEnroll(teacherTok, course.courseCode, teacherEmail, 'teacher')
  await apiEnroll(teacherTok, course.courseCode, studentEmail, 'student')

  const studentRes = await fetch(`${API_BASE}/api/v1/courses/${course.courseCode}/checklist`, {
    headers: { Authorization: `Bearer ${studentTok}` },
  })
  expect(studentRes.status).toBe(403)
  const studentBody = await studentRes.text()
  expect(studentBody).not.toContain('Set course start')

  const listRes = await fetch(`${API_BASE}/api/v1/courses/${course.courseCode}/checklist`, {
    headers: { Authorization: `Bearer ${teacherTok}` },
  })
  expect(listRes.status).toBe(200)
  const list = await listRes.json()
  expect(list.engineVersion).toBeGreaterThanOrEqual(1)
  expect(typeof list.catalogVersion).toBe('string')
  const beforeEssential = list.summary.outstandingEssential as number
  expect(beforeEssential).toBeGreaterThanOrEqual(1)

  const dismissRes = await fetch(
    `${API_BASE}/api/v1/courses/${course.courseCode}/checklist/items/course.dates/dismiss`,
    {
      method: 'POST',
      headers: {
        Authorization: `Bearer ${teacherTok}`,
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ reason: 'not_applicable', note: 'e2e dismiss' }),
    },
  )
  expect(dismissRes.status).toBe(200)

  const summaryAfterDismiss = await fetch(
    `${API_BASE}/api/v1/courses/${course.courseCode}/checklist/summary`,
    { headers: { Authorization: `Bearer ${teacherTok}` } },
  )
  expect(summaryAfterDismiss.status).toBe(200)
  const summary1 = await summaryAfterDismiss.json()
  expect(summary1.dismissed).toBe(1)
  expect(summary1.outstandingEssential).toBe(beforeEssential - 1)

  const restoreRes = await fetch(
    `${API_BASE}/api/v1/courses/${course.courseCode}/checklist/items/course.dates/restore`,
    {
      method: 'POST',
      headers: { Authorization: `Bearer ${teacherTok}` },
    },
  )
  expect(restoreRes.status).toBe(200)

  const summaryAfterRestore = await fetch(
    `${API_BASE}/api/v1/courses/${course.courseCode}/checklist/summary`,
    { headers: { Authorization: `Bearer ${teacherTok}` } },
  )
  const summary2 = await summaryAfterRestore.json()
  expect(summary2.dismissed).toBe(0)
  expect(summary2.outstandingEssential).toBe(beforeEssential)
})
