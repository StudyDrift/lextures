/**
 * CT.22 — Inline Discussion: post-before-you-see, peers, reply, report/remove.
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
    prompt: 'What claim in this paragraph do you want to challenge?',
    postBeforeYouSee: true,
    allowReplies: true,
    requiredPosts: 1,
    requiredReplies: 1,
    anonymity: 'anonymous_to_peers',
    editWindowMinutes: 5,
    allowDelete: true,
    sort: 'oldest',
    pageSize: 20,
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
        toolId: 'inline_discussion',
        hostKind: 'content_page',
        structureItemId,
        title: 'CT22 Inline Discussion',
        config,
      }),
    },
  )
  if (!res.ok) throw new Error(`create instance failed (${res.status}): ${await res.text()}`)
  return (await res.json()) as { id: string }
}

function fenceMarkdown(instanceId: string): string {
  return [
    '# CT22 page',
    '',
    '```lex-tool',
    JSON.stringify({ instanceId, toolId: 'inline_discussion', v: 1 }),
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

test.describe('Content Tools Inline Discussion (CT.22)', () => {
  test('locked → post → peers → reply → endorse → report/remove → UI', async ({
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
          allowedToolIds: ['inline_discussion'],
          studentResetAllowed: true,
          maxInstancesPerItem: 50,
        }),
      ).toBe(200)

      const module = await apiCreateModule(instructorToken, courseCode, 'CT22 Module')
      const contentPage = await apiCreateContentPage(
        instructorToken,
        courseCode,
        module.id,
        'CT22 Page',
      )
      const inst = await createInstance(instructorToken, courseCode, contentPage.id)
      await apiPatchContentPage(instructorToken, courseCode, contentPage.id, {
        markdown: fenceMarkdown(inst.id),
      })

      const locked = await runAction(
        studentToken,
        courseCode,
        inst.id,
        'thread',
        { page: 1 },
        'ct22-thread-1',
      )
      expect(locked.status).toBe(200)
      expect((locked.body.result as { canSeePeers?: boolean; locked?: boolean }).canSeePeers).toBe(
        false,
      )
      expect((locked.body.result as { locked?: boolean }).locked).toBe(true)

      const filtered = await runAction(
        studentToken,
        courseCode,
        inst.id,
        'post',
        { text: 'this is fucking awful' },
        'ct22-filter',
      )
      expect(filtered.status).toBe(200)
      expect((filtered.body.result as { error?: string }).error).toBe('filtered')
      expect((filtered.body.result as { preserveInput?: boolean }).preserveInput).toBe(true)

      const posted = await runAction(
        studentToken,
        courseCode,
        inst.id,
        'post',
        { text: 'I challenge the causality claim in paragraph two.' },
        'ct22-post-1',
      )
      expect(posted.status).toBe(200)
      const postResult = posted.body.result as {
        error?: string
        canSeePeers?: boolean
        post?: { id: string }
      }
      expect(postResult.error).toBeFalsy()
      expect(postResult.canSeePeers).toBe(true)
      expect(postResult.post?.id).toBeTruthy()
      const studentPostId = postResult.post!.id

      const instructorSeed = await runAction(
        instructorToken,
        courseCode,
        inst.id,
        'post',
        { text: 'Here is a classmate-facing seed from the instructor enrollment.' },
        'ct22-instructor-seed',
      )
      expect(instructorSeed.status).toBe(200)
      const seedId = (instructorSeed.body.result as { post?: { id: string } }).post?.id
      expect(seedId).toBeTruthy()

      const unlocked = await runAction(
        studentToken,
        courseCode,
        inst.id,
        'thread',
        { page: 1 },
        'ct22-thread-2',
      )
      expect(unlocked.status).toBe(200)
      const posts =
        ((unlocked.body.result as { posts?: Array<Record<string, unknown>> }).posts) ?? []
      const peer = posts.find((p) => p.isOwn !== true)
      if (peer) {
        expect(peer.authorId).toBeUndefined()
        expect(peer.anonymous).toBe(true)
      }

      const replied = await runAction(
        studentToken,
        courseCode,
        inst.id,
        'post',
        {
          text: 'Building on that — the sample size looks thin.',
          parentPostId: seedId,
        },
        'ct22-reply-1',
      )
      expect(replied.status).toBe(200)
      const replyState = (replied.body.result as { state?: { completedAt?: string; myReplyIds?: string[] } })
        .state
      expect((replyState?.myReplyIds ?? []).length).toBeGreaterThanOrEqual(1)
      expect(replyState?.completedAt).toBeTruthy()

      const endorse = await runAction(
        instructorToken,
        courseCode,
        inst.id,
        'endorse',
        { postId: studentPostId },
        'ct22-endorse',
      )
      expect(endorse.status).toBe(200)
      expect((endorse.body.result as { post?: { endorsed?: boolean } }).post?.endorsed).toBe(true)

      const report = await runAction(
        studentToken,
        courseCode,
        inst.id,
        'report',
        { postId: seedId, category: 'inappropriate' },
        'ct22-report',
      )
      expect(report.status).toBe(200)

      const moderate = await runAction(
        instructorToken,
        courseCode,
        inst.id,
        'moderate',
        { postId: seedId, action: 'removed', reason: 'e2e' },
        'ct22-moderate',
      )
      expect(moderate.status).toBe(200)

      const getRemoved = await runAction(
        studentToken,
        courseCode,
        inst.id,
        'get_post',
        { postId: seedId },
        'ct22-get-removed',
      )
      expect((getRemoved.body.result as { error?: string }).error).toBe('not_found')

      await injectToken(page, studentToken)
      await page.goto(`/courses/${courseCode}/modules/content/${contentPage.id}`)
      const tool = page.locator('[data-content-tool="inline_discussion"]').first()
      await expect(tool).toBeVisible({ timeout: 20_000 })
      await expect(page.getByTestId('inline-discussion-composer')).toBeVisible()
      await expect(page.getByTestId('inline-discussion-anonymity-note')).toBeVisible()
    })
  })
})
