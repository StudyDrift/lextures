/**
 * CT.20 — Explain It Back: formative self-explanation with AI feedback.
 *
 * Checklist:
 *   [x] Student payload has no key points
 *   [x] Submit → dry-run AI feedback → revise → second attempt persists
 *   [x] aiFeedback:false runs in review mode
 *   [x] Instructor reset clears attempts
 *   [x] Analytics expose key-point facets / representatives
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

const sampleText =
  'Stoichiometry uses mole ratios from balanced equations so chemists can predict how much product forms before a reactant runs out in the lab during practice.'

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
    prompt: 'In your own words, explain why stoichiometry matters in the lab.',
    minWords: 10,
    maxWords: 150,
    keyPoints: [
      { id: 'kp1', label: 'ratio', description: 'Mentions mole ratios' },
      { id: 'kp2', label: 'balance', description: 'Mentions balanced equations' },
      { id: 'kp3', label: 'limit', description: 'Mentions limiting reactant' },
    ],
    revealKeyPointsAfterSubmit: true,
    aiFeedback: true,
    feedbackStyle: 'encouraging',
    attempts: 3,
    includeProbeQuestion: true,
    allowInstructorNote: true,
    maxSubmissionsPerDay: 10,
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
        toolId: 'explain_it_back',
        hostKind: 'content_page',
        structureItemId,
        title: 'CT20 Explain It Back',
        config,
      }),
    },
  )
  if (!res.ok) throw new Error(`create instance failed (${res.status}): ${await res.text()}`)
  return (await res.json()) as { id: string }
}

function fenceMarkdown(instanceId: string): string {
  return [
    '# CT20 page',
    '',
    'Stoichiometry connects ratios to lab amounts.',
    '',
    '```lex-tool',
    JSON.stringify({ instanceId, toolId: 'explain_it_back', v: 1 }),
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

async function runSubmit(
  token: string,
  courseCode: string,
  instanceId: string,
  text: string,
): Promise<{ status: number; body: Record<string, unknown> }> {
  const res = await fetch(
    `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/content-tools/instances/${instanceId}/actions/submit`,
    {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${token}`,
      },
      body: JSON.stringify({
        input: { text },
        idempotencyKey: crypto.randomUUID(),
      }),
    },
  )
  const body = (await res.json()) as Record<string, unknown>
  return { status: res.status, body }
}

test.describe('Content Tools Explain It Back (CT.20)', () => {
  test('redacted config, AI feedback, revise, reset, analytics', async ({
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
          allowedToolIds: ['explain_it_back', 'noop_probe'],
          studentResetAllowed: true,
          maxInstancesPerItem: 50,
        }),
      ).toBe(200)

      const mod = await apiCreateModule(instructorToken, courseCode, `CT20 ${Date.now()}`)
      const contentPage = await apiCreateContentPage(
        instructorToken,
        courseCode,
        mod.id,
        'CT20 page',
      )
      const inst = await createInstance(instructorToken, courseCode, contentPage.id)
      await apiPatchContentPage(instructorToken, courseCode, contentPage.id, {
        markdown: fenceMarkdown(inst.id),
      })

      // AC-1: student payload must not include keyPoints.
      const studentInst = await getStudentInstance(studentToken, courseCode, inst.id)
      expect(studentInst.config?.keyPoints).toBeUndefined()
      expect(studentInst.config?.prompt).toBeTruthy()

      const submit1 = await runSubmit(studentToken, courseCode, inst.id, sampleText)
      expect(submit1.status).toBe(200)
      const result1 = submit1.body.result as Record<string, unknown>
      expect(result1.error).toBeUndefined()
      expect(result1.mode).toBe('ai')
      expect(result1.feedback).toBeTruthy()
      const state1 = submit1.body.state as {
        state?: { attempts?: unknown[]; completedAt?: string }
        status?: string
      }
      expect((state1.state?.attempts ?? []).length).toBe(1)
      expect(state1.status).toBe('completed')

      const submit2 = await runSubmit(
        studentToken,
        courseCode,
        inst.id,
        `${sampleText} A revision mentioning the limiting reactant more clearly.`,
      )
      expect(submit2.status).toBe(200)
      const state2 = submit2.body.state as { state?: { attempts?: unknown[] } }
      expect((state2.state?.attempts ?? []).length).toBe(2)

      await injectToken(page, studentToken)
      await page.goto(`/courses/${courseCode}/modules/content/${contentPage.id}`)
      await expect(page.getByTestId('explain-it-back')).toBeVisible({ timeout: 20_000 })
      await expect(page.getByTestId('explain-it-back-feedback')).toBeVisible()

      const analyticsRes = await fetch(
        `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/content-tools/instances/${inst.id}/analytics`,
        { headers: { Authorization: `Bearer ${instructorToken}` } },
      )
      expect(analyticsRes.status).toBe(200)
      const analytics = (await analyticsRes.json()) as {
        facets?: Array<{ key: string }>
        explainItBackRepresentatives?: unknown[]
        completed?: number
      }
      expect(analytics.facets?.some((f) => f.key === 'keyPointId')).toBeTruthy()
      expect((analytics.explainItBackRepresentatives ?? []).length).toBeGreaterThan(0)

      const rosterRes = await fetch(
        `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/content-tools/instances/${inst.id}/states?page=1&pageSize=50`,
        { headers: { Authorization: `Bearer ${instructorToken}` } },
      )
      expect(rosterRes.status).toBe(200)
      const roster = (await rosterRes.json()) as {
        items: Array<{ enrollmentId: string; status: string }>
      }
      const started = roster.items.find((r) => r.status === 'completed' || r.status === 'in_progress')
      expect(started).toBeTruthy()

      const resetRes = await fetch(
        `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/content-tools/state-resets`,
        {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            Authorization: `Bearer ${instructorToken}`,
          },
          body: JSON.stringify({
            scope: 'instance_enrollment',
            instanceId: inst.id,
            enrollmentId: started!.enrollmentId,
            dryRun: false,
            notify: false,
            reason: 'ct20 e2e reset',
          }),
        },
      )
      expect(resetRes.status).toBe(200)

      const after = await fetch(
        `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/content-tools/instances/${inst.id}/state`,
        { headers: { Authorization: `Bearer ${studentToken}` } },
      )
      expect(after.status).toBe(200)
      const afterBody = (await after.json()) as { state?: { attempts?: unknown[] } }
      expect(Array.isArray(afterBody.state?.attempts) ? afterBody.state!.attempts!.length : 0).toBe(0)
    })
  })

  test('review mode when aiFeedback is false', async ({ seededCourse }) => {
    const { instructorToken, studentToken, courseCode } = seededCourse

    await withCourseFeatureRestore(instructorToken, courseCode, async () => {
      await apiPatchCourseFeatures(instructorToken, courseCode, {
        contentToolsEnabled: true,
      })
      expect(
        await putSettings(instructorToken, courseCode, {
          allowedToolIds: ['explain_it_back'],
          studentResetAllowed: false,
          maxInstancesPerItem: 50,
        }),
      ).toBe(200)

      const mod = await apiCreateModule(instructorToken, courseCode, `CT20 review ${Date.now()}`)
      const contentPage = await apiCreateContentPage(
        instructorToken,
        courseCode,
        mod.id,
        'CT20 review page',
      )
      const inst = await createInstance(
        instructorToken,
        courseCode,
        contentPage.id,
        sampleConfig({ aiFeedback: false }),
      )

      const submit = await runSubmit(studentToken, courseCode, inst.id, sampleText)
      expect(submit.status).toBe(200)
      const result = submit.body.result as Record<string, unknown>
      expect(result.mode).toBe('review')
      const fb = result.feedback as { mode?: string; strength?: string }
      expect(fb.mode).toBe('review')
      expect(String(fb.strength || '')).toMatch(/instructor|review|saved/i)
    })
  })
})
