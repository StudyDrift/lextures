/**
 * Course marketplace discovery / storefront (plan MKT3)
 *
 * Coverage:
 *   [x] unauthenticated list rejected (401)
 *   [x] listed free + paid courses appear; draft listed course excluded
 *   [x] free_only filter
 *   [x] detail 404 for non-listed course
 *   [x] owned overlay via entitlement
 *   [x] flag off → 404
 *   [x] UI: sidenav → storefront → filter → detail → Buy CTA handoff
 *   [x] UI: flag off shows not-available (no sidenav link)
 */
import { execFileSync } from 'node:child_process'
import { expect, test } from '@playwright/test'
import { injectToken, mainNav, uniqueEmail } from '../fixtures/test.js'
import { apiSignup } from '../fixtures/api.js'

const API_BASE = process.env.E2E_API_URL ?? 'http://localhost:8080'
const PASSWORD = 'E2eTestPass1!'

function uid(prefix = 'mkt3') {
  return `${prefix}-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`
}

function authHeaders(token: string) {
  return { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' }
}

function databaseUrl(): string {
  return (
    process.env.DATABASE_URL ??
    process.env.E2E_DATABASE_URL ??
    'postgres://studydrift:studydrift@localhost:5432/studydrift?sslmode=disable'
  )
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
    body: JSON.stringify({ title, description: `${title} marketplace storefront test` }),
  })
  expect(res.ok).toBeTruthy()
  const body = (await res.json()) as { courseCode: string; id: string }
  return body
}

async function publishCourse(token: string, courseCode: string, title: string) {
  const res = await fetch(`${API_BASE}/api/v1/courses/${encodeURIComponent(courseCode)}`, {
    method: 'PUT',
    headers: authHeaders(token),
    body: JSON.stringify({
      title,
      description: `${title} marketplace storefront test`,
      published: true,
    }),
  })
  expect(res.ok).toBeTruthy()
}

async function putListing(
  token: string,
  courseCode: string,
  patch: Record<string, unknown>,
): Promise<{ slug: string }> {
  const getRes = await fetch(
    `${API_BASE}/api/v1/courses/${encodeURIComponent(courseCode)}/catalog-listing`,
    { headers: authHeaders(token) },
  )
  expect(getRes.ok).toBeTruthy()
  const existing = (await getRes.json()) as {
    listing: { priceCents: number; priceCurrency: string; slug: string }
  }
  const res = await fetch(
    `${API_BASE}/api/v1/courses/${encodeURIComponent(courseCode)}/catalog-listing`,
    {
      method: 'PUT',
      headers: authHeaders(token),
      body: JSON.stringify({
        isPublic: false,
        category: patch.category ?? 'Computer Science',
        difficultyLevel: patch.difficultyLevel ?? 'beginner',
        language: 'en',
        priceCents: existing.listing.priceCents,
        priceCurrency: existing.listing.priceCurrency,
        slug: patch.slug ?? existing.listing.slug ?? '',
        marketplaceListed: false,
        ...patch,
      }),
    },
  )
  expect(res.ok).toBeTruthy()
  const body = (await res.json()) as { listing: { slug: string } }
  return body.listing
}

function grantCoursePurchase(userEmail: string, courseId: string) {
  const sql = `
INSERT INTO billing.user_entitlements (
  user_id, entitlement_type, course_id, amount_paid_cents, currency, status, acquisition_source
)
SELECT u.id, 'course_purchase', '${courseId}'::uuid, 0, 'usd', 'active', 'free'
FROM "user".users u
WHERE lower(u.email) = lower('${userEmail.replace(/'/g, "''")}')
ON CONFLICT DO NOTHING;
`
  execFileSync('psql', [databaseUrl(), '-v', 'ON_ERROR_STOP=1', '-c', sql], { stdio: 'pipe' })
}

