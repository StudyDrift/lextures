/**
 * CT.10 — Ask Questions: grounded AI Q&A about this activity.
 *
 * Checklist:
 *   [x] Student asks → cited dry-run answer persists in state
 *   [x] Reload restores conversation
 *   [x] Instructor reset clears transcript for one student
 *   [x] Rate-limit message when daily cap exceeded
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

async function createAskInstance(
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
        toolId: 'ask_questions',
        hostKind: 'content_page',
        structureItemId,
        title: 'CT10 Ask Questions',
        config: {
          intro: 'Ask about stoichiometry on this page.',
          stance: 'explain',
          maxQuestionsPerDay: 2,
          maxTurns: 40,
          showCitations: true,
        },
      }),
    },
  )
  if (!res.ok) throw new Error(`create instance failed (${res.status}): ${await res.text()}`)
  return (await res.json()) as { id: string }
}

function fenceMarkdown(instanceId: string): string {
  return [
    '# CT10 page',
    '',
    'Read about [stoichiometry](https://example.com/stoich).',
    '',
    '```lex-tool',
    JSON.stringify({ instanceId, toolId: 'ask_questions', v: 1 }),
    '```',
    '',
  ].join('\n')
}

async function runAsk(
  token: string,
  courseCode: string,
  instanceId: string,
  question: string,
  idempotencyKey?: string,
): Promise<{ status: number; body: Record<string, unknown> }> {
  const res = await fetch(
    `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/content-tools/instances/${instanceId}/actions/ask`,
    {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${token}`,
      },
      body: JSON.stringify({
        input: { question },
        idempotencyKey: idempotencyKey ?? crypto.randomUUID(),
      }),
    },
  )
  const body = (await res.json()) as Record<string, unknown>
  return { status: res.status, body }
}

test.describe('Content Tools Ask Questions (CT.10)', () => {
  test('ask → cited answer → reload persists; reset clears; rate limit', async ({
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
          allowedToolIds: ['ask_questions', 'noop_probe'],
          studentResetAllowed: true,
          maxInstancesPerItem: 50,
        }),
      ).toBe(200)

      const mod = await apiCreateModule(instructorToken, courseCode, `CT10 ${Date.now()}`)
      const contentPage = await apiCreateContentPage(
        instructorToken,
        courseCode,
        mod.id,
        'CT10 page',
      )
      const inst = await createAskInstance(instructorToken, courseCode, contentPage.id)
      await apiPatchContentPage(instructorToken, courseCode, contentPage.id, {
        markdown: fenceMarkdown(inst.id),
      })

      const ask1 = await runAsk(
        studentToken,
        courseCode,
        inst.id,
        'What does stoichiometric mean here?',
      )
      expect(ask1.status).toBe(200)
      const result1 = ask1.body.result as Record<string, unknown>
      expect(result1.error).toBeUndefined()
      expect(result1.turn).toBeTruthy()
      const state1 = ask1.body.state as {
        state?: { turns?: unknown[] }
        revision?: number
      }
      expect((state1.state?.turns ?? []).length).toBeGreaterThanOrEqual(2)

      const ask2 = await runAsk(studentToken, courseCode, inst.id, 'How is the ratio used?')
      expect(ask2.status).toBe(200)
      expect((ask2.body.result as Record<string, unknown>).error).toBeUndefined()

      const ask3 = await runAsk(studentToken, courseCode, inst.id, 'One more?')
      expect(ask3.status).toBe(200)
      const result3 = ask3.body.result as Record<string, unknown>
      expect(result3.error).toBe('rate_limited')
      expect(String(result3.message || '')).toMatch(/limit|reset/i)

      await injectToken(page, studentToken)
      await page.goto(`/courses/${courseCode}/modules/content/${contentPage.id}`)
      await expect(page.getByTestId('ask-questions')).toBeVisible({ timeout: 20_000 })
      await expect(page.getByTestId('ask-questions-log')).toContainText(/stoichiometric|Dry-run/i)

      const rosterRes = await fetch(
        `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/content-tools/instances/${inst.id}/states?page=1&pageSize=50`,
        { headers: { Authorization: `Bearer ${instructorToken}` } },
      )
      expect(rosterRes.status).toBe(200)
      const roster = (await rosterRes.json()) as {
        items: Array<{ enrollmentId: string; status: string }>
      }
      const started = roster.items.find((r) => r.status === 'in_progress')
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
            reason: 'ct10 e2e reset',
          }),
        },
      )
      expect(resetRes.status).toBe(200)

      const after = await fetch(
        `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/content-tools/instances/${inst.id}/state`,
        { headers: { Authorization: `Bearer ${studentToken}` } },
      )
      expect(after.status).toBe(200)
      const afterBody = (await after.json()) as { state?: { turns?: unknown[] } }
      expect(Array.isArray(afterBody.state?.turns) ? afterBody.state!.turns!.length : 0).toBe(0)
    })
  })
})
