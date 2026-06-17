/**
 * Organization provisioning + org-scoped login
 *
 *   [x] Global Admin can create an organization with a custom slug
 *   [x] Creator account is assigned to the new org (org-role-capabilities)
 *   [x] Creator retains global:app:rbac:manage
 *   [x] Org-scoped login at /login/:slug succeeds for the creator
 *   [x] Org-scoped login rejects users who belong to a different org
 *   [x] UI: organizations panel lists the new org with its sign-in path
 */
import { execSync } from 'node:child_process'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { test, expect, type Page } from '@playwright/test'
import { injectToken, mainNav } from '../fixtures/test.js'

const repoRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../..')

const API_BASE = process.env.E2E_API_URL ?? 'http://localhost:8080'
const PASSWORD = process.env.E2E_ADMIN_PASSWORD ?? 'E2eTestPass1!'
const ADMIN_EMAIL = process.env.E2E_ADMIN_EMAIL ?? 'admin@e2e.test'
const GLOBAL_RBAC_PERM = 'global:app:rbac:manage'

function authHeaders(token: string) {
  return { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' }
}

function uniqueSlug(prefix = 'e2e-org') {
  return `${prefix}-${Date.now().toString(36).slice(-10)}`
}

function uniqueEmail(prefix = 'e2e-org-user') {
  return `${prefix}-${Date.now()}-${Math.random().toString(36).slice(2, 10)}@test.invalid`
}

function databaseUrlForBootstrap(): string | null {
  return (
    process.env.DATABASE_URL ??
    process.env.E2E_DATABASE_URL ??
    'postgres://studydrift:studydrift@localhost:5432/studydrift?sslmode=disable'
  )
}

async function bootstrapGlobalAdmin(email: string): Promise<void> {
  const dsn = databaseUrlForBootstrap()
  if (!dsn) return
  execSync(`go run ./cmd/bootstrap-admin -email=${email}`, {
    cwd: path.join(repoRoot, 'server'),
    env: { ...process.env, DATABASE_URL: dsn },
    stdio: 'pipe',
  })
}

async function ensureGlobalAdminCredentials(): Promise<{ email: string; password: string; token: string }> {
  let token = await getAdminToken()
  if (await adminCanManageOrgs(token)) {
    return { email: ADMIN_EMAIL, password: PASSWORD, token }
  }

  const email = uniqueEmail('e2e-ga')
  const signupRes = await fetch(`${API_BASE}/api/v1/auth/signup`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email, password: PASSWORD, display_name: 'E2E GA' }),
  })
  if (!signupRes.ok && signupRes.status !== 409) {
    const body = await signupRes.text()
    throw new Error(`GA signup failed (${signupRes.status}): ${body}`)
  }

  try {
    await bootstrapGlobalAdmin(email)
  } catch (err) {
    const msg = err instanceof Error ? err.message : String(err)
    test.skip(true, `bootstrap Global Admin unavailable: ${msg}`)
  }

  const login = await apiLogin(email, PASSWORD)
  if (login.status !== 200 || !login.access_token) {
    test.skip(true, 'promoted Global Admin could not log in')
  }
  token = login.access_token
  if (!(await adminCanManageOrgs(token))) {
    test.skip(true, 'bootstrap Global Admin did not grant rbac:manage')
  }
  return { email, password: PASSWORD, token }
}

async function getAdminToken(): Promise<string> {
  const loginRes = await fetch(`${API_BASE}/api/v1/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email: ADMIN_EMAIL, password: PASSWORD }),
  })
  if (loginRes.ok) {
    const { access_token } = (await loginRes.json()) as { access_token: string }
    return access_token
  }

  const signupRes = await fetch(`${API_BASE}/api/v1/auth/signup`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      email: ADMIN_EMAIL,
      password: PASSWORD,
      display_name: 'E2E Admin',
    }),
  })
  if (!signupRes.ok && signupRes.status !== 409) {
    const body = await signupRes.text()
    throw new Error(`Admin bootstrap failed (${signupRes.status}): ${body}`)
  }

  const retry = await fetch(`${API_BASE}/api/v1/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email: ADMIN_EMAIL, password: PASSWORD }),
  })
  if (!retry.ok) {
    const body = await retry.text()
    throw new Error(`Admin login failed (${retry.status}): ${body}`)
  }
  const { access_token } = (await retry.json()) as { access_token: string }
  return access_token
}

async function adminCanManageOrgs(token: string): Promise<boolean> {
  const res = await fetch(`${API_BASE}/api/v1/me/permissions`, {
    headers: authHeaders(token),
  })
  if (!res.ok) return false
  const data = (await res.json()) as { permissionStrings?: string[] }
  return (data.permissionStrings ?? []).includes(GLOBAL_RBAC_PERM)
}

async function fetchOrgRoleCapabilities(token: string): Promise<{ orgId: string }> {
  const res = await fetch(`${API_BASE}/api/v1/me/org-role-capabilities`, {
    headers: authHeaders(token),
  })
  if (!res.ok) {
    const body = await res.text()
    throw new Error(`org-role-capabilities failed (${res.status}): ${body}`)
  }
  return (await res.json()) as { orgId: string }
}

