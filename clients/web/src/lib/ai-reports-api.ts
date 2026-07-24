import { authorizedFetch } from './api'
import { readApiErrorMessage } from './errors'

export type AIReportsPayload = {
  range: { from: string; to: string }
  providers: string[]
  cost: {
    summary: {
      totalCostUsd: number
      totalCalls: number
      totalTokens: number
    }
    byDay: { day: string; costUsd: number; calls: number; tokens: number }[]
    byFeature: { feature: string; costUsd: number; calls: number; tokens: number }[]
    byProvider: { provider: string; costUsd: number; calls: number; tokens: number }[]
  }
  byUser: {
    userId: string
    email: string
    displayName: string
    calls: number
    promptTokens: number
    completionTokens: number
    totalTokens: number
    costUsd: number
  }[]
  byCourse: {
    courseId: string
    courseCode: string
    title: string
    calls: number
    totalTokens: number
    costUsd: number
  }[]
}

export type AIReportsQuery = {
  from?: string
  to?: string
  feature?: string
  provider?: string
  userQuery?: string
  courseCode?: string
}

/** GET `/api/v1/settings/ai/reports` — requires global admin (PERM_RBAC_MANAGE). */
export async function fetchAIReports(params: AIReportsQuery): Promise<AIReportsPayload> {
  const search = new URLSearchParams()
  if (params.from) search.set('from', params.from)
  if (params.to) search.set('to', params.to)
  if (params.feature) search.set('feature', params.feature)
  if (params.provider) search.set('provider', params.provider)
  if (params.userQuery) search.set('userQuery', params.userQuery)
  if (params.courseCode) search.set('courseCode', params.courseCode)
  const qs = search.toString()
  const res = await authorizedFetch(`/api/v1/settings/ai/reports${qs ? `?${qs}` : ''}`)
  const raw: unknown = await res.json().catch(() => ({}))
  if (!res.ok) throw new Error(readApiErrorMessage(raw))
  return raw as AIReportsPayload
}

export type AIReportsPreset = '24h' | '7d' | '30d' | '90d'

export function aiReportsUtcRange(preset: AIReportsPreset): { from: string; to: string } {
  const to = new Date()
  const from = new Date(to)
  const hours =
    preset === '24h' ? 24 : preset === '7d' ? 7 * 24 : preset === '30d' ? 30 * 24 : 90 * 24
  from.setTime(from.getTime() - hours * 60 * 60 * 1000)
  return { from: from.toISOString(), to: to.toISOString() }
}

export const AI_FEATURE_LABELS: Record<string, string> = {
  ai_tutor: 'AI Tutor',
  modules_ai_assistant: 'Modules AI assistant',
  rag_notebook: 'Notebook AI',
  syllabus_generation: 'Syllabus generation',
  outcomes_extraction: 'Learning outcomes extraction',
  badges_extraction: 'Badge extraction',
  translation: 'Translation',
  quiz_generation: 'Quiz generation',
  reading_level_simplification: 'Reading level',
  content_translation: 'Content translation',
  alt_text_suggestion: 'Alt text',
  vibe_generation: 'Vibe activities',
  grader_agent: 'Grading agent',
  lesson_generation: 'Lesson generator',
  ai_study_buddy: 'AI study buddy',
  unknown: 'Unknown',
}

export function aiFeatureLabel(feature: string): string {
  return AI_FEATURE_LABELS[feature] ?? feature.replaceAll('_', ' ')
}
