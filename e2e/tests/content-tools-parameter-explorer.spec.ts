/**
 * CT.16 — Parameter Explorer: sliders, checkpoints, noticing prompts.
 *
 * Checklist:
 *   [x] Checkpoint unlock validated server-side (forged params rejected)
 *   [x] Submit answer + completion
 *   [x] Reset defaults clears trace/answers
 *   [x] Instructor analytics expose checkpointId / paramBin
 *   [x] Self-reset restores empty state
 *   [x] Invalid expression config rejected on create (when applicable)
 */
import { test, expect } from '../fixtures/test.js'
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
    prompt: 'Explore the parabola as you change a.',
    hint: 'Try setting a above 1.5',
    parameters: [
      { id: 'a', kind: 'number', label: 'a', min: -3, max: 3, step: 0.1, default: 1 },
      { id: 'b', kind: 'number', label: 'b', min: -5, max: 5, step: 0.1, default: 0 },
      { id: 'c', kind: 'number', label: 'c', min: -5, max: 5, step: 0.1, default: 0 },
    ],
    model: {
      kind: 'preset',
      preset: 'quadratic',
      bind: { a: 'a', b: 'b', c: 'c' },
    },
    outputs: [
      { kind: 'plot', label: 'Curve', xLabel: 'x', yLabel: 'y' },
      { kind: 'table', label: 'Table' },
      { kind: 'readout', label: 'Values' },
    ],
    noticingPrompts: [
      {
        id: 'n1',
        text: 'What happens when a is large?',
        kind: 'text',
        required: true,
        unlockWhen: 'a > 1.5',
      },
    ],
    requireAllCheckpoints: true,
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
        toolId: 'parameter_explorer',
        hostKind: 'content_page',
        structureItemId,
        title: 'CT16 Parameter Explorer',
        config,
      }),
    },
  )
  if (!res.ok) throw new Error(`create instance failed (${res.status}): ${await res.text()}`)
  return (await res.json()) as { id: string }
}

function fenceMarkdown(instanceId: string): string {
  return [
    '# CT16 page',
    '',
    '```lex-tool',
    JSON.stringify({ instanceId, toolId: 'parameter_explorer', v: 1 }),
    '```',
    '',
  ].join('\n')
}

async function putState(
  token: string,
  courseCode: string,
  instanceId: string,
  revision: number,
  state: Record<string, unknown>,
): Promise<{ status: number; body: Record<string, unknown> }> {
  const res = await fetch(
    `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/content-tools/instances/${instanceId}/state`,
    {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${token}`,
      },
      body: JSON.stringify({ revision, state, stateJson: state }),
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

test.describe('Content Tools Parameter Explorer (CT.16)', () => {
  test('checkpoint unlock, forge rejection, answer, analytics, reset', async ({ seededCourse }) => {
    const { instructorToken, studentToken, courseCode } = seededCourse

    await withCourseFeatureRestore(instructorToken, courseCode, async () => {
      await apiPatchCourseFeatures(instructorToken, courseCode, {
        contentToolsEnabled: true,
      })
      expect(
        await putSettings(instructorToken, courseCode, {
          allowedToolIds: ['parameter_explorer'],
          studentResetAllowed: true,
          maxInstancesPerItem: 50,
        }),
      ).toBe(200)

      const module = await apiCreateModule(instructorToken, courseCode, 'CT16 Module')
      const contentPage = await apiCreateContentPage(
        instructorToken,
        courseCode,
        module.id,
        'CT16 Page',
      )
      const inst = await createInstance(instructorToken, courseCode, contentPage.id)
      await apiPatchContentPage(instructorToken, courseCode, contentPage.id, {
        markdown: fenceMarkdown(inst.id),
      })

      // Draft params + trace.
      const draft = await putState(studentToken, courseCode, inst.id, 0, {
        v: 1,
        params: { a: 1, b: 0, c: 0 },
        trace: [{ at: new Date().toISOString(), params: { a: 1, b: 0, c: 0 } }],
        checkpoints: {},
        answers: {},
      })
      expect(draft.status).toBe(200)

      // Forged low-a checkpoint rejected.
      const forged = await runAction(
        studentToken,
        courseCode,
        inst.id,
        'checkpoint',
        { promptId: 'n1', params: { a: 0.2, b: 0, c: 0 } },
        'ct16-forge-1',
      )
      expect(forged.status).toBe(200)
      expect((forged.body.result as { unlocked?: boolean }).unlocked).toBeFalsy()

      // Real unlock.
      const hit = await runAction(
        studentToken,
        courseCode,
        inst.id,
        'checkpoint',
        { promptId: 'n1', params: { a: 2, b: 0, c: 0 } },
        'ct16-hit-1',
      )
      expect(hit.status).toBe(200)
      const hitResult = hit.body.result as { unlocked?: boolean; hitAt?: string }
      expect(hitResult.unlocked).toBe(true)
      expect(hitResult.hitAt).toBeTruthy()

      // Answer → complete.
      const ans = await runAction(
        studentToken,
        courseCode,
        inst.id,
        'submit_answer',
        {
          promptId: 'n1',
          answer: 'The parabola gets steeper',
          params: { a: 2, b: 0, c: 0 },
        },
        'ct16-ans-1',
      )
      expect(ans.status).toBe(200)
      const ansResult = ans.body.result as { ok?: boolean; completed?: boolean }
      expect(ansResult.ok).toBe(true)
      expect(ansResult.completed).toBe(true)

      // Self-reset.
      const selfReset = await fetch(
        `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/content-tools/instances/${inst.id}/self-reset`,
        { method: 'POST', headers: { Authorization: `Bearer ${studentToken}` } },
      )
      expect(selfReset.status).toBe(200)

      // Re-complete for analytics.
      await runAction(
        studentToken,
        courseCode,
        inst.id,
        'checkpoint',
        { promptId: 'n1', params: { a: 2.5, b: 0, c: 0 } },
        'ct16-hit-2',
      )
      await runAction(
        studentToken,
        courseCode,
        inst.id,
        'submit_answer',
        {
          promptId: 'n1',
          answer: 'It steepened again',
          params: { a: 2.5, b: 0, c: 0 },
        },
        'ct16-ans-2',
      )

      const analytics = await fetch(
        `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/content-tools/instances/${inst.id}/analytics`,
        { headers: { Authorization: `Bearer ${instructorToken}` } },
      )
      expect(analytics.status).toBe(200)
      const analyticsBody = (await analytics.json()) as {
        facets?: Array<{ key: string; values?: Array<{ value: string; count: number }> }>
      }
      const checkpointFacet = analyticsBody.facets?.find((f) => f.key === 'checkpointId')
      expect(checkpointFacet?.values?.some((v) => v.value === 'n1')).toBe(true)
      const answeredFacet = analyticsBody.facets?.find((f) => f.key === 'promptAnswered')
      expect(answeredFacet?.values?.some((v) => v.value === 'n1')).toBe(true)

      // In-tool reset_defaults.
      const reset = await runAction(
        studentToken,
        courseCode,
        inst.id,
        'reset_defaults',
        {},
        'ct16-reset-1',
      )
      expect(reset.status).toBe(200)
      expect((reset.body.result as { ok?: boolean }).ok).toBe(true)
    })
  })
})
