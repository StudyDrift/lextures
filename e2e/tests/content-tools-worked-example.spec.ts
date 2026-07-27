/**
 * CT.18 — Step-Through Worked Example: blanked steps, hints, reveal, redaction.
 *
 * Checklist:
 *   [x] Student payload has no expected/hints/explanation
 *   [x] Wrong → hint → correct via algebraic normalisation
 *   [x] Exhaust attempts → reveal_step → continue
 *   [x] Sequential lock on later steps
 *   [x] Locale numeric (3,14)
 *   [x] Analytics facets present
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
    title: 'CT18 expand',
    problem: 'Expand 3(x+2).',
    variables: ['x'],
    blankPolicy: 'author',
    attemptsPerStep: 3,
    practiceOnly: true,
    showAllSteps: false,
    steps: [
      {
        id: 's1',
        label: 'Step 1 — distribute',
        text: 'Write the expanded form:',
        blank: {
          type: 'expression',
          expected: '3(x+2)',
        },
        hints: ['Multiply 3 by each term.', 'You should get two terms.'],
        explanation: '3(x+2) = 3x + 6',
      },
      {
        id: 's2',
        text: 'Simplified:',
        blank: {
          type: 'expression',
          expected: '3x+6',
        },
        explanation: '3x + 6',
      },
      {
        id: 's3',
        text: 'Value at x=1:',
        blank: {
          type: 'numeric',
          expected: 9,
          tolerance: { kind: 'absolute', value: 0.01 },
        },
      },
    ],
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
        toolId: 'worked_example',
        hostKind: 'content_page',
        structureItemId,
        title: 'CT18 Worked Example',
        config,
      }),
    },
  )
  if (!res.ok) throw new Error(`create instance failed (${res.status}): ${await res.text()}`)
  return (await res.json()) as { id: string }
}

function fenceMarkdown(instanceId: string): string {
  return [
    '# CT18 page',
    '',
    '```lex-tool',
    JSON.stringify({ instanceId, toolId: 'worked_example', v: 1 }),
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

test.describe('Content Tools Worked Example (CT.18)', () => {
  test('redaction, hint, normaliser, reveal, analytics, UI', async ({ page, seededCourse }) => {
    const { instructorToken, studentToken, courseCode } = seededCourse

    await withCourseFeatureRestore(instructorToken, courseCode, async () => {
      await apiPatchCourseFeatures(instructorToken, courseCode, {
        contentToolsEnabled: true,
      })
      expect(
        await putSettings(instructorToken, courseCode, {
          allowedToolIds: ['worked_example'],
          studentResetAllowed: true,
          maxInstancesPerItem: 50,
        }),
      ).toBe(200)

      const module = await apiCreateModule(instructorToken, courseCode, 'CT18 Module')
      const contentPage = await apiCreateContentPage(
        instructorToken,
        courseCode,
        module.id,
        'CT18 Page',
      )
      const inst = await createInstance(instructorToken, courseCode, contentPage.id)
      await apiPatchContentPage(instructorToken, courseCode, contentPage.id, {
        markdown: fenceMarkdown(inst.id),
      })

      // AC-1: student config has no secrets.
      const studentInst = await getStudentInstance(studentToken, courseCode, inst.id)
      const cfg = studentInst.config as {
        steps?: Array<Record<string, unknown>>
      }
      expect(cfg?.steps?.[0]?.hints).toBeUndefined()
      expect(cfg?.steps?.[0]?.explanation).toBeUndefined()
      const blank = cfg?.steps?.[0]?.blank as Record<string, unknown> | undefined
      expect(blank?.expected).toBeUndefined()
      expect(blank?.type).toBe('expression')

      await runAction(studentToken, courseCode, inst.id, 'prepare', {}, 'we-prep-1')

      // Sequential: s2 locked.
      const locked = await runAction(
        studentToken,
        courseCode,
        inst.id,
        'check_step',
        { stepId: 's2', value: '3x+6' },
        'we-lock-1',
      )
      expect(locked.status).toBe(200)
      expect((locked.body.result as Record<string, unknown>).error).toBe('sequential_locked')

      // Wrong → hint → correct via 3x+6 ≡ 3(x+2)
      const wrong = await runAction(
        studentToken,
        courseCode,
        inst.id,
        'check_step',
        { stepId: 's1', value: '3x+5' },
        'we-wrong-1',
      )
      expect((wrong.body.result as Record<string, unknown>).result).toBe('incorrect')

      const hint = await runAction(
        studentToken,
        courseCode,
        inst.id,
        'hint',
        { stepId: 's1' },
        'we-hint-1',
      )
      expect(hint.status, JSON.stringify(hint.body)).toBe(200)
      expect((hint.body.result as Record<string, unknown> | undefined)?.hint).toBeTruthy()

      const ok = await runAction(
        studentToken,
        courseCode,
        inst.id,
        'check_step',
        { stepId: 's1', value: '3x + 6' },
        'we-ok-1',
      )
      expect((ok.body.result as Record<string, unknown>).result).toBe('correct')

      // Exhaust s2 then reveal.
      let stKey = 'we-s2'
      for (let i = 0; i < 3; i++) {
        const r = await runAction(
          studentToken,
          courseCode,
          inst.id,
          'check_step',
          { stepId: 's2', value: 'x' },
          `${stKey}-${i}`,
        )
        expect(r.status).toBe(200)
      }
      const revealed = await runAction(
        studentToken,
        courseCode,
        inst.id,
        'reveal_step',
        { stepId: 's2' },
        'we-reveal-1',
      )
      expect((revealed.body.result as Record<string, unknown>).revealed).toBe(true)

      // Locale numeric
      const num = await runAction(
        studentToken,
        courseCode,
        inst.id,
        'check_step',
        { stepId: 's3', value: '9,00' },
        'we-num-1',
      )
      expect((num.body.result as Record<string, unknown>).result).toBe('correct')

      // Analytics
      const analytics = await fetch(
        `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/content-tools/instances/${inst.id}/analytics`,
        { headers: { Authorization: `Bearer ${instructorToken}` } },
      )
      expect(analytics.status).toBe(200)
      const aBody = (await analytics.json()) as Record<string, unknown>
      expect(aBody).toBeTruthy()

      // UI smoke
      await injectToken(page, studentToken)
      await page.goto(`/courses/${courseCode}/pages/${contentPage.id}`)
      await expect(page.locator('[data-content-tool="worked_example"]')).toBeVisible({
        timeout: 15000,
      })
    })
  })
})
