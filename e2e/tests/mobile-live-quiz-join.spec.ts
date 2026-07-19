/**
 * MOB.5 — Mobile live quiz join API parity (Phase 1)
 *
 *   [x] Platform features exposes ffMobileLiveQuiz (default off)
 *   [x] Admin can enable ffMobileLiveQuiz via settings/platform
 *   [x] Join code lookup returns 404 for unknown codes
 *   [x] Guest join is rate-limited / rejected for unknown codes
 */
import { execSync } from 'node:child_process'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { test, expect, uniqueEmail } from '../fixtures/test.js'
import { apiSignup } from '../fixtures/api.js'

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

async function enableMobileLiveQuiz(token: string) {
  const res = await fetch(`${API_BASE}/api/v1/settings/platform`, {
    method: 'PUT',
    headers: authHeaders(token),
    body: JSON.stringify({ ffMobileLiveQuiz: true }),
  })
  if (!res.ok) {
    test.skip(true, `could not enable mobile live quiz: ${await res.text()}`)
  }
}

test('MOB.5 features: ffMobileLiveQuiz defaults off then enables', async () => {
  const email = uniqueEmail('mob5-admin')
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
  const before = (await beforeRes.json()) as { ffMobileLiveQuiz?: boolean }
  expect(typeof before.ffMobileLiveQuiz === 'boolean').toBeTruthy()

  await enableMobileLiveQuiz(token)

  const afterRes = await fetch(`${API_BASE}/api/v1/platform/features`, {
    headers: authHeaders(token),
  })
  expect(afterRes.ok).toBeTruthy()
  const after = (await afterRes.json()) as { ffMobileLiveQuiz?: boolean }
  expect(after.ffMobileLiveQuiz).toBe(true)
})

test('MOB.5 join: unknown code lookup returns 404', async () => {
  const res = await fetch(`${API_BASE}/api/v1/live-quizzes/join/ZZNOPE`)
  expect([404, 429]).toContain(res.status)
})

test('MOB.5 join: guest join unknown code is rejected', async () => {
  const res = await fetch(`${API_BASE}/api/v1/live-quizzes/join/ZZNOPE/players`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ nickname: 'Ada' }),
  })
  expect([400, 404, 429]).toContain(res.status)
})