test.describe('Marketplace storefront — API', () => {
  test('rejects unauthenticated list', async () => {
    const admin = await getAdminToken()
    await setCourseMarketplaceFlag(admin, true)
    const res = await fetch(`${API_BASE}/api/v1/marketplace/courses`)
    expect(res.status).toBe(401)
  })

  test('lists free and paid courses; excludes draft; free_only; detail; owned', async () => {
    const admin = await getAdminToken()
    await setCourseMarketplaceFlag(admin, true)

    const freeTitle = `MKT3 Free ${uid()}`
    const paidTitle = `MKT3 Paid ${uid()}`
    const draftTitle = `MKT3 Draft ${uid()}`
    const freeSlug = `mkt3-free-${uid()}`
    const paidSlug = `mkt3-paid-${uid()}`

    const free = await createCourse(admin, freeTitle)
    const paid = await createCourse(admin, paidTitle)
    const draft = await createCourse(admin, draftTitle)

    await publishCourse(admin, free.courseCode, freeTitle)
    await publishCourse(admin, paid.courseCode, paidTitle)

    await putListing(admin, free.courseCode, {
      marketplaceListed: true,
      priceCents: 0,
      priceCurrency: 'usd',
      slug: freeSlug,
      category: 'Computer Science',
      difficultyLevel: 'beginner',
    })
    await putListing(admin, paid.courseCode, {
      marketplaceListed: true,
      priceCents: 2000,
      priceCurrency: 'usd',
      slug: paidSlug,
      category: 'Computer Science',
      difficultyLevel: 'intermediate',
    })

    // Draft cannot be listed via API (422); force-list via SQL to prove storefront still excludes it.
    execFileSync(
      'psql',
      [
        databaseUrl(),
        '-v',
        'ON_ERROR_STOP=1',
        '-c',
        `UPDATE course.courses SET marketplace_listed = TRUE, marketplace_listed_at = NOW() WHERE id = '${draft.id}'::uuid;`,
      ],
      { stdio: 'pipe' },
    )

    const learnerEmail = uniqueEmail('mkt3-learner')
    const { access_token: learnerToken } = await apiSignup({
      email: learnerEmail,
      password: PASSWORD,
      displayName: 'MKT3 Learner',
    })

    const listRes = await fetch(`${API_BASE}/api/v1/marketplace/courses?q=${encodeURIComponent('MKT3')}`, {
      headers: authHeaders(learnerToken),
    })
    expect(listRes.status).toBe(200)
    const listBody = (await listRes.json()) as {
      courses: Array<{
        title: string
        slug: string
        priceCents: number
        owned: boolean
        courseCode: string
      }>
    }
    const titles = listBody.courses.map((c) => c.title)
    expect(titles).toContain(freeTitle)
    expect(titles).toContain(paidTitle)
    expect(titles).not.toContain(draftTitle)

    const freeCard = listBody.courses.find((c) => c.title === freeTitle)
    const paidCard = listBody.courses.find((c) => c.title === paidTitle)
    expect(freeCard?.priceCents).toBe(0)
    expect(paidCard?.priceCents).toBe(2000)
    expect(freeCard?.owned).toBe(false)

    const freeOnlyRes = await fetch(
      `${API_BASE}/api/v1/marketplace/courses?free_only=true&category=${encodeURIComponent('Computer Science')}`,
      { headers: authHeaders(learnerToken) },
    )
    expect(freeOnlyRes.status).toBe(200)
    const freeOnly = (await freeOnlyRes.json()) as { courses: Array<{ title: string; priceCents: number }> }
    expect(freeOnly.courses.some((c) => c.title === freeTitle)).toBe(true)
    expect(freeOnly.courses.every((c) => c.priceCents === 0)).toBe(true)
    expect(freeOnly.courses.some((c) => c.title === paidTitle)).toBe(false)

    const detailRes = await fetch(`${API_BASE}/api/v1/marketplace/courses/${encodeURIComponent(paidSlug)}`, {
      headers: authHeaders(learnerToken),
    })
    expect(detailRes.status).toBe(200)
    const detail = (await detailRes.json()) as {
      owned: boolean
      priceCents: number
      course: { title: string }
      whatsIncluded: { moduleCount: number; itemCount: number }
    }
    expect(detail.course.title).toBe(paidTitle)
    expect(detail.priceCents).toBe(2000)
    expect(detail.owned).toBe(false)
    expect(detail.whatsIncluded).toBeTruthy()

    const missingRes = await fetch(
      `${API_BASE}/api/v1/marketplace/courses/${encodeURIComponent(draft.courseCode)}`,
      { headers: authHeaders(learnerToken) },
    )
    expect(missingRes.status).toBe(404)

    grantCoursePurchase(learnerEmail, free.id)
    const ownedList = await fetch(
      `${API_BASE}/api/v1/marketplace/courses?q=${encodeURIComponent(freeTitle)}`,
      { headers: authHeaders(learnerToken) },
    )
    expect(ownedList.status).toBe(200)
    const ownedBody = (await ownedList.json()) as { courses: Array<{ title: string; owned: boolean }> }
    const ownedCard = ownedBody.courses.find((c) => c.title === freeTitle)
    expect(ownedCard?.owned).toBe(true)

    const ownedDetail = await fetch(
      `${API_BASE}/api/v1/marketplace/courses/${encodeURIComponent(freeSlug)}`,
      { headers: authHeaders(learnerToken) },
    )
    expect(ownedDetail.status).toBe(200)
    const ownedDetailBody = (await ownedDetail.json()) as { owned: boolean }
    expect(ownedDetailBody.owned).toBe(true)
  })

  test('flag off returns 404', async () => {
    const admin = await getAdminToken()
    await setCourseMarketplaceFlag(admin, false)
    try {
      const { access_token } = await apiSignup({
        email: uniqueEmail('mkt3-off'),
        password: PASSWORD,
      })
      const res = await fetch(`${API_BASE}/api/v1/marketplace/courses`, {
        headers: authHeaders(access_token),
      })
      expect(res.status).toBe(404)
    } finally {
      await setCourseMarketplaceFlag(admin, true)
    }
  })
})

