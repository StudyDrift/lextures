import { authorizedFetch } from './api'
import { readApiErrorMessage } from './errors'

export type QuizScoreBucket = {
  label: string
  min: number
  max: number
  count: number
}

export type QuizQuestionStat = {
  questionIndex: number
  questionText: string
  nResponses: number
  pctCorrect: number
}

export type QuizAnalyticsReport = {
  quizId: string
  nAttempts: number
  meanScore: number | null
  scoreBuckets: QuizScoreBucket[]
  questionStats: QuizQuestionStat[]
}

/** GET `/api/v1/courses/:code/quizzes/:id/analytics` — instructor only. */
export async function fetchQuizAnalytics(
  courseCode: string,
  itemId: string,
): Promise<QuizAnalyticsReport> {
  const res = await authorizedFetch(
    `/api/v1/courses/${encodeURIComponent(courseCode)}/quizzes/${encodeURIComponent(itemId)}/analytics`,
  )
  const raw = (await res.json().catch(() => ({}))) as Record<string, unknown>
  if (!res.ok) throw new Error(readApiErrorMessage(raw))

  return {
    quizId: raw.quizId as string,
    nAttempts: raw.nAttempts as number,
    meanScore: (raw.meanScore as number | null | undefined) ?? null,
    scoreBuckets: (raw.scoreBuckets as QuizScoreBucket[]) ?? [],
    questionStats: (raw.questionStats as QuizQuestionStat[]) ?? [],
  }
}
