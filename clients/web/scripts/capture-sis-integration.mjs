import { spawn } from 'node:child_process'
import { mkdir } from 'node:fs/promises'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { chromium } from 'playwright'

function sleep(ms) {
  return new Promise((r) => setTimeout(r, ms))
}

function b64urlJson(obj) {
  const s = JSON.stringify(obj)
  return Buffer.from(s, 'utf8')
    .toString('base64')
    .replace(/=/g, '')
    .replace(/\+/g, '-')
    .replace(/\//g, '_')
}

function fakeJwt(payload) {
  const header = { alg: 'none', typ: 'JWT' }
  return `${b64urlJson(header)}.${b64urlJson(payload)}.`
}

async function main() {
  const port = Number(process.env.PORT || 5177)
  const host = process.env.HOST || '127.0.0.1'
  const base = `http://${host}:${port}`
  const orgId = 'a0000000-0000-4000-8000-0000000000a0'
  const connId = 'b0000000-0000-4000-8000-0000000000b1'

  const server = spawn('npm', ['run', 'dev', '--', '--host', host, '--port', String(port)], {
    stdio: ['ignore', 'pipe', 'pipe'],
    env: { ...process.env, VITE_API_URL: 'http://localhost:8080' },
  })

  let ready = false
  server.stdout.on('data', (buf) => {
    const s = buf.toString()
    if (s.includes('Local:') || s.includes('ready')) ready = true
  })
  server.stderr.on('data', (buf) => {
    const s = buf.toString()
    if (s.includes('Local:') || s.includes('ready')) ready = true
  })

  for (let i = 0; i < 60 && !ready; i++) await sleep(500)
  if (!ready) {
    server.kill()
    throw new Error('Vite dev server did not start')
  }
  await sleep(1500)

  const access = fakeJwt({
    sub: 'demo-user',
    org_id: orgId,
    org_slug: 'default',
    iat: Math.floor(Date.now() / 1000),
    exp: Math.floor(Date.now() / 1000) + 60 * 60,
  })

  const browser = await chromium.launch()
  const page = await browser.newPage({ viewport: { width: 1280, height: 900 } })

  const api = 'http://localhost:8080'

  await page.goto(`${base}/`, { waitUntil: 'domcontentloaded' })
  await page.evaluate((token) => {
    localStorage.setItem('studydrift_access_token', token)
  }, access)

  await page.route(`${api}/api/v1/settings/account`, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        email: 'registrar@university.example.edu',
        displayName: 'Registrar IT',
        firstName: 'Registrar',
        lastName: 'IT',
        avatarUrl: null,
        uiTheme: 'light',
      }),
    })
  })

  await page.route(`${api}/api/v1/platform/features`, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ ffSisIntegration: true }),
    })
  })

  await page.route(`${api}/api/v1/me/permissions**`, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        permissionStrings: ['tenant:org:roles:manage'],
      }),
    })
  })

  await page.route(`${api}/api/v1/public/branding/resolve`, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        primaryColor: '#4F46E5',
        secondaryColor: '#7C3AED',
      }),
    })
  })

  await page.route(`${api}/api/v1/me/notifications`, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ notifications: [], unreadCount: 0 }),
    })
  })

  await page.route(`${api}/api/v1/me/reading-preferences`, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({}),
    })
  })

  await page.route(`${api}/api/v1/admin/orgs/${orgId}/sis/connections`, async (route) => {
    if (route.request().method() === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          connections: [
            {
              id: connId,
              orgId,
              vendor: 'banner',
              market: 'he',
              baseUrl: 'https://banner.university.example.edu',
              clientIdRef: 'secrets/banner-client-id',
              clientSecretRef: 'secrets/banner-client-secret',
              syncSchedule: '0 2 * * *',
              syncMode: 'incremental',
              active: true,
              lastSyncAt: '2026-06-06T02:00:00Z',
              createdAt: '2026-06-01T10:00:00Z',
            },
          ],
        }),
      })
      return
    }
    await route.continue()
  })

  await page.route(`${api}/api/v1/admin/orgs/${orgId}/sis/sync-logs`, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        logs: [
          {
            id: 'c0000000-0000-4000-8000-0000000000c1',
            connectionId: connId,
            startedAt: '2026-06-06T02:00:00Z',
            finishedAt: '2026-06-06T02:00:12Z',
            status: 'success',
            summary: { enrollments_created: 42 },
            errors: [],
          },
        ],
      }),
    })
  })

  page.on('console', (msg) => {
    if (msg.type() === 'error') console.error('browser:', msg.text())
  })
  page.on('pageerror', (err) => console.error('pageerror:', err.message))

  await page.goto(`${base}/admin/sis?orgId=${orgId}`, { waitUntil: 'networkidle' })
  try {
    await page.getByRole('heading', { name: 'Configured connections' }).waitFor({ timeout: 20000 })
  } catch {
    const body = await page.locator('body').innerText().catch(() => '(no body)')
    console.error('Page body:', body.slice(0, 2000))
    throw new Error('SIS integration page did not render')
  }
  await sleep(500)

  const here = dirname(fileURLToPath(import.meta.url))
  const out = join(here, '../../../docs/completed/14-higher-ed-specific/14.1-sis-integration.png')
  await mkdir(dirname(out), { recursive: true })
  await page.screenshot({ path: out, fullPage: true })
  console.log('Wrote', out)

  await browser.close()
  server.kill()
}

main().catch((err) => {
  console.error(err)
  process.exit(1)
})
