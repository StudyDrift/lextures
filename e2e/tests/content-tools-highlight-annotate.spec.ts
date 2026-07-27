/**
 * CT.13 — Highlight & Annotate: prompted passage tagging with instructor heat map.
 *
 * Checklist:
 *   [x] Student payload has no expectedRegions
 *   [x] Annotation save stores quote/anchor/tag; status in_progress → completed at min
 *   [x] filterNote blocks filtered content with preserveInput
 *   [x] Analytics facets include tagId / unitIndex
 *   [x] Reset clears annotations
 *   [x] UI: keyboard unit path creates an annotation
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

async function putSettings(
  token: string,
  courseCode: string,
  body: {
    allowedToolIds: string[]
    studentResetAllowed: boolean
    maxInstancesPerItem: number
  },
): Promise<number> {
  const res = await fetch(
    `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/content-tools/settings`,
    {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${token}`,
      },
      body: JSON.stringify(body),
    },
  )
  return res.status
}

function sampleConfig(overrides: Record<string, unknown> = {}) {
  return {
    prompt: 'Highlight every unsupported claim.',
    passageSource: 'inline',
    passageMarkdown:
      'The author claims energy is conserved. However, no evidence is given. The conclusion follows anyway.',
    unitGranularity: 'sentence',
    tags: [
      { id: 'claim', label: 'Claim', color: '#0f766e' },
      { id: 'evidence', label: 'Evidence', color: '#b45309' },
    ],
    minAnnotations: 2,
    maxAnnotations: 10,
    requireNote: false,
    expectedRegions: [{ tagId: 'claim', quote: 'energy is conserved' }],
    ...overrides,
  }
}

async function createInstance(
  token: string,
  courseCode: string,
  structureItemId: string,
  config: Record<string, unknown> = sampleConfig(),
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
        toolId: 'highlight_annotate',
        hostKind: 'content_page',
        structureItemId,
        title: 'CT13 Highlight & Annotate',
        config,
      }),
    },
  )
  if (!res.ok) throw new Error(`create instance failed (${res.status}): ${await res.text()}`)
  return (await res.json()) as { id: string }
}

function fenceMarkdown(instanceId: string): string {
  return [
    '# CT13 page',
    '',
    '```lex-tool',
    JSON.stringify({ instanceId, toolId: 'highlight_annotate', v: 1 }),
    '```',
    '',
  ].join('\n')
}

async function getStudentInstance(
  token: string,
  courseCode: string,
  instanceId: string,
): Promise<{ config?: Record<string, unknown>; state?: Record<string, unknown> }> {
  const res = await fetch(
    `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/content-tools/instances?withState=1`,
    { headers: { Authorization: `Bearer ${token}` } },
  )
  if (!res.ok) throw new Error(`list instances failed (${res.status}): ${await res.text()}`)
  const body = (await res.json()) as {
    instances?: Array<{ id: string; config?: Record<string, unknown>; state?: Record<string, unknown> }>
  }
  const inst = (body.instances ?? []).find((i) => i.id === instanceId)
  if (!inst) throw new Error('instance not found in student list')
  return inst
}

async function putState(
  token: string,
  courseCode: string,
  instanceId: string,
  state: Record<string, unknown>,
  revision = 0,
): Promise<{ status: number; body: Record<string, unknown> }> {
  const res = await fetch(
    `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/content-tools/instances/${instanceId}/state`,
    {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${token}`,
      },
      body: JSON.stringify({ state, revision }),
    },
  )
  const body = (await res.json()) as Record<string, unknown>
  return { status: res.status, body }
}

async function runAction(
  token: string,
  courseCode: string,
  instanceId: string,
  action: string,
  input: Record<string, unknown>,
  idempotencyKey: string,
): Promise<{ status: number; body: Record<string, unknown> }> {
  const res = await fetch(
    `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/content-tools/instances/${instanceId}/actions/${action}`,
    {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${token}`,
      },
      body: JSON.stringify({ input, idempotencyKey }),
    },
  )
  const body = (await res.json()) as Record<string, unknown>
  return { status: res.status, body }
}

async function resetEnrollment(
  token: string,
  courseCode: string,
  instanceId: string,
  enrollmentId: string,
): Promise<number> {
  const res = await fetch(
    `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/content-tools/state-resets`,
    {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${token}`,
      },
      body: JSON.stringify({
        scope: 'instance_enrollment',
        instanceId,
        enrollmentId,
        dryRun: false,
        notify: false,
        reason: 'ct13 e2e reset',
      }),
    },
  )
  return res.status
}

function annotation(
  id: string,
  tagId: string,
  quote: string,
  offset: number,
  unitIndex: number,
) {
  return {
    id,
    tagId,
    quote,
    anchor: {
      prefix: '',
      suffix: '',
      approxOffset: offset,
      unitIndex,
    },
    createdAt: new Date().toISOString(),
  }
}

test.describe('Content Tools Highlight & Annotate (CT.13)', () => {
  test('redaction, completion, filterNote, analytics, UI, reset', async ({
    page,
    seededCourse,
  }) => {
    const { instructorToken, studentToken, courseCode } = seededCourse

    await withCourseFeatureRestore(instructorToken, courseCode, async () => {
      await apiPatchCourseFeatures(instructorToken, courseCode, {
        contentToolsEnabled: true,
      })
      expect(
        await putSettings(instructorToken, courseCode, {
          allowedToolIds: ['highlight_annotate'],
          studentResetAllowed: true,
          maxInstancesPerItem: 50,
        }),
      ).toBe(200)

      const module = await apiCreateModule(instructorToken, courseCode, 'CT13 Module')
      const contentPage = await apiCreateContentPage(
        instructorToken,
        courseCode,
        module.id,
        'CT13 Page',
      )
      const inst = await createInstance(instructorToken, courseCode, contentPage.id)
      await apiPatchContentPage(instructorToken, courseCode, contentPage.id, {
        markdown: fenceMarkdown(inst.id),
      })

      // AC-7: student config has no expectedRegions.
      const studentInst = await getStudentInstance(studentToken, courseCode, inst.id)
      expect(studentInst.config?.expectedRegions).toBeUndefined()
      expect(studentInst.config?.prompt).toBe('Highlight every unsupported claim.')
      expect(Array.isArray(studentInst.config?.tags)).toBe(true)

      // AC-4: one annotation → in_progress; two → completed.
      const one = await putState(studentToken, courseCode, inst.id, {
        v: 1,
        annotations: [
          annotation('a1', 'claim', 'The author claims energy is conserved.', 0, 0),
        ],
      })
      expect(one.status).toBe(200)
      expect((one.body as { status?: string }).status).toBe('in_progress')

      const two = await putState(
        studentToken,
        courseCode,
        inst.id,
        {
          v: 1,
          annotations: [
            annotation('a1', 'claim', 'The author claims energy is conserved.', 0, 0),
            annotation('a2', 'evidence', 'However, no evidence is given.', 40, 1),
          ],
        },
        (one.body as { revision?: number }).revision ?? 1,
      )
      expect(two.status).toBe(200)
      expect((two.body as { status?: string }).status).toBe('completed')
      const st = (two.body as { state?: { annotations?: unknown[]; completedAt?: string } }).state
      expect(st?.annotations).toHaveLength(2)
      expect(st?.completedAt).toBeTruthy()

      // AC-6: filterNote blocks filtered content and preserves input.
      const filtered = await runAction(
        studentToken,
        courseCode,
        inst.id,
        'filter_note',
        { note: 'this contains fuck language' },
        'ct13-filter-1',
      )
      expect(filtered.status).toBe(200)
      const filterResult = filtered.body.result as {
        error?: string
        preserveInput?: boolean
        message?: string
      }
      expect(filterResult.error).toBe('filtered')
      expect(filterResult.preserveInput).toBe(true)
      expect(filterResult.message).toBeTruthy()

      // Instructor analytics facets.
      const analyticsRes = await fetch(
        `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/content-tools/instances/${inst.id}/analytics`,
        { headers: { Authorization: `Bearer ${instructorToken}` } },
      )
      expect(analyticsRes.status).toBe(200)
      const analytics = (await analyticsRes.json()) as {
        facets?: Array<{ key: string; values: Array<{ value: string; count: number }> }>
      }
      const tagFacet = analytics.facets?.find((f) => f.key === 'tagId')
      const unitFacet = analytics.facets?.find((f) => f.key === 'unitIndex')
      expect(tagFacet).toBeTruthy()
      expect((tagFacet?.values?.length ?? 0) > 0).toBe(true)
      expect(unitFacet).toBeTruthy()

      // AC-9: reset clears annotations.
      const rosterRes = await fetch(
        `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/content-tools/instances/${inst.id}/states?page=1&pageSize=50`,
        { headers: { Authorization: `Bearer ${instructorToken}` } },
      )
      expect(rosterRes.status).toBe(200)
      const roster = (await rosterRes.json()) as {
        items: Array<{ enrollmentId: string; status: string }>
      }
      const started = roster.items.find((r) => r.status === 'completed' || r.status === 'in_progress')
      expect(started?.enrollmentId).toBeTruthy()
      expect(
        await resetEnrollment(instructorToken, courseCode, inst.id, started!.enrollmentId),
      ).toBe(200)

      const clearedRes = await fetch(
        `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/content-tools/instances/${inst.id}/state`,
        { headers: { Authorization: `Bearer ${studentToken}` } },
      )
      expect(clearedRes.status).toBe(200)
      const cleared = (await clearedRes.json()) as {
        status: string
        state?: { annotations?: unknown[] }
      }
      expect(cleared.status).toBe('not_started')
      expect(cleared.state?.annotations?.length ?? 0).toBe(0)

      // Fresh instance for UI keyboard path (avoids stale host state after API reset).
      const uiInst = await createInstance(instructorToken, courseCode, contentPage.id)
      await apiPatchContentPage(instructorToken, courseCode, contentPage.id, {
        markdown: fenceMarkdown(uiInst.id),
      })

      await injectToken(page, studentToken)
      await page.goto(`/courses/${courseCode}/modules/content/${contentPage.id}`)
      const tool = page.locator('[data-content-tool="highlight_annotate"]').first()
      await expect(tool).toBeVisible({ timeout: 20_000 })
      await expect(page.getByText('Highlight every unsupported claim.')).toBeVisible()
      await expect(page.getByTestId('ha-progress')).toContainText(/0 of 2/i)

      const unit0 = page.locator('[data-testid="ha-unit-0"]').first()
      await unit0.focus()
      await unit0.press('Enter')
      await expect(page.locator('[data-testid="ha-tag-menu"]')).toBeVisible()
      await page.locator('[data-testid="ha-tag-claim"]').click()
      await expect(page.getByTestId('ha-progress')).toContainText(/1 of 2/i, { timeout: 10_000 })
      await expect(page.locator('[data-testid="ha-tag-menu"]')).toHaveCount(0)
    })
  })
})
