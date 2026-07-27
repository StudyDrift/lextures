/**
 * CT.12 — Predict & Reveal: commit a prediction + confidence, then see the answer.
 *
 * Checklist:
 *   [x] Student payload has no reveal / correct flags
 *   [x] Commit returns reveal; state stores prediction, confidence, committedAt
 *   [x] Post-commit mutation refused; PUT refused
 *   [x] Confidence required soft-error
 *   [x] Reset restores the gate
 *   [x] Peer results suppressed at n<5
 *   [x] UI: predict → commit → reveal → reflect → reload
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
    question: 'What happens when we heat the sealed balloon?',
    mode: 'choice',
    confidenceScale: 'three',
    confidenceRequired: true,
    showPeerResults: true,
    reflectionPrompt: 'What surprised you?',
    outcomes: [
      { id: 'expand', text: 'It expands', correct: true },
      { id: 'shrink', text: 'It shrinks', correct: false },
      { id: 'same', text: 'Nothing changes', correct: false },
    ],
    reveal: {
      markdown: 'The balloon expands as air molecules move faster.',
    },
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
        toolId: 'predict_reveal',
        hostKind: 'content_page',
        structureItemId,
        title: 'CT12 Predict & Reveal',
        config,
      }),
    },
  )
  if (!res.ok) throw new Error(`create instance failed (${res.status}): ${await res.text()}`)
  return (await res.json()) as { id: string }
}

function fenceMarkdown(instanceId: string): string {
  return [
    '# CT12 page',
    '',
    '```lex-tool',
    JSON.stringify({ instanceId, toolId: 'predict_reveal', v: 1 }),
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

async function putState(
  token: string,
  courseCode: string,
  instanceId: string,
  state: Record<string, unknown>,
  revision = 0,
): Promise<number> {
  const res = await fetch(
    `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/content-tools/instances/${instanceId}/state`,
    {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${token}`,
      },
      body: JSON.stringify({ state, revision }),
    },
  )
  return res.status
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
        reason: 'ct12 e2e reset',
      }),
    },
  )
  return res.status
}

test.describe('Content Tools Predict & Reveal (CT.12)', () => {
  test('redaction, commit gate, irreversibility, peers, UI, reset', async ({
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
          allowedToolIds: ['predict_reveal'],
          studentResetAllowed: true,
          maxInstancesPerItem: 50,
        }),
      ).toBe(200)

      const module = await apiCreateModule(instructorToken, courseCode, 'CT12 Module')
      const contentPage = await apiCreateContentPage(
        instructorToken,
        courseCode,
        module.id,
        'CT12 Page',
      )
      const inst = await createInstance(instructorToken, courseCode, contentPage.id)
      await apiPatchContentPage(instructorToken, courseCode, contentPage.id, {
        markdown: fenceMarkdown(inst.id),
      })

      // AC-1: student config has no reveal / correct.
      const studentInst = await getStudentInstance(studentToken, courseCode, inst.id)
      expect(studentInst.config?.reveal).toBeUndefined()
      const outcomes = studentInst.config?.outcomes as Array<Record<string, unknown>>
      expect(outcomes?.[0]?.correct).toBeUndefined()
      expect(outcomes?.[0]?.text).toBeTruthy()

      // AC-4: commit without confidence.
      const noConf = await runAction(
        studentToken,
        courseCode,
        inst.id,
        'commit',
        { prediction: { outcomeId: 'expand' } },
        'ct12-noconf',
      )
      expect(noConf.status).toBe(200)
      expect((noConf.body.result as { error?: string }).error).toBe('confidence_required')

      // AC-2: commit with prediction + confidence.
      const committed = await runAction(
        studentToken,
        courseCode,
        inst.id,
        'commit',
        { prediction: { outcomeId: 'shrink' }, confidence: 3 },
        'ct12-commit-1',
      )
      expect(committed.status).toBe(200)
      const result = committed.body.result as {
        reveal?: { markdown?: string }
        peerResults?: { suppressed?: boolean; reason?: string; learners?: number }
      }
      expect(result.reveal?.markdown).toMatch(/expands/i)
      expect(result.peerResults?.suppressed).toBe(true)
      expect(result.peerResults?.reason).toBe('small_n')
      const envelope = committed.body.state as {
        status?: string
        state?: { committedAt?: string; prediction?: { outcomeId?: string }; confidence?: number }
      }
      expect(envelope.state?.committedAt).toBeTruthy()
      expect(envelope.state?.prediction?.outcomeId).toBe('shrink')
      expect(envelope.status).toBe('completed')

      // AC-3: refuse change after commit.
      const again = await runAction(
        studentToken,
        courseCode,
        inst.id,
        'commit',
        { prediction: { outcomeId: 'expand' }, confidence: 1 },
        'ct12-commit-2',
      )
      expect(again.status).toBe(200)
      expect((again.body.result as { error?: string }).error).toBe('already_committed')

      const putStatus = await putState(
        studentToken,
        courseCode,
        inst.id,
        { v: 1, prediction: { outcomeId: 'expand' } },
        (envelope as { revision?: number }).revision ?? 1,
      )
      expect(putStatus).toBe(409)

      // Reflect.
      const reflected = await runAction(
        studentToken,
        courseCode,
        inst.id,
        'reflect',
        { text: 'I thought heat made it shrink.' },
        'ct12-reflect-1',
      )
      expect(reflected.status).toBe(200)

      // UI path.
      await injectToken(page, studentToken)
      await page.goto(`/courses/${courseCode}/modules/content/${contentPage.id}`)
      const tool = page.locator('[data-content-tool="predict_reveal"]').first()
      await expect(tool).toBeVisible({ timeout: 20_000 })
      await expect(page.getByText(/heat the sealed balloon/i)).toBeVisible()
      await expect(page.getByTestId('predict-reveal-side-by-side')).toBeVisible({ timeout: 15_000 })
      await expect(page.getByText(/balloon expands/i)).toBeVisible()

      // Reload persists revealed view.
      await page.reload()
      await expect(page.locator('[data-content-tool="predict_reveal"]').first()).toBeVisible({
        timeout: 20_000,
      })
      await expect(page.getByTestId('predict-reveal-side-by-side')).toBeVisible({ timeout: 15_000 })

      // AC-7: reset restores gate.
      const rosterRes = await fetch(
        `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/content-tools/instances/${inst.id}/states?page=1&pageSize=50`,
        { headers: { Authorization: `Bearer ${instructorToken}` } },
      )
      expect(rosterRes.status).toBe(200)
      const roster = (await rosterRes.json()) as {
        items: Array<{ enrollmentId: string; status: string }>
      }
      const started = roster.items.find((r) => r.status === 'completed' || r.status === 'submitted')
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
        state: { committedAt?: string }
      }
      expect(cleared.status).toBe('not_started')
      expect(cleared.state?.committedAt).toBeFalsy()

      const afterReset = await getStudentInstance(studentToken, courseCode, inst.id)
      expect(afterReset.config?.reveal).toBeUndefined()

      await page.reload()
      await expect(page.locator('[data-content-tool="predict_reveal"]').first()).toBeVisible({
        timeout: 20_000,
      })
      await expect(page.getByTestId('predict-reveal-commit')).toBeVisible({ timeout: 10_000 })
    })
  })
})
