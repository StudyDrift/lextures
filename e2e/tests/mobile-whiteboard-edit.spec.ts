/**
 * MOB.6 — Mobile whiteboard authoring API parity
 *
 *   [x] Platform features exposes ffMobileWhiteboardEdit (always on)
 *   [x] Staff can create → update canvasData → delete a course whiteboard
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

async function enableMobileWhiteboardEdit(token: string) {
  const res = await fetch(`${API_BASE}/api/v1/settings/platform`, {
    method: 'PUT',
    headers: authHeaders(token),
    body: JSON.stringify({ ffMobileWhiteboardEdit: true }),
  })
  if (!res.ok) {
    test.skip(true, `could not enable mobile whiteboard edit: ${await res.text()}`)
  }
}

test('MOB.6 features: ffMobileWhiteboardEdit is always on', async () => {
  const email = uniqueEmail('mob6-admin')
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
  const features = (await res.json()) as { ffMobileWhiteboardEdit?: boolean }
  expect(features.ffMobileWhiteboardEdit).toBe(true)
})

test('MOB.6 whiteboard: create edit delete with web-compatible canvasData', async () => {
  const email = uniqueEmail('mob6-wb')
  await apiSignup({ email, password: PASSWORD })
  try {
    await bootstrapGlobalAdmin(email)
  } catch (err) {
    test.skip(true, `bootstrap unavailable: ${err}`)
  }
  const { access_token: token } = await apiLogin(email)
  await enableMobileWhiteboardEdit(token)

  const course = await apiCreateCourse(token, { title: 'MOB.6 Whiteboard Course' })
  const courseCode = course.courseCode

  const canvasData = [
    { type: 'stroke', color: '#ef4444', width: 4, pts: [[10, 10], [40, 40]] },
    { type: 'rect', color: '#3b82f6', width: 2, x: 50, y: 50, w: 80, h: 40 },
    { type: 'line', color: '#22c55e', width: 2, x1: 0, y1: 0, x2: 20, y2: 20 },
  ]

  const createRes = await fetch(`${API_BASE}/api/v1/courses/${encodeURIComponent(courseCode)}/whiteboards`, {
    method: 'POST',
    headers: authHeaders(token),
    body: JSON.stringify({ title: 'Mobile board', canvasData: [] }),
  })
  expect(createRes.status).toBe(201)
  const created = (await createRes.json()) as { id: string; title: string; canvasData?: unknown[] }
  expect(created.id).toBeTruthy()
  expect(created.title).toBe('Mobile board')

  const updateRes = await fetch(
    `${API_BASE}/api/v1/courses/${encodeURIComponent(courseCode)}/whiteboards/${encodeURIComponent(created.id)}`,
    {
      method: 'PUT',
      headers: authHeaders(token),
      body: JSON.stringify({ title: 'Mobile board', canvasData }),
    },
  )
  expect(updateRes.ok).toBeTruthy()
  const updated = (await updateRes.json()) as { canvasData?: Array<{ type: string }> }
  expect(updated.canvasData?.length).toBe(3)
  expect(updated.canvasData?.map((el) => el.type)).toEqual(['stroke', 'rect', 'line'])

  const getRes = await fetch(
    `${API_BASE}/api/v1/courses/${encodeURIComponent(courseCode)}/whiteboards/${encodeURIComponent(created.id)}`,
    { headers: authHeaders(token) },
  )
  expect(getRes.ok).toBeTruthy()
  const loaded = (await getRes.json()) as { canvasData?: unknown[] }
  expect(loaded.canvasData?.length).toBe(3)

  const deleteRes = await fetch(
    `${API_BASE}/api/v1/courses/${encodeURIComponent(courseCode)}/whiteboards/${encodeURIComponent(created.id)}`,
    { method: 'DELETE', headers: authHeaders(token) },
  )
  expect([200, 204]).toContain(deleteRes.status)
})