async function createOrganization(
  token: string,
  name: string,
  slug: string,
): Promise<{ id: string; slug: string; name: string }> {
  const res = await fetch(`${API_BASE}/api/v1/admin/orgs`, {
    method: 'POST',
    headers: authHeaders(token),
    body: JSON.stringify({ name, slug }),
  })
  if (!res.ok) {
    const body = await res.text()
    throw new Error(`Create org failed (${res.status}): ${body}`)
  }
  return (await res.json()) as { id: string; slug: string; name: string }
}

/** Re-login after org creation — JWT org_id is updated in DB but not on the existing access token. */
async function freshSessionToken(email: string, password: string): Promise<string> {
  const login = await apiLogin(email, password)
  if (login.status !== 200 || !login.access_token) {
    throw new Error(`Fresh login failed (${login.status})`)
  }
  return login.access_token
}

async function apiLogin(
  email: string,
  password: string,
  orgSlug?: string,
): Promise<{ status: number; access_token: string; raw: unknown }> {
  const res = await fetch(`${API_BASE}/api/v1/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      email,
      password,
      ...(orgSlug ? { org_slug: orgSlug } : {}),
    }),
  })
  const raw = await res.json().catch(() => ({}))
  return {
    status: res.status,
    access_token: (raw as { access_token?: string }).access_token ?? '',
    raw,
  }
}

async function loginViaOrgSlugPage(page: Page, slug: string, email: string, password: string) {
  await page.goto(`/login/${encodeURIComponent(slug)}`)
  await expect(page.getByText(new RegExp(`Sign in to`, 'i'))).toBeVisible({ timeout: 10_000 })
  await page.getByLabel('Email', { exact: true }).fill(email)
  await page.getByLabel('Password').fill(password)
  await page.getByRole('button', { name: /^sign in$/i }).click()
}

test.describe('Organizations — creator assignment and org-scoped login', () => {
  test('API: creating an org moves the creator into the tenant and keeps Global Admin', async () => {
    const { email: adminEmail, password: adminPassword, token: adminToken } =
      await ensureGlobalAdminCredentials()

    const slug = uniqueSlug()
    const orgName = `E2E Chase Org ${slug}`
    const created = await createOrganization(adminToken, orgName, slug)
    expect(created.slug).toBe(slug)

    const refreshedToken = await freshSessionToken(adminEmail, adminPassword)
    const caps = await fetchOrgRoleCapabilities(refreshedToken)
    expect(caps.orgId).toBe(created.id)

    const permsRes = await fetch(`${API_BASE}/api/v1/me/permissions`, {
      headers: authHeaders(refreshedToken),
    })
    expect(permsRes.ok).toBeTruthy()
    const perms = (await permsRes.json()) as { permissionStrings?: string[] }
    expect(perms.permissionStrings ?? []).toContain(GLOBAL_RBAC_PERM)

    const scopedLogin = await apiLogin(adminEmail, adminPassword, slug)
    expect(scopedLogin.status).toBe(200)
    expect(scopedLogin.access_token).toBeTruthy()

    const otherEmail = uniqueEmail()
    const otherSignup = await fetch(`${API_BASE}/api/v1/auth/signup`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email: otherEmail, password: PASSWORD }),
    })
    expect(otherSignup.ok).toBeTruthy()

    const wrongOrgLogin = await apiLogin(otherEmail, PASSWORD, slug)
    expect(wrongOrgLogin.status).toBe(401)
  })

  test('UI: creator signs in via /login/:slug after provisioning in settings', async ({ page }) => {
    const { email: adminEmail, password: adminPassword, token: adminToken } =
      await ensureGlobalAdminCredentials()

    const slug = uniqueSlug('e2e-ui')
    const orgName = `E2E UI Org ${slug}`

    await injectToken(page, adminToken)
    await page.goto('/settings/organizations')
    await expect(page.getByRole('heading', { name: /^organizations$/i })).toBeVisible({
      timeout: 10_000,
    })

    await page.getByLabel(/^name$/i).fill(orgName)
    await page.getByLabel(/short name/i).fill(slug)
    await page.getByRole('button', { name: /^create$/i }).click()

    await expect(page.getByRole('link', { name: `/login/${slug}` })).toBeVisible({
      timeout: 15_000,
    })

    const createdRow = page.getByRole('row', { name: new RegExp(orgName, 'i') })
    await expect(createdRow).toBeVisible()

    await page.evaluate(() => {
      localStorage.removeItem('studydrift_access_token')
      localStorage.removeItem('studydrift_refresh_token')
    })

    const refreshedToken = await freshSessionToken(adminEmail, adminPassword)
    const caps = await fetchOrgRoleCapabilities(refreshedToken)
    expect(caps.orgId).toBeTruthy()

    await loginViaOrgSlugPage(page, slug, adminEmail, adminPassword)
    await expect(page).toHaveURL('/')
    await expect(mainNav(page)).toBeVisible()

    const sessionToken = await page.evaluate(() => localStorage.getItem('studydrift_access_token'))
    expect(sessionToken).toBeTruthy()
    const afterLogin = await fetchOrgRoleCapabilities(sessionToken!)
    expect(afterLogin.orgId).toBe(caps.orgId)
  })
})