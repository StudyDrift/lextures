/**
 * CT.11 — Inline Questions: formative checks with server-side scoring.
 *
 * Checklist:
 *   [x] Student payload has no answer key
 *   [x] Wrong → feedback → retry → correct; reviewed state persists
 *   [x] Attempt exhaustion refuses third submit
 *   [x] Sequential mode locks question 2
 *   [x] Instructor analytics expose option distribution
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
    label: 'CT11 check',
    attempts: 2,
    revealCorrectAfter: 'last_attempt',
    sequential: true,
    shuffleOptions: false,
    scorePolicy: 'best',
    questions: [
      {
        id: 'q1',
        type: 'single',
        prompt: 'Capital of France?',
        options: [
          { id: 'a', text: 'London', correct: false, feedback: 'That is in England.' },
          { id: 'b', text: 'Paris', correct: true, feedback: 'Yes — Paris.' },
          { id: 'c', text: 'Berlin', correct: false, feedback: 'That is in Germany.' },
        ],
        explanation: 'Paris is the capital of France.',
        points: 1,
      },
      {
        id: 'q2',
        type: 'short_text',
        prompt: 'Process plants use to make food?',
        acceptedAnswers: ['photosynthesis'],
        points: 1,
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
        toolId: 'inline_questions',
        hostKind: 'content_page',
        structureItemId,
        title: 'CT11 Inline Questions',
        config,
      }),
    },
  )
  if (!res.ok) throw new Error(`create instance failed (${res.status}): ${await res.text()}`)
  return (await res.json()) as { id: string }
}

function fenceMarkdown(instanceId: string): string {
  return [
    '# CT11 page',
    '',
    '```lex-tool',
    JSON.stringify({ instanceId, toolId: 'inline_questions', v: 1 }),
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
  const body = (await res.json()) as { instances?: Array<{ id: string; config?: Record<string, unknown> }> }
  const inst = (body.instances ?? []).find((i) => i.id === instanceId)
  if (!inst) throw new Error('instance not found in student list')
  return inst
}

async function submit(
  token: string,
  courseCode: string,
  instanceId: string,
  questionId: string,
  value: unknown,
  idempotencyKey: string,
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
        input: { questionId, value },
        idempotencyKey,
      }),
    },
  )
  const body = (await res.json()) as Record<string, unknown>
  return { status: res.status, body }
}

test.describe('Content Tools Inline Questions (CT.11)', () => {
  test('redaction, retry, sequential, exhaustion, analytics', async ({ page, seededCourse }) => {
    const { instructorToken, studentToken, courseCode } = seededCourse

    await withCourseFeatureRestore(instructorToken, courseCode, async () => {
      await apiPatchCourseFeatures(instructorToken, courseCode, {
        contentToolsEnabled: true,
      })
      expect(
        await putSettings(instructorToken, courseCode, {
          allowedToolIds: ['inline_questions'],
          studentResetAllowed: true,
          maxInstancesPerItem: 50,
        }),
      ).toBe(200)

      const module = await apiCreateModule(instructorToken, courseCode, 'CT11 Module')
      const contentPage = await apiCreateContentPage(
        instructorToken,
        courseCode,
        module.id,
        'CT11 Page',
      )
      const inst = await createInstance(instructorToken, courseCode, contentPage.id)
      await apiPatchContentPage(instructorToken, courseCode, contentPage.id, {
        markdown: fenceMarkdown(inst.id),
      })

      // AC-1: student config has no answer key / feedback.
      const studentInst = await getStudentInstance(studentToken, courseCode, inst.id)
      const cfg = studentInst.config as {
        questions?: Array<Record<string, unknown>>
      }
      expect(cfg?.questions?.[0]?.explanation).toBeUndefined()
      const opts = cfg?.questions?.[0]?.options as Array<Record<string, unknown>>
      expect(opts?.[0]?.correct).toBeUndefined()
      expect(opts?.[0]?.feedback).toBeUndefined()
      expect(opts?.[0]?.text).toBeTruthy()
      expect(cfg?.questions?.[1]?.acceptedAnswers).toBeUndefined()

      // Sequential: q2 locked until q1 answered.
      const locked = await submit(
        studentToken,
        courseCode,
        inst.id,
        'q2',
        'photosynthesis',
        'ct11-seq-1',
      )
      expect(locked.status).toBe(200)
      expect((locked.body.result as { error?: string }).error).toBe('sequential_locked')

      // Wrong then correct on q1.
      const wrong = await submit(studentToken, courseCode, inst.id, 'q1', 'a', 'ct11-q1-1')
      expect(wrong.status).toBe(200)
      const wrongResult = wrong.body.result as {
        correct?: boolean
        feedback?: string
        correctAnswer?: unknown
      }
      expect(wrongResult.correct).toBe(false)
      expect(wrongResult.feedback).toMatch(/England/i)
      expect(wrongResult.correctAnswer).toBeUndefined()

      const right = await submit(studentToken, courseCode, inst.id, 'q1', 'b', 'ct11-q1-2')
      expect(right.status).toBe(200)
      const rightResult = right.body.result as {
        correct?: boolean
        feedback?: string
        explanation?: string
      }
      expect(rightResult.correct).toBe(true)
      expect(rightResult.feedback).toMatch(/Paris/i)
      expect(rightResult.explanation).toMatch(/capital/i)

      // Exhaustion on q2 with attempts=2.
      await submit(studentToken, courseCode, inst.id, 'q2', 'respiration', 'ct11-q2-1')
      await submit(studentToken, courseCode, inst.id, 'q2', 'digestion', 'ct11-q2-2')
      const third = await submit(studentToken, courseCode, inst.id, 'q2', 'photosynthesis', 'ct11-q2-3')
      expect(third.status).toBe(200)
      expect((third.body.result as { error?: string }).error).toBe('max_attempts')

      // Idempotent double submit does not burn an extra attempt on a fresh instance.
      const inst2 = await createInstance(
        instructorToken,
        courseCode,
        contentPage.id,
        sampleConfig({ sequential: false, attempts: 2, questions: sampleConfig().questions.slice(0, 1) }),
      )
      const first = await submit(studentToken, courseCode, inst2.id, 'q1', 'a', 'ct11-idem-1')
      const again = await submit(studentToken, courseCode, inst2.id, 'q1', 'a', 'ct11-idem-1')
      expect(again.status).toBe(200)
      const firstState = first.body.state as { revision?: number }
      const againState = again.body.state as { revision?: number }
      expect(againState.revision).toBe(firstState.revision)

      // UI: host mounts and shows prompt.
      await injectToken(page, studentToken)
      await page.goto(`/courses/${courseCode}/modules/content/${contentPage.id}`)
      const tool = page.locator('[data-content-tool="inline_questions"]').first()
      await expect(tool).toBeVisible({ timeout: 20_000 })
      await expect(page.getByText('Capital of France?')).toBeVisible()

      // Instructor analytics facets include option distribution.
      const analyticsRes = await fetch(
        `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/content-tools/instances/${inst.id}/analytics`,
        { headers: { Authorization: `Bearer ${instructorToken}` } },
      )
      expect(analyticsRes.status).toBe(200)
      const analytics = (await analyticsRes.json()) as {
        facets?: Array<{ key: string; values: Array<{ value: string; count: number }> }>
      }
      const optionFacet = analytics.facets?.find((f) => f.key === 'optionId')
      expect(optionFacet).toBeTruthy()
      expect((optionFacet?.values?.length ?? 0) > 0).toBe(true)
    })
  })
})
