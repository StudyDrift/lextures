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

const COURSE_CODE = 'INCOMP-101'
const STUDENT_ID = 's0000000-0000-4000-8000-000000000003'
const ENROLLMENT_ID = 'e0000000-0000-4000-8000-000000000003'
const ASSIGNMENT_ID = 'a0000000-0000-4000-8000-000000000002'

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
    sub: 'demo-instructor',
    org_id: orgId,
    org_slug: 'default',
    iat: Math.floor(Date.now() / 1000),
    exp: Math.floor(Date.now() / 1000) + 60 * 60,
  })

  const browser = await chromium.launch()
  const page = await browser.newPage({ viewport: { width: 1400, height: 900 } })
  await page.addInitScript((token) => {
    localStorage.setItem('studydrift_access_token', token)
  }, access)

  await page.route('**/api/v1/settings/account', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        email: 'instructor@university.example.edu',
        displayName: 'Dr. Instructor',
        firstName: 'Dr.',
        lastName: 'Instructor',
        avatarUrl: null,
        uiTheme: 'light',
      }),
    })
  })

  await page.route('**/api/v1/platform/features', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        ffEnrollmentStateMachine: true,
        ffIncompleteGradeWorkflow: true,
      }),
    })
  })

  await page.route('**/api/v1/me/permissions**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        permissionStrings: [
          `course:${COURSE_CODE}:gradebook:view`,
          `course:${COURSE_CODE}:item:create`,
        ],
      }),
    })
  })

  await page.route('**/api/v1/public/branding/resolve', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ primaryColor: '#4F46E5', secondaryColor: '#7C3AED' }),
    })
  })

  await page.route('**/api/v1/me/notifications', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ notifications: [], unreadCount: 0 }),
    })
  })

  await page.route('**/api/v1/me/reading-preferences', async (route) => {
    await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({}) })
  })

  await page.route(`**/api/v1/courses/${COURSE_CODE}`, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        courseCode: COURSE_CODE,
        title: 'Organic Chemistry II',
        sectionsEnabled: false,
      }),
    })
  })

  await page.route(`**/api/v1/courses/${COURSE_CODE}/gradebook/grid**`, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        students: [
          {
            userId: STUDENT_ID,
            displayName: 'Sam Incomplete',
            enrollmentId: ENROLLMENT_ID,
            state: 'incomplete',
            incompleteRecord: {
              extensionDeadline: '2026-09-15',
              status: 'open',
              outstandingItemIds: [ASSIGNMENT_ID],
            },
          },
        ],
        columns: [
          {
            id: ASSIGNMENT_ID,
            kind: 'assignment',
            title: 'Final Lab Report',
            maxPoints: 100,
          },
        ],
        grades: {
          [STUDENT_ID]: { [ASSIGNMENT_ID]: '' },
        },
      }),
    })
  })

  await page.route(`**/api/v1/courses/${COURSE_CODE}/grading`, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ gradingScheme: null }),
    })
  })

  await page.route(`**/api/v1/courses/${COURSE_CODE}/enrollments`, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ enrollments: [] }),
    })
  })

  await page.route('**/api/v1/me', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ id: 'demo-instructor', email: 'instructor@university.example.edu' }),
    })
  })

  await page.route(`**/api/v1/courses/${COURSE_CODE}/sections`, async (route) => {
    await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ sections: [] }) })
  })

  page.on('console', (msg) => {
    if (msg.type() === 'error') console.error('browser:', msg.text())
  })
  page.on('pageerror', (err) => console.error('pageerror:', err.message))

  await page.goto(`${base}/courses/${COURSE_CODE}/gradebook`, { waitUntil: 'networkidle' })
  await page.getByText('Due 2026-09-15').waitFor({ timeout: 20000 })
  await page.getByRole('button', { name: /resolve i/i }).waitFor({ timeout: 20000 })

  const here = dirname(fileURLToPath(import.meta.url))
  const out = join(here, '../../../docs/completed/14-higher-ed-specific/14.4-incomplete-grade-workflow.png')
  await mkdir(dirname(out), { recursive: true })
  await page.screenshot({ path: out, fullPage: false })
  console.log('Wrote', out)

  await browser.close()
  server.kill()
}

main().catch((err) => {
  console.error(err)
  process.exit(1)
})
