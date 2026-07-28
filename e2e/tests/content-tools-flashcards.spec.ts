/**
 * CT.23 — Flashcards & Spaced Recall: inline deck feeding shipped SRS.
 *
 * Checklist:
 *   [x] Start session queues all new cards
 *   [x] Rate each card; state records ratings; first pass completes
 *   [x] SRS-off path still works and reports srsEnabled=false
 *   [x] UI: start → reveal → rate
 *   [x] Reset clears tool state
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
    title: 'Spanish basics',
    cards: [
      { id: 'c1', front: 'hola', back: 'hello' },
      { id: 'c2', front: 'adiós', back: 'goodbye' },
      { id: 'c3', front: 'gracias', back: 'thank you' },
      { id: 'c4', front: 'por favor', back: 'please' },
      { id: 'c5', front: 'sí', back: 'yes' },
      { id: 'c6', front: 'no', back: 'no' },
    ],
    reversePractice: false,
    sessionCap: 20,
    shuffle: false,
    requireFirstPass: true,
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
        toolId: 'flashcards',
        hostKind: 'content_page',
        structureItemId,
        title: 'CT23 Flashcards',
        config,
      }),
    },
  )
  if (!res.ok) throw new Error(`create instance failed (${res.status}): ${await res.text()}`)
  return (await res.json()) as { id: string }
}

function fenceMarkdown(instanceId: string): string {
  return [
    '# CT23 page',
    '',
    '```lex-tool',
    JSON.stringify({ instanceId, toolId: 'flashcards', v: 1 }),
    '```',
    '',
  ].join('\n')
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
  schedulingHandling: 'keep' | 'clear' = 'keep',
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
        reason: 'ct23 e2e reset',
        schedulingHandling,
      }),
    },
  )
  return res.status
}

test.describe('Content Tools Flashcards (CT.23)', () => {
  test('session ratings, SRS-off path, UI, reset', async ({ page, seededCourse }) => {
    const { instructorToken, studentToken, courseCode } = seededCourse

    await withCourseFeatureRestore(instructorToken, courseCode, async () => {
      await apiPatchCourseFeatures(instructorToken, courseCode, {
        contentToolsEnabled: true,
        srsEnabled: false,
      })
      expect(
        await putSettings(instructorToken, courseCode, {
          allowedToolIds: ['flashcards'],
          studentResetAllowed: true,
          maxInstancesPerItem: 50,
        }),
      ).toBe(200)

      const module = await apiCreateModule(instructorToken, courseCode, 'CT23 Module')
      const contentPage = await apiCreateContentPage(
        instructorToken,
        courseCode,
        module.id,
        'CT23 Page',
      )
      const inst = await createInstance(instructorToken, courseCode, contentPage.id)
      await apiPatchContentPage(instructorToken, courseCode, contentPage.id, {
        markdown: fenceMarkdown(inst.id),
      })

      const started = await runAction(
        studentToken,
        courseCode,
        inst.id,
        'start_session',
        {},
        'ct23-start-1',
      )
      expect(started.status).toBe(200)
      const startResult = started.body.result as {
        caughtUp?: boolean
        srsEnabled?: boolean
        queue?: Array<{ cardId: string; side: string }>
        state?: { activeSession?: { queue: Array<{ cardId: string; side: string }> } }
      }
      expect(startResult.caughtUp).toBeFalsy()
      expect(startResult.srsEnabled).toBe(false)
      const queue =
        startResult.queue ??
        startResult.state?.activeSession?.queue ??
        []
      expect(queue.length).toBe(6)

      for (let i = 0; i < queue.length; i++) {
        const item = queue[i]
        const rated = await runAction(
          studentToken,
          courseCode,
          inst.id,
          'rate',
          {
            cardId: item.cardId,
            side: item.side,
            rating: i % 2 === 0 ? 'good' : 'hard',
            idempotencyKey: `ct23-rate-${i}`,
          },
          `ct23-rate-${i}`,
        )
        expect(rated.status).toBe(200)
        const rr = rated.body.result as {
          error?: string
          sessionComplete?: boolean
          srsEnabled?: boolean
          state?: {
            cards?: Record<string, { seen: number }>
            firstPassCompletedAt?: string
          }
        }
        expect(rr.error).toBeUndefined()
        expect(rr.srsEnabled).toBe(false)
        if (i === queue.length - 1) {
          expect(rr.sessionComplete).toBe(true)
          expect(Object.keys(rr.state?.cards ?? {}).length).toBe(6)
          expect(rr.state?.firstPassCompletedAt).toBeTruthy()
        }
      }

      await injectToken(page, studentToken)
      await page.goto(`/courses/${courseCode}/modules/content/${contentPage.id}`)
      const tool = page.locator('[data-content-tool="flashcards"]').first()
      await expect(tool).toBeVisible({ timeout: 20_000 })
      await expect(page.getByTestId('flashcards-srs-off')).toBeVisible()
      await page.getByTestId('flashcards-start').click()
      await expect(page.getByTestId('flashcards-card')).toBeVisible({ timeout: 15_000 })
      await page.getByTestId('flashcards-reveal').click()
      await expect(page.getByTestId('flashcards-answer')).toBeVisible()
      await page.getByTestId('flashcards-rate-good').click()

      const rosterRes = await fetch(
        `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/content-tools/instances/${inst.id}/states?page=1&pageSize=50`,
        { headers: { Authorization: `Bearer ${instructorToken}` } },
      )
      expect(rosterRes.status).toBe(200)
      const roster = (await rosterRes.json()) as {
        items: Array<{ enrollmentId: string; status: string }>
      }
      const row = roster.items.find((r) => r.status === 'completed' || r.status === 'in_progress')
      expect(row?.enrollmentId).toBeTruthy()
      expect(
        await resetEnrollment(instructorToken, courseCode, inst.id, row!.enrollmentId, 'keep'),
      ).toBe(200)

      const stateRes = await fetch(
        `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/content-tools/instances/${inst.id}/state`,
        { headers: { Authorization: `Bearer ${studentToken}` } },
      )
      expect(stateRes.status).toBe(200)
      const cleared = (await stateRes.json()) as {
        status: string
        state: { cards?: Record<string, unknown>; firstPassCompletedAt?: string }
      }
      expect(cleared.status).toBe('not_started')
      expect(Object.keys(cleared.state?.cards ?? {}).length).toBe(0)
      expect(cleared.state?.firstPassCompletedAt ?? '').toBe('')
    })
  })
})
