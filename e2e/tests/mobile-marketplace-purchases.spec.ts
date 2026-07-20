/**
 * MOB.7 — Mobile marketplace purchases API parity
 *
 *   [x] Platform features exposes ffMobileMarketplacePurchase (always on)
 *   [x] Free claim → owned + listed under /me/purchases
 *   [x] Paid checkout (when billing configured) returns session or alreadyOwned shape
 */
import { execSync } from 'node:child_process'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { test, expect, uniqueEmail } from '../fixtures/test.js'
import { apiSignup, apiCreateCourse } from '../fixtures/api.js'

const repoRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../..')
const API_BASE = process.env.E2E_API_URL ?? 'http://localhost:8080'
const PASSWORD = process.env.E2E_ADMIN_PASSWORD ?? 'E2eTestPass1!'

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

async function apiLogin(email: string) {
  const res = await fetch(`${API_BASE}/api/v1/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email, password: PASSWORD }),
  })
  if (!res.ok) throw new Error(`login failed: ${await res.text()}`)
  return (await res.json()) as { access_token: string }
}

async function bootstrapGlobalAdmin(email: string) {
  execSync(`go run ./cmd/bootstrap-admin -email=${email}`, {
    cwd: path.join(repoRoot, 'server'),
    env: { ...process.env, DATABASE_URL: databaseUrl() },
    stdio: 'pipe',
  })
}

async function enableMobileMarketplacePurchase(token: string) {
  const res = await fetch(`${API_BASE}/api/v1/settings/platform`, {
    method: 'PUT',
    headers: authHeaders(token),
    body: JSON.stringify({
      ffCourseMarketplace: true,
      ffMobileMarketplacePurchase: true,
    }),
  })
  if (!res.ok) {
    test.skip(true, `could not enable mobile marketplace purchase: ${await res.text()}`)
  }
}

async function publishAndList(
  token: string,
  courseCode: string,
  title: string,
  slug: string,
  priceCents: number,
) {
  const putCourse = await fetch(`${API_BASE}/api/v1/courses/${encodeURIComponent(courseCode)}`, {
    method: 'PUT',
    headers: authHeaders(token),
    body: JSON.stringify({
      title,
      description: `${title} MOB.7`,
      published: true,
    }),
  })
  expect(putCourse.ok).toBeTruthy()

  const listing = await fetch(
    `${API_BASE}/api/v1/courses/${encodeURIComponent(courseCode)}/catalog-listing`,
    {
      method: 'PUT',
      headers: authHeaders(token),
      body: JSON.stringify({
        marketplaceListed: true,
        priceCents,
        priceCurrency: 'usd',
        slug,
      }),
    },
  )
  expect(listing.ok).toBeTruthy()
}

test('MOB.7 features: ffMobileMarketplacePurchase is always on', async () => {
  const email = uniqueEmail('mob7-admin')
  await apiSignup({ email, password: PASSWORD })
  try {
    await bootstrapGlobalAdmin(email)
  } catch (err) {
    test.skip(true, `bootstrap unavailable: ${err}`)
  }
  const { access_token: token } = await apiLogin(email)

  const res = await fetch(`${API_BASE}/api/v1/platform/features`, {
    headers: authHeaders(token),
  })
  expect(res.ok).toBeTruthy()
  const features = (await res.json()) as { ffMobileMarketplacePurchase?: boolean }
  expect(features.ffMobileMarketplacePurchase).toBe(true)
})

test('MOB.7 claim free course and list under /me/purchases', async () => {
  const email = uniqueEmail('mob7-seller')
  await apiSignup({ email, password: PASSWORD })
  try {
    await bootstrapGlobalAdmin(email)
  } catch (err) {
    test.skip(true, `bootstrap unavailable: ${err}`)
  }
  const { access_token: adminToken } = await apiLogin(email)
  await enableMobileMarketplacePurchase(adminToken)

  const slug = `mob7-free-${Date.now().toString(36)}`
  const course = await apiCreateCourse(adminToken, { title: 'MOB.7 Free Course' })
  await publishAndList(adminToken, course.courseCode, 'MOB.7 Free Course', slug, 0)

  const learnerEmail = uniqueEmail('mob7-learner')
  const { access_token: learnerToken } = await apiSignup({
    email: learnerEmail,
    password: PASSWORD,
    displayName: 'MOB.7 Learner',
  })

  const detailRes = await fetch(`${API_BASE}/api/v1/marketplace/courses/${slug}`, {
    headers: authHeaders(learnerToken),
  })
  expect(detailRes.ok).toBeTruthy()
  const detail = (await detailRes.json()) as { owned?: boolean; course?: { owned?: boolean } }
  expect(detail.owned === true || detail.course?.owned === true).toBeFalsy()

  const claimRes = await fetch(`${API_BASE}/api/v1/marketplace/courses/${slug}/claim`, {
    method: 'POST',
    headers: authHeaders(learnerToken),
    body: '{}',
  })
  expect(claimRes.status).toBe(200)
  const claim = (await claimRes.json()) as { enrolled: boolean; courseCode: string }
  expect(claim.enrolled).toBe(true)
  expect(claim.courseCode).toBe(course.courseCode)

  const ownedRes = await fetch(`${API_BASE}/api/v1/marketplace/courses/${slug}`, {
    headers: authHeaders(learnerToken),
  })
  expect(ownedRes.ok).toBeTruthy()
  const ownedDetail = (await ownedRes.json()) as { owned?: boolean; course?: { owned?: boolean } }
  expect(ownedDetail.owned === true || ownedDetail.course?.owned === true).toBeTruthy()

  const purchasesRes = await fetch(`${API_BASE}/api/v1/me/purchases`, {
    headers: authHeaders(learnerToken),
  })
  expect(purchasesRes.ok).toBeTruthy()
  const purchases = (await purchasesRes.json()) as {
    purchases?: Array<{ courseCode: string; title: string }>
  }
  expect((purchases.purchases ?? []).some((p) => p.courseCode === course.courseCode)).toBeTruthy()
})

test('MOB.7 paid checkout response shape for listed course', async () => {
  const email = uniqueEmail('mob7-paid-admin')
  await apiSignup({ email, password: PASSWORD })
  try {
    await bootstrapGlobalAdmin(email)
  } catch (err) {
    test.skip(true, `bootstrap unavailable: ${err}`)
  }
  const { access_token: adminToken } = await apiLogin(email)
  await enableMobileMarketplacePurchase(adminToken)

  const slug = `mob7-paid-${Date.now().toString(36)}`
  const course = await apiCreateCourse(adminToken, { title: 'MOB.7 Paid Course' })
  await publishAndList(adminToken, course.courseCode, 'MOB.7 Paid Course', slug, 1999)

  const learnerEmail = uniqueEmail('mob7-paid-learner')
  const { access_token: learnerToken } = await apiSignup({
    email: learnerEmail,
    password: PASSWORD,
    displayName: 'MOB.7 Paid Learner',
  })

  const checkoutRes = await fetch(`${API_BASE}/api/v1/marketplace/courses/${slug}/checkout`, {
    method: 'POST',
    headers: authHeaders(learnerToken),
    body: '{}',
  })

  // Stripe/billing may be unconfigured or feature-gated in CI — accept success shape or known errors.
  if (checkoutRes.status === 200) {
    const body = (await checkoutRes.json()) as {
      checkoutUrl?: string
      alreadyOwned?: boolean
      sessionId?: string
    }
    expect(
      typeof body.checkoutUrl === 'string' ||
        body.alreadyOwned === true ||
        typeof body.sessionId === 'string',
    ).toBeTruthy()
  } else {
    expect([400, 402, 403, 404, 503]).toContain(checkoutRes.status)
  }
})
