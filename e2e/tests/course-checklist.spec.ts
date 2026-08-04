/**
 * Course Checklist state API (CC.2) + foundation rule pack (CC.3) + structure/outcomes (CC.4)
 * + assessment/grading/feedback (CC.5).
 *
 * Checklist coverage:
 *   [x] Unauthenticated GET /checklist returns 401
 *   [x] Student GET /checklist returns 403
 *   [x] Teacher GET /checklist returns 200 with summary
 *   [x] Teacher dismisses/restores item (CC.3: all rules recommended — badge may stay 0)
 *   [x] Bare course shows orientation.welcome-message + people.students-enrolled as todo
 *   [x] Post welcome announcement → orientation.welcome-message becomes done
 *   [x] Enroll a student → people.students-enrolled becomes done
 *   [x] CC.4: 5 assignments / 2 mapped → outcomes.assessment-mapping shows 3 evidence rows
 *   [x] CC.4: map a third assignment → evidence shrinks after refresh
 *   [x] CC.5: weights 40/30/17 → grading.group-weights todo with 87% detail; fix to 100% → done
 *   [x] CC.5: 30% assignment without rubric → feedback.rubrics-on-high-stakes evidence targets rubric
 *   [x] CC.6: image without alt → a11y.image-alt-text evidence; add alt → done
 *   [x] CC.6: View as student stamps launch.student-preview → done
 */

import { test, expect } from '@playwright/test'
import {
  apiSignup,
  apiCreateCourse,
  apiEnroll,
  apiGetFeedChannels,
  apiPostFeedMessage,
  apiCreateModule,
  apiCreateAssignment,
  apiPatchAssignment,
  apiCreateOutcome,
  apiCreateOutcomeLink,
  apiPutCourseGrading,
  apiGetCourseGrading,
  apiCreateContentPage,
  apiPatchContentPage,
} from '../fixtures/api.js'

const API_BASE = process.env.E2E_API_URL ?? 'http://localhost:8080'
const PASSWORD = 'E2eTestPass1!'

let _seq = 0
function uniqueEmail(prefix = 'cc3') {
  return `e2e-${prefix}-${Date.now()}-${++_seq}@test.invalid`
}

type ChecklistItem = {
  id: string
  status: string
  detail?: string | null
  finding?: { status?: string; detailDefault?: string }
  progress?: { done: number; total: number }
  evidence?: {
    columns?: string[]
    rows?: Array<{
      label?: string
      sublabel?: string
      target?: { route?: string; anchor?: string }
    }>
  }
}

type ChecklistResponse = {
  engineVersion: number
  catalogVersion: string
  summary: {
    outstandingEssential: number
    dismissed: number
    todo?: number
  }
  categories?: Array<{ items?: ChecklistItem[] }>
  items?: ChecklistItem[]
}

function findItem(list: ChecklistResponse, id: string): ChecklistItem | undefined {
  for (const cat of list.categories ?? []) {
    for (const it of cat.items ?? []) {
      if (it.id === id) return it
    }
  }
  for (const it of list.items ?? []) {
    if (it.id === id) return it
  }
  return undefined
}

function itemStatus(it: ChecklistItem | undefined): string {
  return it?.status ?? it?.finding?.status ?? ''
}

async function fetchChecklist(token: string, courseCode: string): Promise<ChecklistResponse> {
  const res = await fetch(`${API_BASE}/api/v1/courses/${courseCode}/checklist`, {
    headers: { Authorization: `Bearer ${token}` },
  })
  expect(res.status).toBe(200)
  return res.json() as Promise<ChecklistResponse>
}

test('Course checklist: unauthenticated GET returns 401', async () => {
  const res = await fetch(`${API_BASE}/api/v1/courses/C-FAKE/checklist`)
  expect(res.status).toBe(401)
})

