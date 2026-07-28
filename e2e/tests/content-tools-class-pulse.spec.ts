/**
 * CT.21 — Class Pulse: vote, then see how the class answered.
 *
 * Checklist:
 *   [x] Student payload has no correctOptionId / explanation / aggregate
 *   [x] Vote stores once; aggregate (or suppression) returned
 *   [x] Double-vote refused
 *   [x] Revote stores both rounds and reveals correct when configured
 *   [x] Aggregate denied before vote
 *   [x] UI: vote → suppressed/results → revote
 *   [x] Reset clears votes
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
    question: 'Which explanation best fits the data?',
    options: [
      { id: 'a', text: 'Option A' },
      { id: 'b', text: 'Option B' },
      { id: 'c', text: 'Option C' },
    ],
    correctOptionId: 'a',
    explanation: 'A matches the trend in the table.',
    allowSecondVote: true,
    revealCorrect: 'after_revote',
    minRespondents: 5,
    scopeToSection: false,
    showPercentages: true,
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
        toolId: 'class_pulse',
        hostKind: 'content_page',
        structureItemId,
        title: 'CT21 Class Pulse',
        config,
      }),
    },
  )
  if (!res.ok) throw new Error(`create instance failed (${res.status}): ${await res.text()}`)
  return (await res.json()) as { id: string }
}

function fenceMarkdown(instanceId: string): string {
  return [
    '# CT21 page',
    '',
    '```lex-tool',
    JSON.stringify({ instanceId, toolId: 'class_pulse', v: 1 }),
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
        reason: 'ct21 e2e reset',
      }),
    },
  )
  return res.status
}

test.describe('Content Tools Class Pulse (CT.21)', () => {
  test('redaction, vote gate, suppression, revote, UI, reset', async ({ page, seededCourse }) => {
    const { instructorToken, studentToken, courseCode } = seededCourse

    await withCourseFeatureRestore(instructorToken, courseCode, async () => {
      await apiPatchCourseFeatures(instructorToken, courseCode, {
        contentToolsEnabled: true,
      })
      expect(
        await putSettings(instructorToken, courseCode, {
          allowedToolIds: ['class_pulse'],
          studentResetAllowed: true,
          maxInstancesPerItem: 50,
        }),
      ).toBe(200)

      const module = await apiCreateModule(instructorToken, courseCode, 'CT21 Module')
      const contentPage = await apiCreateContentPage(
        instructorToken,
        courseCode,
        module.id,
        'CT21 Page',
      )
      const inst = await createInstance(instructorToken, courseCode, contentPage.id)
      await apiPatchContentPage(instructorToken, courseCode, contentPage.id, {
        markdown: fenceMarkdown(inst.id),
      })

      // AC-1: student config has options but no correct marker / explanation / aggregate.
      const studentInst = await getStudentInstance(studentToken, courseCode, inst.id)
      expect(studentInst.config?.correctOptionId).toBeUndefined()
      expect(studentInst.config?.explanation).toBeUndefined()
      expect(studentInst.config?.aggregate).toBeUndefined()
      const options = studentInst.config?.options as Array<Record<string, unknown>>
      expect(options?.length).toBe(3)
      expect(options?.[0]?.text).toBeTruthy()

      const denied = await runAction(
        studentToken,
        courseCode,
        inst.id,
        'aggregate',
        {},
        'ct21-agg-before',
      )
      expect(denied.status).toBe(200)
      expect((denied.body.result as { error?: string }).error).toBe('vote_required')

      const voted = await runAction(
        studentToken,
        courseCode,
        inst.id,
        'vote',
        { optionId: 'b', round: 1 },
        'ct21-vote-1',
      )
      expect(voted.status).toBe(200)
      const result = voted.body.result as {
        aggregate?: { suppressed?: boolean; reason?: string; learners?: number }
        reveal?: { correctOptionId?: string }
        error?: string
      }
      expect(result.error).toBeFalsy()
      expect(result.aggregate?.suppressed).toBe(true)
      expect(result.aggregate?.reason).toBe('small_n')
      expect(result.reveal).toBeUndefined()
      const envelope = voted.body.state as {
        status?: string
        state?: { votes?: Array<{ round: number; optionId: string }> }
      }
      expect(envelope.state?.votes?.[0]?.optionId).toBe('b')

      const again = await runAction(
        studentToken,
        courseCode,
        inst.id,
        'vote',
        { optionId: 'a', round: 1 },
        'ct21-vote-1b',
      )
      expect(again.status).toBe(200)
      expect((again.body.result as { error?: string }).error).toBe('already_voted')

      const revoted = await runAction(
        studentToken,
        courseCode,
        inst.id,
        'vote',
        { optionId: 'a', round: 2 },
        'ct21-vote-2',
      )
      expect(revoted.status).toBe(200)
      const r2 = revoted.body.result as {
        reveal?: { correctOptionId?: string; explanation?: string }
        aggregateRound2?: { learners?: number }
        error?: string
      }
      expect(r2.error).toBeFalsy()
      expect(r2.reveal?.correctOptionId).toBe('a')
      expect(r2.reveal?.explanation).toMatch(/trend/i)
      expect(r2.aggregateRound2).toBeTruthy()

      await injectToken(page, studentToken)
      await page.goto(`/courses/${courseCode}/modules/content/${contentPage.id}`)
      const tool = page.locator('[data-content-tool="class_pulse"]').first()
      await expect(tool).toBeVisible({ timeout: 20_000 })
      await expect(page.getByText(/which explanation best fits/i)).toBeVisible()
      await expect(page.getByTestId('class-pulse-aggregate')).toBeVisible({ timeout: 15_000 })
      await expect(page.getByTestId('class-pulse-reveal')).toBeVisible()

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

      const stateRes = await fetch(
        `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/content-tools/instances/${inst.id}/state`,
        { headers: { Authorization: `Bearer ${studentToken}` } },
      )
      expect(stateRes.status).toBe(200)
      const cleared = (await stateRes.json()) as {
        status: string
        state: { votes?: unknown[] }
      }
      expect(cleared.status).toBe('not_started')
      expect(cleared.state?.votes?.length ?? 0).toBe(0)

      await page.reload()
      await expect(page.locator('[data-content-tool="class_pulse"]').first()).toBeVisible({
        timeout: 20_000,
      })
      await expect(page.getByTestId('class-pulse-submit')).toBeVisible({ timeout: 10_000 })
    })
  })
})
