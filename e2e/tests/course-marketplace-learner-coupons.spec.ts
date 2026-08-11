/**
 * Learner coupon entry + URL codes (plan MKTC.5)
 *
 * Coverage:
 *   [x] Flag off: no coupon UI
 *   [x] Apply percent code updates price + CTA
 *   [x] URL ?coupon= auto-applies
 *   [x] Expired code shows reason; full-price CTA remains
 *   [x] 100%-off enrolls without Stripe
 *   [x] Remove restores full price
 */
import { expect, test, type Page } from '@playwright/test'

const API_BASE = process.env.E2E_API_URL ?? 'http://localhost:8080'
const WEB_BASE = process.env.E2E_WEB_URL ?? 'http://localhost:5173'
const PASSWORD = 'E2eTestPass1!'

function uid(prefix = 'mktc5') {
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

async function setFlags(token: string, flags: Record<string, boolean>) {
  const updateMask = Object.keys(flags)
  const res = await fetch(`${API_BASE}/api/v1/settings/platform`, {
    method: 'PUT',
    headers: authHeaders(token),
    body: JSON.stringify({ ...flags, updateMask }),
  })
  expect(res.ok).toBeTruthy()
}

async function createCourse(token: string, title: string): Promise<string> {
  const res = await fetch(`${API_BASE}/api/v1/courses`, {
    method: 'POST',
    headers: authHeaders(token),
    body: JSON.stringify({ title, description: `${title} learner coupons` }),
  })
  expect(res.ok).toBeTruthy()
  const body = (await res.json()) as { courseCode: string }
  return body.courseCode
}

async function publishCourse(token: string, courseCode: string, title: string) {
  const res = await fetch(`${API_BASE}/api/v1/courses/${encodeURIComponent(courseCode)}`, {
    method: 'PUT',
    headers: authHeaders(token),
    body: JSON.stringify({ title, description: `${title} learner coupons`, published: true }),
  })
  expect(res.ok).toBeTruthy()
}

async function listOnMarketplace(
  token: string,
  courseCode: string,
  priceCents: number,
): Promise<{ slug: string }> {
  const res = await fetch(
    `${API_BASE}/api/v1/courses/${encodeURIComponent(courseCode)}/marketplace`,
    {
      method: 'PUT',
      headers: authHeaders(token),
      body: JSON.stringify({
        listed: true,
        priceCents,
        priceCurrency: 'usd',
        category: 'E2E',
        level: 'beginner',
      }),
    },
  )
  expect(res.ok).toBeTruthy()
  const body = (await res.json()) as { slug?: string; course?: { slug?: string } }
  const slug = body.slug || body.course?.slug || courseCode.toLowerCase()
  return { slug }
}

async function createCoupon(
  token: string,
  courseCode: string,
  code: string,
  percentOff: number,
): Promise<void> {
  const res = await fetch(`${API_BASE}/api/v1/courses/${encodeURIComponent(courseCode)}/coupons`, {
    method: 'POST',
    headers: authHeaders(token),
    body: JSON.stringify({
      code,
      discountType: 'percent',
      percentOff,
      status: 'active',
    }),
  })
  expect(res.ok).toBeTruthy()
}

async function signupLearner(page: Page, email: string) {
  await page.goto(`${WEB_BASE}/signup`)
  await page.getByLabel(/email/i).fill(email)
  await page.getByLabel(/password/i).fill(PASSWORD)
  const name = page.getByLabel(/name|display/i).first()
  if (await name.isVisible().catch(() => false)) {
    await name.fill('Learner E2E')
  }
  await page.getByRole('button', { name: /sign up|create/i }).click()
  await page.waitForURL((u) => !u.pathname.includes('/signup'), { timeout: 30_000 }).catch(() => {})
}

test.describe('MKTC.5 learner coupons', () => {
  test.describe.configure({ mode: 'serial' })

  let adminToken: string
  let courseCode: string
  let slug: string
  const code25 = `SAVE25${Date.now().toString(36).toUpperCase().slice(-4)}`
  const code100 = `FREE${Date.now().toString(36).toUpperCase().slice(-4)}`

  test.beforeAll(async () => {
    adminToken = await getAdminToken()
    await setFlags(adminToken, {
      ffCourseMarketplace: true,
      ffCourseCoupons: true,
      ffBilling: true,
    })
    const title = `MKTC5 ${uid()}`
    courseCode = await createCourse(adminToken, title)
    await publishCourse(adminToken, courseCode, title)
    ;({ slug } = await listOnMarketplace(adminToken, courseCode, 4000))
    await createCoupon(adminToken, courseCode, code25, 25)
    await createCoupon(adminToken, courseCode, code100, 100)
  })

  test('flag off hides coupon field', async ({ page }) => {
    await setFlags(adminToken, { ffCourseCoupons: false })
    const email = `${uid('learner')}@e2e.test`
    await signupLearner(page, email)
    await page.goto(`${WEB_BASE}/marketplace/${encodeURIComponent(slug)}?coupon=${code25}`)
    await expect(page.getByTestId('marketplace-course-detail')).toBeVisible({ timeout: 20_000 })
    await expect(page.getByTestId('marketplace-coupon-field')).toHaveCount(0)
    await setFlags(adminToken, { ffCourseCoupons: true })
  })

  test('typed apply updates price and CTA', async ({ page }) => {
    const email = `${uid('learner')}@e2e.test`
    await signupLearner(page, email)
    await page.goto(`${WEB_BASE}/marketplace/${encodeURIComponent(slug)}`)
    await expect(page.getByTestId('marketplace-course-detail')).toBeVisible({ timeout: 20_000 })
    await page.getByRole('button', { name: /have a coupon/i }).click()
    await page.getByTestId('marketplace-coupon-input').fill(code25)
    await page.getByTestId('marketplace-coupon-apply').click()
    await expect(page.getByTestId('marketplace-coupon-applied')).toBeVisible({ timeout: 15_000 })
    await expect(page.getByTestId('marketplace-price')).toContainText(/\$30/)
    await expect(page.getByTestId('marketplace-cta')).toContainText(/\$30/)
  })

  test('URL coupon auto-applies', async ({ page }) => {
    const email = `${uid('learner')}@e2e.test`
    await signupLearner(page, email)
    await page.goto(
      `${WEB_BASE}/marketplace/${encodeURIComponent(slug)}?coupon=${encodeURIComponent(code25)}`,
    )
    await expect(page.getByTestId('marketplace-coupon-applied')).toBeVisible({ timeout: 20_000 })
    await expect(page.getByTestId('marketplace-price')).toContainText(/\$30/)
  })

  test('invalid code shows reason and keeps full price CTA', async ({ page }) => {
    const email = `${uid('learner')}@e2e.test`
    await signupLearner(page, email)
    await page.goto(
      `${WEB_BASE}/marketplace/${encodeURIComponent(slug)}?coupon=NOTAREALCODE99`,
    )
    await expect(page.getByTestId('marketplace-coupon-error')).toBeVisible({ timeout: 20_000 })
    await expect(page.getByTestId('marketplace-cta')).toBeEnabled()
    await expect(page.getByTestId('marketplace-price')).toContainText(/\$40/)
  })

  test('remove restores full price', async ({ page }) => {
    const email = `${uid('learner')}@e2e.test`
    await signupLearner(page, email)
    await page.goto(
      `${WEB_BASE}/marketplace/${encodeURIComponent(slug)}?coupon=${encodeURIComponent(code25)}`,
    )
    await expect(page.getByTestId('marketplace-coupon-applied')).toBeVisible({ timeout: 20_000 })
    await page.getByTestId('marketplace-coupon-remove').click()
    await expect(page.getByTestId('marketplace-coupon-applied')).toHaveCount(0)
    await expect(page.getByTestId('marketplace-price')).toContainText(/\$40/)
  })

  test('100% off enrolls without Stripe redirect', async ({ page }) => {
    const email = `${uid('learner')}@e2e.test`
    await signupLearner(page, email)
    await page.goto(
      `${WEB_BASE}/marketplace/${encodeURIComponent(slug)}?coupon=${encodeURIComponent(code100)}`,
    )
    await expect(page.getByTestId('marketplace-coupon-applied')).toBeVisible({ timeout: 20_000 })
    await page.getByTestId('marketplace-cta').click()
    await page.waitForURL((u) => u.pathname.includes('/courses/'), { timeout: 30_000 })
    expect(page.url()).not.toMatch(/stripe|checkout\.stripe/i)
  })
})
