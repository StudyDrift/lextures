import { expect, test } from '@playwright/test'
import { apiCreateCourse, apiEnroll, apiSignup } from '../fixtures/api.js'
import { injectToken, mainNav, uniqueEmail } from '../fixtures/test.js'

const API_BASE = process.env.E2E_API_URL ?? 'http://localhost:8080'
const PASSWORD = 'E2eTestPass1!'

test.describe('Family progress view (#653)', () => {
  test('managed learner courses deep-link and family gradebook/reports copy', async ({
    page,
    request,
  }) => {
    const email = uniqueEmail('family-progress')
    const { access_token } = await apiSignup({
      email,
      password: PASSWORD,
      displayName: 'Family Teacher',
    })

    const featuresRes = await request.get(`${API_BASE}/api/v1/platform/features`, {
      headers: { Authorization: `Bearer ${access_token}` },
    })
    expect(featuresRes.ok()).toBeTruthy()
    const features = (await featuresRes.json()) as { ffHomeschoolManagedLearners?: boolean }
    test.skip(!features.ffHomeschoolManagedLearners, 'managed learners disabled')

    const course = await apiCreateCourse(access_token, {
      title: 'Family Math',
      gradeLevels: ['3-5'],
    })
    await apiEnroll(access_token, course.courseCode, email, 'teacher')

    const depRes = await request.post(`${API_BASE}/api/v1/me/dependents`, {
      headers: { Authorization: `Bearer ${access_token}`, 'Content-Type': 'application/json' },
      data: { displayName: 'Alex Learner', gradeLevel: '4' },
    })
    expect(depRes.ok(), await depRes.text()).toBeTruthy()
    const dependent = (await depRes.json()) as { id: string; displayName: string }
    expect(dependent.id).toBeTruthy()

    const enrollRes = await request.post(
      `${API_BASE}/api/v1/courses/${encodeURIComponent(course.courseCode)}/enrollments`,
      {
        headers: { Authorization: `Bearer ${access_token}`, 'Content-Type': 'application/json' },
        data: { learnerUserIds: [dependent.id], courseRole: 'student' },
      },
    )
    expect(enrollRes.ok(), await enrollRes.text()).toBeTruthy()

    const coursesRes = await request.get(`${API_BASE}/api/v1/me/dependents/courses`, {
      headers: { Authorization: `Bearer ${access_token}` },
    })
    expect(coursesRes.ok(), await coursesRes.text()).toBeTruthy()
    const body = (await coursesRes.json()) as {
      enrollments?: { dependentId: string; courseCode: string; enrollmentId: string; courseTitle: string }[]
    }
    const link = (body.enrollments ?? []).find(
      (row) => row.dependentId === dependent.id && row.courseCode === course.courseCode,
    )
    expect(link?.enrollmentId).toBeTruthy()

    await injectToken(page, access_token)
    await page.goto(`/courses/${encodeURIComponent(course.courseCode)}/gradebook`)
    await expect(mainNav(page)).toBeVisible()
    await expect(page.getByText('Scores for each learner')).toBeVisible()
    await expect(page.getByLabel('Learner')).toBeVisible()

    await page.goto(`/courses/${encodeURIComponent(course.courseCode)}/reports`)
    await expect(page.getByRole('heading', { name: 'Alex Learner' })).toBeVisible()
    await page.getByRole('link', { name: 'See progress' }).click()
    await expect(page).toHaveURL(new RegExp(`/students/${link!.enrollmentId}/progress`))

    await page.goto('/learners')
    await expect(page.getByRole('link', { name: 'Progress in Family Math' })).toBeVisible()
  })
})