test('Course checklist: student receives 403; teacher dismiss/restore updates dismissed count', async () => {
  const teacherEmail = uniqueEmail('cc3-teacher')
  const studentEmail = uniqueEmail('cc3-student')
  const { access_token: teacherTok } = await apiSignup({ email: teacherEmail, password: PASSWORD })
  const { access_token: studentTok } = await apiSignup({ email: studentEmail, password: PASSWORD })
  const course = await apiCreateCourse(teacherTok, { title: 'CC3 Checklist Course' })
  await apiEnroll(teacherTok, course.courseCode, teacherEmail, 'teacher')
  await apiEnroll(teacherTok, course.courseCode, studentEmail, 'student', studentTok)

  const studentRes = await fetch(`${API_BASE}/api/v1/courses/${course.courseCode}/checklist`, {
    headers: { Authorization: `Bearer ${studentTok}` },
  })
  expect(studentRes.status).toBe(403)
  const studentBody = await studentRes.text()
  expect(studentBody).not.toContain('Set course start')

  const list = await fetchChecklist(teacherTok, course.courseCode)
  expect(list.engineVersion).toBeGreaterThanOrEqual(1)
  expect(typeof list.catalogVersion).toBe('string')
  // CC.3 FR-37: all foundation rules ship as recommended — essentials may be 0.
  const beforeEssential = list.summary.outstandingEssential
  expect(beforeEssential).toBeGreaterThanOrEqual(0)

  const dismissRes = await fetch(
    `${API_BASE}/api/v1/courses/${course.courseCode}/checklist/items/course.dates/dismiss`,
    {
      method: 'POST',
      headers: {
        Authorization: `Bearer ${teacherTok}`,
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ reason: 'not_applicable', note: 'e2e dismiss' }),
    },
  )
  expect(dismissRes.status).toBe(200)

  const summaryAfterDismiss = await fetch(
    `${API_BASE}/api/v1/courses/${course.courseCode}/checklist/summary`,
    { headers: { Authorization: `Bearer ${teacherTok}` } },
  )
  expect(summaryAfterDismiss.status).toBe(200)
  const summary1 = await summaryAfterDismiss.json()
  expect(summary1.dismissed).toBe(1)
  expect(summary1.outstandingEssential).toBe(beforeEssential)

  const restoreRes = await fetch(
    `${API_BASE}/api/v1/courses/${course.courseCode}/checklist/items/course.dates/restore`,
    {
      method: 'POST',
      headers: { Authorization: `Bearer ${teacherTok}` },
    },
  )
  expect(restoreRes.status).toBe(200)

  const summaryAfterRestore = await fetch(
    `${API_BASE}/api/v1/courses/${course.courseCode}/checklist/summary`,
    { headers: { Authorization: `Bearer ${teacherTok}` } },
  )
  const summary2 = await summaryAfterRestore.json()
  expect(summary2.dismissed).toBe(0)
  expect(summary2.outstandingEssential).toBe(beforeEssential)
})

