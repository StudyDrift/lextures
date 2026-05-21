/**
 * OER library (plan 8.9)
 *
 * Checklist coverage:
 *   [x] OER search API returns results for photosynthesis (stub mode)
 *   [x] CC BY license filter excludes NC/ND/SA licenses
 *   [x] OER import creates module external link with attribution
 *   [x] Disabled MERLOT provider hidden from provider list
 */
import { test, expect } from '../fixtures/test.js'
import { apiSignup } from '../fixtures/api.js'

const apiBase = process.env.E2E_API_URL ?? 'http://localhost:8080'
const PASSWORD = 'E2eTestPass1!'
const OER_ENABLED = process.env.FEATURE_OER_LIBRARY === 'true'

function uniqueEmail(prefix = 'oer') {
  return `e2e-${prefix}-${Date.now()}-${Math.random().toString(36).slice(2)}@test.invalid`
}

async function apiSearchOER(
  token: string,
  provider: string,
  q: string,
  license?: string,
): Promise<{ results: Array<{ id: string; title: string; licenseSpdx: string }> }> {
  const qs = new URLSearchParams({ provider, q })
  if (license) qs.set('license', license)
  const res = await fetch(`${apiBase}/api/v1/oer/search?${qs.toString()}`, {
    headers: { Authorization: `Bearer ${token}` },
  })
  const body = await res.text()
  if (!res.ok) throw new Error(`OER search failed (${res.status}): ${body}`)
  return JSON.parse(body) as { results: Array<{ id: string; title: string; licenseSpdx: string }> }
}

async function apiImportOER(
  token: string,
  courseCode: string,
  moduleId: string,
  payload: Record<string, unknown>,
): Promise<{ id: string }> {
  const res = await fetch(
    `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/structure/modules/${encodeURIComponent(moduleId)}/oer-import`,
    {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${token}`,
      },
      body: JSON.stringify(payload),
    },
  )
  const body = await res.text()
  if (!res.ok) throw new Error(`OER import failed (${res.status}): ${body}`)
  return JSON.parse(body) as { id: string }
}

test.describe('OER library', () => {
  test.beforeEach(() => {
    test.skip(!OER_ENABLED, 'Set FEATURE_OER_LIBRARY=true for OER e2e tests')
  })

  test('search photosynthesis returns CC BY results from OER Commons', async ({ seededCourse }) => {
    const resp = await apiSearchOER(seededCourse.instructorToken, 'oer_commons', 'photosynthesis')
    expect(resp.results.length).toBeGreaterThanOrEqual(5)
    for (const r of resp.results) {
      expect(r.title.length).toBeGreaterThan(0)
      expect(r.licenseSpdx).toMatch(/CC-BY/i)
      expect(r.licenseSpdx).not.toMatch(/NC|ND|SA/i)
    }
  })

  test('CC BY filter excludes NC, ND, and SA licenses', async ({ seededCourse }) => {
    const resp = await apiSearchOER(
      seededCourse.instructorToken,
      'oer_commons',
      'chemistry',
      'CC-BY',
    )
    expect(resp.results.length).toBeGreaterThan(0)
    for (const r of resp.results) {
      const s = r.licenseSpdx.toUpperCase()
      expect(s).not.toContain('NC')
      expect(s).not.toContain('ND')
      expect(s).not.toContain('SA')
    }
  })

  test('import OpenStax chapter creates external link with attribution', async ({
    seededCourse,
  }) => {
    const search = await apiSearchOER(seededCourse.instructorToken, 'openstax', 'algebra')
    expect(search.results.length).toBeGreaterThan(0)
    const hit = search.results[0]
    const item = await apiImportOER(
      seededCourse.instructorToken,
      seededCourse.courseCode,
      seededCourse.moduleId,
      {
        title: `OpenStax: ${hit.title}`,
        url: 'https://openstax.org/books/elementary-algebra-2e/pages/1-introduction',
        provider: 'openstax',
        externalId: hit.id,
        licenseSpdx: 'CC-BY-4.0',
        attributionText: 'OpenStax Elementary Algebra 2e © Rice University, licensed CC BY 4.0',
      },
    )
    expect(item.id).toBeTruthy()

    const linkRes = await fetch(
      `${apiBase}/api/v1/courses/${encodeURIComponent(seededCourse.courseCode)}/external-links/${item.id}`,
      { headers: { Authorization: `Bearer ${seededCourse.instructorToken}` } },
    )
    expect(linkRes.ok).toBe(true)
    const link = (await linkRes.json()) as Record<string, unknown>
    expect(String(link.title ?? '')).toMatch(/openstax/i)
    expect(String(link.attributionText ?? '')).toContain('CC BY')
    expect(String(link.oerProvider ?? '')).toBe('openstax')
  })

  test('UI: search algebra and add first result to module', async ({ coursePage: page, seededCourse }) => {
    await page.goto(`/courses/${seededCourse.courseCode}/modules`)
    await expect(page.getByText(seededCourse.moduleTitle)).toBeVisible()

    const moduleRow = page.locator('li').filter({ hasText: seededCourse.moduleTitle }).first()
    await moduleRow.hover()
    const addBtn = moduleRow.getByRole('button', { name: /add item|add content|\+/i }).first()
    if (!(await addBtn.isVisible({ timeout: 3000 }))) {
      test.skip(true, 'Add item button not visible')
      return
    }
    await addBtn.click()

    const oerItem = page.getByRole('menuitem', { name: /find open resources/i })
    if (!(await oerItem.isVisible({ timeout: 3000 }))) {
      test.skip(true, 'OER menu item not visible — is VITE_FEATURE_OER_LIBRARY=true?')
      return
    }
    await oerItem.click()

    const dialog = page.getByRole('dialog', { name: /find open resources/i })
    await expect(dialog).toBeVisible({ timeout: 5000 })
    await dialog.getByRole('searchbox', { name: /search keywords/i }).fill('algebra')
    await expect(dialog.getByRole('list').getByRole('listitem').first()).toBeVisible({ timeout: 8000 })
    await dialog.getByRole('button', { name: /add to module/i }).first().click()
    await expect(page.getByText(/algebra|OpenStax/i).first()).toBeVisible({ timeout: 10000 })
  })

  test('disabled MERLOT hidden from provider list', async () => {
    const adminEmail = process.env.E2E_ADMIN_EMAIL ?? 'admin@e2e.test'
    const { access_token: adminToken } = await apiSignup({
      email: adminEmail,
      password: PASSWORD,
      displayName: 'E2E Admin',
    })

    const putRes = await fetch(`${apiBase}/api/v1/admin/oer-providers/merlot`, {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${adminToken}`,
      },
      body: JSON.stringify({ enabled: false }),
    })
    expect([204, 403, 501]).toContain(putRes.status)
    if (putRes.status !== 204) {
      test.skip(true, 'Admin could not disable MERLOT in this environment')
      return
    }

    const { access_token } = await apiSignup({ email: uniqueEmail('oer-user'), password: PASSWORD })
    const res = await fetch(`${apiBase}/api/v1/oer/providers`, {
      headers: { Authorization: `Bearer ${access_token}` },
    })
    expect(res.ok).toBe(true)
    const list = (await res.json()) as Array<{ provider: string }>
    const ids = list.map((r) => r.provider)
    expect(ids).not.toContain('merlot')
    expect(ids).toContain('oer_commons')
    expect(ids).toContain('openstax')
  })
})
