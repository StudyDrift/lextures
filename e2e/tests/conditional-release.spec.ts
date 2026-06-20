/**
 * Conditional release & module requirements (plan 1.11)
 */
import { test, expect } from '@playwright/test'
import {
  apiSignup,
  apiCreateCourse,
  apiCreateModule,
  apiCreateContentPage,
  apiEnroll,
} from '../fixtures/api.js'

const apiBase = process.env.E2E_API_URL ?? 'http://localhost:8080'
const PASSWORD = 'E2eTestPass1!'

function uid(prefix = 'cr') {
  return `e2e-${prefix}-${Date.now()}-${Math.random().toString(36).slice(2)}`
}
function authHeaders(token: string) {
  return { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' }
}

async function getAdminToken(): Promise<string> {
  const adminEmail = process.env.E2E_ADMIN_EMAIL ?? 'admin@e2e.test'
  const adminPassword = process.env.E2E_ADMIN_PASSWORD ?? PASSWORD
  const loginRes = await fetch(`${apiBase}/api/v1/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email: adminEmail, password: adminPassword }),
  })
  if (loginRes.ok) {
    const { access_token } = (await loginRes.json()) as { access_token: string }
    return access_token
  }
  const { access_token } = await apiSignup({
    email: adminEmail,
    password: adminPassword,
    displayName: 'E2E Admin',
  })
  return access_token
}

async function enableConditionalRelease(adminToken: string): Promise<boolean> {
  const res = await fetch(`${apiBase}/api/v1/settings/platform`, {
    method: 'PUT',
    headers: authHeaders(adminToken),
    body: JSON.stringify({
      ffConditionalRelease: true,
      updateMask: ['ffConditionalRelease'],
    }),
  })
  return res.ok
}

test.describe('Conditional release', () => {
  test('module prerequisite blocks content until prior module requirement met', async () => {
    const adminToken = await getAdminToken()
    const enabled = await enableConditionalRelease(adminToken)
    if (!enabled) {
      test.skip(true, 'Could not enable ff_conditional_release (no platform admin)')
      return
    }

    const instructor = await apiSignup({
      email: `${uid('inst')}@e2e.test`,
      password: PASSWORD,
      displayName: 'Instructor',
    })
    const studentEmail = `${uid('stu')}@e2e.test`
    const student = await apiSignup({
      email: studentEmail,
      password: PASSWORD,
      displayName: 'Student',
    })

    const course = await apiCreateCourse(instructor.access_token, { title: 'Gating 101' })
    const modA = await apiCreateModule(instructor.access_token, course.courseCode, 'Module A')
    const modB = await apiCreateModule(instructor.access_token, course.courseCode, 'Module B')
    const itemA = await apiCreateContentPage(
      instructor.access_token,
      course.courseCode,
      modA.id,
      'Lesson A',
    )
    const itemB = await apiCreateContentPage(
      instructor.access_token,
      course.courseCode,
      modB.id,
      'Lesson B',
    )

    const reqRes = await fetch(
      `${apiBase}/api/v1/courses/${encodeURIComponent(course.courseCode)}/structure/modules/${encodeURIComponent(modB.id)}/requirements`,
      {
        method: 'PUT',
        headers: authHeaders(instructor.access_token),
        body: JSON.stringify({
          completionMode: 'all_items',
          prerequisiteModuleIds: [modA.id],
        }),
      },
    )
    expect(reqRes.ok).toBeTruthy()

    const ruleRes = await fetch(
      `${apiBase}/api/v1/courses/${encodeURIComponent(course.courseCode)}/items/${encodeURIComponent(itemA.id)}/completion-rule`,
      {
        method: 'PUT',
        headers: authHeaders(instructor.access_token),
        body: JSON.stringify({ ruleType: 'must_view' }),
      },
    )
    expect(ruleRes.ok).toBeTruthy()

    await apiEnroll(instructor.access_token, course.courseCode, studentEmail, 'student', student.access_token)

    const blocked = await fetch(
      `${apiBase}/api/v1/courses/${encodeURIComponent(course.courseCode)}/content-pages/${encodeURIComponent(itemB.id)}`,
      { headers: authHeaders(student.access_token) },
    )
    expect(blocked.status).toBe(403)

    const viewA = await fetch(
      `${apiBase}/api/v1/courses/${encodeURIComponent(course.courseCode)}/content-pages/${encodeURIComponent(itemA.id)}`,
      { headers: authHeaders(student.access_token) },
    )
    expect(viewA.ok).toBeTruthy()

    const unlocked = await fetch(
      `${apiBase}/api/v1/courses/${encodeURIComponent(course.courseCode)}/content-pages/${encodeURIComponent(itemB.id)}`,
      { headers: authHeaders(student.access_token) },
    )
    expect(unlocked.ok).toBeTruthy()
  })
})
