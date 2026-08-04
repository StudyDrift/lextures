/**
 * Course Checklist web UI (CC.7) — nav badge, page, dismiss/restore, student no-access.
 */
import { expect, test } from '@playwright/test'
import { apiSignup, apiCreateCourse, apiEnroll } from '../fixtures/api.js'
import { injectToken, mainNav, uniqueEmail } from '../fixtures/test.js'

const PASSWORD = 'E2eTestPass1!'
const API_BASE = process.env.E2E_API_URL ?? 'http://localhost:8080'

test.describe('Course checklist UI (CC.7)', () => {
  test('teacher sees nav entry and can open checklist page', async ({ page }) => {
    const email = uniqueEmail('cc7-teacher')
    const { access_token: token } = await apiSignup({
      email,
      password: PASSWORD,
      displayName: 'CC7 Teacher',
    })
    const course = await apiCreateCourse(token, { title: `CC7 UI ${Date.now()}` })
    await apiEnroll(token, course.courseCode, email, 'teacher')

    await injectToken(page, token)
    await page.goto(`/courses/${encodeURIComponent(course.courseCode)}`)
    await expect(mainNav(page)).toBeVisible()

    const checklistNav = page.getByRole('link', { name: /checklist/i })
    await expect(checklistNav).toBeVisible()
    await expect(checklistNav).toHaveAttribute(
      'href',
      `/courses/${encodeURIComponent(course.courseCode)}/checklist`,
    )

    await checklistNav.click()
    await expect(page.getByRole('heading', { name: /course checklist/i })).toBeVisible({
      timeout: 15_000,
    })
    await expect(page.getByRole('button', { name: /re-check/i })).toBeVisible()
    // Bare course should surface foundation items (CC.3).
    await expect(page.getByText(/welcome/i).first()).toBeVisible({ timeout: 15_000 })
  })

  test('teacher can dismiss and restore an item', async ({ page }) => {
    const email = uniqueEmail('cc7-dismiss')
    const { access_token: token } = await apiSignup({
      email,
      password: PASSWORD,
      displayName: 'CC7 Dismiss',
    })
    const course = await apiCreateCourse(token, { title: `CC7 Dismiss ${Date.now()}` })
    await apiEnroll(token, course.courseCode, email, 'teacher')

    await injectToken(page, token)
    await page.goto(`/courses/${encodeURIComponent(course.courseCode)}/checklist`)
    await expect(page.getByRole('heading', { name: /course checklist/i })).toBeVisible({
      timeout: 15_000,
    })

    const firstActions = page.getByRole('button', { name: /item actions/i }).first()
    await expect(firstActions).toBeVisible({ timeout: 15_000 })
    await firstActions.click()
    await page.getByRole('menuitem', { name: /^dismiss$/i }).click()
    await expect(page.getByRole('dialog', { name: /dismiss checklist item/i })).toBeVisible()
    await page.getByLabel(/^reason$/i).selectOption('not_applicable')
    await page.getByRole('button', { name: /dismiss item/i }).click()
    await expect(page.getByText(/item dismissed/i)).toBeVisible({ timeout: 10_000 })

    await page.getByRole('button', { name: /dismissed \(/i }).click()
    await expect(page.getByRole('button', { name: /^restore$/i }).first()).toBeVisible()
    await page.getByRole('button', { name: /^restore$/i }).first().click()
    await expect(page.getByText(/item restored/i)).toBeVisible({ timeout: 10_000 })
  })

  test('student has no nav item and sees no-access on direct URL', async ({ page }) => {
    const teacherEmail = uniqueEmail('cc7-t')
    const studentEmail = uniqueEmail('cc7-s')
    const { access_token: teacherTok } = await apiSignup({
      email: teacherEmail,
      password: PASSWORD,
    })
    const { access_token: studentTok } = await apiSignup({
      email: studentEmail,
      password: PASSWORD,
    })
    const course = await apiCreateCourse(teacherTok, { title: `CC7 Student ${Date.now()}` })
    await apiEnroll(teacherTok, course.courseCode, teacherEmail, 'teacher')
    await apiEnroll(teacherTok, course.courseCode, studentEmail, 'student', studentTok)

    // Confirm API still returns items for teacher so the course is not empty.
    const teacherRes = await fetch(
      `${API_BASE}/api/v1/courses/${encodeURIComponent(course.courseCode)}/checklist`,
      { headers: { Authorization: `Bearer ${teacherTok}` } },
    )
    expect(teacherRes.status).toBe(200)
    const teacherBody = (await teacherRes.json()) as {
      categories?: Array<{ items?: Array<{ title?: string }> }>
    }
    const sampleTitle = teacherBody.categories?.[0]?.items?.[0]?.title
    expect(sampleTitle).toBeTruthy()

    await injectToken(page, studentTok)
    await page.goto(`/courses/${encodeURIComponent(course.courseCode)}`)
    await expect(mainNav(page)).toBeVisible()
    await expect(page.getByRole('link', { name: /checklist/i })).toHaveCount(0)

    await page.goto(`/courses/${encodeURIComponent(course.courseCode)}/checklist`)
    await expect(page.getByText(/don't have access/i)).toBeVisible({ timeout: 15_000 })
    if (sampleTitle) {
      await expect(page.getByText(sampleTitle)).toHaveCount(0)
    }
  })
})
