/**
 * Bookstore / Textbook linking (plan 14.11)
 *
 *   [x] GET inclusive-access unauthenticated returns 401
 *   [x] POST textbook-resources unauthenticated returns 401
 *   [x] GET admin bookstore config unauthenticated returns 401
 *   [x] Instructor creates a textbook resource module item (AC-1 content item)
 *   [x] Textbook resource round-trips provider + metadata (ISBN, chapter)
 *   [x] PATCH updates textbook metadata
 *   [x] Recording a launch event returns 204 and stores no PII (AC-5)
 *   [x] Launch events list includes provider, excludes user identity
 *   [x] Instructor configures Inclusive Access; GET reflects opt-out URL (AC-2)
 *   [x] Inclusive Access POST without required fields returns 400
 *   [x] Inclusive Access POST with invalid provider returns 400
 *   [x] Admin bookstore config GET/POST round-trips default provider (AC-4 prep)
 *   [x] Invalid provider on textbook resource returns 400
 */
import { test, expect } from '@playwright/test'
import { apiSignup, apiCreateCourse, apiCreateModule } from '../fixtures/api.js'

const API_BASE = process.env.E2E_API_URL ?? 'http://localhost:8080'
const PASSWORD = 'E2eTestPass1!'

function uid(prefix = 'bk') {
  return `e2e-${prefix}-${Date.now()}-${Math.random().toString(36).slice(2)}`
}
function uniqueEmail(prefix = 'bk') {
  return `${uid(prefix)}@test.invalid`
}
function authHeaders(token: string) {
  return { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' }
}

async function getAdminToken(): Promise<string> {
  const adminEmail = process.env.E2E_ADMIN_EMAIL ?? 'admin@e2e.test'
  const adminPassword = process.env.E2E_ADMIN_PASSWORD ?? 'E2eTestPass1!'
  const loginRes = await fetch(`${API_BASE}/api/v1/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email: adminEmail, password: adminPassword }),
  })
  if (!loginRes.ok) {
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
  const { access_token } = (await loginRes.json()) as { access_token: string }
  return access_token
}

// ─────────────────────────────────────────────────────────────────────────────
// Auth guards (no token → 401)
// ─────────────────────────────────────────────────────────────────────────────

test('Bookstore: GET inclusive-access unauthenticated returns 401', async () => {
  const res = await fetch(`${API_BASE}/api/v1/courses/some-course/inclusive-access`)
  expect(res.status).toBe(401)
})

test('Bookstore: POST textbook-resources unauthenticated returns 401', async () => {
  const res = await fetch(
    `${API_BASE}/api/v1/courses/some-course/structure/modules/00000000-0000-0000-0000-000000000001/textbook-resources`,
    { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: '{}' },
  )
  expect(res.status).toBe(401)
})

test('Bookstore: GET admin bookstore config unauthenticated returns 401', async () => {
  const res = await fetch(`${API_BASE}/api/v1/admin/bookstore/config`)
  expect(res.status).toBe(401)
})

// ─────────────────────────────────────────────────────────────────────────────
// Textbook resource lifecycle
// ─────────────────────────────────────────────────────────────────────────────

async function seedCourseWithModule() {
  const { access_token: token } = await apiSignup({
    email: uniqueEmail('teacher'),
    password: PASSWORD,
  })
  const course = await apiCreateCourse(token, { title: `Bookstore Course ${uid()}` })
  const moduleItem = await apiCreateModule(token, course.courseCode, 'Unit 1')
  return { token, courseCode: course.courseCode, moduleId: moduleItem.id }
}

test('Bookstore: instructor creates and reads a textbook resource (AC-1)', async () => {
  const { token, courseCode, moduleId } = await seedCourseWithModule()

  const createRes = await fetch(
    `${API_BASE}/api/v1/courses/${encodeURIComponent(courseCode)}/structure/modules/${encodeURIComponent(moduleId)}/textbook-resources`,
    {
      method: 'POST',
      headers: authHeaders(token),
      body: JSON.stringify({
        title: 'Chapter 3 Reading',
        provider: 'vitalsource',
        metadata: { isbn: '9780131103627', title: 'The C Programming Language', chapter: 'Chapter 3' },
      }),
    },
  )
  // Feature must be enabled in the e2e environment.
  if (createRes.status === 501) {
    test.skip(true, 'bookstore integration not enabled')
    return
  }
  expect(createRes.status).toBe(201)
  const item = (await createRes.json()) as { id: string; kind: string; title: string }
  expect(item.kind).toBe('textbook_resource')
  expect(item.title).toBe('Chapter 3 Reading')

  const getRes = await fetch(
    `${API_BASE}/api/v1/courses/${encodeURIComponent(courseCode)}/textbook-resources/${encodeURIComponent(item.id)}`,
    { headers: authHeaders(token) },
  )
  expect(getRes.status).toBe(200)
  const detail = (await getRes.json()) as {
    provider: string
    metadata: { isbn?: string; chapter?: string }
  }
  expect(detail.provider).toBe('vitalsource')
  expect(detail.metadata.isbn).toBe('9780131103627')
  expect(detail.metadata.chapter).toBe('Chapter 3')
})

test('Bookstore: PATCH updates textbook metadata', async () => {
  const { token, courseCode, moduleId } = await seedCourseWithModule()
  const createRes = await fetch(
    `${API_BASE}/api/v1/courses/${encodeURIComponent(courseCode)}/structure/modules/${encodeURIComponent(moduleId)}/textbook-resources`,
    {
      method: 'POST',
      headers: authHeaders(token),
      body: JSON.stringify({ title: 'Reading', provider: 'redshelf', metadata: { isbn: '111' } }),
    },
  )
  if (createRes.status === 501) {
    test.skip(true, 'bookstore integration not enabled')
    return
  }
  const item = (await createRes.json()) as { id: string }

  const patchRes = await fetch(
    `${API_BASE}/api/v1/courses/${encodeURIComponent(courseCode)}/textbook-resources/${encodeURIComponent(item.id)}`,
    {
      method: 'PATCH',
      headers: authHeaders(token),
      body: JSON.stringify({ metadata: { isbn: '222', pageRange: '10-20' } }),
    },
  )
  expect(patchRes.status).toBe(200)
  const updated = (await patchRes.json()) as { metadata: { isbn?: string; pageRange?: string } }
  expect(updated.metadata.isbn).toBe('222')
  expect(updated.metadata.pageRange).toBe('10-20')
})

test('Bookstore: invalid provider on textbook resource returns 400', async () => {
  const { token, courseCode, moduleId } = await seedCourseWithModule()
  const res = await fetch(
    `${API_BASE}/api/v1/courses/${encodeURIComponent(courseCode)}/structure/modules/${encodeURIComponent(moduleId)}/textbook-resources`,
    {
      method: 'POST',
      headers: authHeaders(token),
      body: JSON.stringify({ title: 'Reading', provider: 'chegg' }),
    },
  )
  if (res.status === 501) {
    test.skip(true, 'bookstore integration not enabled')
    return
  }
  expect(res.status).toBe(400)
})

// ─────────────────────────────────────────────────────────────────────────────
// Launch events (COUNTER, anonymized — AC-5)
// ─────────────────────────────────────────────────────────────────────────────

test('Bookstore: launch event records with provider and no PII (AC-5)', async () => {
  const { token, courseCode, moduleId } = await seedCourseWithModule()
  const createRes = await fetch(
    `${API_BASE}/api/v1/courses/${encodeURIComponent(courseCode)}/structure/modules/${encodeURIComponent(moduleId)}/textbook-resources`,
    {
      method: 'POST',
      headers: authHeaders(token),
      body: JSON.stringify({ title: 'Reading', provider: 'vitalsource', metadata: { isbn: '999' } }),
    },
  )
  if (createRes.status === 501) {
    test.skip(true, 'bookstore integration not enabled')
    return
  }
  const item = (await createRes.json()) as { id: string }

  const accessRes = await fetch(
    `${API_BASE}/api/v1/courses/${encodeURIComponent(courseCode)}/textbook-resources/${encodeURIComponent(item.id)}/access`,
    { method: 'POST', headers: authHeaders(token) },
  )
  expect(accessRes.status).toBe(204)

  const eventsRes = await fetch(
    `${API_BASE}/api/v1/courses/${encodeURIComponent(courseCode)}/textbook-launch-events`,
    { headers: authHeaders(token) },
  )
  expect(eventsRes.status).toBe(200)
  const body = (await eventsRes.json()) as {
    events: Array<{ id: string; itemId: string; provider: string; accessedAt: string }>
  }
  const found = body.events.find((e) => e.itemId === item.id)
  expect(found).toBeTruthy()
  expect(found?.provider).toBe('vitalsource')
  // AC-5: launch event rows must not expose student identity.
  expect(JSON.stringify(found)).not.toContain('userId')
  expect(JSON.stringify(found)).not.toContain('studentId')
})

// ─────────────────────────────────────────────────────────────────────────────
// Inclusive Access (AC-2)
// ─────────────────────────────────────────────────────────────────────────────

test('Bookstore: instructor configures Inclusive Access, GET reflects opt-out (AC-2)', async () => {
  const { token, courseCode } = await seedCourseWithModule()

  const initial = await fetch(
    `${API_BASE}/api/v1/courses/${encodeURIComponent(courseCode)}/inclusive-access`,
    { headers: authHeaders(token) },
  )
  if (initial.status === 501) {
    test.skip(true, 'bookstore integration not enabled')
    return
  }
  expect(initial.status).toBe(200)
  expect(((await initial.json()) as { enabled: boolean }).enabled).toBe(false)

  const optOutUrl = 'https://bookstore.example.edu/opt-out'
  const postRes = await fetch(
    `${API_BASE}/api/v1/courses/${encodeURIComponent(courseCode)}/inclusive-access`,
    {
      method: 'POST',
      headers: authHeaders(token),
      body: JSON.stringify({
        isbn: '9780262033848',
        title: 'Introduction to Algorithms',
        optOutUrl,
        provider: 'vitalsource',
        enabled: true,
      }),
    },
  )
  expect(postRes.status).toBe(200)

  const getRes = await fetch(
    `${API_BASE}/api/v1/courses/${encodeURIComponent(courseCode)}/inclusive-access`,
    { headers: authHeaders(token) },
  )
  const status = (await getRes.json()) as {
    enabled: boolean
    optOutUrl: string
    title: string
    isbn: string
  }
  expect(status.enabled).toBe(true)
  expect(status.optOutUrl).toBe(optOutUrl)
  expect(status.title).toBe('Introduction to Algorithms')
  expect(status.isbn).toBe('9780262033848')
})

test('Bookstore: Inclusive Access POST missing required fields returns 400', async () => {
  const { token, courseCode } = await seedCourseWithModule()
  const res = await fetch(
    `${API_BASE}/api/v1/courses/${encodeURIComponent(courseCode)}/inclusive-access`,
    {
      method: 'POST',
      headers: authHeaders(token),
      body: JSON.stringify({ isbn: '123' }),
    },
  )
  if (res.status === 501) {
    test.skip(true, 'bookstore integration not enabled')
    return
  }
  expect(res.status).toBe(400)
})

test('Bookstore: Inclusive Access POST invalid provider returns 400', async () => {
  const { token, courseCode } = await seedCourseWithModule()
  const res = await fetch(
    `${API_BASE}/api/v1/courses/${encodeURIComponent(courseCode)}/inclusive-access`,
    {
      method: 'POST',
      headers: authHeaders(token),
      body: JSON.stringify({
        isbn: '123',
        title: 'X',
        optOutUrl: 'https://x.example/opt',
        provider: 'chegg',
      }),
    },
  )
  if (res.status === 501) {
    test.skip(true, 'bookstore integration not enabled')
    return
  }
  expect(res.status).toBe(400)
})

// ─────────────────────────────────────────────────────────────────────────────
// Admin bookstore config (AC-4 prep)
// ─────────────────────────────────────────────────────────────────────────────

test('Bookstore: admin config GET/POST round-trips default provider', async () => {
  const adminToken = await getAdminToken()

  const getRes = await fetch(`${API_BASE}/api/v1/admin/bookstore/config`, {
    headers: authHeaders(adminToken),
  })
  if (getRes.status === 501) {
    test.skip(true, 'bookstore integration not enabled')
    return
  }
  if (getRes.status === 403) {
    test.skip(true, 'admin lacks rbac for bookstore config')
    return
  }
  expect(getRes.status).toBe(200)

  const postRes = await fetch(`${API_BASE}/api/v1/admin/bookstore/config`, {
    method: 'POST',
    headers: authHeaders(adminToken),
    body: JSON.stringify({ defaultProvider: 'redshelf' }),
  })
  expect(postRes.status).toBe(200)
  const cfg = (await postRes.json()) as { defaultProvider: string }
  expect(cfg.defaultProvider).toBe('redshelf')
})
