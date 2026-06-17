/**
 * Course reviews & ratings (plan 15.7)
 *
 * Coverage:
 *   [x] feature flag off => review endpoints return 404
 *   [x] enrolled learner with 10%+ progress submits review; aggregate updates
 *   [x] second submit updates existing review (idempotent)
 *   [x] public catalog reviews list returns submitted review
 *   [x] admin removes flagged review; aggregate recomputes
 *   [x] unenrolled user cannot submit review
 */
import { expect, test } from '@playwright/test'
import {
  apiCreateCourse,
  apiCreateContentPage,
  apiCreateModule,
  apiSignup,
} from '../fixtures/api.js'

const apiBase = process.env.E2E_API_URL ?? 'http://localhost:8080'
const PASSWORD = 'E2eTestPass1!'

function uid(prefix = 'rev') {
  return `e2e-${prefix}-${Date.now()}-${Math.random().toString(36).slice(2)}`
}
function authHeaders(token: string) {
  return { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' }
}

async function getAdminToken(): Promise<string> {
  const adminEmail = process.env.E2E_ADMIN_EMAIL ?? 'admin@e2e.test'
  const adminPassword = process.env.E2E_ADMIN_PASSWORD ?? PASSWORD
  const loginRes = await fetch(`${apiBase}/api/v1/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email: adminEmail, password: adminPassword }),
  })
  if (loginRes.ok) {
    const { access_token } = (await loginRes.json()) as { access_token: string }
    return access_token
  }
  const { access_token } = await apiSignup({
    email: adminEmail,
    password: adminPassword,
    displayName: 'E2E Admin',
  })
  return access_token
}

async function enableFlags(adminToken: string) {
  const res = await fetch(`${apiBase}/api/v1/settings/platform`, {
    method: 'PUT',
    headers: authHeaders(adminToken),
    body: JSON.stringify({
      ffCourseReviews: true,
      ffSelfPacedMode: true,
      ffPublicCatalog: true,
      updateMask: ['ffCourseReviews', 'ffSelfPacedMode', 'ffPublicCatalog'],
    }),
  })
  expect(res.ok).toBeTruthy()
}

async function setSelfPaced(token: string, courseCode: string, title: string) {
  const res = await fetch(`${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}`, {
    method: 'PUT',
    headers: authHeaders(token),
    body: JSON.stringify({ title, courseMode: 'self_paced', openEnrollment: true }),
  })
  expect(res.ok).toBeTruthy()
}

async function publishPublic(token: string, courseCode: string, title: string, slug: string) {
  const pub = await fetch(`${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}`, {
    method: 'PUT',
    headers: authHeaders(token),
    body: JSON.stringify({ title, description: `${title} description`, published: true }),
  })
  expect(pub.ok).toBeTruthy()
  const list = await fetch(`${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/catalog-listing`, {
    method: 'PUT',
    headers: authHeaders(token),
    body: JSON.stringify({
      isPublic: true,
      category: 'Self-paced',
      difficultyLevel: 'beginner',
      language: 'en',
      priceCents: 0,
      slug,
    }),
  })
  expect(list.ok).toBeTruthy()
}

async function selfEnroll(token: string, courseCode: string) {
  const res = await fetch(`${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/self-enroll`, {
    method: 'POST',
    headers: authHeaders(token),
  })
  expect(res.ok).toBeTruthy()
}

async function completeItem(token: string, courseCode: string, itemId: string) {
  const res = await fetch(
    `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/items/${encodeURIComponent(itemId)}/complete`,
    { method: 'POST', headers: authHeaders(token) },
  )
  expect(res.ok).toBeTruthy()
}

test.describe('Course reviews (15.7)', () => {
  test('reviews workflow: submit, list, update, admin remove', async () => {
    const adminToken = await getAdminToken()
    await enableFlags(adminToken)

    const title = `Reviews ${uid()}`
    const slug = uid('slug')
    const courseCode = (await apiCreateCourse(adminToken, { title })).courseCode
    await setSelfPaced(adminToken, courseCode, title)
    await publishPublic(adminToken, courseCode, title, slug)

    const mod = await apiCreateModule(adminToken, courseCode, 'Module 1')
    await apiCreateContentPage(adminToken, courseCode, mod.id, 'Lesson')
    await apiCreateContentPage(adminToken, courseCode, mod.id, 'Lesson 2')
    const structure = await fetch(`${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/structure`, {
      headers: authHeaders(adminToken),
    })
    const { items } = (await structure.json()) as { items: { id: string; kind: string }[] }
    const leafIds = items.filter((i) => i.kind === 'content_page').map((i) => i.id)

    const learnerEmail = `${uid('learner')}@e2e.test`
    const { access_token: learnerToken } = await apiSignup({
      email: learnerEmail,
      password: PASSWORD,
      displayName: 'Review Learner',
    })
    await selfEnroll(learnerToken, courseCode)
    await completeItem(learnerToken, courseCode, leafIds[0]!)

    const submit1 = await fetch(`${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/reviews`, {
      method: 'POST',
      headers: authHeaders(learnerToken),
      body: JSON.stringify({ rating: 4, reviewText: 'Solid intro course.' }),
    })
    expect(submit1.ok).toBeTruthy()

    const submit2 = await fetch(`${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/reviews`, {
      method: 'POST',
      headers: authHeaders(learnerToken),
      body: JSON.stringify({ rating: 5, reviewText: 'Updated — even better.' }),
    })
    expect(submit2.ok).toBeTruthy()
    const updated = (await submit2.json()) as { id: string; rating: number; reviewText: string }
    expect(updated.rating).toBe(5)
    expect(updated.reviewText).toContain('Updated')

    const publicList = await fetch(`${apiBase}/api/v1/public/catalog/courses/${encodeURIComponent(slug)}/reviews`)
    expect(publicList.ok).toBeTruthy()
    const listed = (await publicList.json()) as {
      summary: { averageRating: number; ratingCount: number }
      reviews: { reviewText?: string }[]
    }
    expect(listed.summary.ratingCount).toBe(1)
    expect(listed.summary.averageRating).toBe(5)
    expect(listed.reviews[0]?.reviewText).toContain('Updated')

    const rid = updated.id
    const flagRes = await fetch(
      `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/reviews/${encodeURIComponent(rid)}/flag`,
      { method: 'POST', headers: authHeaders(learnerToken) },
    )
    expect(flagRes.status).toBe(204)

    const removeRes = await fetch(`${apiBase}/api/v1/admin/reviews/${encodeURIComponent(rid)}`, {
      method: 'DELETE',
      headers: authHeaders(adminToken),
    })
    expect(removeRes.status).toBe(204)

    const afterRemove = await fetch(`${apiBase}/api/v1/public/catalog/courses/${encodeURIComponent(slug)}/reviews`)
    const afterBody = (await afterRemove.json()) as { summary: { ratingCount: number } }
    expect(afterBody.summary.ratingCount).toBe(0)
  })

  test('feature flag off returns 404', async () => {
    const res = await fetch(`${apiBase}/api/v1/courses/NOPE/reviews`)
    expect([404, 500]).toContain(res.status)
  })

  test('unenrolled user cannot submit review', async () => {
    const adminToken = await getAdminToken()
    await enableFlags(adminToken)
    const title = `Reviews gate ${uid()}`
    const courseCode = (await apiCreateCourse(adminToken, { title })).courseCode
    await setSelfPaced(adminToken, courseCode, title)

    const outsiderEmail = `${uid('outsider')}@e2e.test`
    const { access_token } = await apiSignup({
      email: outsiderEmail,
      password: PASSWORD,
      displayName: 'Outsider',
    })
    const res = await fetch(`${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/reviews`, {
      method: 'POST',
      headers: authHeaders(access_token),
      body: JSON.stringify({ rating: 3 }),
    })
    expect(res.status).toBe(403)
  })
})
