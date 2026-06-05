import { authorizedFetch } from './api'

export type HallPassStatus = 'requested' | 'approved' | 'returned' | 'denied'

export type HallPassDestination = 'bathroom' | 'office' | 'library' | 'nurse' | 'other'

export const HALL_PASS_DESTINATIONS: HallPassDestination[] = [
  'bathroom',
  'office',
  'library',
  'nurse',
  'other',
]

export type HallPass = {
  id: string
  studentId?: string
  sectionId: string
  destination: HallPassDestination | string
  estimatedMins: number | null
  status: HallPassStatus
  requestedAt: string
  approvedAt: string | null
  returnedAt: string | null
  approvedBy: string | null
  overdue: boolean
}

export type AnonymousQuestion = {
  id: string
  courseId: string
  question: string
  addressed: boolean
  createdAt: string
  authorId?: string // only present on teacher routes
}

async function readError(res: Response, fallback: string): Promise<string> {
  try {
    const data = (await res.json()) as { error?: { message?: string } }
    return data.error?.message ?? fallback
  } catch {
    return fallback
  }
}

export async function requestHallPass(
  sectionId: string,
  destination: HallPassDestination,
  estimatedMins?: number,
): Promise<HallPass> {
  const res = await authorizedFetch(
    `/api/v1/sections/${encodeURIComponent(sectionId)}/hall-passes`,
    {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ destination, estimatedMins }),
    },
  )
  if (!res.ok) throw new Error(await readError(res, 'Failed to request hall pass'))
  const data = (await res.json()) as { pass: HallPass }
  return data.pass
}

export async function listActiveHallPasses(sectionId: string): Promise<HallPass[]> {
  const res = await authorizedFetch(
    `/api/v1/sections/${encodeURIComponent(sectionId)}/hall-passes/active`,
  )
  if (!res.ok) return []
  const data = (await res.json()) as { passes: HallPass[] }
  return data.passes ?? []
}

export async function updateHallPassStatus(
  passId: string,
  status: 'approved' | 'denied' | 'returned',
): Promise<HallPass> {
  const res = await authorizedFetch(`/api/v1/hall-passes/${encodeURIComponent(passId)}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ status }),
  })
  if (!res.ok) throw new Error(await readError(res, 'Failed to update hall pass'))
  const data = (await res.json()) as { pass: HallPass }
  return data.pass
}

export async function submitAnonymousQuestion(
  courseId: string,
  question: string,
): Promise<AnonymousQuestion> {
  const res = await authorizedFetch(
    `/api/v1/courses/${encodeURIComponent(courseId)}/questions`,
    {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ question }),
    },
  )
  if (!res.ok) throw new Error(await readError(res, 'Failed to submit question'))
  const data = (await res.json()) as { question: AnonymousQuestion }
  return data.question
}

export async function listCourseQuestions(
  courseId: string,
  includeAddressed = false,
): Promise<AnonymousQuestion[]> {
  const qs = includeAddressed ? '?includeAddressed=true' : ''
  const res = await authorizedFetch(
    `/api/v1/courses/${encodeURIComponent(courseId)}/questions${qs}`,
  )
  if (!res.ok) return []
  const data = (await res.json()) as { questions: AnonymousQuestion[] }
  return data.questions ?? []
}

export async function markQuestionAddressed(
  courseId: string,
  questionId: string,
): Promise<void> {
  const res = await authorizedFetch(
    `/api/v1/courses/${encodeURIComponent(courseId)}/questions/${encodeURIComponent(questionId)}`,
    { method: 'PATCH' },
  )
  if (!res.ok) throw new Error(await readError(res, 'Failed to update question'))
}
