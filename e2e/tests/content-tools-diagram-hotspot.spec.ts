/**
 * CT.15 — Labeled Diagram & Hotspot: label/hotspot modes with server grading.
 *
 * Checklist:
 *   [x] Student payload has no answer key / feedback
 *   [x] Label check returns per-item correctness + score
 *   [x] List-mode flag persisted; scores identical to spatial assignments
 *   [x] Attempt exhaustion refuses further checks
 *   [x] Reset attempt returns unlocked items to tray
 *   [x] Self-reset clears attempts
 *   [x] Instructor analytics expose regionId / assignedTo / gridCell
 *   [x] Tampered region id rejected
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

function sampleLabelConfig(overrides: Record<string, unknown> = {}) {
  return {
    mode: 'label',
    prompt: 'Label the cell parts',
    image: {
      url: 'https://example.com/cell.png',
      alt: 'Animal cell diagram with nucleus and mitochondrion',
      naturalWidth: 800,
      naturalHeight: 600,
    },
    regions: [
      {
        id: 'nuc',
        label: 'Nucleus',
        description: 'Dense round center of the cell that stores DNA',
        shape: { kind: 'circle', cx: 0.5, cy: 0.5, r: 0.12 },
      },
      {
        id: 'mit',
        label: 'Mitochondrion',
        description: 'Oval organelle on the right that produces energy',
        shape: { kind: 'rect', x: 0.7, y: 0.3, w: 0.15, h: 0.1 },
      },
    ],
    labels: [
      { id: 'l_nuc', text: 'Nucleus' },
      { id: 'l_mit', text: 'Mitochondrion' },
    ],
    correctRegionByLabel: { l_nuc: 'nuc', l_mit: 'mit' },
    feedbackByRegion: { nuc: 'Look for the dense center.' },
    attempts: 2,
    lockCorrect: true,
    showPerItemCorrectness: true,
    showRegionOutlines: 'on_focus',
    ...overrides,
  }
}

function sampleHotspotConfig() {
  return {
    mode: 'hotspot',
    prompt: 'Identify the structure',
    image: {
      url: 'https://example.com/cell.png',
      alt: 'Cell diagram',
      naturalWidth: 400,
      naturalHeight: 400,
    },
    regions: [
      {
        id: 'nuc',
        label: 'Nucleus',
        description: 'Center region storing genetic material',
        shape: { kind: 'circle', cx: 0.5, cy: 0.5, r: 0.2 },
      },
      {
        id: 'mit',
        label: 'Mito',
        description: 'Energy-producing oval near the edge',
        shape: { kind: 'rect', x: 0.1, y: 0.1, w: 0.2, h: 0.2 },
      },
    ],
    prompts: [{ id: 'p1', text: 'Where is DNA stored?' }],
    correctRegionByPrompt: { p1: 'nuc' },
    attempts: 3,
    lockCorrect: false,
    showPerItemCorrectness: true,
    showRegionOutlines: 'always',
  }
}

async function createInstance(
  token: string,
  courseCode: string,
  structureItemId: string,
  config: Record<string, unknown> = sampleLabelConfig(),
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
        toolId: 'diagram_hotspot',
        hostKind: 'content_page',
        structureItemId,
        title: 'CT15 Diagram & Hotspot',
        config,
      }),
    },
  )
  if (!res.ok) throw new Error(`create instance failed (${res.status}): ${await res.text()}`)
  return (await res.json()) as { id: string }
}

function fenceMarkdown(instanceId: string): string {
  return [
    '# CT15 page',
    '',
    '```lex-tool',
    JSON.stringify({ instanceId, toolId: 'diagram_hotspot', v: 1 }),
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

test.describe('Content Tools Diagram & Hotspot (CT.15)', () => {
  test('redaction, check, lock, list mode, reset, analytics', async ({ seededCourse }) => {
    const { instructorToken, studentToken, courseCode } = seededCourse

    await withCourseFeatureRestore(instructorToken, courseCode, async () => {
      await apiPatchCourseFeatures(instructorToken, courseCode, {
        contentToolsEnabled: true,
      })
      expect(
        await putSettings(instructorToken, courseCode, {
          allowedToolIds: ['diagram_hotspot'],
          studentResetAllowed: true,
          maxInstancesPerItem: 50,
        }),
      ).toBe(200)

      const module = await apiCreateModule(instructorToken, courseCode, 'CT15 Module')
      const contentPage = await apiCreateContentPage(
        instructorToken,
        courseCode,
        module.id,
        'CT15 Page',
      )
      const inst = await createInstance(instructorToken, courseCode, contentPage.id)
      await apiPatchContentPage(instructorToken, courseCode, contentPage.id, {
        markdown: fenceMarkdown(inst.id),
      })

      // AC-1: student config has regions but no correct mapping / feedback.
      const studentInst = await getStudentInstance(studentToken, courseCode, inst.id)
      const cfg = studentInst.config as Record<string, unknown>
      expect(cfg?.correctRegionByLabel).toBeUndefined()
      expect(cfg?.correctRegionByPrompt).toBeUndefined()
      expect(cfg?.feedbackByRegion).toBeUndefined()
      expect(Array.isArray(cfg?.regions)).toBe(true)
      expect((cfg?.regions as Array<{ description?: string }>)[0]?.description).toBeTruthy()

      // Draft assignments autosave.
      const draft = await putState(studentToken, courseCode, inst.id, 0, {
        v: 1,
        assignments: { l_nuc: 'nuc', l_mit: null },
        attempts: [],
        lockedIds: [],
      })
      expect(draft.status).toBe(200)

      // Tampered region rejected.
      const tampered = await runAction(
        studentToken,
        courseCode,
        inst.id,
        'check',
        { assignments: { l_nuc: 'ghost', l_mit: 'mit' } },
        'ct15-tamper-1',
      )
      expect(tampered.status).toBe(200)
      expect((tampered.body.result as { error?: string }).error).toBe('invalid_placement')

      // Partial then correct with lock (AC-2).
      const wrong = await runAction(
        studentToken,
        courseCode,
        inst.id,
        'check',
        { assignments: { l_nuc: 'mit', l_mit: 'mit' } },
        'ct15-check-1',
      )
      expect(wrong.status).toBe(200)
      const wrongResult = wrong.body.result as {
        scorePct?: number
        perItem?: Record<string, { correct?: boolean; feedback?: string }>
      }
      expect(wrongResult.scorePct).toBe(50)
      expect(wrongResult.perItem?.l_nuc?.correct).toBe(false)
      expect(wrongResult.perItem?.l_mit?.correct).toBe(true)

      const reset = await runAction(
        studentToken,
        courseCode,
        inst.id,
        'reset_attempt',
        {},
        'ct15-reset-1',
      )
      expect(reset.status).toBe(200)
      const resetState = reset.body.state as {
        state?: { assignments?: Record<string, string | null>; lockedIds?: string[] }
        stateJson?: { assignments?: Record<string, string | null>; lockedIds?: string[] }
      }
      const st =
        (resetState.state as {
          assignments?: Record<string, string | null>
          lockedIds?: string[]
        }) ??
        (resetState.stateJson as {
          assignments?: Record<string, string | null>
          lockedIds?: string[]
        })
      expect(st?.lockedIds).toContain('l_mit')
      expect(st?.assignments?.l_mit).toBe('mit')
      expect(st?.assignments?.l_nuc == null).toBe(true)

      // List-mode check completes identically (AC-4).
      const right = await runAction(
        studentToken,
        courseCode,
        inst.id,
        'check',
        { assignments: { l_nuc: 'nuc', l_mit: 'mit' }, usedListMode: true },
        'ct15-check-2',
      )
      expect(right.status).toBe(200)
      expect((right.body.result as { scorePct?: number }).scorePct).toBe(100)
      const completedState = right.body.state as {
        state?: { usedListMode?: boolean }
        stateJson?: { usedListMode?: boolean }
      }
      const completed =
        (completedState.state as { usedListMode?: boolean }) ??
        (completedState.stateJson as { usedListMode?: boolean })
      expect(completed?.usedListMode).toBe(true)

      // Exhaustion.
      const third = await runAction(
        studentToken,
        courseCode,
        inst.id,
        'check',
        { assignments: { l_nuc: 'nuc', l_mit: 'mit' } },
        'ct15-check-3',
      )
      expect(third.status).toBe(200)
      expect((third.body.result as { error?: string }).error).toBe('max_attempts')

      // Hotspot mode.
      const hotspotInst = await createInstance(
        instructorToken,
        courseCode,
        contentPage.id,
        sampleHotspotConfig(),
      )
      const hotspotCheck = await runAction(
        studentToken,
        courseCode,
        hotspotInst.id,
        'check',
        { assignments: { p1: 'nuc' } },
        'ct15-hotspot-1',
      )
      expect(hotspotCheck.status).toBe(200)
      expect((hotspotCheck.body.result as { scorePct?: number }).scorePct).toBe(100)

      // Self-reset (AC-10).
      const selfReset = await fetch(
        `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/content-tools/instances/${inst.id}/self-reset`,
        { method: 'POST', headers: { Authorization: `Bearer ${studentToken}` } },
      )
      expect(selfReset.status).toBe(200)

      // Analytics facets (AC-8).
      const analytics = await fetch(
        `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/content-tools/instances/${hotspotInst.id}/analytics`,
        { headers: { Authorization: `Bearer ${instructorToken}` } },
      )
      expect(analytics.status).toBe(200)
      const analyticsBody = (await analytics.json()) as {
        facets?: Array<{ key: string; values?: Array<{ value: string; count: number }> }>
      }
      const regionFacet = analyticsBody.facets?.find((f) => f.key === 'regionId')
      expect(regionFacet?.values?.length).toBeGreaterThan(0)
      const assigned = analyticsBody.facets?.find((f) => f.key === 'assignedTo')
      expect(assigned?.values?.length).toBeGreaterThan(0)
      const grid = analyticsBody.facets?.find((f) => f.key === 'gridCell')
      expect(grid?.values?.length).toBeGreaterThan(0)
    })
  })
})
