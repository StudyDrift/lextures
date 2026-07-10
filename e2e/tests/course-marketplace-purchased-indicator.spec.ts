/**
 * Course marketplace purchased indicator + My purchases (plan MKT5)
 *
 * Coverage:
 *   [x] free claim → Courses list acquiredViaMarketplace + badge
 *   [x] instructor-added course has no purchased badge
 *   [x] GET /api/v1/me/purchases lists active acquisitions
 *   [x] My purchases page renders rows
 */
import { expect, test } from '@playwright/test'
import { injectToken, uniqueEmail } from '../fixtures/test.js'
import { apiEnroll, apiSignup } from '../fixtures/api.js'

const API_BASE = process.env.E2E_API_URL ?? 'http://localhost:8080'
const PASSWORD = 'E2eTestPass1!'

function uid(prefix = 'mkt5') {
  return `${prefix}-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`
}

function authHeaders(token: string) {
  return { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' }
}

async function getAdminToken(): Promise<string> {
  const adminEmail = process.env.E2E_ADMIN_EMAIL ?? 'admin@e2e.test'
  const adminPassword = process.env.E2E_ADMIN_PASSWORD ?? PASSWORD
  const login = await fetch(`${API_BASE}/api/v1/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email: adminEmail, password: adminPassword }),
  })
  if (login.ok) {
    const { access_token } = (await login.json()) as { access_token: string }
    return access_token
  }
  await fetch(`${API_BASE}/api/v1/auth/signup`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email: adminEmail, password: adminPassword, display_name: 'E2E Admin' }),
  })
  const retry = await fetch(`${API_BASE}/api/v1/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email: adminEmail, password: adminPassword }),
  })
  const { access_token } = (await retry.json()) as { access_token: string }
  return access_token
}

async function setCourseMarketplaceFlag(token: string, on: boolean) {
  const res = await fetch(`${API_BASE}/api/v1/settings/platform`, {
    method: 'PUT',
    headers: authHeaders(token),
    body: JSON.stringify({ ffCourseMarketplace: on, updateMask: ['ffCourseMarketplace'] }),
  })
  expect(res.ok).toBeTruthy()
}

async function createCourse(token: string, title: string): Promise<{ courseCode: string; id: string }> {
  const res = await fetch(`${API_BASE}/api/v1/courses`, {
    method: 'POST',
    headers: authHeaders(token),
    body: JSON.stringify({ title, description: `${title} marketplace purchased indicator test` }),
  })
  expect(res.ok).toBeTruthy()
  return (await res.json()) as { courseCode: string; id: string }
}

async function publishCourse(token: string, courseCode: string, title: string) {
  const res = await fetch(`${API_BASE}/api/v1/courses/${encodeURIComponent(courseCode)}`, {
    method: 'PUT',
    headers: authHeaders(token),
    body: JSON.stringify({
      title,
      description: `${title} marketplace purchased indicator test`,
      published: true,
    }),
  })
  expect(res.ok).toBeTruthy()
}

async function putListing(
  token: string,
  courseCode: string,
  patch: Record<string, unknown>,
) {
  const res = await fetch(
    `${API_BASE}/api/v1/courses/${encodeURIComponent(courseCode)}/catalog-listing`,
    {
      method: 'PUT',
      headers: authHeaders(token),
      body: JSON.stringify(patch),
    },
  )
  expect(res.ok).toBeTruthy()
}

test.describe('MKT5 purchased indicator', () => {
  test('claim free course → badge on Courses + My purchases', async ({ page }) => {
    const admin = await getAdminToken()
    await setCourseMarketplaceFlag(admin, true)

    const title = `MKT5 Free ${uid()}`
    const slug = `mkt5-free-${uid()}`
    const course = await createCourse(admin, title)
    await publishCourse(admin, course.courseCode, title)
    await putListing(admin, course.courseCode, {
      marketplaceListed: true,
      priceCents: 0,
      priceCurrency: 'usd',
      slug,
    })

    const learnerEmail = uniqueEmail('mkt5-learner')
    const { access_token: token } = await apiSignup({
      email: learnerEmail,
      password: PASSWORD,
      displayName: 'MKT5 Learner',
    })

    const claim = await fetch(`${API_BASE}/api/v1/marketplace/courses/${slug}/claim`, {
      method: 'POST',
      headers: authHeaders(token),
      body: '{}',
    })
    expect(claim.status).toBe(200)

    const list = await fetch(`${API_BASE}/api/v1/courses`, { headers: authHeaders(token) })
    expect(list.ok).toBeTruthy()
    const listBody = (await list.json()) as {
      courses: Array<{
        id: string
        acquiredViaMarketplace?: boolean
        acquisitionSource?: string | null
      }>
    }
    const row = listBody.courses.find((c) => c.id === course.id)
    expect(row?.acquiredViaMarketplace).toBe(true)
    expect(row?.acquisitionSource).toBe('free')

    const purchases = await fetch(`${API_BASE}/api/v1/me/purchases`, {
      headers: authHeaders(token),
    })
    expect(purchases.ok).toBeTruthy()
    const purchasesBody = (await purchases.json()) as {
      purchases: Array<{ courseId: string; source: string; receiptUrl?: string }>
    }
    expect(purchasesBody.purchases.some((p) => p.courseId === course.id && p.source === 'free')).toBe(
      true,
    )

    await injectToken(page, token)
    await page.goto('/courses')
    await expect(page.getByTestId('course-purchased-badge').first()).toBeVisible()

    await page.goto('/me/purchases')
    await expect(page.getByTestId('my-purchase-row').first()).toBeVisible()
    await expect(page.getByText(title)).toBeVisible()
  })

  test('instructor-added course has no purchased badge', async ({ page }) => {
    const admin = await getAdminToken()
    await setCourseMarketplaceFlag(admin, true)

    const title = `MKT5 Added ${uid()}`
    const course = await createCourse(admin, title)
    await publishCourse(admin, course.courseCode, title)

    const learnerEmail = uniqueEmail('mkt5-added')
    const { access_token: token } = await apiSignup({
      email: learnerEmail,
      password: PASSWORD,
      displayName: 'MKT5 Added',
    })

    await apiEnroll(admin, course.courseCode, learnerEmail, 'student', { memberToken: token })

    const list = await fetch(`${API_BASE}/api/v1/courses`, { headers: authHeaders(token) })
    expect(list.ok).toBeTruthy()
    const listBody = (await list.json()) as {
      courses: Array<{ id: string; acquiredViaMarketplace?: boolean }>
    }
    const row = listBody.courses.find((c) => c.id === course.id)
    expect(row).toBeTruthy()
    expect(row?.acquiredViaMarketplace).toBeFalsy()

    await injectToken(page, token)
    await page.goto('/courses')
    await expect(page.getByRole('heading', { name: title, exact: true })).toBeVisible()
    await expect(page.getByTestId('course-purchased-badge')).toHaveCount(0)
  })
})
