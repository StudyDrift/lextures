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

const DEMO_SECTIONS = [
  {
    id: 'd0000000-0000-4000-8000-0000000000d1',
    orgId: 'a0000000-0000-4000-8000-0000000000a0',
    termId: 't0000000-0000-4000-8000-0000000000t1',
    sisCourseId: 'CS-201',
    sisSectionId: 'CS-201-001-SPRING',
    crn: '12345',
    subject: 'CS',
    courseNumber: '201',
    sectionNumber: '001',
    title: 'Data Structures',
    credits: 3,
    meetingPattern: { days: 'MWF', startTime: '10:00', endTime: '10:50', instructor: 'Dr. Alice Chen' },
    room: 'SCI 201',
    department: 'CS',
    instructorName: 'Dr. Alice Chen',
    prerequisites: [{ code: 'CS 101', title: 'Intro to Computer Science' }],
    status: 'active',
    syncedAt: '2026-06-06T02:00:00Z',
  },
  {
    id: 'd0000000-0000-4000-8000-0000000000d2',
    orgId: 'a0000000-0000-4000-8000-0000000000a0',
    termId: 't0000000-0000-4000-8000-0000000000t1',
    sisCourseId: 'CS-301',
    sisSectionId: 'CS-301-002-SPRING',
    crn: '12346',
    subject: 'CS',
    courseNumber: '301',
    sectionNumber: '002',
    title: 'Algorithms',
    credits: 4,
    meetingPattern: { days: 'TR', startTime: '14:00', endTime: '15:15', instructor: 'Prof. Bob Martinez' },
    room: 'SCI 105',
    department: 'CS',
    instructorName: 'Prof. Bob Martinez',
    status: 'active',
    syncedAt: '2026-06-06T02:00:00Z',
  },
  {
    id: 'd0000000-0000-4000-8000-0000000000d3',
    orgId: 'a0000000-0000-4000-8000-0000000000a0',
    termId: 't0000000-0000-4000-8000-0000000000t1',
    sisCourseId: 'MATH-150',
    sisSectionId: 'MATH-150-001-SPRING',
    subject: 'MATH',
    courseNumber: '150',
    sectionNumber: '001',
    title: 'Calculus I',
    credits: 4,
    meetingPattern: { days: 'MWF', startTime: '09:00', endTime: '09:50' },
    department: 'MATH',
    status: 'active',
    syncedAt: '2026-06-06T02:00:00Z',
  },
]

async function main() {
  const port = Number(process.env.PORT || 5178)
  const host = process.env.HOST || '127.0.0.1'
  let base = `http://${host}:${port}`
  const orgId = 'a0000000-0000-4000-8000-0000000000a0'

  const server = spawn('npm', ['run', 'dev', '--', '--host', host, '--port', String(port), '--strictPort'], {
    stdio: ['ignore', 'pipe', 'pipe'],
    env: { ...process.env, VITE_API_URL: 'http://localhost:8080' },
  })

  let ready = false
  let actualPort = port
  const onData = (buf) => {
    const s = buf.toString()
    const m = s.match(/Local:\s+https?:\/\/[^:]+:(\d+)/)
    if (m) actualPort = Number(m[1])
    if (s.includes('Local:') || s.includes('ready')) ready = true
  }
  server.stdout.on('data', onData)
  server.stderr.on('data', onData)

  for (let i = 0; i < 60 && !ready; i++) await sleep(500)
  if (!ready) {
    server.kill()
    throw new Error('Vite dev server did not start')
  }
  await sleep(1500)
  base = `http://${host}:${actualPort}`

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
        email: 'student@university.example.edu',
        displayName: 'Alex Student',
        firstName: 'Alex',
        lastName: 'Student',
        avatarUrl: null,
        uiTheme: 'light',
      }),
    })
  })

  await page.route(`${api}/api/v1/platform/features`, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ ffCatalogIntegration: true, ffSisIntegration: true }),
    })
  })

  await page.route(`${api}/api/v1/me/permissions**`, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ permissionStrings: [] }),
    })
  })

  await page.route(`${api}/api/v1/public/branding/resolve`, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ primaryColor: '#4F46E5', secondaryColor: '#7C3AED' }),
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
    await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({}) })
  })

  await page.route(`${api}/api/v1/catalog/sections**`, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        sections: DEMO_SECTIONS,
        lastSyncedAt: '2026-06-06T02:00:00Z',
      }),
    })
  })

  page.on('console', (msg) => {
    if (msg.type() === 'error') console.error('browser:', msg.text())
  })
  page.on('pageerror', (err) => console.error('pageerror:', err.message))

  await page.goto(`${base}/catalog`, { waitUntil: 'networkidle' })
  try {
    await page.getByRole('heading', { name: 'Course catalog' }).waitFor({ timeout: 20000 })
    await page.getByText('Data Structures').waitFor({ timeout: 10000 })
  } catch {
    const body = await page.locator('body').innerText().catch(() => '(no body)')
    console.error('Page body:', body.slice(0, 2000))
    throw new Error('Course catalog page did not render')
  }
  await sleep(500)

  const here = dirname(fileURLToPath(import.meta.url))
  const out = join(here, '../../../docs/completed/14-higher-ed-specific/14.2-course-catalog-registration.png')
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
