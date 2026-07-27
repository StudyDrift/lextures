/**
 * CT.7 — Analytics insights, student progress, grade link badge.
 *
 * Checklist:
 *   [x] State write produces summary visible in instance analytics
 *   [x] Instructor insights panel shows engagement rates
 *   [x] Grade link enable shows graded badge to student
 *   [x] Student my-progress reports completed/total
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

async function putSettings(token: string, courseCode: string): Promise<void> {
  const res = await fetch(
    `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/content-tools/settings`,
    {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${token}`,
      },
      body: JSON.stringify({
        allowedToolIds: ['noop_probe'],
        studentResetAllowed: false,
        maxInstancesPerItem: 50,
      }),
    },
  )
  if (!res.ok) throw new Error(`settings failed (${res.status}): ${await res.text()}`)
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
        title: 'CT7 probe',
        config: { prompt: 'Capital of France?', answerKey: 'paris', maxAttempts: 3 },
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
): Promise<void> {
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
        status: 'in_progress',
      }),
    },
  )
  if (!res.ok) throw new Error(`put state failed (${res.status}): ${await res.text()}`)
}

async function submitAndGrade(
  token: string,
  courseCode: string,
  instanceId: string,
): Promise<void> {
  const res = await fetch(
    `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/content-tools/instances/${instanceId}/actions/grade`,
    {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${token}`,
      },
      body: JSON.stringify({ input: { response: 'paris' }, idempotencyKey: `ct7-${Date.now()}` }),
    },
  )
  if (!res.ok) throw new Error(`grade action failed (${res.status}): ${await res.text()}`)
}

test.describe('Content Tools analytics (CT.7)', () => {
  test('insights panel + grade badge + my-progress', async ({ page, seededCourse }) => {
    test.setTimeout(90_000)
    const { instructorToken, studentToken, courseCode } = seededCourse

    await withCourseFeatureRestore(instructorToken, courseCode, async () => {
      await apiPatchCourseFeatures(instructorToken, courseCode, {
        contentToolsEnabled: true,
      })
      await putSettings(instructorToken, courseCode)

      const mod = await apiCreateModule(instructorToken, courseCode, 'CT7 Module')
      const content = await apiCreateContentPage(instructorToken, courseCode, mod.id, 'CT7 Page')
      const inst = await createInstance(instructorToken, courseCode, content.id)

      await apiPatchContentPage(instructorToken, courseCode, content.id, {
        markdown: [
          '# CT7 page',
          '',
          'Intro',
          '',
          '```lex-tool',
          JSON.stringify({ instanceId: inst.id, toolId: 'noop_probe', v: 1 }),
          '```',
          '',
        ].join('\n'),
      })

      await putStudentState(studentToken, courseCode, inst.id, 0, 'paris')
      await submitAndGrade(studentToken, courseCode, inst.id)

      const analyticsRes = await fetch(
        `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/content-tools/instances/${inst.id}/analytics`,
        { headers: { Authorization: `Bearer ${instructorToken}` } },
      )
      expect(analyticsRes.status).toBe(200)
      const analytics = (await analyticsRes.json()) as {
        learners: number
        engaged: number
        completed: number
        suppressed: boolean
      }
      expect(analytics.engaged).toBeGreaterThanOrEqual(1)
      expect(analytics.completed).toBeGreaterThanOrEqual(1)

      const progressRes = await fetch(
        `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/content-tools/my-progress?itemId=${content.id}`,
        { headers: { Authorization: `Bearer ${studentToken}` } },
      )
      expect(progressRes.status).toBe(200)
      const progress = (await progressRes.json()) as { completed: number; total: number }
      expect(progress.total).toBe(1)
      expect(progress.completed).toBe(1)

      // Enable grade link pointing at the content page item itself (structure item).
      const gradeRes = await fetch(
        `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/content-tools/instances/${inst.id}/grade-link`,
        {
          method: 'PUT',
          headers: {
            'Content-Type': 'application/json',
            Authorization: `Bearer ${instructorToken}`,
          },
          body: JSON.stringify({
            countsForGrade: true,
            assignmentItemId: content.id,
            pointsPossible: 10,
            latePolicy: 'accept',
          }),
        },
      )
      expect(gradeRes.status).toBe(200)

      await injectToken(page, instructorToken)
      await page.goto(`/courses/${encodeURIComponent(courseCode)}/content-tools`)
      await expect(page.getByTestId('content-tools-insights')).toBeVisible()
      await page.getByTestId(`ct-open-insights-${inst.id}`).click()
      await expect(page.getByTestId('instance-analytics-panel')).toBeVisible()
      await expect(page.getByTestId('analytics-rates')).toBeVisible()
      await page.getByTestId('open-grade-link').click()
      await expect(page.getByTestId('grade-link-dialog')).toBeVisible()

      await injectToken(page, studentToken)
      await page.goto(
        `/courses/${encodeURIComponent(courseCode)}/modules/content/${content.id}`,
      )
      // Graded badge may appear on the tool frame after grade link is enabled.
      await expect(page.getByTestId('tool-graded-badge').first()).toBeVisible({ timeout: 15000 })
    })
  })
})
