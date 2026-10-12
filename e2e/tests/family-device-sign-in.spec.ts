import { expect, test } from '@playwright/test'
import { apiSignup } from '../fixtures/api.js'
import { injectToken, uniqueEmail } from '../fixtures/test.js'

const API_BASE = process.env.E2E_API_URL ?? 'http://localhost:8080'
const PASSWORD = 'E2eTestPass1!'

test.describe('Family device learner sign-in (#771)', () => {
  test('parent marks a family device; learner signs in with a PIN and signs out to the picker', async ({
    page,
    request,
  }) => {
    const { access_token } = await apiSignup({
      email: uniqueEmail('family-device'),
      password: PASSWORD,
      displayName: 'Family Parent',
    })
    const auth = { Authorization: `Bearer ${access_token}` }

    const featuresRes = await request.get(`${API_BASE}/api/v1/platform/features`, { headers: auth })
    expect(featuresRes.ok()).toBeTruthy()
    const features = (await featuresRes.json()) as { ffHomeschoolManagedLearners?: boolean }
    test.skip(!features.ffHomeschoolManagedLearners, 'managed learners disabled')

    const depRes = await request.post(`${API_BASE}/api/v1/me/dependents`, {
      headers: { ...auth, 'Content-Type': 'application/json' },
      data: { displayName: 'Robin Learner', gradeLevel: '2' },
    })
    expect(depRes.ok(), await depRes.text()).toBeTruthy()
    const dependent = (await depRes.json()) as { id: string }

    const pinRes = await request.put(`${API_BASE}/api/v1/me/dependents/${dependent.id}/sign-in`, {
      headers: { ...auth, 'Content-Type': 'application/json' },
      data: { method: 'pin', pin: '2468' },
    })
    expect(pinRes.ok(), await pinRes.text()).toBeTruthy()
    expect(((await pinRes.json()) as { signInMethod: string }).signInMethod).toBe('pin')

    await injectToken(page, access_token)
    await page.goto('/learners')
    await expect(page.getByText('PIN sign-in on')).toBeVisible()
    await page.getByRole('button', { name: 'Use this browser as a family device' }).click()
    await page.getByRole('link', { name: `Open "Who's learning?"` }).click()

    await expect(page).toHaveURL(/\/family$/)
    await expect(page.getByRole('heading', { name: "Who's learning?" })).toBeVisible()
    await page.getByRole('button', { name: /Robin Learner/ }).click()

    // Wrong PIN first: rate-limited attempts count down.
    for (const d of ['1', '1', '1', '1']) await page.getByRole('button', { name: d, exact: true }).click()
    await page.getByRole('button', { name: 'Start learning' }).click()
    await expect(page.getByText(/tries left/)).toBeVisible()

    for (const d of ['2', '4', '6', '8']) await page.getByRole('button', { name: d, exact: true }).click()
    await page.getByRole('button', { name: 'Start learning' }).click()
    await expect(page).not.toHaveURL(/\/family/)
    await expect(page.getByText('Signed in as Robin Learner.')).toBeVisible()

    // Learner sessions cannot manage learners or devices.
    const learnerToken = await page.evaluate(() => localStorage.getItem('studydrift_impersonation_token'))
    expect(learnerToken).toBeTruthy()
    const blocked = await request.get(`${API_BASE}/api/v1/me/dependents/family-devices`, {
      headers: { Authorization: `Bearer ${learnerToken}` },
    })
    expect(blocked.status()).toBe(403)

    await page.getByRole('button', { name: 'Sign out', exact: true }).first().click()
    await expect(page).toHaveURL(/\/family$/)
    await expect(page.getByRole('heading', { name: "Who's learning?" })).toBeVisible()

    // Parent session is untouched by the learner signing in and out.
    const parentToken = await page.evaluate(() => localStorage.getItem('studydrift_access_token'))
    expect(parentToken).toBe(access_token)
    const me = await request.get(`${API_BASE}/api/v1/me/dependents`, { headers: auth })
    expect(me.ok()).toBeTruthy()
  })
})
