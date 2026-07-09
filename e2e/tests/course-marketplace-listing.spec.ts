/**
 * Course marketplace listing settings (plan MKT2)
 *
 * Coverage:
 *   [x] list a published course with free default
 *   [x] set paid price and reload persists
 *   [x] draft course rejected when listing (422)
 *   [x] negative price rejected (400)
 */
import { expect, test } from '@playwright/test'

const API_BASE = process.env.E2E_API_URL ?? 'http://localhost:8080'
const PASSWORD = 'E2eTestPass1!'

function uid(prefix = 'mkt2') {
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

async function createCourse(token: string, title: string): Promise<string> {
  const res = await fetch(`${API_BASE}/api/v1/courses`, {
    method: 'POST',
    headers: authHeaders(token),
    body: JSON.stringify({ title, description: `${title} marketplace test` }),
  })
  expect(res.ok).toBeTruthy()
  const body = (await res.json()) as { courseCode: string }
  return body.courseCode
}

async function publishCourse(token: string, courseCode: string, title: string) {
  const res = await fetch(`${API_BASE}/api/v1/courses/${encodeURIComponent(courseCode)}`, {
    method: 'PUT',
    headers: authHeaders(token),
    body: JSON.stringify({ title, description: `${title} marketplace test`, published: true }),
  })
  expect(res.ok).toBeTruthy()
}

type CatalogListing = {
  marketplaceListed: boolean
  priceCents: number
  priceCurrency: string
  publishState: string
}

async function getListing(token: string, courseCode: string): Promise<CatalogListing> {
  const res = await fetch(`${API_BASE}/api/v1/courses/${encodeURIComponent(courseCode)}/catalog-listing`, {
    headers: authHeaders(token),
  })
  expect(res.ok).toBeTruthy()
  const body = (await res.json()) as { listing: CatalogListing }
  return body.listing
}

async function putListing(
  token: string,
  courseCode: string,
  patch: Record<string, unknown>,
  expectOk = true,
): Promise<Response> {
  const existing = await getListing(token, courseCode)
  const res = await fetch(`${API_BASE}/api/v1/courses/${encodeURIComponent(courseCode)}/catalog-listing`, {
    method: 'PUT',
    headers: authHeaders(token),
    body: JSON.stringify({
      isPublic: false,
      category: null,
      difficultyLevel: null,
      language: 'en',
      priceCents: existing.priceCents,
      priceCurrency: existing.priceCurrency,
      slug: '',
      ...patch,
    }),
  })
  if (expectOk) expect(res.ok).toBeTruthy()
  return res
}

test('Marketplace listing: list free course, set price, draft rejected', async () => {
  const token = await getAdminToken()
  await setCourseMarketplaceFlag(token, true)

  const title = `Marketplace Course ${uid()}`
  const courseCode = await createCourse(token, title)

  const draftListing = await getListing(token, courseCode)
  expect(draftListing.publishState).toBe('draft')
  expect(draftListing.marketplaceListed).toBe(false)
  expect(draftListing.priceCents).toBe(0)

  const draftRes = await putListing(
    token,
    courseCode,
    { marketplaceListed: true },
    false,
  )
  expect(draftRes.status).toBe(422)
  const draftErr = (await draftRes.json()) as { error?: { message?: string } }
  expect(draftErr.error?.message).toContain('Publish the course')

  await publishCourse(token, courseCode, title)
  await putListing(token, courseCode, { marketplaceListed: true, priceCents: 0, priceCurrency: 'usd' })

  const listed = await getListing(token, courseCode)
  expect(listed.marketplaceListed).toBe(true)
  expect(listed.priceCents).toBe(0)
  expect(listed.priceCurrency).toBe('usd')

  await putListing(token, courseCode, { marketplaceListed: true, priceCents: 1999, priceCurrency: 'usd' })
  const paid = await getListing(token, courseCode)
  expect(paid.priceCents).toBe(1999)

  const negRes = await putListing(token, courseCode, { priceCents: -1 }, false)
  expect(negRes.status).toBe(400)

  await putListing(token, courseCode, { marketplaceListed: false })
  const unlisted = await getListing(token, courseCode)
  expect(unlisted.marketplaceListed).toBe(false)
})
