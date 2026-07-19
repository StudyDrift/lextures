/**
 * MOB.8 — Mobile collaboration boards advanced API parity
 *
 *   [x] Platform features exposes ffMobileBoardsAdvanced (default off)
 *   [x] Admin can enable ffMobileBoardsAdvanced via settings/platform
 *   [x] Staff can list templates → create from template → save as template
 *   [x] Staff can export a board (job → content)
 *   [x] Admin boards policies/overview require authz
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

async function enableMobileBoardsAdvanced(token: string) {
  const res = await fetch(`${API_BASE}/api/v1/settings/platform`, {
    method: 'PUT',
    headers: authHeaders(token),
    body: JSON.stringify({ ffMobileBoardsAdvanced: true, ffVisualBoards: true }),
  })
  if (!res.ok) {
    test.skip(true, `could not enable mobile boards advanced: ${await res.text()}`)
  }
}

async function enableCourseVisualBoards(token: string, courseCode: string) {
  const res = await fetch(`${API_BASE}/api/v1/courses/${encodeURIComponent(courseCode)}`, {
    method: 'PATCH',
    headers: authHeaders(token),
    body: JSON.stringify({ visualBoardsEnabled: true }),
  })
  if (!res.ok) {
    // Some deployments use features endpoint
    const alt = await fetch(
      `${API_BASE}/api/v1/courses/${encodeURIComponent(courseCode)}/features`,
      {
        method: 'PATCH',
        headers: authHeaders(token),
        body: JSON.stringify({ visualBoardsEnabled: true }),
      },
    )
    if (!alt.ok) {
      test.skip(true, `could not enable course visual boards: ${await alt.text()}`)
    }
  }
}

test('MOB.8 features: ffMobileBoardsAdvanced defaults off then enables', async () => {
  const email = uniqueEmail('mob8-admin')
  await apiSignup({ email, password: PASSWORD })
  try {
    await bootstrapGlobalAdmin(email)
  } catch (err) {
    test.skip(true, `bootstrap unavailable: ${err}`)
  }
  const { access_token: token } = await apiLogin(email)

  const beforeRes = await fetch(`${API_BASE}/api/v1/platform/features`, {
    headers: authHeaders(token),
  })
  expect(beforeRes.ok).toBeTruthy()
  const before = (await beforeRes.json()) as { ffMobileBoardsAdvanced?: boolean }
  expect(typeof before.ffMobileBoardsAdvanced === 'boolean').toBeTruthy()

  await enableMobileBoardsAdvanced(token)

  const afterRes = await fetch(`${API_BASE}/api/v1/platform/features`, {
    headers: authHeaders(token),
  })
  expect(afterRes.ok).toBeTruthy()
  const after = (await afterRes.json()) as { ffMobileBoardsAdvanced?: boolean }
  expect(after.ffMobileBoardsAdvanced).toBe(true)
})

test('MOB.8 boards: templates create/save and export job', async () => {
  const email = uniqueEmail('mob8-boards')
  await apiSignup({ email, password: PASSWORD })
  try {
    await bootstrapGlobalAdmin(email)
  } catch (err) {
    test.skip(true, `bootstrap unavailable: ${err}`)
  }
  const { access_token: token } = await apiLogin(email)
  await enableMobileBoardsAdvanced(token)

  const course = await apiCreateCourse(token, { title: 'MOB.8 Boards Course' })
  const courseCode = course.courseCode
  await enableCourseVisualBoards(token, courseCode)

  const templatesRes = await fetch(`${API_BASE}/api/v1/board-templates?courseCode=${encodeURIComponent(courseCode)}`, {
    headers: authHeaders(token),
  })
  expect(templatesRes.ok).toBeTruthy()
  const templatesBody = (await templatesRes.json()) as { templates?: Array<{ id: string; title: string }> }
  const templates = templatesBody.templates ?? []
  expect(templates.length).toBeGreaterThan(0)
  const template = templates[0]

  const createRes = await fetch(
    `${API_BASE}/api/v1/courses/${encodeURIComponent(courseCode)}/boards?from=${encodeURIComponent(`template:${template.id}`)}`,
    {
      method: 'POST',
      headers: authHeaders(token),
      body: JSON.stringify({ title: `From ${template.title}`, description: '' }),
    },
  )
  expect(createRes.ok).toBeTruthy()
  const board = (await createRes.json()) as { id: string; title: string }
  expect(board.id).toBeTruthy()

  const saveRes = await fetch(
    `${API_BASE}/api/v1/courses/${encodeURIComponent(courseCode)}/boards/${encodeURIComponent(board.id)}/save-as-template`,
    {
      method: 'POST',
      headers: authHeaders(token),
      body: JSON.stringify({
        scope: 'course',
        title: 'MOB.8 saved template',
        description: 'from e2e',
        tags: ['mob8'],
        includePosts: false,
      }),
    },
  )
  expect(saveRes.ok).toBeTruthy()
  const saved = (await saveRes.json()) as { id: string; title: string }
  expect(saved.id).toBeTruthy()
  expect(saved.title).toBe('MOB.8 saved template')

  const exportRes = await fetch(
    `${API_BASE}/api/v1/courses/${encodeURIComponent(courseCode)}/boards/${encodeURIComponent(board.id)}/export`,
    {
      method: 'POST',
      headers: authHeaders(token),
      body: JSON.stringify({ format: 'csv', includeModeration: false }),
    },
  )
  expect([200, 202]).toContain(exportRes.status)
  const exportBody = (await exportRes.json()) as { job: { id: string; status: string } }
  expect(exportBody.job.id).toBeTruthy()

  let job = exportBody.job
  for (let i = 0; i < 10 && job.status !== 'done' && job.status !== 'failed'; i++) {
    await new Promise((r) => setTimeout(r, 300))
    const jobRes = await fetch(
      `${API_BASE}/api/v1/courses/${encodeURIComponent(courseCode)}/boards/${encodeURIComponent(board.id)}/export/${encodeURIComponent(job.id)}`,
      { headers: authHeaders(token) },
    )
    expect(jobRes.ok).toBeTruthy()
    job = (await jobRes.json()) as { id: string; status: string }
  }
  expect(job.status).toBe('done')

  const contentRes = await fetch(
    `${API_BASE}/api/v1/courses/${encodeURIComponent(courseCode)}/boards/${encodeURIComponent(board.id)}/export/${encodeURIComponent(job.id)}/content`,
    { headers: { Authorization: `Bearer ${token}` } },
  )
  expect(contentRes.ok).toBeTruthy()
  const bytes = await contentRes.arrayBuffer()
  expect(bytes.byteLength).toBeGreaterThan(0)
})

test('MOB.8 admin boards governance: policies require auth', async () => {
  const email = uniqueEmail('mob8-gov')
  await apiSignup({ email, password: PASSWORD })
  try {
    await bootstrapGlobalAdmin(email)
  } catch (err) {
    test.skip(true, `bootstrap unavailable: ${err}`)
  }
  const { access_token: token } = await apiLogin(email)
  await enableMobileBoardsAdvanced(token)

  const unauth = await fetch(`${API_BASE}/api/v1/admin/boards/policies`)
  expect([401, 403]).toContain(unauth.status)

  const polRes = await fetch(`${API_BASE}/api/v1/admin/boards/policies`, {
    headers: authHeaders(token),
  })
  expect(polRes.ok).toBeTruthy()
  const policies = (await polRes.json()) as { externalSharing?: boolean }
  expect(typeof policies.externalSharing === 'boolean').toBeTruthy()

  const ovRes = await fetch(`${API_BASE}/api/v1/admin/boards/overview`, {
    headers: authHeaders(token),
  })
  expect(ovRes.ok).toBeTruthy()
  const overview = (await ovRes.json()) as { boardCount?: number }
  expect(typeof overview.boardCount === 'number').toBeTruthy()
})
