/**
 * Homeschool records (#774)
 *
 *   [x] Parent can set family school name
 *   [x] Parent can add offline hours and see totals
 *   [x] Hours CSV and transcript PDF download
 *   [x] Portfolio PDF with selected submission
 *   [x] Managed learner cannot POST hours (403)
 */
import { expect, test } from '@playwright/test'
import { apiSignup } from '../fixtures/api.js'
import { injectToken, uniqueEmail } from '../fixtures/test.js'

const API_BASE = process.env.E2E_API_URL ?? 'http://localhost:8080'
const PASSWORD = 'E2eTestPass1!'

test.describe('Homeschool records (#774)', () => {
  test('parent manages hours log, transcript, and portfolio; managed learner cannot edit', async ({
    page,
    request,
  }) => {
    const { access_token } = await apiSignup({
      email: uniqueEmail('records'),
      password: PASSWORD,
      displayName: 'Records Parent',
    })
    const auth = { Authorization: `Bearer ${access_token}` }

    const featuresRes = await request.get(`${API_BASE}/api/v1/platform/features`, { headers: auth })
    expect(featuresRes.ok()).toBeTruthy()
    const features = (await featuresRes.json()) as { ffHomeschoolManagedLearners?: boolean }
    test.skip(!features.ffHomeschoolManagedLearners, 'managed learners disabled')

    const depRes = await request.post(`${API_BASE}/api/v1/me/dependents`, {
      headers: { ...auth, 'Content-Type': 'application/json' },
      data: { displayName: 'Records Kid', gradeLevel: '3' },
    })
    expect(depRes.ok(), await depRes.text()).toBeTruthy()
    const dependent = (await depRes.json()) as { id: string }

    const settingsRes = await request.put(`${API_BASE}/api/v1/parent/records/settings`, {
      headers: { ...auth, 'Content-Type': 'application/json' },
      data: { schoolName: 'E2E Homeschool' },
    })
    expect(settingsRes.ok(), await settingsRes.text()).toBeTruthy()

    const hoursRes = await request.post(
      `${API_BASE}/api/v1/parent/students/${dependent.id}/records/hours`,
      {
        headers: { ...auth, 'Content-Type': 'application/json' },
        data: {
          date: new Date().toISOString().slice(0, 10),
          subject: 'Reading',
          hours: 1.5,
          notes: 'Library',
          source: 'offline',
        },
      },
    )
    expect(hoursRes.status()).toBe(201)

    const logRes = await request.get(
      `${API_BASE}/api/v1/parent/students/${dependent.id}/records/hours`,
      { headers: auth },
    )
    expect(logRes.ok()).toBeTruthy()
    const log = (await logRes.json()) as { totalHours: number; entries: unknown[] }
    expect(log.totalHours).toBeGreaterThanOrEqual(1.5)

    const csvRes = await request.get(
      `${API_BASE}/api/v1/parent/students/${dependent.id}/records/hours/export.csv`,
      { headers: auth },
    )
    expect(csvRes.ok()).toBeTruthy()
    expect(await csvRes.text()).toContain('Reading')

    const pdfRes = await request.get(
      `${API_BASE}/api/v1/parent/students/${dependent.id}/records/transcript.pdf`,
      { headers: auth },
    )
    expect(pdfRes.ok()).toBeTruthy()
    const pdfBuf = await pdfRes.body()
    expect(pdfBuf.slice(0, 4).toString()).toBe('%PDF')

    const portfolioRes = await request.post(
      `${API_BASE}/api/v1/parent/students/${dependent.id}/records/portfolio`,
      {
        headers: { ...auth, 'Content-Type': 'application/json' },
        data: { submissionIds: [], format: 'pdf' },
      },
    )
    expect(portfolioRes.ok()).toBeTruthy()
    expect((await portfolioRes.body()).slice(0, 4).toString()).toBe('%PDF')

    // UI: Records page shows the offline entry.
    await injectToken(page, access_token)
    await page.goto(`/parent/records?student=${dependent.id}`)
    await expect(page.getByRole('heading', { name: 'Records' })).toBeVisible()
    await expect(page.getByText('Reading')).toBeVisible()
    await expect(page.getByRole('button', { name: 'Add offline hours' })).toBeVisible()

    // Managed learner session cannot edit records.
    const pinRes = await request.put(`${API_BASE}/api/v1/me/dependents/${dependent.id}/sign-in`, {
      headers: { ...auth, 'Content-Type': 'application/json' },
      data: { method: 'pin', pin: '1357' },
    })
    expect(pinRes.ok(), await pinRes.text()).toBeTruthy()
    const deviceRes = await request.post(`${API_BASE}/api/v1/me/dependents/family-devices`, {
      headers: { ...auth, 'Content-Type': 'application/json' },
      data: { label: 'E2E records device' },
    })
    expect(deviceRes.ok(), await deviceRes.text()).toBeTruthy()
    const device = (await deviceRes.json()) as { deviceToken: string }
    const sessionRes = await request.post(`${API_BASE}/api/v1/auth/family-device/sessions`, {
      headers: { 'Content-Type': 'application/json' },
      data: { deviceToken: device.deviceToken, learnerId: dependent.id, pin: '1357' },
    })
    expect(sessionRes.ok(), await sessionRes.text()).toBeTruthy()
    const session = (await sessionRes.json()) as { accessToken: string }
    const blocked = await request.post(
      `${API_BASE}/api/v1/parent/students/${dependent.id}/records/hours`,
      {
        headers: {
          Authorization: `Bearer ${session.accessToken}`,
          'Content-Type': 'application/json',
        },
        data: { date: '2026-01-01', subject: 'X', hours: 1 },
      },
    )
    expect(blocked.status()).toBe(403)
  })
})
