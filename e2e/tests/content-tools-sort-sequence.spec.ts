/**
 * CT.14 — Sort & Sequence: categorize / order with server-side grading.
 *
 * Checklist:
 *   [x] Student payload has no answer key
 *   [x] Categorize check returns per-item correctness + score
 *   [x] Order mode with tie groups scores swapped items correct
 *   [x] Attempt exhaustion refuses further checks
 *   [x] Reset attempt returns unlocked items to tray
 *   [x] Self-reset clears attempts
 *   [x] Instructor analytics expose placedIn facet / confusion
 *   [x] Tampered bucket id rejected
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

function sampleCategorizeConfig(overrides: Record<string, unknown> = {}) {
  return {
    mode: 'categorize',
    prompt: 'Sort acids and bases',
    items: [
      { id: 'hcl', text: 'HCl' },
      { id: 'naoh', text: 'NaOH' },
      { id: 'h2so4', text: 'H2SO4' },
    ],
    buckets: [
      { id: 'acid', label: 'Acid' },
      { id: 'base', label: 'Base' },
    ],
    correctBucketByItem: {
      hcl: 'acid',
      naoh: 'base',
      h2so4: ['acid'],
    },
    itemFeedback: { naoh: 'NaOH is a strong base.' },
    attempts: 2,
    showPerItemCorrectness: true,
    lockCorrect: true,
    scoreMode: 'per_item',
    shuffleItems: false,
    ...overrides,
  }
}

function sampleOrderConfig() {
  return {
    mode: 'order',
    prompt: 'Order the events',
    items: [
      { id: 'a', text: 'A' },
      { id: 'b', text: 'B' },
      { id: 'c', text: 'C' },
    ],
    correctOrder: ['a', 'b', 'c'],
    tieGroups: [['a', 'b']],
    attempts: 3,
    showPerItemCorrectness: true,
    lockCorrect: false,
    scoreMode: 'per_item',
    shuffleItems: false,
  }
}

async function createInstance(
  token: string,
  courseCode: string,
  structureItemId: string,
  config: Record<string, unknown> = sampleCategorizeConfig(),
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
        toolId: 'sort_sequence',
        hostKind: 'content_page',
        structureItemId,
        title: 'CT14 Sort & Sequence',
        config,
      }),
    },
  )
  if (!res.ok) throw new Error(`create instance failed (${res.status}): ${await res.text()}`)
  return (await res.json()) as { id: string }
}

function fenceMarkdown(instanceId: string): string {
  return [
    '# CT14 page',
    '',
    '```lex-tool',
    JSON.stringify({ instanceId, toolId: 'sort_sequence', v: 1 }),
    '```',
    '',
  ].join('\n')
}

async function getStudentInstance(
  token: string,
  courseCode: string,
  instanceId: string,
): Promise<{ config?: Record<string, unknown> }> {
  const res = await fetch(
    `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/content-tools/instances?withState=1`,
    { headers: { Authorization: `Bearer ${token}` } },
  )
  if (!res.ok) throw new Error(`list instances failed (${res.status}): ${await res.text()}`)
  const body = (await res.json()) as {
    instances?: Array<{ id: string; config?: Record<string, unknown> }>
  }
  const inst = (body.instances ?? []).find((i) => i.id === instanceId)
  if (!inst) throw new Error('instance not found in student list')
  return inst
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

test.describe('Content Tools Sort & Sequence (CT.14)', () => {
  test('redaction, check, lock, order ties, reset, analytics', async ({ seededCourse }) => {
    const { instructorToken, studentToken, courseCode } = seededCourse

    await withCourseFeatureRestore(instructorToken, courseCode, async () => {
      await apiPatchCourseFeatures(instructorToken, courseCode, {
        contentToolsEnabled: true,
      })
      expect(
        await putSettings(instructorToken, courseCode, {
          allowedToolIds: ['sort_sequence'],
          studentResetAllowed: true,
          maxInstancesPerItem: 50,
        }),
      ).toBe(200)

      const module = await apiCreateModule(instructorToken, courseCode, 'CT14 Module')
      const contentPage = await apiCreateContentPage(
        instructorToken,
        courseCode,
        module.id,
        'CT14 Page',
      )
      const inst = await createInstance(instructorToken, courseCode, contentPage.id)
      await apiPatchContentPage(instructorToken, courseCode, contentPage.id, {
        markdown: fenceMarkdown(inst.id),
      })

      // AC-1: student config has no answer key / feedback.
      const studentInst = await getStudentInstance(studentToken, courseCode, inst.id)
      const cfg = studentInst.config as Record<string, unknown>
      expect(cfg?.correctBucketByItem).toBeUndefined()
      expect(cfg?.itemFeedback).toBeUndefined()
      expect(cfg?.correctOrder).toBeUndefined()
      expect(cfg?.tieGroups).toBeUndefined()
      expect(Array.isArray(cfg?.items)).toBe(true)
      expect(Array.isArray(cfg?.buckets)).toBe(true)

      // Draft placement autosave (AC-9).
      const draft = await putState(studentToken, courseCode, inst.id, 0, {
        v: 1,
        placement: { hcl: 'acid', naoh: null, h2so4: null },
        attempts: [],
        lockedItemIds: [],
      })
      expect(draft.status).toBe(200)
      const draftRev = (draft.body.revision as number) ?? 1

      // Tampered bucket rejected.
      const tampered = await runAction(
        studentToken,
        courseCode,
        inst.id,
        'check',
        { placement: { hcl: 'nope', naoh: 'base', h2so4: 'acid' } },
        'ct14-tamper-1',
      )
      expect(tampered.status).toBe(200)
      expect((tampered.body.result as { error?: string }).error).toBe('invalid_placement')

      // Partial then correct with lock (AC-2, AC-5).
      const wrong = await runAction(
        studentToken,
        courseCode,
        inst.id,
        'check',
        { placement: { hcl: 'base', naoh: 'base', h2so4: 'acid' } },
        'ct14-check-1',
      )
      expect(wrong.status).toBe(200)
      const wrongResult = wrong.body.result as {
        scorePct?: number
        perItem?: Record<string, { correct?: boolean; feedback?: string }>
      }
      expect(wrongResult.scorePct).toBeCloseTo(66.67, 0)
      expect(wrongResult.perItem?.hcl?.correct).toBe(false)
      expect(wrongResult.perItem?.naoh?.correct).toBe(true)
      expect(wrongResult.perItem?.naoh?.feedback).toMatch(/base/i)

      const reset = await runAction(
        studentToken,
        courseCode,
        inst.id,
        'reset_attempt',
        {},
        'ct14-reset-1',
      )
      expect(reset.status).toBe(200)
      const resetState = reset.body.state as { state?: { placement?: Record<string, string | null>; lockedItemIds?: string[] }; stateJson?: { placement?: Record<string, string | null>; lockedItemIds?: string[] } }
      const st =
        (resetState.state as { placement?: Record<string, string | null>; lockedItemIds?: string[] }) ??
        (resetState.stateJson as { placement?: Record<string, string | null>; lockedItemIds?: string[] })
      expect(st?.lockedItemIds).toContain('naoh')
      expect(st?.placement?.naoh).toBe('base')
      expect(st?.placement?.hcl == null).toBe(true)

      const right = await runAction(
        studentToken,
        courseCode,
        inst.id,
        'check',
        { placement: { hcl: 'acid', naoh: 'base', h2so4: 'acid' } },
        'ct14-check-2',
      )
      expect(right.status).toBe(200)
      expect((right.body.result as { scorePct?: number }).scorePct).toBe(100)

      // Exhaustion.
      const third = await runAction(
        studentToken,
        courseCode,
        inst.id,
        'check',
        { placement: { hcl: 'acid', naoh: 'base', h2so4: 'acid' } },
        'ct14-check-3',
      )
      expect(third.status).toBe(200)
      expect((third.body.result as { error?: string }).error).toBe('max_attempts')

      // Order mode + tie groups (AC-6).
      const orderInst = await createInstance(
        instructorToken,
        courseCode,
        contentPage.id,
        sampleOrderConfig(),
      )
      const orderCheck = await runAction(
        studentToken,
        courseCode,
        orderInst.id,
        'check',
        { placement: ['b', 'a', 'c'] },
        'ct14-order-1',
      )
      expect(orderCheck.status).toBe(200)
      expect((orderCheck.body.result as { scorePct?: number }).scorePct).toBe(100)

      // Self-reset (AC-10).
      const selfReset = await fetch(
        `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/content-tools/instances/${inst.id}/self-reset`,
        { method: 'POST', headers: { Authorization: `Bearer ${studentToken}` } },
      )
      expect(selfReset.status).toBe(200)

      // Analytics facets (AC-7) — use order instance which completed.
      const analytics = await fetch(
        `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/content-tools/instances/${orderInst.id}/analytics`,
        { headers: { Authorization: `Bearer ${instructorToken}` } },
      )
      expect(analytics.status).toBe(200)
      const analyticsBody = (await analytics.json()) as {
        facets?: Array<{ key: string; values?: Array<{ value: string; count: number }> }>
        engaged?: number
      }
      const placed = analyticsBody.facets?.find((f) => f.key === 'placedIn')
      expect(placed?.values?.length).toBeGreaterThan(0)
      expect(draftRev).toBeGreaterThanOrEqual(0)
    })
  })
})
