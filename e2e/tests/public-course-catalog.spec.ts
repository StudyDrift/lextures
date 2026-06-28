/**
 * Public Course Catalog & Search (plan 15.1)
 *
 * Coverage:
 *   [x] unauthenticated browse of published public courses (AC-1)
 *   [x] full-text search returns the matching course (AC-2, relevance sort)
 *   [x] Free price filter (AC-4)
 *   [x] course landing detail returns Schema.org Course JSON-LD (AC-3)
 *   [x] draft/unpublished courses excluded from the public API without auth (AC-5)
 *   [x] feature flag off => endpoints return 404 (rollback path)
 *   [x] /explore page renders for a logged-out visitor (UI smoke)
 */
import { expect, test } from '@playwright/test'

const API_BASE = process.env.E2E_API_URL ?? 'http://localhost:8080'
const PASSWORD = 'E2eTestPass1!'

function uid(prefix = 'catalog') {
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

async function setPublicCatalogFlag(token: string, on: boolean) {
  const res = await fetch(`${API_BASE}/api/v1/settings/platform`, {
    method: 'PUT',
    headers: authHeaders(token),
    body: JSON.stringify({ ffPublicCatalog: on, updateMask: ['ffPublicCatalog'] }),
  })
  expect(res.ok).toBeTruthy()
}

async function createCourse(token: string, title: string): Promise<string> {
  const res = await fetch(`${API_BASE}/api/v1/courses`, {
    method: 'POST',
    headers: authHeaders(token),
    body: JSON.stringify({ title, description: `${title} — full description for the catalog.` }),
  })
  expect(res.ok).toBeTruthy()
  const body = (await res.json()) as { courseCode: string }
  return body.courseCode
}

async function publishCourse(token: string, courseCode: string, title: string) {
  const res = await fetch(`${API_BASE}/api/v1/courses/${encodeURIComponent(courseCode)}`, {
    method: 'PUT',
    headers: authHeaders(token),
    body: JSON.stringify({ title, description: `${title} — full description for the catalog.`, published: true }),
  })
  expect(res.ok).toBeTruthy()
}

async function setListing(
  token: string,
  courseCode: string,
  listing: { isPublic: boolean; category?: string; difficultyLevel?: string; priceCents?: number; slug?: string },
) {
  const res = await fetch(`${API_BASE}/api/v1/courses/${encodeURIComponent(courseCode)}/catalog-listing`, {
    method: 'PUT',
    headers: authHeaders(token),
    body: JSON.stringify({
      isPublic: listing.isPublic,
      category: listing.category ?? null,
      difficultyLevel: listing.difficultyLevel ?? null,
      language: 'en',
      priceCents: listing.priceCents ?? 0,
      slug: listing.slug ?? '',
    }),
  })
  expect(res.ok).toBeTruthy()
  return (await res.json()) as { listing: { slug: string } }
}

test('Public catalog: unauthenticated browse, search, filter, and JSON-LD', async ({ request }) => {
  const token = await getAdminToken()
  await setPublicCatalogFlag(token, true)

  const tag = uid()
  const publicTitle = `Astrophysics ${tag}`
  const draftTitle = `Hidden Draft ${tag}`

  // A published, public, free course.
  const publicCode = await createCourse(token, publicTitle)
  await publishCourse(token, publicCode, publicTitle)
  const slug = uid('astro')
  await setListing(token, publicCode, {
    isPublic: true,
    category: `Science ${tag}`,
    difficultyLevel: 'beginner',
    priceCents: 0,
    slug,
  })

  // A draft course that must never appear publicly.
  const draftCode = await createCourse(token, draftTitle)
  await setListing(token, draftCode, { isPublic: true, priceCents: 0 }) // public flag but not published

  // AC-1 + AC-2: anonymous search finds the published public course by title.
  const searchRes = await fetch(
    `${API_BASE}/api/v1/public/catalog/courses?q=${encodeURIComponent(publicTitle)}&sort=relevance`,
  )
  expect(searchRes.ok).toBeTruthy()
  expect(searchRes.headers.get('cache-control')).toContain('max-age=3600')
  const search = (await searchRes.json()) as {
    courses: Array<{ courseCode: string; slug: string; priceCents: number }>
    total: number
  }
  const found = search.courses.find((c) => c.courseCode === publicCode)
  expect(found).toBeTruthy()
  expect(found?.priceCents).toBe(0)

  // AC-5: the draft course is absent even though is_public was set (not published).
  const draftLeak = search.courses.find((c) => c.courseCode === draftCode)
  expect(draftLeak).toBeUndefined()

  // AC-4: Free filter keeps the $0 course.
  const freeRes = await fetch(`${API_BASE}/api/v1/public/catalog/courses?price_max=0&q=${encodeURIComponent(publicTitle)}`)
  const free = (await freeRes.json()) as { courses: Array<{ courseCode: string }> }
  expect(free.courses.some((c) => c.courseCode === publicCode)).toBeTruthy()

  // Categories taxonomy includes the new category.
  const catsRes = await fetch(`${API_BASE}/api/v1/public/catalog/categories`)
  const cats = (await catsRes.json()) as { categories: Array<{ category: string }> }
  expect(cats.categories.some((c) => c.category === `Science ${tag}`)).toBeTruthy()

  // AC-3: course landing detail carries valid Schema.org Course JSON-LD.
  const detailRes = await fetch(`${API_BASE}/api/v1/public/catalog/courses/${encodeURIComponent(slug)}`)
  expect(detailRes.ok).toBeTruthy()
  const detail = (await detailRes.json()) as {
    course: { title: string }
    jsonLd: { '@type': string; name: string; provider: { name: string }; offers: { price: string } }
  }
  expect(detail.course.title).toBe(publicTitle)
  expect(detail.jsonLd['@type']).toBe('Course')
  expect(detail.jsonLd.name).toBe(publicTitle)
  expect(detail.jsonLd.provider.name).toBe('Lextures')
  expect(detail.jsonLd.offers.price).toBe('0.00')

  // The dedicated SSR JSON-LD endpoint returns the same document type.
  const ldRes = await fetch(`${API_BASE}/api/v1/internal/catalog/courses/${encodeURIComponent(slug)}/json-ld`)
  expect(ldRes.ok).toBeTruthy()
  const ld = (await ldRes.json()) as { '@type': string }
  expect(ld['@type']).toBe('Course')
})

test('Public catalog: feature flag off returns 404', async () => {
  const token = await getAdminToken()
  await setPublicCatalogFlag(token, false)
  try {
    const res = await fetch(`${API_BASE}/api/v1/public/catalog/courses`)
    expect(res.status).toBe(404)
  } finally {
    await setPublicCatalogFlag(token, true)
  }
})

test('Public catalog: /explore renders for a logged-out visitor', async ({ page }) => {
  const token = await getAdminToken()
  await setPublicCatalogFlag(token, true)

  await page.goto('/explore')
  await expect(page.getByRole('heading', { name: 'Explore courses' })).toBeVisible()
  await expect(page.getByPlaceholder(/Search courses/i)).toBeVisible()
  // No redirect to the login wall.
  expect(page.url()).toContain('/explore')
})
