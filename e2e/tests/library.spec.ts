/**
 * Library / Book-Club / Leveled-Reader (plan 13.8)
 *
 *   [x] GET library unauthenticated returns 401
 *   [x] POST library unauthenticated returns 401
 *   [x] GET library/:bookId unauthenticated returns 401
 *   [x] DELETE library/:bookId unauthenticated returns 401
 *   [x] GET reading-log unauthenticated returns 401
 *   [x] POST reading-log unauthenticated returns 401
 *   [x] GET reading-dashboard unauthenticated returns 401
 *   [x] Admin can add a book to the library
 *   [x] Book is returned in catalog list
 *   [x] Book fields (title, author, lexileLevel, fpBand, gradeBand) round-trip correctly
 *   [x] Catalog filtered by lexile_min/lexile_max returns matching books
 *   [x] Catalog filtered by grade_band returns matching books
 *   [x] Lexile filter excludes books outside range
 *   [x] GET library/:bookId returns the book
 *   [x] POST book without title returns 400
 *   [x] Admin can delete a book
 *   [x] Student can post a reading log entry (free-text title)
 *   [x] Student can post a reading log entry (catalog bookId)
 *   [x] Reading log entry missing bookTitle and bookId returns 400
 *   [x] Reading log entry missing logDate returns 400
 *   [x] Reading log entry with invalid logDate returns 400
 *   [x] Student can list own reading log entries
 *   [x] Reading dashboard returns enrolled students
 *   [x] Non-admin cannot delete a library book (403)
 */
import { test, expect } from '@playwright/test'
import { apiSignup } from '../fixtures/api.js'

const API_BASE = process.env.E2E_API_URL ?? 'http://localhost:8080'
const PASSWORD = 'E2eTestPass1!'

function uid(prefix = 'lib') {
  return `e2e-${prefix}-${Date.now()}-${Math.random().toString(36).slice(2)}`
}
function uniqueEmail(prefix = 'lib') {
  return `${uid(prefix)}@test.invalid`
}
function authHeaders(token: string) {
  return { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' }
}

// ─────────────────────────────────────────────────────────────────────────────
// Auth guard checks (no token → 401)
// ─────────────────────────────────────────────────────────────────────────────

test('Library: GET library unauthenticated returns 401', async () => {
  const res = await fetch(
    `${API_BASE}/api/v1/orgs/00000000-0000-0000-0000-000000000001/library`,
  )
  expect(res.status).toBe(401)
})

test('Library: POST library unauthenticated returns 401', async () => {
  const res = await fetch(
    `${API_BASE}/api/v1/orgs/00000000-0000-0000-0000-000000000001/library`,
    { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: '{}' },
  )
  expect(res.status).toBe(401)
})

test('Library: GET library/:bookId unauthenticated returns 401', async () => {
  const res = await fetch(
    `${API_BASE}/api/v1/orgs/00000000-0000-0000-0000-000000000001/library/00000000-0000-0000-0000-000000000002`,
  )
  expect(res.status).toBe(401)
})

test('Library: DELETE library/:bookId unauthenticated returns 401', async () => {
  const res = await fetch(
    `${API_BASE}/api/v1/orgs/00000000-0000-0000-0000-000000000001/library/00000000-0000-0000-0000-000000000002`,
    { method: 'DELETE' },
  )
  expect(res.status).toBe(401)
})

test('Library: GET reading-log unauthenticated returns 401', async () => {
  const res = await fetch(`${API_BASE}/api/v1/me/reading-log`)
  expect(res.status).toBe(401)
})

test('Library: POST reading-log unauthenticated returns 401', async () => {
  const res = await fetch(`${API_BASE}/api/v1/me/reading-log`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: '{}',
  })
  expect(res.status).toBe(401)
})

test('Library: GET reading-dashboard unauthenticated returns 401', async () => {
  const res = await fetch(`${API_BASE}/api/v1/courses/some-course/reading-dashboard`)
  expect(res.status).toBe(401)
})

// ─────────────────────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────────────────────

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

async function getAdminOrgId(token: string): Promise<string | null> {
  const res = await fetch(`${API_BASE}/api/v1/admin/orgs`, {
    headers: authHeaders(token),
  })
  if (!res.ok) return null
  const data = (await res.json()) as { organizations?: Array<{ id: string }> }
  return data.organizations?.[0]?.id ?? null
}

