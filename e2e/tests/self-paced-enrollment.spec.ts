/**
 * Self-paced enrollment with no instructor (plan 15.2)
 *
 * Checklist coverage:
 *   [x] Admin enables ff_self_paced_mode; creator marks a course self_paced + open_enrollment
 *   [x] Learner self-enrolls in one call with no instructor action (AC-1)
 *   [x] Progress reports 0% before any items, then 100% after completing the only item (AC-2, AC-5)
 *   [x] Completing the final item signals course completion (justCompleted)
 *   [x] Module gating blocks completing a later module before the first is done (AC-3)
 *   [x] A learner cannot mark items complete in a course they are not enrolled in (security)
 *   [x] Learner sees the accessible progress bar on the modules page
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

function uid(prefix = 'sp') {
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

async function enableSelfPaced(adminToken: string): Promise<boolean> {
  const res = await fetch(`${apiBase}/api/v1/settings/platform`, {
    method: 'PUT',
    headers: authHeaders(adminToken),
    body: JSON.stringify({
      ffSelfPacedMode: true,
      updateMask: ['ffSelfPacedMode'],
    }),
  })
  return res.ok
}

async function setSelfPaced(
  token: string,
  courseCode: string,
  title: string,
  opts: { openEnrollment?: boolean; moduleGatingEnabled?: boolean } = {},
): Promise<Response> {
  return fetch(`${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}`, {
    method: 'PUT',
    headers: authHeaders(token),
    body: JSON.stringify({
      title,
      courseMode: 'self_paced',
      openEnrollment: opts.openEnrollment ?? true,
      moduleGatingEnabled: opts.moduleGatingEnabled ?? false,
    }),
  })
}

async function selfEnroll(token: string, courseCode: string): Promise<Response> {
  return fetch(`${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/self-enroll`, {
    method: 'POST',
    headers: authHeaders(token),
  })
}

async function getProgress(token: string, courseCode: string) {
  const res = await fetch(`${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/my-progress`, {
    headers: authHeaders(token),
  })
  return { status: res.status, body: res.ok ? await res.json() : null }
}

async function completeItem(token: string, courseCode: string, itemId: string): Promise<Response> {
  return fetch(
    `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/items/${encodeURIComponent(itemId)}/complete`,
    { method: 'POST', headers: authHeaders(token) },
  )
}

test.describe('Self-paced enrollment', () => {
  test('learner self-enrolls, progresses, and completes without an instructor', async ({ page }) => {
    const adminToken = await getAdminToken()
    const enabled = await enableSelfPaced(adminToken)
    if (!enabled) {
      test.skip(true, 'Could not enable ff_self_paced_mode (no platform admin)')
      return
    }

    // Creator builds a one-item self-paced course.
    const creator = await apiSignup({ email: `${uid('creator')}@e2e.test`, password: PASSWORD, displayName: 'Creator' })
    const course = await apiCreateCourse(creator.access_token, { title: 'Self-Paced 101' })
    const mod = await apiCreateModule(creator.access_token, course.courseCode, 'Unit 1')
    const item = await apiCreateContentPage(creator.access_token, course.courseCode, mod.id, 'Lesson 1')
    const putRes = await setSelfPaced(creator.access_token, course.courseCode, 'Self-Paced 101')
    expect(putRes.ok).toBeTruthy()

    // Learner self-enrolls in one click — no instructor approval (AC-1).
    const learner = await apiSignup({ email: `${uid('learner')}@e2e.test`, password: PASSWORD, displayName: 'Learner' })
    const enrollRes = await selfEnroll(learner.access_token, course.courseCode)
    expect(enrollRes.ok).toBeTruthy()
    const enrollBody = (await enrollRes.json()) as { enrolled: boolean; firstItemId?: string }
    expect(enrollBody.enrolled).toBeTruthy()
    expect(enrollBody.firstItemId).toBe(item.id)

    // 0% before completing anything (AC-2).
    const before = await getProgress(learner.access_token, course.courseCode)
    expect(before.status).toBe(200)
    expect(before.body.progressPercent).toBe(0)
    expect(before.body.totalItems).toBe(1)
    expect(before.body.completed).toBe(false)

    // Completing the only item reaches 100% and signals completion (AC-5).
    const compRes = await completeItem(learner.access_token, course.courseCode, item.id)
    expect(compRes.ok).toBeTruthy()
    const compBody = (await compRes.json()) as {
      progressPercent: number
      completed: boolean
      justCompleted?: boolean
    }
    expect(compBody.progressPercent).toBe(100)
    expect(compBody.completed).toBe(true)
    expect(compBody.justCompleted).toBe(true)

    // Learner sees the accessible progress bar on the modules page.
    await injectToken(page, learner.access_token)
    await page.goto(`/courses/${course.courseCode}/modules`)
    const bar = page.getByRole('progressbar', { name: /complete/i })
    await expect(bar).toBeVisible()
    await expect(bar).toHaveAttribute('aria-valuenow', '100')
  })

  test('module gating blocks a later module until the first is complete', async () => {
    const adminToken = await getAdminToken()
    const enabled = await enableSelfPaced(adminToken)
    if (!enabled) {
      test.skip(true, 'Could not enable ff_self_paced_mode (no platform admin)')
      return
    }

    const creator = await apiSignup({ email: `${uid('creator')}@e2e.test`, password: PASSWORD, displayName: 'Creator' })
    const course = await apiCreateCourse(creator.access_token, { title: 'Gated Course' })
    const mod1 = await apiCreateModule(creator.access_token, course.courseCode, 'Module 1')
    const item1 = await apiCreateContentPage(creator.access_token, course.courseCode, mod1.id, 'M1 Lesson')
    const mod2 = await apiCreateModule(creator.access_token, course.courseCode, 'Module 2')
    const item2 = await apiCreateContentPage(creator.access_token, course.courseCode, mod2.id, 'M2 Lesson')
    const putRes = await setSelfPaced(creator.access_token, course.courseCode, 'Gated Course', {
      moduleGatingEnabled: true,
    })
    expect(putRes.ok).toBeTruthy()

    const learner = await apiSignup({ email: `${uid('learner')}@e2e.test`, password: PASSWORD, displayName: 'Learner' })
    expect((await selfEnroll(learner.access_token, course.courseCode)).ok).toBeTruthy()

    // Module 2 is locked until Module 1 is complete (AC-3).
    const blocked = await completeItem(learner.access_token, course.courseCode, item2.id)
    expect(blocked.status).toBe(403)

    // Complete Module 1, which unlocks Module 2.
    expect((await completeItem(learner.access_token, course.courseCode, item1.id)).ok).toBeTruthy()
    const unblocked = await completeItem(learner.access_token, course.courseCode, item2.id)
    expect(unblocked.ok).toBeTruthy()
  })

  test('a non-enrolled learner cannot mark items complete', async () => {
    const adminToken = await getAdminToken()
    const enabled = await enableSelfPaced(adminToken)
    if (!enabled) {
      test.skip(true, 'Could not enable ff_self_paced_mode (no platform admin)')
      return
    }

    const creator = await apiSignup({ email: `${uid('creator')}@e2e.test`, password: PASSWORD, displayName: 'Creator' })
    const course = await apiCreateCourse(creator.access_token, { title: 'Secure Course' })
    const mod = await apiCreateModule(creator.access_token, course.courseCode, 'Unit 1')
    const item = await apiCreateContentPage(creator.access_token, course.courseCode, mod.id, 'Lesson 1')
    expect((await setSelfPaced(creator.access_token, course.courseCode, 'Secure Course')).ok).toBeTruthy()

    const stranger = await apiSignup({ email: `${uid('stranger')}@e2e.test`, password: PASSWORD, displayName: 'Stranger' })
    const res = await completeItem(stranger.access_token, course.courseCode, item.id)
    expect(res.status).toBe(403)
  })
})
