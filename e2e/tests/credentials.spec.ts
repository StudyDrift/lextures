/**
 * Completion certificates / Open Badges (plan 15.5)
 */
import { test, expect } from '@playwright/test'
import { injectToken } from '../fixtures/test.js'
import {
  apiSignup,
  apiCreateCourse,
  apiCreateModule,
  apiCreateContentPage,
} from '../fixtures/api.js'

const apiBase = process.env.E2E_API_URL ?? 'http://localhost:8080'
const PASSWORD = 'E2eTestPass1!'

function uid(prefix = 'cred') {
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

async function enableCredentials(adminToken: string): Promise<boolean> {
  const res = await fetch(`${apiBase}/api/v1/settings/platform`, {
    method: 'PUT',
    headers: authHeaders(adminToken),
    body: JSON.stringify({
      ffSelfPacedMode: true,
      ffCompletionCredentials: true,
      updateMask: ['ffSelfPacedMode', 'ffCompletionCredentials'],
    }),
  })
  return res.ok
}

test.describe('Credentials — API auth', () => {
  test('GET /api/v1/me/credentials returns 401 without auth', async () => {
    const res = await fetch(`${apiBase}/api/v1/me/credentials`)
    expect(res.status).toBe(401)
  })

  test('GET credential verify endpoint is public (not 401)', async () => {
    const res = await fetch(`${apiBase}/api/v1/credentials/00000000-0000-0000-0000-000000000099/verify`)
    expect(res.status).not.toBe(401)
  })
})

test.describe('Credentials — issuance flow', () => {
  test('learner earns verifiable credential after course completion', async ({ page }) => {
    const adminToken = await getAdminToken()
    const enabled = await enableCredentials(adminToken)
    if (!enabled) {
      test.skip(true, 'Could not enable ff_completion_credentials')
      return
    }

    const creator = await apiSignup({ email: `${uid('creator')}@e2e.test`, password: PASSWORD, displayName: 'Creator' })
    const course = await apiCreateCourse(creator.access_token, { title: 'Certificate Course' })
    const mod = await apiCreateModule(creator.access_token, course.courseCode, 'Unit 1')
    const item = await apiCreateContentPage(creator.access_token, course.courseCode, mod.id, 'Lesson 1')

    await fetch(`${apiBase}/api/v1/courses/${encodeURIComponent(course.courseCode)}`, {
      method: 'PUT',
      headers: authHeaders(creator.access_token),
      body: JSON.stringify({ title: 'Certificate Course', courseMode: 'self_paced', openEnrollment: true }),
    })

    const learner = await apiSignup({ email: `${uid('learner')}@e2e.test`, password: PASSWORD, displayName: 'Learner' })
    await fetch(`${apiBase}/api/v1/courses/${encodeURIComponent(course.courseCode)}/self-enroll`, {
      method: 'POST',
      headers: authHeaders(learner.access_token),
    })

    const compRes = await fetch(
      `${apiBase}/api/v1/courses/${encodeURIComponent(course.courseCode)}/items/${encodeURIComponent(item.id)}/complete`,
      { method: 'POST', headers: authHeaders(learner.access_token) },
    )
    expect(compRes.ok).toBeTruthy()
    const compBody = (await compRes.json()) as { justCompleted?: boolean; credentialId?: string }
    expect(compBody.justCompleted).toBe(true)

    const listRes = await fetch(`${apiBase}/api/v1/me/credentials`, {
      headers: authHeaders(learner.access_token),
    })
    if (listRes.status === 404) {
      test.skip(true, 'ff_completion_credentials not enabled')
    }
    expect(listRes.status).toBe(200)
    const listBody = (await listRes.json()) as { credentials: { id: string; title: string }[] }
    expect(listBody.credentials.length).toBeGreaterThanOrEqual(1)

    const credentialId = compBody.credentialId ?? listBody.credentials[0]?.id
    expect(credentialId).toBeTruthy()

    const verifyRes = await fetch(`${apiBase}/api/v1/credentials/${credentialId}/verify`)
    expect(verifyRes.ok).toBeTruthy()
    const verifyBody = (await verifyRes.json()) as { valid: boolean; status: string; learnerName?: string }
    expect(verifyBody.valid).toBe(true)
    expect(verifyBody.status).toBe('Valid')

    const pdfRes = await fetch(`${apiBase}/api/v1/credentials/${credentialId}/download`, {
      headers: authHeaders(learner.access_token),
    })
    expect(pdfRes.ok).toBeTruthy()
    expect(pdfRes.headers.get('content-type')).toContain('application/pdf')

    await injectToken(page, learner.access_token)
    await page.goto('/me/credentials')
    await expect(page.getByRole('heading', { name: /my credentials/i })).toBeVisible({ timeout: 10_000 })
    await expect(page.getByLabel(/certificate: certificate course/i)).toBeVisible()
  })
})