test('Course checklist CC.3: welcome message and enrollment flip to done', async () => {
  const teacherEmail = uniqueEmail('cc3-rules-teacher')
  const studentEmail = uniqueEmail('cc3-rules-student')
  const { access_token: teacherTok } = await apiSignup({ email: teacherEmail, password: PASSWORD })
  const { access_token: studentTok } = await apiSignup({ email: studentEmail, password: PASSWORD })
  const course = await apiCreateCourse(teacherTok, { title: 'CC3 Rules Course' })
  await apiEnroll(teacherTok, course.courseCode, teacherEmail, 'teacher')

  async function refreshChecklist(token: string, courseCode: string): Promise<ChecklistResponse> {
    const res = await fetch(`${API_BASE}/api/v1/courses/${courseCode}/checklist/refresh`, {
      method: 'POST',
      headers: { Authorization: `Bearer ${token}` },
    })
    expect(res.status).toBe(200)
    return res.json() as Promise<ChecklistResponse>
  }

  let list = await refreshChecklist(teacherTok, course.courseCode)
  expect(itemStatus(findItem(list, 'orientation.welcome-message'))).toBe('todo')
  expect(itemStatus(findItem(list, 'people.students-enrolled'))).toBe('todo')

  // apiGetFeedChannels historically typed as an array; the API returns { channels }.
  const channelsRaw = (await apiGetFeedChannels(teacherTok, course.courseCode)) as
    | Array<{ id: string; name: string }>
    | { channels?: Array<{ id: string; name: string }> }
  const channels = Array.isArray(channelsRaw) ? channelsRaw : (channelsRaw.channels ?? [])
  const announcements = channels.find((c) => c.name.toLowerCase() === 'announcements')
  expect(announcements, 'announcements channel should exist').toBeTruthy()
  const welcomeBody = 'Welcome to the course! '.repeat(20) // ≥ 200 chars
  await apiPostFeedMessage(teacherTok, course.courseCode, announcements!.id, welcomeBody)

  list = await refreshChecklist(teacherTok, course.courseCode)
  expect(itemStatus(findItem(list, 'orientation.welcome-message'))).toBe('done')

  await apiEnroll(teacherTok, course.courseCode, studentEmail, 'student', studentTok)
  list = await refreshChecklist(teacherTok, course.courseCode)
  expect(itemStatus(findItem(list, 'people.students-enrolled'))).toBe('done')
})

test('Course checklist CC.4: assessment-mapping evidence rows shrink after mapping', async () => {
  const teacherEmail = uniqueEmail('cc4-map-teacher')
  const { access_token: teacherTok } = await apiSignup({ email: teacherEmail, password: PASSWORD })
  const course = await apiCreateCourse(teacherTok, { title: 'CC4 Mapping Course' })
  await apiEnroll(teacherTok, course.courseCode, teacherEmail, 'teacher')

  const mod = await apiCreateModule(teacherTok, course.courseCode, 'Unit 1')
  const assignments = []
  for (let i = 1; i <= 5; i++) {
    const a = await apiCreateAssignment(teacherTok, course.courseCode, mod.id, `Assignment ${i}`)
    await apiPatchAssignment(teacherTok, course.courseCode, a.id, { pointsWorth: 10 })
    assignments.push(a)
  }
  const outcome = await apiCreateOutcome(teacherTok, course.courseCode, 'Apply course concepts', 'Learners apply concepts.')
  await apiCreateOutcomeLink(teacherTok, course.courseCode, outcome.id, {
    structureItemId: assignments[0].id,
    targetKind: 'assignment',
  })
  await apiCreateOutcomeLink(teacherTok, course.courseCode, outcome.id, {
    structureItemId: assignments[1].id,
    targetKind: 'assignment',
  })

  async function refreshChecklist(token: string, courseCode: string): Promise<ChecklistResponse> {
    const res = await fetch(`${API_BASE}/api/v1/courses/${courseCode}/checklist/refresh`, {
      method: 'POST',
      headers: { Authorization: `Bearer ${token}` },
    })
    expect(res.status).toBe(200)
    return res.json() as Promise<ChecklistResponse>
  }

  let list = await refreshChecklist(teacherTok, course.courseCode)
  const mapping = findItem(list, 'outcomes.assessment-mapping')
  expect(mapping, 'outcomes.assessment-mapping should be present').toBeTruthy()
  expect(itemStatus(mapping)).toBe('in_progress')
  expect(mapping!.progress).toEqual({ done: 2, total: 5 })
  expect(mapping!.evidence?.columns).toEqual(['Item', 'Type', 'Module', 'Points'])
  expect(mapping!.evidence?.rows?.length).toBe(3)
  const firstRow = mapping!.evidence!.rows![0]
  expect(firstRow.target?.route).toContain('/modules/assignment/')
  expect(firstRow.target?.anchor).toBe('assignment.outcomes-mapping')

  await apiCreateOutcomeLink(teacherTok, course.courseCode, outcome.id, {
    structureItemId: assignments[2].id,
    targetKind: 'assignment',
  })
  list = await refreshChecklist(teacherTok, course.courseCode)
  const after = findItem(list, 'outcomes.assessment-mapping')
  expect(itemStatus(after)).toBe('in_progress')
  expect(after!.progress).toEqual({ done: 3, total: 5 })
  expect(after!.evidence?.rows?.length).toBe(2)
})

