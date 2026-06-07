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
  const port = Number(process.env.PORT || 5176)
  const host = process.env.HOST || '127.0.0.1'
  const base = `http://${host}:${port}`
  const orgId = 'a0000000-0000-4000-8000-0000000000a0'

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

  await page.addInitScript((token) => {
    localStorage.setItem('studydrift_access_token', token)
  }, access)

  const api = 'http://localhost:8080'

  await page.route(`${api}/api/v1/settings/account`, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        email: 'it@district.example.edu',
        displayName: 'District IT',
        firstName: 'District',
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
      body: JSON.stringify({ ffContentFilterIntegration: true }),
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

  await page.route(`${api}/api/v1/orgs/${orgId}/settings/content-filter`, async (route) => {
    if (route.request().method() === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          orgId,
          goGuardianEnabled: true,
          goGuardianApiKey: '••••••••••••',
          hasGoGuardianApiKey: true,
          securlyEnabled: true,
          updatedAt: '2026-06-06T12:00:00Z',
          allowlistUrl: '/.well-known/content-filter-allowlist.json',
        }),
      })
      return
    }
    await route.continue()
  })

  page.on('console', (msg) => {
    if (msg.type() === 'error') console.error('browser:', msg.text())
  })
  page.on('pageerror', (err) => console.error('pageerror:', err.message))

  await page.goto(`${base}/admin/content-filter?orgId=${orgId}`, { waitUntil: 'networkidle' })
  try {
    await page.waitForSelector('text=GoGuardian', { timeout: 20000 })
  } catch {
    const body = await page.locator('body').innerText().catch(() => '(no body)')
    console.error('Page body:', body.slice(0, 2000))
    throw new Error('Content filter settings page did not render')
  }
  await sleep(500)

  const here = dirname(fileURLToPath(import.meta.url))
  const out = join(here, '../../../docs/completed/13-k12-specific/13.14-content-filter-settings.png')
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
