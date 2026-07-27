/**
 * CT.6 — Grounded context sources: discover links, block private URLs, exclude.
 *
 * Checklist:
 *   [x] Context pack preview discovers body links with section-first ordering
 *   [x] Private / metadata URLs surface as blocked: private network
 *   [x] Instructor can exclude a source via PATCH
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
        allowedToolIds: [],
        studentResetAllowed: false,
        maxInstancesPerItem: 50,
        monthlyAiTokenBudget: 0,
        dailyAiCallsPerUser: 50,
        linkIngestionMode: 'public',
        linkHostAllowlist: [],
      }),
    },
  )
  if (!res.ok) throw new Error(`settings put failed: ${res.status} ${await res.text()}`)
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
        title: 'CT6 context probe',
        config: {
          prompt: 'Ask about the linked sources',
          answerKey: 'x',
          maxAttempts: 3,
        },
      }),
    },
  )
  if (!res.ok) throw new Error(`create instance failed (${res.status}): ${await res.text()}`)
  return (await res.json()) as { id: string }
}

test.describe('Content Tools grounded context (CT.6)', () => {
  test('discover sources, block private URL, exclude link', async ({ page, seededCourse }) => {
    test.setTimeout(90_000)
    const { instructorToken, courseCode } = seededCourse

    await withCourseFeatureRestore(instructorToken, courseCode, async () => {
      await apiPatchCourseFeatures(instructorToken, courseCode, {
        contentToolsEnabled: true,
      })
      await putSettings(instructorToken, courseCode)

      const mod = await apiCreateModule(instructorToken, courseCode, 'CT6 Module')
      const pageItem = await apiCreateContentPage(instructorToken, courseCode, mod.id, 'CT6 Page')
      const inst = await createInstance(instructorToken, courseCode, pageItem.id)

      const md = [
        '# Linked lesson',
        '',
        'Read [Standards](https://standards.example/a), also https://config.example/extra, and see http://169.254.169.254/latest/meta-data.',
        '',
        '```lex-tool',
        JSON.stringify({ instanceId: inst.id, toolId: 'noop_probe', v: 1 }),
        '```',
        '',
      ].join('\n')
      await apiPatchContentPage(instructorToken, courseCode, pageItem.id, { markdown: md })

      const previewRes = await fetch(
        `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/content-tools/context/preview`,
        {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            Authorization: `Bearer ${instructorToken}`,
          },
          body: JSON.stringify({ instanceId: inst.id }),
        },
      )
      expect(previewRes.status).toBe(200)
      const pack = (await previewRes.json()) as {
        segments: Array<{ kind: string; text: string }>
        pendingSources: Array<{ url: string; status: string; reason?: string }>
      }
      expect(pack.segments.length).toBeGreaterThan(0)
      expect(pack.segments[0]?.kind).toBe('section')

      const blocked = pack.pendingSources.find((p) => p.url.includes('169.254.169.254'))
      expect(blocked).toBeTruthy()
      expect(blocked?.status).toBe('blocked')
      expect(blocked?.reason ?? '').toMatch(/private network/i)

      const listRes = await fetch(
        `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/content-tools/context/sources?itemId=${pageItem.id}&instanceId=${inst.id}`,
        { headers: { Authorization: `Bearer ${instructorToken}` } },
      )
      expect(listRes.status).toBe(200)
      const list = (await listRes.json()) as {
        items: Array<{ id: string; url: string; status: string; excluded: boolean }>
      }
      expect(list.items.length).toBeGreaterThan(0)

      const standards = list.items.find((i) => i.url.includes('standards.example'))
      expect(standards).toBeTruthy()
      if (!standards) return

      const patchRes = await fetch(
        `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/content-tools/context/sources/${standards.id}`,
        {
          method: 'PATCH',
          headers: {
            'Content-Type': 'application/json',
            Authorization: `Bearer ${instructorToken}`,
          },
          body: JSON.stringify({ excluded: true }),
        },
      )
      expect(patchRes.status).toBe(200)
      const patched = (await patchRes.json()) as { excluded: boolean }
      expect(patched.excluded).toBe(true)

      // Confirm instances list is non-empty before UI assertions.
      const instListRes = await fetch(
        `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/content-tools/instances`,
        { headers: { Authorization: `Bearer ${instructorToken}` } },
      )
      expect(instListRes.status).toBe(200)
      const instList = (await instListRes.json()) as { instances?: unknown[] }
      expect((instList.instances ?? []).length).toBeGreaterThan(0)

      // UI: insights page shows sources panel after opening responses for the instance.
      await injectToken(page, instructorToken)
      await page.goto(`/courses/${encodeURIComponent(courseCode)}/content-tools`)
      await expect(page.getByTestId('content-tools-insights')).toBeVisible({ timeout: 15000 })
      const ack = page.getByRole('button', { name: /i acknowledge/i })
      if (await ack.isVisible().catch(() => false)) {
        await ack.click()
      }
      await page.getByTestId(`ct-open-sources-${inst.id}`).click()
      await expect(page.getByTestId('content-tools-sources-panel')).toBeVisible({ timeout: 15000 })
      await expect(page.getByTestId('ct-sources-empty').or(page.locator('table'))).toBeVisible()
    })
  })
})
