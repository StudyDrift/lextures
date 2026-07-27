/**
 * CT.3 — Content Tools student runtime: host render, autosave, restore, grade action.
 *
 * Checklist:
 *   [x] Student sees noop_probe host (not a raw lex-tool code fence)
 *   [x] Typing autosaves; reload restores response
 *   [x] Grade action sets score server-side
 *   [x] Stale revision returns 409 with current envelope
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
): Promise<{ id: string; toolId: string }> {
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
        title: 'CT3 probe',
        config: { prompt: 'CT3 runtime prompt', answerKey: 'paris', maxAttempts: 5 },
      }),
    },
  )
  if (!res.ok) {
    throw new Error(`create instance failed (${res.status}): ${await res.text()}`)
  }
  return (await res.json()) as { id: string; toolId: string }
}

function fenceMarkdown(instanceId: string): string {
  return [
    '# CT3 page',
    '',
    'Intro paragraph.',
    '',
    '```lex-tool',
    JSON.stringify({ instanceId, toolId: 'noop_probe', v: 1 }),
    '```',
    '',
  ].join('\n')
}

test.describe('Content Tools runtime (CT.3)', () => {
  test('student interact → autosave → reload restores; grade + 409 conflict', async ({
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
          allowedToolIds: ['noop_probe'],
          studentResetAllowed: false,
          maxInstancesPerItem: 50,
        }),
      ).toBe(200)

      const module = await apiCreateModule(instructorToken, courseCode, 'CT3 Runtime Module')
      const contentPage = await apiCreateContentPage(
        instructorToken,
        courseCode,
        module.id,
        'CT3 Runtime Page',
      )
      const inst = await createInstance(instructorToken, courseCode, contentPage.id)
      await apiPatchContentPage(instructorToken, courseCode, contentPage.id, {
        markdown: fenceMarkdown(inst.id),
      })

      // Student opens reader — host mounts (not raw fence JSON).
      await injectToken(page, studentToken)
      await page.goto(`/courses/${courseCode}/modules/content/${contentPage.id}`)
      const tool = page.locator('[data-content-tool="noop_probe"]').first()
      await expect(tool).toBeVisible({ timeout: 20_000 })
      await expect(page.getByText('CT3 runtime prompt')).toBeVisible()
      await expect(page.locator('pre code.language-lex-tool')).toHaveCount(0)

      const answer = page.getByLabel(/your answer/i).first()
      await expect(answer).toBeVisible({ timeout: 20_000 })
      await expect(answer).toBeEditable()
      const statePut = page.waitForResponse(
        (res) =>
          res.request().method() === 'PUT' &&
          res.url().includes(`/content-tools/instances/${inst.id}/state`) &&
          res.ok(),
        { timeout: 45_000 },
      )
      // Prefer sequential key events so controlled React inputs fire onChange under load.
      await answer.click()
      await answer.fill('')
      await answer.pressSequentially('paris', { delay: 20 })
      await answer.blur()
      // Autosave debounce is 1.5s; wait for state PUT + sync chip (not a flaky toast).
      await statePut
      await expect(
        page.locator('[data-content-tool-frame] [data-sync-status="saved"]').first(),
      ).toBeVisible({ timeout: 12_000 })

      await page.reload()
      await expect(page.locator('[data-content-tool="noop_probe"]').first()).toBeVisible({
        timeout: 20_000,
      })
      await expect(page.getByLabel(/your answer/i).first()).toHaveValue('paris', {
        timeout: 10_000,
      })

      // Grade action (API) — server sets score; client cannot forge.
      const gradeRes = await fetch(
        `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/content-tools/instances/${encodeURIComponent(inst.id)}/actions/grade`,
        {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            Authorization: `Bearer ${studentToken}`,
          },
          body: JSON.stringify({ input: { response: 'paris' }, idempotencyKey: 'ct3-e2e-grade-1' }),
        },
      )
      expect(gradeRes.status).toBe(200)
      const gradeJson = (await gradeRes.json()) as {
        result: { correct?: boolean }
        state: { score?: { raw: number; max: number } | null; revision: number; state?: { response?: string } }
      }
      expect(gradeJson.result.correct).toBe(true)
      expect(gradeJson.state.score?.raw).toBe(1)
      expect(gradeJson.state.score?.max).toBe(1)

      // Idempotent retry
      const gradeRes2 = await fetch(
        `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/content-tools/instances/${encodeURIComponent(inst.id)}/actions/grade`,
        {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            Authorization: `Bearer ${studentToken}`,
          },
          body: JSON.stringify({ input: { response: 'paris' }, idempotencyKey: 'ct3-e2e-grade-1' }),
        },
      )
      expect(gradeRes2.status).toBe(200)
      const gradeJson2 = (await gradeRes2.json()) as { state: { revision: number } }
      expect(gradeJson2.state.revision).toBe(gradeJson.state.revision)

      // Stale revision → 409 with current envelope
      const conflictRes = await fetch(
        `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/content-tools/instances/${encodeURIComponent(inst.id)}/state`,
        {
          method: 'PUT',
          headers: {
            'Content-Type': 'application/json',
            Authorization: `Bearer ${studentToken}`,
          },
          body: JSON.stringify({ revision: 0, state: { response: 'stale' } }),
        },
      )
      expect(conflictRes.status).toBe(409)
      const conflictJson = (await conflictRes.json()) as {
        error: string
        current: { revision: number; state?: { response?: string } }
      }
      expect(conflictJson.error).toBe('revision_conflict')
      expect(conflictJson.current.revision).toBeGreaterThan(0)
      expect(conflictJson.current.state?.response).not.toBe('stale')

      // Schema invalid → 422, state unchanged
      const badRes = await fetch(
        `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/content-tools/instances/${encodeURIComponent(inst.id)}/state`,
        {
          method: 'PUT',
          headers: {
            'Content-Type': 'application/json',
            Authorization: `Bearer ${studentToken}`,
          },
          body: JSON.stringify({
            revision: conflictJson.current.revision,
            state: { attempts: 'nope' },
          }),
        },
      )
      expect(badRes.status).toBe(422)
      const badJson = (await badRes.json()) as { error: string }
      expect(badJson.error).toBe('schema_invalid')
    })
  })
})