test('Course checklist CC.5: group weights 87% → todo, then 100% → done', async () => {
  const teacherEmail = uniqueEmail('cc5-weights-teacher')
  const { access_token: teacherTok } = await apiSignup({ email: teacherEmail, password: PASSWORD })
  const course = await apiCreateCourse(teacherTok, { title: 'CC5 Weights Course' })
  await apiEnroll(teacherTok, course.courseCode, teacherEmail, 'teacher')

  async function refreshChecklist(token: string, courseCode: string): Promise<ChecklistResponse> {
    const res = await fetch(`${API_BASE}/api/v1/courses/${courseCode}/checklist/refresh`, {
      method: 'POST',
      headers: { Authorization: `Bearer ${token}` },
    })
    expect(res.status).toBe(200)
    return res.json() as Promise<ChecklistResponse>
  }

  await apiPutCourseGrading(teacherTok, course.courseCode, {
    gradingScale: 'percent',
    assignmentGroups: [
      { name: 'Essays', sortOrder: 0, weightPercent: 40 },
      { name: 'Quizzes', sortOrder: 1, weightPercent: 30 },
      { name: 'Participation', sortOrder: 2, weightPercent: 17 },
    ],
  })

  let list = await refreshChecklist(teacherTok, course.courseCode)
  const weights = findItem(list, 'grading.group-weights')
  expect(weights, 'grading.group-weights should be present').toBeTruthy()
  expect(itemStatus(weights)).toBe('todo')
  const detail = weights!.detail ?? weights!.finding?.detailDefault ?? ''
  expect(detail).toMatch(/87%/)
  expect(detail).toMatch(/100%/)

  const current = await apiGetCourseGrading(teacherTok, course.courseCode)
  const groups = current.assignmentGroups ?? []
  expect(groups.length).toBeGreaterThanOrEqual(3)
  await apiPutCourseGrading(teacherTok, course.courseCode, {
    gradingScale: 'percent',
    assignmentGroups: groups.map((g) => ({
      id: g.id,
      name: g.name,
      sortOrder: g.sortOrder,
      weightPercent: g.name === 'Participation' ? 30 : g.weightPercent,
    })),
  })
  list = await refreshChecklist(teacherTok, course.courseCode)
  expect(itemStatus(findItem(list, 'grading.group-weights'))).toBe('done')
})

test('Course checklist CC.5: high-stakes assignment without rubric targets rubric section', async () => {
  const teacherEmail = uniqueEmail('cc5-rubric-teacher')
  const { access_token: teacherTok } = await apiSignup({ email: teacherEmail, password: PASSWORD })
  const course = await apiCreateCourse(teacherTok, { title: 'CC5 Rubric Course' })
  await apiEnroll(teacherTok, course.courseCode, teacherEmail, 'teacher')

  const mod = await apiCreateModule(teacherTok, course.courseCode, 'Unit 1')
  const capstone = await apiCreateAssignment(teacherTok, course.courseCode, mod.id, 'Capstone Essay')
  await apiPatchAssignment(teacherTok, course.courseCode, capstone.id, {
    pointsWorth: 30,
    markdown: '',
  })
  const filler = await apiCreateAssignment(teacherTok, course.courseCode, mod.id, 'Weekly quiz points')
  await apiPatchAssignment(teacherTok, course.courseCode, filler.id, {
    pointsWorth: 70,
    markdown: 'Complete the weekly work.',
  })

  const res = await fetch(`${API_BASE}/api/v1/courses/${course.courseCode}/checklist/refresh`, {
    method: 'POST',
    headers: { Authorization: `Bearer ${teacherTok}` },
  })
  expect(res.status).toBe(200)
  const list = (await res.json()) as ChecklistResponse
  const item = findItem(list, 'feedback.rubrics-on-high-stakes')
  expect(item, 'feedback.rubrics-on-high-stakes should be present').toBeTruthy()
  expect(itemStatus(item)).toBe('todo')
  const row = item!.evidence?.rows?.find((r) => (r.label ?? '').includes('Capstone'))
  expect(row, 'capstone should appear in evidence').toBeTruthy()
  expect(row!.target?.route).toContain(`/modules/assignment/${capstone.id}`)
  expect(row!.target?.anchor).toBe('assignment.rubric')
})

