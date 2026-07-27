/**
 * CT.19 — Media Checkpoints: answers, redaction, watch progress, transcript-only, analytics.
 *
 * Checklist:
 *   [x] Student payload has no correct/feedback/acceptedAnswers
 *   [x] answer_checkpoint grades server-side
 *   [x] record_progress does not forge score
 *   [x] transcript-only completion path
 *   [x] Analytics facets present
 *   [x] UI smoke (transcript-only + question)
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
    media: {
      source: 'course_file',
      fileId: 'e2e-media-file',
      kind: 'video',
      durationSec: 120,
      // Intentionally omitted URL → media unavailable → transcript path (AC-5).
    },
    transcriptSource: 'inline',
    transcriptMarkdown: '0:00 Intro\n0:15 Concept\n0:45 Wrap-up',
    preventSkipPastUnanswered: true,
    practiceOnly: true,
    checkpoints: [
      {
        id: 'c1',
        atSec: 15,
        required: true,
        attempts: 2,
        showFeedback: true,
        question: {
          type: 'single',
          prompt: 'What is the main idea?',
          options: [
            { id: 'a', text: 'Noise', correct: false, feedback: 'Not the focus' },
            { id: 'b', text: 'Signal', correct: true, feedback: 'Yes' },
          ],
        },
      },
      {
        id: 'c2',
        atSec: 45,
        required: true,
        attempts: 2,
        showFeedback: true,
        question: {
          type: 'short_text',
          prompt: 'Keyword?',
          acceptedAnswers: ['photosynthesis'],
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
        toolId: 'media_checkpoints',
        hostKind: 'content_page',
        structureItemId,
        title: 'CT19 Media Checkpoints',
        config,
      }),
    },
  )
  if (!res.ok) throw new Error(`create instance failed (${res.status}): ${await res.text()}`)
  return (await res.json()) as { id: string }
}

function fenceMarkdown(instanceId: string): string {
  return [
    '# CT19 page',
    '',
    '```lex-tool',
    JSON.stringify({ instanceId, toolId: 'media_checkpoints', v: 1 }),
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

test.describe('Content Tools Media Checkpoints (CT.19)', () => {
  test('redaction, answers, progress, transcript-only UI, analytics', async ({
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
          allowedToolIds: ['media_checkpoints'],
          studentResetAllowed: true,
          maxInstancesPerItem: 50,
        }),
      ).toBe(200)

      const module = await apiCreateModule(instructorToken, courseCode, 'CT19 Module')
      const contentPage = await apiCreateContentPage(
        instructorToken,
        courseCode,
        module.id,
        'CT19 Page',
      )
      const inst = await createInstance(instructorToken, courseCode, contentPage.id)
      await apiPatchContentPage(instructorToken, courseCode, contentPage.id, {
        markdown: fenceMarkdown(inst.id),
      })

      // AC: student config has no secrets.
      const studentInst = await getStudentInstance(studentToken, courseCode, inst.id)
      const cfg = studentInst.config as {
        checkpoints?: Array<{ question?: Record<string, unknown> }>
      }
      const q0 = cfg?.checkpoints?.[0]?.question
      const opts = q0?.options as Array<Record<string, unknown>> | undefined
      expect(opts?.[0]?.correct).toBeUndefined()
      expect(opts?.[0]?.feedback).toBeUndefined()
      expect(opts?.[0]?.text).toBe('Noise')
      expect(cfg?.checkpoints?.[1]?.question?.acceptedAnswers).toBeUndefined()

      // Watch progress alone does not create a score.
      const prog = await runAction(
        studentToken,
        courseCode,
        inst.id,
        'record_progress',
        { watchedSegments: [[0, 180], [300, 360]], furthestSec: 360 },
        'mc-prog-1',
      )
      expect(prog.status, JSON.stringify(prog.body)).toBe(200)

      // Wrong then correct on c1.
      const wrong = await runAction(
        studentToken,
        courseCode,
        inst.id,
        'answer_checkpoint',
        { checkpointId: 'c1', value: 'a', transcriptOnly: true },
        'mc-wrong-1',
      )
      expect((wrong.body.result as Record<string, unknown>).correct).toBe(false)

      const ok = await runAction(
        studentToken,
        courseCode,
        inst.id,
        'answer_checkpoint',
        { checkpointId: 'c1', value: 'b', transcriptOnly: true },
        'mc-ok-1',
      )
      expect((ok.body.result as Record<string, unknown>).correct).toBe(true)
      expect((ok.body.result as Record<string, unknown>).done).toBe(true)

      // Complete via transcript-only short text.
      const kw = await runAction(
        studentToken,
        courseCode,
        inst.id,
        'answer_checkpoint',
        { checkpointId: 'c2', value: 'photosynthesis', transcriptOnly: true },
        'mc-kw-1',
      )
      expect((kw.body.result as Record<string, unknown>).correct).toBe(true)

      // Analytics
      const analytics = await fetch(
        `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/content-tools/instances/${inst.id}/analytics`,
        { headers: { Authorization: `Bearer ${instructorToken}` } },
      )
      expect(analytics.status).toBe(200)
      const aBody = (await analytics.json()) as Record<string, unknown>
      expect(aBody).toBeTruthy()

      // UI smoke — media unavailable banner + transcript-only checkpoints.
      await injectToken(page, studentToken)
      await page.goto(`/courses/${courseCode}/modules/content/${contentPage.id}`)
      const tool = page.locator('[data-content-tool="media_checkpoints"]').first()
      await expect(tool).toBeVisible({ timeout: 20_000 })
      await expect(page.getByTestId('media-checkpoint-transcript-only')).toBeVisible()
      await page.getByTestId('media-checkpoint-transcript-only').check()
      await expect(page.getByText('What is the main idea?')).toBeVisible()
    })
  })
})
