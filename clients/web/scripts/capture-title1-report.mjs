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
  const port = Number(process.env.PORT || 5175)
  const host = process.env.HOST || '127.0.0.1'
  const base = `http://${host}:${port}`
  const schoolId = 'b0000000-0000-4000-8000-000000000001'

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
    org_id: 'a0000000-0000-4000-8000-0000000000a0',
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
        email: 'title1@example.edu',
        displayName: 'Title I Coordinator',
        firstName: 'Title',
        lastName: 'Coordinator',
        avatarUrl: null,
        uiTheme: 'light',
      }),
    })
  })

  await page.route(`${api}/api/v1/platform/features`, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ ffDemographics: true }),
    })
  })
  await page.route(`${api}/api/v1/admin/org-units/${schoolId}/demographics/report`, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        schoolId,
        totalStudents: 500,
        freeLunchCount: 200,
        reducedLunchCount: 0,
        economicDisadvantaged: 200,
        economicDisadvantagePct: 40,
        ellCount: 45,
        disabilityCount: 38,
        homelessCount: 5,
        migrantCount: 2,
        raceBreakdown: { '1': 120, '4': 85, '6': 210, unknown: 85 },
      }),
    })
  })
  await page.route(
    `${api}/api/v1/admin/org-units/${schoolId}/demographics/disaggregated-performance**`,
    async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          dimension: 'ell',
          subgroups: [
            { label: 'ELL', count: 45, suppressed: false, passRate: 68.2 },
            { label: 'Non-ELL', count: 455, suppressed: false, passRate: 81.4 },
          ],
        }),
      })
    },
  )

  await page.goto(`${base}/admin/demographics/title1?schoolId=${schoolId}`)
  try {
    await page.waitForSelector('text=Economic disadvantage', { timeout: 15000 })
  } catch {
    const body = await page.locator('main').innerText().catch(() => '(no main)')
    console.error('Page body:', body)
    throw new Error('Title I report did not render')
  }
  await sleep(500)

  const here = dirname(fileURLToPath(import.meta.url))
  const out = join(here, '../../../docs/completed/13-k12-specific/13.13-title1-report.png')
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