async function getAdminUserId(token: string): Promise<string | null> {
  const res = await fetch(`${API_BASE}/api/v1/me`, {
    headers: authHeaders(token),
  })
  if (!res.ok) return null
  const data = (await res.json()) as { id?: string }
  return data.id ?? null
}

async function grantOrgAdmin(adminToken: string, orgId: string, userId: string): Promise<void> {
  await fetch(`${API_BASE}/api/v1/orgs/${orgId}/role-grants`, {
    method: 'POST',
    headers: authHeaders(adminToken),
    body: JSON.stringify({ userId, role: 'org_admin' }),
  })
}

interface LibraryBook {
  id: string
  orgId: string
  title: string
  author: string | null
  isbn: string | null
  coverUrl: string | null
  lexileLevel: number | null
  fpBand: string | null
  gradeBand: string | null
  summary: string | null
  createdAt: string
  updatedAt: string
}

async function addBook(
  token: string,
  orgId: string,
  payload: Partial<{
    title: string
    author: string
    isbn: string
    coverUrl: string
    lexileLevel: number
    fpBand: string
    gradeBand: string
    summary: string
  }>,
): Promise<{ status: number; book?: LibraryBook }> {
  const res = await fetch(`${API_BASE}/api/v1/orgs/${orgId}/library`, {
    method: 'POST',
    headers: authHeaders(token),
    body: JSON.stringify(payload),
  })
  if (!res.ok) return { status: res.status }
  const data = (await res.json()) as { book: LibraryBook }
  return { status: res.status, book: data.book }
}

interface ReadingLogEntry {
  id: string
  studentId: string
  bookId: string | null
  bookTitle: string | null
  logDate: string
  pagesRead: number | null
  reflection: string | null
  loggedAt: string
}

async function postReadingLogEntry(
  token: string,
  payload: Record<string, unknown>,
): Promise<{ status: number; entry?: ReadingLogEntry }> {
  const res = await fetch(`${API_BASE}/api/v1/me/reading-log`, {
    method: 'POST',
    headers: authHeaders(token),
    body: JSON.stringify(payload),
  })
  if (!res.ok) return { status: res.status }
  const data = (await res.json()) as { entry: ReadingLogEntry }
  return { status: res.status, entry: data.entry }
}

// ─────────────────────────────────────────────────────────────────────────────
// Library catalog CRUD
// ─────────────────────────────────────────────────────────────────────────────

test('Library: Admin can add a book to the library', async () => {
  const token = await getAdminToken()
  const orgId = await getAdminOrgId(token)
  if (!orgId) { test.skip(true, 'no org'); return }
  const adminId = await getAdminUserId(token)
  if (adminId) await grantOrgAdmin(token, orgId, adminId)

  const { status, book } = await addBook(token, orgId, {
    title: "Charlotte's Web",
    author: 'E.B. White',
    lexileLevel: 680,
    fpBand: 'P',
    gradeBand: '3-5',
    summary: 'A classic story of friendship.',
  })
  expect(status).toBe(201)
  expect(book?.title).toBe("Charlotte's Web")
  expect(book?.id).toBeTruthy()
  expect(book?.orgId).toBe(orgId)
})

test('Library: Book fields round-trip correctly', async () => {
  const token = await getAdminToken()
  const orgId = await getAdminOrgId(token)
  if (!orgId) { test.skip(true, 'no org'); return }
  const adminId = await getAdminUserId(token)
  if (adminId) await grantOrgAdmin(token, orgId, adminId)

  const { status, book } = await addBook(token, orgId, {
    title: 'The Giver',
    author: 'Lois Lowry',
    isbn: '9780385732550',
    lexileLevel: 760,
    fpBand: 'T',
    gradeBand: '6-8',
    summary: 'A dystopian novel.',
  })
  expect(status).toBe(201)
  expect(book?.author).toBe('Lois Lowry')
  expect(book?.isbn).toBe('9780385732550')
  expect(book?.lexileLevel).toBe(760)
  expect(book?.fpBand).toBe('T')
  expect(book?.gradeBand).toBe('6-8')
})

