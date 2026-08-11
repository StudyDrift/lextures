/**
 * Creator coupon manager UI (plan MKTC.4)
 *
 * Coverage:
 *   [x] Coupons panel absent when flag off
 *   [x] Creator creates a percent coupon and sees it in the table
 *   [x] Copy share link writes the expected URL
 *   [x] Pause flips status badge
 *   [x] Archive hides from default view
 */
import { expect, test, type Page } from '@playwright/test'

const API_BASE = process.env.E2E_API_URL ?? 'http://localhost:8080'
const WEB_BASE = process.env.E2E_WEB_URL ?? 'http://localhost:5173'
const PASSWORD = 'E2eTestPass1!'

function uid(prefix = 'mktc4') {
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
    body: JSON.stringify({ title, description: `${title} coupons test` }),
  })
  expect(res.ok).toBeTruthy()
  const body = (await res.json()) as { courseCode: string }
  return body.courseCode
}

async function publishCourse(token: string, courseCode: string, title: string) {
  const res = await fetch(`${API_BASE}/api/v1/courses/${encodeURIComponent(courseCode)}`, {
    method: 'PUT',
    headers: authHeaders(token),
    body: JSON.stringify({ title, description: `${title} coupons test`, published: true }),
  })
  expect(res.ok).toBeTruthy()
}

async function putListing(token: string, courseCode: string, priceCents: number) {
  const get = await fetch(
    `${API_BASE}/api/v1/courses/${encodeURIComponent(courseCode)}/catalog-listing`,
    { headers: authHeaders(token) },
  )
  expect(get.ok).toBeTruthy()
  const { listing } = (await get.json()) as {
    listing: {
      isPublic: boolean
      category: string | null
      difficultyLevel: string | null
      language: string
      priceCents: number
      priceCurrency: string
      slug: string
      marketplaceListed: boolean
    }
  }
  const res = await fetch(
    `${API_BASE}/api/v1/courses/${encodeURIComponent(courseCode)}/catalog-listing`,
    {
      method: 'PUT',
      headers: authHeaders(token),
      body: JSON.stringify({
        isPublic: listing.isPublic,
        category: listing.category,
        difficultyLevel: listing.difficultyLevel,
        language: listing.language || 'en',
        priceCents,
        priceCurrency: 'usd',
        slug: listing.slug || courseCode.toLowerCase(),
        marketplaceListed: true,
      }),
    },
  )
  expect(res.ok).toBeTruthy()
}

async function loginAsAdmin(page: Page, token: string) {
  await page.goto(`${WEB_BASE}/`)
  await page.evaluate((t) => {
    localStorage.setItem('access_token', t)
    localStorage.setItem('token', t)
  }, token)
}

test.describe('MKTC.4 creator coupon manager', () => {
  test('panel is absent when ffCourseCoupons is off', async ({ page }) => {
    const token = await getAdminToken()
    await setFlags(token, { ffCourseMarketplace: true, ffCourseCoupons: false })
    const title = `Coupons Off ${uid()}`
    const code = await createCourse(token, title)
    await publishCourse(token, code, title)
    await putListing(token, code, 4000)
    await loginAsAdmin(page, token)
    await page.goto(`${WEB_BASE}/courses/${encodeURIComponent(code)}/settings/features`)
    await expect(page.getByRole('heading', { name: /Marketplace/i })).toBeVisible({ timeout: 30_000 })
    await expect(page.getByRole('heading', { name: /Coupon codes/i })).toHaveCount(0)
  })

  test('create percent coupon, copy share link, pause and archive', async ({ page, context }) => {
    await context.grantPermissions(['clipboard-read', 'clipboard-write'])
    const token = await getAdminToken()
    await setFlags(token, { ffCourseMarketplace: true, ffCourseCoupons: true })
    const title = `Coupons On ${uid()}`
    const courseCode = await createCourse(token, title)
    await publishCourse(token, courseCode, title)
    await putListing(token, courseCode, 4000)
    await loginAsAdmin(page, token)
    await page.goto(`${WEB_BASE}/courses/${encodeURIComponent(courseCode)}/settings/features`)

    await expect(page.getByRole('heading', { name: /Coupon codes/i })).toBeVisible({ timeout: 30_000 })
    await page.getByRole('button', { name: /New coupon/i }).first().click()

    const couponCode = `SAVE${Math.floor(Math.random() * 9000 + 1000)}`
    await page.getByLabel(/^Code$/i).fill(couponCode)
    await page.getByLabel(/Percent off/i).fill('25')
    await page.getByRole('button', { name: /Create coupon/i }).click()

    await expect(page.getByText(couponCode.toUpperCase())).toBeVisible({ timeout: 15_000 })

    await page.getByRole('button', { name: new RegExp(`Copy share link for ${couponCode}`, 'i') }).click()
    const clip = await page.evaluate(() => navigator.clipboard.readText())
    expect(clip).toContain(`coupon=${couponCode.toUpperCase()}`)
    expect(clip).toMatch(/marketplace\//)

    await page.getByRole('button', { name: new RegExp(`Actions for ${couponCode}`, 'i') }).click()
    await page.getByRole('menuitem', { name: /Pause/i }).click()
    await expect(page.getByText(/Paused/i).first()).toBeVisible({ timeout: 10_000 })

    await page.getByRole('button', { name: new RegExp(`Actions for ${couponCode}`, 'i') }).click()
    await page.getByRole('menuitem', { name: /Archive/i }).click()
    await page.getByRole('button', { name: /^Archive$/i }).click()
    await expect(page.getByText(couponCode.toUpperCase())).toHaveCount(0, { timeout: 10_000 })
  })
})
