/**
 * CT.17 — Code Sandbox: runnable cell with instructor tests.
 *
 * Checklist:
 *   [x] Student payload redacts test input/expected/feedback
 *   [x] Run returns stdout and persists code + history
 *   [x] Check returns pass/fail without hidden secrets; scores when auto
 *   [x] Rate limit returns typed error with resetAt
 *   [x] Reset code restores starter and keeps history
 *   [x] Analytics facets include testId / passed
 */
import { test, expect } from '../fixtures/test.js'
import {
  apiCreateContentPage,
  apiCreateModule,
  apiGetPlatformAdminToken,
  apiPatchContentPage,
  apiPatchCourseFeatures,
  apiPatchPlatformSettings,
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
    language: 'python',
    prompt: 'Double the input integer',
    starterCode: 'n = int(input())\nprint(n)\n',
    sampleInput: '3',
    tests: [
      {
        id: 't1',
        name: 'doubles 3',
        input: '3',
        expectedOutput: '6',
        hidden: false,
        feedback: 'Print n * 2',
      },
      {
        id: 't2',
        name: 'doubles 10',
        input: '10',
        expectedOutput: '20',
        hidden: true,
        feedback: 'Hidden case',
      },
    ],
    runLimitPerHour: 30,
    checkLimitPerHour: 20,
    editorMode: 'plain',
    scoringMode: 'auto',
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
        toolId: 'code_sandbox',
        hostKind: 'content_page',
        structureItemId,
        title: 'CT17 Code Sandbox',
        config,
      }),
    },
  )
  if (!res.ok) throw new Error(`create instance failed (${res.status}): ${await res.text()}`)
  return (await res.json()) as { id: string }
}

function fenceMarkdown(instanceId: string): string {
  return [
    '# CT17 page',
    '',
    '```lex-tool',
    JSON.stringify({ instanceId, toolId: 'code_sandbox', v: 1 }),
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

function stateDoc(body: Record<string, unknown>): Record<string, unknown> {
  const envelope = body.state as { state?: Record<string, unknown>; stateJson?: Record<string, unknown> } | undefined
  return (envelope?.state ?? envelope?.stateJson ?? {}) as Record<string, unknown>
}

test.describe('Content Tools Code Sandbox (CT.17)', () => {
  test('redaction, run, check, rate limit, reset, analytics', async ({ seededCourse }) => {
    const { instructorToken, studentToken, courseCode } = seededCourse
    const adminToken = await apiGetPlatformAdminToken()
    await apiPatchPlatformSettings(adminToken, { codeExecutionEnabled: true })

    await withCourseFeatureRestore(instructorToken, courseCode, async () => {
      await apiPatchCourseFeatures(instructorToken, courseCode, { contentToolsEnabled: true })
      expect(
        await putSettings(instructorToken, courseCode, {
          allowedToolIds: ['code_sandbox'],
          studentResetAllowed: true,
          maxInstancesPerItem: 10,
        }),
      ).toBe(200)

      const mod = await apiCreateModule(instructorToken, courseCode, 'CT17 Module')
      const page = await apiCreateContentPage(instructorToken, courseCode, mod.id, 'CT17 Page')
      const inst = await createInstance(instructorToken, courseCode, page.id)
      await apiPatchContentPage(instructorToken, courseCode, page.id, {
        markdown: fenceMarkdown(inst.id),
      })

      const studentView = await getStudentInstance(studentToken, courseCode, inst.id)
      const tests = studentView.config?.tests as Array<Record<string, unknown>> | undefined
      expect(Array.isArray(tests)).toBe(true)
      for (const tc of tests ?? []) {
        expect(tc.input).toBeUndefined()
        expect(tc.expectedOutput).toBeUndefined()
        expect(tc.feedback).toBeUndefined()
        expect(tc.id).toBeTruthy()
        expect(tc.name).toBeTruthy()
      }

      const correct = 'n = int(input())\nprint(n * 2)\n'
      const put = await putState(studentToken, courseCode, inst.id, 0, {
        v: 1,
        code: correct,
        runs: [],
      })
      expect(put.status).toBe(200)

      const run = await runAction(
        studentToken,
        courseCode,
        inst.id,
        'run',
        { code: correct, stdin: '3' },
        'ct17-run-1',
      )
      expect(run.status).toBe(200)
      const runResult = run.body.result as { error?: string; stdout?: string }
      expect(runResult.error).toBeUndefined()
      expect(String(runResult.stdout ?? '')).toContain('6')

      const failCheck = await runAction(
        studentToken,
        courseCode,
        inst.id,
        'check',
        { code: 'n = int(input())\nprint(n)\n' },
        'ct17-check-fail',
      )
      expect(failCheck.status).toBe(200)
      const failResult = failCheck.body.result as {
        passed?: number
        tests?: Array<Record<string, unknown>>
      }
      expect(Number(failResult.passed)).toBe(0)
      expect(failResult.tests?.length).toBe(2)
      for (const tc of failResult.tests ?? []) {
        expect(tc.input).toBeUndefined()
        expect(tc.expectedOutput).toBeUndefined()
      }
      const hidden = failResult.tests?.find((t) => t.hidden === true)
      expect(hidden?.feedback).toBe('Hidden case')

      const passCheck = await runAction(
        studentToken,
        courseCode,
        inst.id,
        'check',
        { code: correct },
        'ct17-check-pass',
      )
      expect(passCheck.status).toBe(200)
      const passResult = passCheck.body.result as { passed?: number; total?: number }
      expect(Number(passResult.passed)).toBe(2)
      expect(Number(passResult.total)).toBe(2)
      const passState = stateDoc(passCheck.body)
      expect(String(passState.code ?? '')).toContain('n * 2')
      expect(Array.isArray(passState.runs) && (passState.runs as unknown[]).length > 0).toBe(true)

      const limited = await createInstance(
        instructorToken,
        courseCode,
        page.id,
        sampleConfig({
          runLimitPerHour: 1,
          starterCode: 'print(1)',
          tests: [],
          scoringMode: 'none',
        }),
      )
      const r1 = await runAction(
        studentToken,
        courseCode,
        limited.id,
        'run',
        { code: 'print(1)' },
        'ct17-lim-1',
      )
      expect(r1.status).toBe(200)
      const r2 = await runAction(
        studentToken,
        courseCode,
        limited.id,
        'run',
        { code: 'print(2)' },
        'ct17-lim-2',
      )
      expect(r2.status).toBe(200)
      const r2Result = r2.body.result as { error?: string; resetAt?: number }
      expect(r2Result.error).toBe('rate_limited')
      expect(r2Result.resetAt).toBeTruthy()

      const reset = await runAction(
        studentToken,
        courseCode,
        inst.id,
        'reset_code',
        {},
        'ct17-reset',
      )
      expect(reset.status).toBe(200)
      const resetResult = reset.body.result as { code?: string }
      expect(String(resetResult.code)).toContain('print(n)')
      const resetState = stateDoc(reset.body)
      expect(Array.isArray(resetState.runs) && (resetState.runs as unknown[]).length > 0).toBe(true)

      const analytics = await fetch(
        `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/content-tools/instances/${inst.id}/analytics`,
        { headers: { Authorization: `Bearer ${instructorToken}` } },
      )
      expect(analytics.status).toBe(200)
      const analyticsBody = (await analytics.json()) as {
        facets?: Array<{ key: string }>
      }
      const facetKeys = (analyticsBody.facets ?? []).map((f) => f.key)
      expect(facetKeys).toEqual(expect.arrayContaining(['testId', 'passed']))
    })
  })
})
