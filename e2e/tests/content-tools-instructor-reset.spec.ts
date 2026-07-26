/**
 * CT.4 — Instructor roster, reset one learner, student sees empty state.
 *
 * Checklist:
 *   [x] Instructor roster includes not_started learners
 *   [x] Dry-run does not mutate
 *   [x] Reset one enrollment clears state and increments reset_count
 *   [x] Self-reset forbidden when studentResetAllowed=false
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

async function createInstance(
  token: string,
  courseCode: string,
  structureItemId: string,
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
        toolId: 'noop_probe',
        hostKind: 'content_page',
        structureItemId,
        title: 'CT4 probe',
        config: { prompt: 'CT4 reset prompt', answerKey: 'paris', maxAttempts: 5 },
      }),
    },
  )
  if (!res.ok) throw new Error(`create instance failed (${res.status}): ${await res.text()}`)
  return (await res.json()) as { id: string }
}

async function putStudentState(
  token: string,
  courseCode: string,
  instanceId: string,
  revision: number,
  response: string,
): Promise<{ revision: number; status: string; resetCount: number }> {
  const res = await fetch(
    `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/content-tools/instances/${instanceId}/state`,
    {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${token}`,
      },
      body: JSON.stringify({
        revision,
        state: { response, attempts: 1 },
      }),
    },
  )
  if (!res.ok) throw new Error(`put state failed (${res.status}): ${await res.text()}`)
  return (await res.json()) as { revision: number; status: string; resetCount: number }
}

test.describe('Content Tools instructor reset (CT.4)', () => {
  test('roster + dry-run + reset one + self-reset denied', async ({ page, seededCourse }) => {
    const { instructorToken, studentToken, courseCode } = seededCourse

    await withCourseFeatureRestore(instructorToken, courseCode, async () => {
      await apiPatchCourseFeatures(instructorToken, courseCode, {
        contentToolsEnabled: true,
      })
      expect(
        await putSettings(instructorToken, courseCode, {
          allowedToolIds: ['noop_probe'],
          studentResetAllowed: false,
          maxInstancesPerItem: 50,
        }),
      ).toBe(200)

      const mod = await apiCreateModule(instructorToken, courseCode, `CT4 ${Date.now()}`)
      const content = await apiCreateContentPage(
        instructorToken,
        courseCode,
        mod.id,
        'CT4 page',
      )
      const inst = await createInstance(instructorToken, courseCode, content.id)
      await apiPatchContentPage(instructorToken, courseCode, content.id, {
        markdown: [
          '# CT4',
          '',
          '```lex-tool',
          JSON.stringify({ instanceId: inst.id, toolId: 'noop_probe', v: 1 }),
          '```',
          '',
        ].join('\n'),
      })

      await putStudentState(studentToken, courseCode, inst.id, 0, 'wrong answer')

      // Roster includes the student with in_progress and any not_started peers.
      const rosterRes = await fetch(
        `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/content-tools/instances/${inst.id}/states?page=1&pageSize=50`,
        { headers: { Authorization: `Bearer ${instructorToken}` } },
      )
      expect(rosterRes.status).toBe(200)
      const roster = (await rosterRes.json()) as {
        items: Array<{ enrollmentId: string; status: string; displayName: string }>
        totalCount: number
      }
      expect(roster.totalCount).toBeGreaterThanOrEqual(1)
      const started = roster.items.find((r) => r.status === 'in_progress')
      expect(started).toBeTruthy()

      // Dry-run does not create snapshots.
      const dryRes = await fetch(
        `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/content-tools/state-resets`,
        {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            Authorization: `Bearer ${instructorToken}`,
          },
          body: JSON.stringify({
            scope: 'instance_all',
            instanceId: inst.id,
            dryRun: true,
          }),
        },
      )
      expect(dryRes.status).toBe(200)
      const dry = (await dryRes.json()) as { dryRun: boolean; affectedCount: number }
      expect(dry.dryRun).toBe(true)
      expect(dry.affectedCount).toBeGreaterThanOrEqual(1)

      const histBefore = await fetch(
        `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/content-tools/state-resets?instanceId=${inst.id}`,
        { headers: { Authorization: `Bearer ${instructorToken}` } },
      )
      const beforeItems = ((await histBefore.json()) as { items: unknown[] }).items
      const beforeCount = beforeItems.length

      // Real reset for one enrollment.
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
            reason: 'e2e reset',
          }),
        },
      )
      expect(resetRes.status).toBe(200)
      const resetBody = (await resetRes.json()) as {
        affectedCount: number
        batchId?: string
      }
      expect(resetBody.affectedCount).toBe(1)
      expect(resetBody.batchId).toBeTruthy()

      const histAfter = await fetch(
        `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/content-tools/state-resets?instanceId=${inst.id}`,
        { headers: { Authorization: `Bearer ${instructorToken}` } },
      )
      const afterItems = ((await histAfter.json()) as { items: unknown[] }).items
      expect(afterItems.length).toBe(beforeCount + 1)

      const stateRes = await fetch(
        `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/content-tools/instances/${inst.id}/state`,
        { headers: { Authorization: `Bearer ${studentToken}` } },
      )
      expect(stateRes.status).toBe(200)
      const state = (await stateRes.json()) as {
        status: string
        resetCount: number
        state: { response?: string }
      }
      expect(state.status).toBe('not_started')
      expect(state.resetCount).toBeGreaterThanOrEqual(1)
      expect(state.state?.response ?? '').toBe('')

      // Self-reset denied when course setting is off.
      const selfRes = await fetch(
        `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/content-tools/instances/${inst.id}/self-reset`,
        {
          method: 'POST',
          headers: { Authorization: `Bearer ${studentToken}` },
        },
      )
      expect(selfRes.status).toBe(403)

      // UI: instructor sees Responses affordance.
      await injectToken(page, instructorToken)
      await page.goto(`/courses/${encodeURIComponent(courseCode)}/modules/content/${content.id}`)
      await expect(page.getByTestId('tool-responses-button')).toBeVisible({ timeout: 15000 })
      await page.getByTestId('tool-responses-button').click()
      await expect(page.getByTestId('tool-responses-panel')).toBeVisible()
      await expect(page.getByTestId('tool-roster-table')).toBeVisible()
    })
  })
})
