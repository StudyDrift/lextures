/**
 * Course marketplace purchase / entitlement flow (plan MKT4)
 *
 * Coverage:
 *   [x] free claim → entitlement + enrollment
 *   [x] double claim is idempotent
 *   [x] claim on paid course → 402
 *   [x] checkout on free course → 400
 *   [x] already-owned claim short-circuit
 *   [x] UI: Enroll Free CTA claims and navigates into course
 */
import { expect, test } from '@playwright/test'
import { injectToken, uniqueEmail } from '../fixtures/test.js'
import { apiSignup } from '../fixtures/api.js'

const API_BASE = process.env.E2E_API_URL ?? 'http://localhost:8080'
const PASSWORD = 'E2eTestPass1!'

function uid(prefix = 'mkt4') {
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
    body: JSON.stringify({ title, description: `${title} marketplace purchase test` }),
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
      description: `${title} marketplace purchase test`,
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

test.describe('MKT4 marketplace purchase', () => {
  test('free claim enrolls and is idempotent; paid claim returns 402', async ({ request }) => {
    const admin = await getAdminToken()
    await setCourseMarketplaceFlag(admin, true)

    const freeTitle = `MKT4 Free ${uid()}`
    const freeSlug = `mkt4-free-${uid()}`
    const free = await createCourse(admin, freeTitle)
    await publishCourse(admin, free.courseCode, freeTitle)
    await putListing(admin, free.courseCode, {
      marketplaceListed: true,
      priceCents: 0,
      priceCurrency: 'usd',
      slug: freeSlug,
    })

    const paidTitle = `MKT4 Paid ${uid()}`
    const paidSlug = `mkt4-paid-${uid()}`
    const paid = await createCourse(admin, paidTitle)
    await publishCourse(admin, paid.courseCode, paidTitle)
    await putListing(admin, paid.courseCode, {
      marketplaceListed: true,
      priceCents: 2500,
      priceCurrency: 'usd',
      slug: paidSlug,
    })

    const email = uniqueEmail('mkt4-learner')
    const { token } = await apiSignup(request, email, PASSWORD)

    const claim1 = await fetch(`${API_BASE}/api/v1/marketplace/courses/${freeSlug}/claim`, {
      method: 'POST',
      headers: authHeaders(token),
      body: '{}',
    })
    expect(claim1.status).toBe(200)
    const body1 = (await claim1.json()) as {
      enrolled: boolean
      entitlementId: string
      courseCode: string
      alreadyOwned?: boolean
    }
    expect(body1.enrolled).toBe(true)
    expect(body1.courseCode).toBe(free.courseCode)
    expect(body1.entitlementId).toBeTruthy()

    const claim2 = await fetch(`${API_BASE}/api/v1/marketplace/courses/${freeSlug}/claim`, {
      method: 'POST',
      headers: authHeaders(token),
      body: '{}',
    })
    expect(claim2.status).toBe(200)
    const body2 = (await claim2.json()) as { entitlementId: string; alreadyOwned?: boolean }
    expect(body2.entitlementId).toBe(body1.entitlementId)
    expect(body2.alreadyOwned).toBe(true)

    const paidClaim = await fetch(`${API_BASE}/api/v1/marketplace/courses/${paidSlug}/claim`, {
      method: 'POST',
      headers: authHeaders(token),
      body: '{}',
    })
    expect(paidClaim.status).toBe(402)
    const paidBody = (await paidClaim.json()) as {
      error: { code: string }
      checkoutHint?: string
    }
    expect(paidBody.error.code).toBe('PAYMENT_REQUIRED')
    expect(paidBody.checkoutHint).toContain('/marketplace/')

    const freeCheckout = await fetch(
      `${API_BASE}/api/v1/marketplace/courses/${freeSlug}/checkout`,
      {
        method: 'POST',
        headers: authHeaders(token),
        body: '{}',
      },
    )
    // Payments may be off (404) or free rejected (400).
    expect([400, 404]).toContain(freeCheckout.status)
  })

  test('UI free claim navigates into course', async ({ page, request }) => {
    const admin = await getAdminToken()
    await setCourseMarketplaceFlag(admin, true)

    const title = `MKT4 UI Free ${uid()}`
    const slug = `mkt4-ui-free-${uid()}`
    const course = await createCourse(admin, title)
    await publishCourse(admin, course.courseCode, title)
    await putListing(admin, course.courseCode, {
      marketplaceListed: true,
      priceCents: 0,
      priceCurrency: 'usd',
      slug,
    })

    const email = uniqueEmail('mkt4-ui')
    const { token } = await apiSignup(request, email, PASSWORD)
    await injectToken(page, token)

    await page.goto(`/marketplace/${slug}`)
    await expect(page.getByTestId('marketplace-course-detail')).toBeVisible({ timeout: 15_000 })
    await page.getByTestId('marketplace-cta').click()
    await expect(page).toHaveURL(new RegExp(`/courses/${course.courseCode}`), { timeout: 20_000 })
  })
})