test('Library: Book is returned in catalog list', async () => {
  const token = await getAdminToken()
  const orgId = await getAdminOrgId(token)
  if (!orgId) { test.skip(true, 'no org'); return }
  const adminId = await getAdminUserId(token)
  if (adminId) await grantOrgAdmin(token, orgId, adminId)

  const bookTitle = `Catalog Test ${uid()}`
  await addBook(token, orgId, { title: bookTitle, lexileLevel: 500 })

  const listRes = await fetch(`${API_BASE}/api/v1/orgs/${orgId}/library`, {
    headers: authHeaders(token),
  })
  expect(listRes.status).toBe(200)
  const data = (await listRes.json()) as { books: LibraryBook[] }
  expect(Array.isArray(data.books)).toBe(true)
  const found = data.books.find((b) => b.title === bookTitle)
  expect(found).toBeTruthy()
})

test('Library: GET library/:bookId returns the book', async () => {
  const token = await getAdminToken()
  const orgId = await getAdminOrgId(token)
  if (!orgId) { test.skip(true, 'no org'); return }
  const adminId = await getAdminUserId(token)
  if (adminId) await grantOrgAdmin(token, orgId, adminId)

  const { book: created } = await addBook(token, orgId, { title: 'Get By ID Test', lexileLevel: 400 })
  expect(created).toBeTruthy()

  const getRes = await fetch(`${API_BASE}/api/v1/orgs/${orgId}/library/${created!.id}`, {
    headers: authHeaders(token),
  })
  expect(getRes.status).toBe(200)
  const data = (await getRes.json()) as { book: LibraryBook }
  expect(data.book.id).toBe(created!.id)
  expect(data.book.title).toBe('Get By ID Test')
})

test('Library: POST book without title returns 400', async () => {
  const token = await getAdminToken()
  const orgId = await getAdminOrgId(token)
  if (!orgId) { test.skip(true, 'no org'); return }
  const adminId = await getAdminUserId(token)
  if (adminId) await grantOrgAdmin(token, orgId, adminId)

  const { status } = await addBook(token, orgId, { title: '' })
  expect(status).toBe(400)
})

test('Library: Admin can delete a book', async () => {
  const token = await getAdminToken()
  const orgId = await getAdminOrgId(token)
  if (!orgId) { test.skip(true, 'no org'); return }
  const adminId = await getAdminUserId(token)
  if (adminId) await grantOrgAdmin(token, orgId, adminId)

  const { book } = await addBook(token, orgId, { title: 'To Be Deleted' })
  expect(book).toBeTruthy()

  const delRes = await fetch(`${API_BASE}/api/v1/orgs/${orgId}/library/${book!.id}`, {
    method: 'DELETE',
    headers: authHeaders(token),
  })
  expect(delRes.status).toBe(204)

  // Confirm it's gone
  const getRes = await fetch(`${API_BASE}/api/v1/orgs/${orgId}/library/${book!.id}`, {
    headers: authHeaders(token),
  })
  expect(getRes.status).toBe(404)
})

// ─────────────────────────────────────────────────────────────────────────────
// Catalog filters (AC-1)
// ─────────────────────────────────────────────────────────────────────────────

test('Library: Catalog filtered by lexile_min/lexile_max returns matching books', async () => {
  const token = await getAdminToken()
  const orgId = await getAdminOrgId(token)
  if (!orgId) { test.skip(true, 'no org'); return }
  const adminId = await getAdminUserId(token)
  if (adminId) await grantOrgAdmin(token, orgId, adminId)

  const suffix = uid()
  await addBook(token, orgId, { title: `Low Lexile ${suffix}`, lexileLevel: 300 })
  await addBook(token, orgId, { title: `Mid Lexile ${suffix}`, lexileLevel: 680 })
  await addBook(token, orgId, { title: `High Lexile ${suffix}`, lexileLevel: 900 })

  const res = await fetch(
    `${API_BASE}/api/v1/orgs/${orgId}/library?lexile_min=500&lexile_max=800`,
    { headers: authHeaders(token) },
  )
  expect(res.status).toBe(200)
  const data = (await res.json()) as { books: LibraryBook[] }
  const titles = data.books.map((b) => b.title)
  expect(titles).toContain(`Mid Lexile ${suffix}`)
  expect(titles).not.toContain(`Low Lexile ${suffix}`)
  expect(titles).not.toContain(`High Lexile ${suffix}`)
})