test.describe('Marketplace storefront — UI', () => {
  test('sidenav → storefront → filter → detail → Buy CTA', async ({ page }) => {
    const admin = await getAdminToken()
    await setCourseMarketplaceFlag(admin, true)

    const paidTitle = `MKT3 UI Paid ${uid()}`
    const freeTitle = `MKT3 UI Free ${uid()}`
    const paidSlug = `mkt3-ui-paid-${uid()}`
    const freeSlug = `mkt3-ui-free-${uid()}`

    const paid = await createCourse(admin, paidTitle)
    const free = await createCourse(admin, freeTitle)
    await publishCourse(admin, paid.courseCode, paidTitle)
    await publishCourse(admin, free.courseCode, freeTitle)
    await putListing(admin, paid.courseCode, {
      marketplaceListed: true,
      priceCents: 2000,
      priceCurrency: 'usd',
      slug: paidSlug,
      category: 'Computer Science',
    })
    await putListing(admin, free.courseCode, {
      marketplaceListed: true,
      priceCents: 0,
      priceCurrency: 'usd',
      slug: freeSlug,
      category: 'Computer Science',
    })

    const learnerEmail = uniqueEmail('mkt3-ui')
    const { access_token } = await apiSignup({
      email: learnerEmail,
      password: PASSWORD,
      displayName: 'MKT3 UI Learner',
    })
    await injectToken(page, access_token)
    await expect(mainNav(page)).toBeVisible()

    const marketplaceNav = mainNav(page).getByRole('link', { name: /^marketplace$/i })
    await expect(marketplaceNav).toBeVisible()
    await marketplaceNav.click()
    await expect(page).toHaveURL(/\/marketplace/)

    await expect(page.getByTestId('marketplace-search')).toBeVisible()
    await page.getByTestId('marketplace-search').fill(paidTitle)
    await expect(page.getByRole('link', { name: new RegExp(paidTitle) })).toBeVisible({
      timeout: 15_000,
    })
    await expect(page.getByTestId('marketplace-price').first()).toContainText(/\$20\.00|Free/)

    await page.getByTestId('marketplace-filter-price').selectOption('free')
    await expect(page.getByRole('link', { name: new RegExp(freeTitle) })).toBeVisible({
      timeout: 15_000,
    })
    await expect(page.getByRole('link', { name: new RegExp(paidTitle) })).toHaveCount(0)

    await page.getByTestId('marketplace-filter-price').selectOption('any')
    await page.getByTestId('marketplace-search').fill(paidTitle)
    await page.getByRole('link', { name: new RegExp(paidTitle) }).click()
    await expect(page).toHaveURL(new RegExp(`/marketplace/${paidSlug}`))
    await expect(page.getByTestId('marketplace-course-detail')).toBeVisible()
    await expect(page.getByTestId('marketplace-cta')).toBeVisible()
    // Paid CTA starts Stripe checkout (or shows unavailable when Stripe is not configured).
    // Do not assert navigation to the old MKT3 stub route.
    await expect(page.getByTestId('marketplace-cta')).toContainText(/Buy|\$20/)
  })

  test('owned course shows Go to course', async ({ page }) => {
    const admin = await getAdminToken()
    await setCourseMarketplaceFlag(admin, true)

    const title = `MKT3 Owned ${uid()}`
    const slug = `mkt3-owned-${uid()}`
    const course = await createCourse(admin, title)
    await publishCourse(admin, course.courseCode, title)
    await putListing(admin, course.courseCode, {
      marketplaceListed: true,
      priceCents: 0,
      priceCurrency: 'usd',
      slug,
    })

    const learnerEmail = uniqueEmail('mkt3-owned')
    const { access_token } = await apiSignup({
      email: learnerEmail,
      password: PASSWORD,
    })
    grantCoursePurchase(learnerEmail, course.id)

    await injectToken(page, access_token)
    await page.goto(`/marketplace/${slug}`)
    await expect(page.getByTestId('marketplace-owned-badge')).toBeVisible({ timeout: 15_000 })
    await expect(page.getByTestId('marketplace-cta')).toHaveText(/go to course/i)
    await page.getByTestId('marketplace-cta').click()
    await expect(page).toHaveURL(new RegExp(`/courses/${course.courseCode}`))
  })

  test('flag off hides sidenav and shows not available', async ({ page }) => {
    const admin = await getAdminToken()
    await setCourseMarketplaceFlag(admin, false)
    try {
      const { access_token } = await apiSignup({
        email: uniqueEmail('mkt3-flagoff'),
        password: PASSWORD,
      })
      await injectToken(page, access_token)
      await expect(mainNav(page)).toBeVisible()
      await expect(mainNav(page).getByRole('link', { name: /^marketplace$/i })).toHaveCount(0)
      await page.goto('/marketplace')
      await expect(page.getByText(/marketplace is not enabled/i)).toBeVisible({ timeout: 15_000 })
    } finally {
      await setCourseMarketplaceFlag(admin, true)
    }
  })
})
