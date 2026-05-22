import { authorizedFetch } from './api'
import { readApiErrorMessage } from './errors'

export type ItemStat = {
  questionIndex: number
  questionText: string
  nResponses: number
  pValue: number | null
  rPb: number | null
  distractorFreqs: Record<string, number> | null
  flag: 'easy' | 'hard' | 'poor_discriminator' | null
}

export type TestStats = {
  nResponses: number
  kr20: number | null
  cronbachAlpha: number | null
  meanScore: number | null
  stdDev: number | null
  computedAt: string
}

export type ItemAnalysisResult =
  | { status: 'ok'; quizId: string; nResponses: number; testStats: TestStats; itemStats: ItemStat[] }
  | { status: 'insufficient'; quizId: string; nResponses: number; minimumRequired: number }
  | { status: 'pending'; quizId: string; nResponses: number }

async function parseJson(res: Response): Promise<unknown> {
  return res.json().catch(() => ({}))
}

/** GET `/api/v1/courses/:code/quizzes/:id/item-analysis` — instructor only. */
export async function fetchItemAnalysis(
  courseCode: string,
  itemId: string,
): Promise<ItemAnalysisResult> {
  const res = await authorizedFetch(
    `/api/v1/courses/${encodeURIComponent(courseCode)}/quizzes/${encodeURIComponent(itemId)}/item-analysis`,
  )
  const raw = (await parseJson(res)) as Record<string, unknown>
  if (!res.ok) throw new Error(readApiErrorMessage(raw))

  if (raw.insufficientData) {
    return {
      status: 'insufficient',
      quizId: raw.quizId as string,
      nResponses: raw.nResponses as number,
      minimumRequired: raw.minimumRequired as number,
    }
  }
  if (raw.statsPending) {
    return {
      status: 'pending',
      quizId: raw.quizId as string,
      nResponses: raw.nResponses as number,
    }
  }
  return {
    status: 'ok',
    quizId: raw.quizId as string,
    nResponses: raw.nResponses as number,
    testStats: raw.testStats as TestStats,
    itemStats: (raw.itemStats as ItemStat[]) ?? [],
  }
}

/** POST `/api/v1/courses/:code/quizzes/:id/item-analysis/compute` — triggers re-computation. */
export async function computeItemAnalysis(
  courseCode: string,
  itemId: string,
): Promise<ItemAnalysisResult> {
  const res = await authorizedFetch(
    `/api/v1/courses/${encodeURIComponent(courseCode)}/quizzes/${encodeURIComponent(itemId)}/item-analysis/compute`,
    { method: 'POST' },
  )
  const raw = (await parseJson(res)) as Record<string, unknown>
  if (!res.ok) throw new Error(readApiErrorMessage(raw))

  if (raw.insufficientData) {
    return {
      status: 'insufficient',
      quizId: raw.quizId as string,
      nResponses: raw.nResponses as number,
      minimumRequired: raw.minimumRequired as number,
    }
  }
  return {
    status: 'ok',
    quizId: raw.quizId as string,
    nResponses: raw.nResponses as number,
    testStats: raw.testStats as TestStats,
    itemStats: (raw.itemStats as ItemStat[]) ?? [],
  }
}

/** Returns the export CSV URL for direct download. */
export function itemAnalysisExportUrl(courseCode: string, itemId: string): string {
  return `/api/v1/courses/${encodeURIComponent(courseCode)}/quizzes/${encodeURIComponent(itemId)}/item-analysis/export.csv`
}