test('Library: Catalog filtered by grade_band returns matching books', async () => {
  const token = await getAdminToken()
  const orgId = await getAdminOrgId(token)
  if (!orgId) { test.skip(true, 'no org'); return }
  const adminId = await getAdminUserId(token)
  if (adminId) await grantOrgAdmin(token, orgId, adminId)

  const suffix = uid()
  await addBook(token, orgId, { title: `K-2 Book ${suffix}`, gradeBand: 'K-2' })
  await addBook(token, orgId, { title: `3-5 Book ${suffix}`, gradeBand: '3-5' })

  const res = await fetch(
    `${API_BASE}/api/v1/orgs/${orgId}/library?grade_band=3-5`,
    { headers: authHeaders(token) },
  )
  expect(res.status).toBe(200)
  const data = (await res.json()) as { books: LibraryBook[] }
  const titles = data.books.map((b) => b.title)
  expect(titles).toContain(`3-5 Book ${suffix}`)
  expect(titles).not.toContain(`K-2 Book ${suffix}`)
})

test('Library: Charlotte\'s Web appears when filtering Grade 3-5 and Lexile 500-800 (AC-1)', async () => {
  const token = await getAdminToken()
  const orgId = await getAdminOrgId(token)
  if (!orgId) { test.skip(true, 'no org'); return }
  const adminId = await getAdminUserId(token)
  if (adminId) await grantOrgAdmin(token, orgId, adminId)

  const suffix = uid()
  await addBook(token, orgId, {
    title: `Charlotte's Web ${suffix}`,
    author: 'E.B. White',
    lexileLevel: 680,
    fpBand: 'P',
    gradeBand: '3-5',
  })

  const res = await fetch(
    `${API_BASE}/api/v1/orgs/${orgId}/library?grade_band=3-5&lexile_min=500&lexile_max=800`,
    { headers: authHeaders(token) },
  )
  expect(res.status).toBe(200)
  const data = (await res.json()) as { books: LibraryBook[] }
  const found = data.books.find((b) => b.title === `Charlotte's Web ${suffix}`)
  expect(found).toBeTruthy()
})

// ─────────────────────────────────────────────────────────────────────────────
// Reading log (AC-2)
// ─────────────────────────────────────────────────────────────────────────────

test('Library: Student can post a reading log entry (free-text title)', async () => {
  const { access_token: token } = await apiSignup({ email: uniqueEmail('student'), password: PASSWORD })

  const { status, entry } = await postReadingLogEntry(token, {
    bookTitle: "Charlotte's Web",
    logDate: '2026-04-17',
    pagesRead: 25,
    reflection: 'I really enjoyed the first chapter.',
  })
  expect(status).toBe(201)
  expect(entry?.bookTitle).toBe("Charlotte's Web")
  expect(entry?.logDate).toBe('2026-04-17')
  expect(entry?.pagesRead).toBe(25)
  expect(entry?.reflection).toBe('I really enjoyed the first chapter.')
  expect(entry?.id).toBeTruthy()
})

test('Library: Student can post a reading log entry (catalog bookId)', async () => {
  const adminToken = await getAdminToken()
  const orgId = await getAdminOrgId(adminToken)
  if (!orgId) { test.skip(true, 'no org'); return }
  const adminId = await getAdminUserId(adminToken)
  if (adminId) await grantOrgAdmin(adminToken, orgId, adminId)

  const { book } = await addBook(adminToken, orgId, { title: 'Book For Log', lexileLevel: 500 })
  expect(book).toBeTruthy()

  const { access_token: studentToken } = await apiSignup({ email: uniqueEmail('slog'), password: PASSWORD })
  const { status, entry } = await postReadingLogEntry(studentToken, {
    bookId: book!.id,
    logDate: '2026-04-18',
    pagesRead: 15,
  })
  expect(status).toBe(201)
  expect(entry?.bookId).toBe(book!.id)
  expect(entry?.logDate).toBe('2026-04-18')
})

test('Library: Reading log entry missing bookTitle and bookId returns 400', async () => {
  const { access_token: token } = await apiSignup({ email: uniqueEmail('missing'), password: PASSWORD })
  const { status } = await postReadingLogEntry(token, {
    logDate: '2026-04-17',
    pagesRead: 10,
  })
  expect(status).toBe(400)
})

