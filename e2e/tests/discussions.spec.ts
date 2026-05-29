/**
 * Discussion forums
 *
 * Checklist coverage (docs/e2e.md):
 *   [x] Discussions list page loads
 *   [x] Create a new discussion forum
 *   [x] Post a reply to a discussion thread
 *   [x] Reply appears under parent post
 */
import { test, expect } from '../fixtures/test.js'
import { apiCreateForum, apiCreateDiscussionThread, apiEnableCourseFeatures } from '../fixtures/api.js'

test.describe('Discussions', () => {
  test('discussions page loads', async ({ coursePage: page, seededCourse }) => {
    await apiEnableCourseFeatures(seededCourse.instructorToken, seededCourse.courseCode, {
      discussionsEnabled: true,
    })
    await page.goto(`/courses/${seededCourse.courseCode}/discussions`)
    await expect(page.getByRole('heading', { name: /discussions?/i })).toBeVisible()
  })

  test('forum created via API appears in the list', async ({
    coursePage: page,
    seededCourse,
  }) => {
    await apiEnableCourseFeatures(seededCourse.instructorToken, seededCourse.courseCode, {
      discussionsEnabled: true,
    })
    const forum = await apiCreateForum(
      seededCourse.instructorToken,
      seededCourse.courseCode,
      'E2E API Forum',
    )
    await page.goto(`/courses/${seededCourse.courseCode}/discussions`)
    await expect(page.getByText(forum.name)).toBeVisible({ timeout: 8000 })
  })

  test('create a forum via UI → forum appears in list', async ({
    coursePage: page,
    seededCourse,
  }) => {
    await apiEnableCourseFeatures(seededCourse.instructorToken, seededCourse.courseCode, {
      discussionsEnabled: true,
    })
    await page.goto(`/courses/${seededCourse.courseCode}/discussions`)

    const newForumBtn = page.getByRole('button', { name: /^New$/i })
    await expect(newForumBtn).toBeVisible({ timeout: 5000 })
    await newForumBtn.click()

    await page.getByLabel(/forum name/i).fill('UI Created Forum')
    await page.getByRole('button', { name: /^Create$/i }).click()

    await expect(page.getByText('UI Created Forum')).toBeVisible({ timeout: 8000 })
  })

  test('post a reply to a discussion thread → reply appears', async ({
    coursePage: page,
    seededCourse,
  }) => {
    await apiEnableCourseFeatures(seededCourse.instructorToken, seededCourse.courseCode, {
      discussionsEnabled: true,
    })
    // Seed a forum and thread via API.
    const forum = await apiCreateForum(
      seededCourse.instructorToken,
      seededCourse.courseCode,
      'Reply Test Forum',
    )
    const thread = await apiCreateDiscussionThread(
      seededCourse.instructorToken,
      seededCourse.courseCode,
      forum.id,
      'Reply Test Thread',
    )

    await page.goto(`/courses/${encodeURIComponent(seededCourse.courseCode)}/discussions`)

    await expect(page.getByRole('button', { name: forum.name })).toBeVisible({ timeout: 10_000 })
    await page.getByRole('button', { name: forum.name }).click()

    const threadBtn = page.getByRole('button', { name: new RegExp(thread.title) })
    await expect(threadBtn).toBeVisible({ timeout: 10_000 })
    await threadBtn.click()

    await expect(page.getByRole('heading', { level: 1, name: thread.title })).toBeVisible({
      timeout: 10_000,
    })

    const postBtn = page.getByRole('button', { name: /^Post$/i })
    await expect(postBtn).toBeVisible({ timeout: 15_000 })

    const composer = page.locator('[contenteditable="true"].ProseMirror').last()
    await expect(composer).toBeVisible({ timeout: 10_000 })
    await composer.click()
    const replyText = `E2E reply ${Date.now()}`
    await composer.fill(replyText)

    await postBtn.scrollIntoViewIfNeeded()
    await postBtn.click({ force: true })

    await expect(page.getByText(replyText)).toBeVisible({ timeout: 10000 })
  })
})
