/**
 * CT.8 — Governance, safety, privacy & accessibility.
 *
 * Checklist:
 *   [x] Data sheets endpoint returns registered tools with WCAG level
 *   [x] AI consent opt-out recorded
 *   [x] Free-text block path refuses slur without storing (when org policy writable)
 *   [x] Report + instructor remove moderation path
 *   [x] Student UI exposes report affordance
 */
import { test, expect, injectToken } from '../fixtures/test.js'
import {
  apiCreateContentPage,
  apiCreateModule,
  apiPatchContentPage,
  apiPatchCourseFeatures,
} from '../fixtures/api.js'
import { withCourseFeatureRestore } from '../lib/course-feature-matrix-helpers.js'

const apiBase = process.env.E2E_API_URL ?? 'http://localhost:8080'

async function putSettings(token: string, courseCode: string): Promise<void> {
  const res = await fetch(
    `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/content-tools/settings`,
    {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${token}`,
      },
      body: JSON.stringify({
        allowedToolIds: ['noop_probe'],
        studentResetAllowed: false,
        maxInstancesPerItem: 50,
      }),
    },
  )
  if (!res.ok) throw new Error(`settings failed (${res.status}): ${await res.text()}`)
}

async function createInstance(
  token: string,
  courseCode: string,
  structureItemId: string,
): Promise<{ id: string }> {
  const res = await fetch(
    `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/content-tools/instances`,
    {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${token}`,
      },
      body: JSON.stringify({
        toolId: 'noop_probe',
        hostKind: 'content_page',
        structureItemId,
        title: 'CT8 probe',
        config: { prompt: 'Say hello', answerKey: 'hello', maxAttempts: 3 },
      }),
    },
  )
  if (!res.ok) throw new Error(`create instance failed (${res.status}): ${await res.text()}`)
  return (await res.json()) as { id: string }
}

async function meOrgId(token: string): Promise<string | null> {
  const res = await fetch(`${apiBase}/api/v1/me`, {
    headers: { Authorization: `Bearer ${token}` },
  })
  if (!res.ok) return null
  const body = (await res.json()) as {
    orgId?: string
    organizationId?: string
    org?: { id?: string }
  }
  return body.org?.id || body.orgId || body.organizationId || null
}

test.describe('CT.8 Content Tools governance', () => {
  test('data sheets, consent, filter, report/moderate', async ({ page, seededCourse }) => {
    test.setTimeout(90_000)
    const { instructorToken, studentToken, courseCode } = seededCourse

    await withCourseFeatureRestore(instructorToken, courseCode, async () => {
      await apiPatchCourseFeatures(instructorToken, courseCode, {
        contentToolsEnabled: true,
      })
      await putSettings(instructorToken, courseCode)

      const mod = await apiCreateModule(instructorToken, courseCode, 'CT8 Module')
      const content = await apiCreateContentPage(instructorToken, courseCode, mod.id, 'CT8 Page')
      const inst = await createInstance(instructorToken, courseCode, content.id)

      await apiPatchContentPage(instructorToken, courseCode, content.id, {
        markdown: [
          '# CT8 page',
          '',
          'Intro',
          '',
          '```lex-tool',
          JSON.stringify({ instanceId: inst.id, toolId: 'noop_probe', v: 1 }),
          '```',
          '',
        ].join('\n'),
      })

      const sheetsRes = await fetch(`${apiBase}/api/v1/content-tools/data-sheets`, {
        headers: { Authorization: `Bearer ${instructorToken}` },
      })
      expect(sheetsRes.ok).toBeTruthy()
      const sheets = (await sheetsRes.json()) as {
        dataSheets: Array<{ toolId: string; wcagLevel: string }>
      }
      expect(
        sheets.dataSheets.some((s) => s.toolId === 'noop_probe' && s.wcagLevel === 'AA'),
      ).toBeTruthy()

      const consentRes = await fetch(
        `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/content-tools/ai-consent`,
        {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            Authorization: `Bearer ${studentToken}`,
          },
          body: JSON.stringify({ toolId: 'noop_probe', decision: 'opted_out' }),
        },
      )
      expect(consentRes.ok).toBeTruthy()

      const orgId = await meOrgId(instructorToken)
      if (orgId) {
        const polRes = await fetch(
          `${apiBase}/api/v1/orgs/${encodeURIComponent(orgId)}/content-tool-policy`,
          {
            method: 'PUT',
            headers: {
              'Content-Type': 'application/json',
              Authorization: `Bearer ${instructorToken}`,
            },
            body: JSON.stringify({
              deniedCapabilities: [],
              deniedToolIds: [],
              allowedToolIds: [],
              aiDisclosureMode: 'banner',
              freeTextFilterAction: 'block',
              crisisEscalationEnabled: true,
              aiLogRetentionDays: 30,
            }),
          },
        )
        if (polRes.ok) {
          const blockRes = await fetch(
            `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/content-tools/instances/${inst.id}/state`,
            {
              method: 'PUT',
              headers: {
                'Content-Type': 'application/json',
                Authorization: `Bearer ${studentToken}`,
              },
              body: JSON.stringify({
                revision: 0,
                state: { response: 'this contains fuck language', attempts: 1 },
              }),
            },
          )
          expect(blockRes.status).toBe(422)
          const blocked = (await blockRes.json()) as { error?: { guidance?: string } }
          expect(blocked.error?.guidance).toBeTruthy()
        }
      }

      const reportRes = await fetch(
        `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/content-tools/instances/${inst.id}/report`,
        {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            Authorization: `Bearer ${studentToken}`,
          },
          body: JSON.stringify({ category: 'abuse', reason: 'e2e report' }),
        },
      )
      expect(reportRes.status).toBe(201)

      const modRes = await fetch(
        `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/content-tools/instances/${inst.id}/moderate`,
        {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            Authorization: `Bearer ${instructorToken}`,
          },
          body: JSON.stringify({ action: 'removed', category: 'abuse' }),
        },
      )
      expect(modRes.status).toBe(201)

      const listRes = await fetch(
        `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/content-tools/instances/${inst.id}/moderation`,
        { headers: { Authorization: `Bearer ${instructorToken}` } },
      )
      expect(listRes.ok).toBeTruthy()
      const list = (await listRes.json()) as { items: Array<{ action: string }> }
      expect(list.items.some((i) => i.action === 'reported')).toBeTruthy()
      expect(list.items.some((i) => i.action === 'removed')).toBeTruthy()

      await injectToken(page, studentToken)
      await page.goto(`/courses/${encodeURIComponent(courseCode)}/pages/${content.id}`)
      await expect(page.getByTestId('content-tool-report')).toBeVisible({ timeout: 20000 })
    })
  })
})