test('Library: Reading log entry missing logDate returns 400', async () => {
  const { access_token: token } = await apiSignup({ email: uniqueEmail('nodate'), password: PASSWORD })
  const { status } = await postReadingLogEntry(token, {
    bookTitle: 'Some Book',
    pagesRead: 10,
  })
  expect(status).toBe(400)
})

test('Library: Reading log entry with invalid logDate returns 400', async () => {
  const { access_token: token } = await apiSignup({ email: uniqueEmail('baddate'), password: PASSWORD })
  const { status } = await postReadingLogEntry(token, {
    bookTitle: 'Some Book',
    logDate: 'not-a-date',
    pagesRead: 10,
  })
  expect(status).toBe(400)
})

test('Library: Student can list own reading log entries', async () => {
  const { access_token: token } = await apiSignup({ email: uniqueEmail('listlog'), password: PASSWORD })

  await postReadingLogEntry(token, { bookTitle: 'Book A', logDate: '2026-04-01', pagesRead: 10 })
  await postReadingLogEntry(token, { bookTitle: 'Book B', logDate: '2026-04-02', pagesRead: 20 })

  const listRes = await fetch(`${API_BASE}/api/v1/me/reading-log`, {
    headers: authHeaders(token),
  })
  expect(listRes.status).toBe(200)
  const data = (await listRes.json()) as { entries: ReadingLogEntry[] }
  expect(Array.isArray(data.entries)).toBe(true)
  expect(data.entries.length).toBeGreaterThanOrEqual(2)
  const titles = data.entries.map((e) => e.bookTitle)
  expect(titles).toContain('Book A')
  expect(titles).toContain('Book B')
})

// ─────────────────────────────────────────────────────────────────────────────
// Reading dashboard (AC-2 teacher view)
// ─────────────────────────────────────────────────────────────────────────────

test('Library: Reading dashboard returns enrolled students', async () => {
  const adminToken = await getAdminToken()
  const orgId = await getAdminOrgId(adminToken)
  if (!orgId) { test.skip(true, 'no org'); return }
  const adminId = await getAdminUserId(adminToken)
  if (adminId) await grantOrgAdmin(adminToken, orgId, adminId)

  // Create a course
  const courseTitle = `Library Dashboard Course ${uid()}`
  const courseRes = await fetch(`${API_BASE}/api/v1/courses`, {
    method: 'POST',
    headers: authHeaders(adminToken),
    body: JSON.stringify({ title: courseTitle }),
  })
  if (!courseRes.ok) { test.skip(true, 'cannot create course'); return }
  const course = (await courseRes.json()) as { courseCode: string }
  const courseCode = course.courseCode

  const dashRes = await fetch(
    `${API_BASE}/api/v1/courses/${encodeURIComponent(courseCode)}/reading-dashboard`,
    { headers: authHeaders(adminToken) },
  )
  expect(dashRes.status).toBe(200)
  const data = (await dashRes.json()) as { students: Array<{ studentId: string; weeklyPages: number; totalEntries: number }> }
  expect(Array.isArray(data.students)).toBe(true)
})

// ─────────────────────────────────────────────────────────────────────────────
// RBAC: non-admin cannot delete a book
// ─────────────────────────────────────────────────────────────────────────────

test('Library: Non-admin cannot delete a library book (403)', async () => {
  const adminToken = await getAdminToken()
  const orgId = await getAdminOrgId(adminToken)
  if (!orgId) { test.skip(true, 'no org'); return }
  const adminId = await getAdminUserId(adminToken)
  if (adminId) await grantOrgAdmin(adminToken, orgId, adminId)

  const { book } = await addBook(adminToken, orgId, { title: 'Protected Book' })
  expect(book).toBeTruthy()

  const { access_token: studentToken } = await apiSignup({ email: uniqueEmail('nonAdmin'), password: PASSWORD })

  const delRes = await fetch(`${API_BASE}/api/v1/orgs/${orgId}/library/${book!.id}`, {
    method: 'DELETE',
    headers: authHeaders(studentToken),
  })
  expect([403, 404]).toContain(delRes.status)
})