test('Course checklist CC.6: missing alt text → evidence, then done after fix', async () => {
  const teacherEmail = uniqueEmail('cc6-alt-teacher')
  const { access_token: teacherTok } = await apiSignup({ email: teacherEmail, password: PASSWORD })
  const course = await apiCreateCourse(teacherTok, { title: 'CC6 Alt Course' })
  await apiEnroll(teacherTok, course.courseCode, teacherEmail, 'teacher')

  const mod = await apiCreateModule(teacherTok, course.courseCode, 'Unit 1')
  const page = await apiCreateContentPage(teacherTok, course.courseCode, mod.id, 'Images')
  await apiPatchContentPage(teacherTok, course.courseCode, page.id, {
    markdown: '![has alt](https://example.com/a.png)\n![](https://example.com/b.png)\n',
  })

  async function refreshChecklist(token: string, courseCode: string): Promise<ChecklistResponse> {
    const res = await fetch(`${API_BASE}/api/v1/courses/${courseCode}/checklist/refresh`, {
      method: 'POST',
      headers: { Authorization: `Bearer ${token}` },
    })
    expect(res.status).toBe(200)
    return res.json() as Promise<ChecklistResponse>
  }

  let list = await refreshChecklist(teacherTok, course.courseCode)
  const item = findItem(list, 'a11y.image-alt-text')
  expect(item, 'a11y.image-alt-text should be present').toBeTruthy()
  expect(itemStatus(item)).toBe('in_progress')
  expect(item!.progress).toEqual({ done: 1, total: 2 })
  expect(item!.evidence?.rows?.length).toBe(1)

  await apiPatchContentPage(teacherTok, course.courseCode, page.id, {
    markdown: '![has alt](https://example.com/a.png)\n![also](https://example.com/b.png)\n',
  })
  list = await refreshChecklist(teacherTok, course.courseCode)
  expect(itemStatus(findItem(list, 'a11y.image-alt-text'))).toBe('done')
})

test('Course checklist CC.6: View as student flips launch.student-preview', async () => {
  const teacherEmail = uniqueEmail('cc6-preview-teacher')
  const { access_token: teacherTok } = await apiSignup({ email: teacherEmail, password: PASSWORD })
  const course = await apiCreateCourse(teacherTok, { title: 'CC6 Preview Course' })
  await apiEnroll(teacherTok, course.courseCode, teacherEmail, 'teacher')

  async function refreshChecklist(token: string, courseCode: string): Promise<ChecklistResponse> {
    const res = await fetch(`${API_BASE}/api/v1/courses/${courseCode}/checklist/refresh`, {
      method: 'POST',
      headers: { Authorization: `Bearer ${token}` },
    })
    expect(res.status).toBe(200)
    return res.json() as Promise<ChecklistResponse>
  }

  let list = await refreshChecklist(teacherTok, course.courseCode)
  expect(itemStatus(findItem(list, 'launch.student-preview'))).toBe('todo')

  const permRes = await fetch(
    `${API_BASE}/api/v1/me/permissions?courseCode=${encodeURIComponent(course.courseCode)}&viewAs=student`,
    { headers: { Authorization: `Bearer ${teacherTok}` } },
  )
  expect(permRes.status).toBe(200)

  list = await refreshChecklist(teacherTok, course.courseCode)
  expect(itemStatus(findItem(list, 'launch.student-preview'))).toBe('done')
})
