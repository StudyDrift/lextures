/**
 * At-risk / early-warning alerts (plan 9.2)
 *
 *   [x] GET at-risk without auth returns 401
 *   [x] GET at-risk when feature disabled returns 404
 *   [x] Student cannot access at-risk (403)
 *   [x] Instructor runs scoring job, sees alert, dismisses (feature on)
 */
import { test, expect } from '../fixtures/test.js'
import { apiSignup, apiLogin } from '../fixtures/api.js'
import { isAtRiskEnabled } from '../fixtures/platform-features.js'

const API_BASE = process.env.E2E_API_URL ?? 'http://localhost:8080'
const PASSWORD = 'E2eTestPass1!'

function uniqueEmail(prefix = 'atrisk') {
  return `e2e-${prefix}-${Date.now()}-${Math.random().toString(36).slice(2)}@test.invalid`
}

test('GET at-risk: unauthenticated returns 401', async () => {
  const res = await fetch(`${API_BASE}/api/v1/courses/demo/at-risk`)
  expect(res.status).toBe(401)
})

test('GET at-risk: feature off returns 404', async () => {
  if (await isAtRiskEnabled()) {
    test.skip(true, 'skipped when at-risk alerts enabled')
  }
  const { access_token } = await apiSignup({ email: uniqueEmail(), password: PASSWORD })
  const res = await fetch(`${API_BASE}/api/v1/courses/any-course/at-risk`, {
    headers: { Authorization: `Bearer ${access_token}` },
  })
  expect(res.status).toBe(404)
})

test('GET at-risk: student returns 403', async ({ seededCourse }) => {
  if (!(await isAtRiskEnabled())) {
    test.skip(true, 'requires at-risk alerts enabled')
  }
  const res = await fetch(
    `${API_BASE}/api/v1/courses/${encodeURIComponent(seededCourse.courseCode)}/at-risk`,
    { headers: { Authorization: `Bearer ${seededCourse.studentToken}` } },
  )
  expect(res.status).toBe(403)
})

test('At-risk tab: scoring, list, dismiss', async ({ coursePage: page, seededCourse }) => {
  if (!(await isAtRiskEnabled())) {
    test.skip(true, 'requires at-risk alerts enabled')
  }

  const adminEmail = process.env.E2E_ADMIN_EMAIL ?? 'admin@e2e.test'
  const adminPassword = process.env.E2E_ADMIN_PASSWORD ?? PASSWORD
  let adminToken: string
  try {
    ;({ access_token: adminToken } = await apiLogin({
      email: adminEmail,
      password: adminPassword,
    }))
  } catch {
    test.skip(true, 'bootstrap admin not available')
    return
  }

  const runRes = await fetch(`${API_BASE}/api/v1/admin/at-risk/run`, {
    method: 'POST',
    headers: {
      Authorization: `Bearer ${adminToken}`,
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ courseCode: seededCourse.courseCode }),
  })
  expect(runRes.ok).toBeTruthy()

  await page.goto(`/courses/${encodeURIComponent(seededCourse.courseCode)}/at-risk`)
  await expect(page.getByRole('heading', { name: /at-risk students/i })).toBeVisible()

  const listRes = await fetch(
    `${API_BASE}/api/v1/courses/${encodeURIComponent(seededCourse.courseCode)}/at-risk`,
    { headers: { Authorization: `Bearer ${seededCourse.instructorToken}` } },
  )
  expect(listRes.ok).toBeTruthy()
  const body = (await listRes.json()) as { alerts: { id: string; displayName: string }[] }
  if (body.alerts.length === 0) {
    await expect(page.getByText(/no at-risk students|on track/i)).toBeVisible()
    return
  }

  const first = body.alerts[0]
  await expect(page.getByText(first.displayName)).toBeVisible()

  const dismissRes = await fetch(
    `${API_BASE}/api/v1/courses/${encodeURIComponent(seededCourse.courseCode)}/at-risk/${first.id}`,
    {
      method: 'PATCH',
      headers: {
        Authorization: `Bearer ${seededCourse.instructorToken}`,
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ status: 'dismissed' }),
    },
  )
  expect(dismissRes.ok).toBeTruthy()

  await page.reload()
  const after = await fetch(
    `${API_BASE}/api/v1/courses/${encodeURIComponent(seededCourse.courseCode)}/at-risk`,
    { headers: { Authorization: `Bearer ${seededCourse.instructorToken}` } },
  )
  const afterBody = (await after.json()) as { alerts: { id: string }[] }
  expect(afterBody.alerts.find((a) => a.id === first.id)).toBeUndefined()
})
