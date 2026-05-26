import { authorizedFetch } from './api'

export type SignalItem = {
  itemId: string
  title: string
  kind: string
  completionRate: number
  avgScore: number | null
  engagement: number
  difficulty: number | null
  compositeScore: number
  narrative: string
}

export type ScatterPoint = {
  itemId: string
  title: string
  kind: string
  difficulty: number
  engagement: number
  flag: string
}

export type Insights = {
  courseId: string
  weekOf: string
  generatedAt: string
  workingWell: SignalItem[]
  needsAttention: SignalItem[]
  scatter: ScatterPoint[]
}

export type CrossSectionRow = {
  sectionId: string
  sectionName: string
  nStudents: number
  avgQuizScore: number | null
  completionRate: number
}

export async function fetchInsights(courseCode: string): Promise<Insights> {
  const res = await authorizedFetch(
    `/api/v1/courses/${encodeURIComponent(courseCode)}/analytics/insights`,
  )
  if (!res.ok) throw new Error(`${res.status}`)
  return res.json() as Promise<Insights>
}

export async function refreshInsights(courseCode: string): Promise<Insights> {
  const res = await authorizedFetch(
    `/api/v1/courses/${encodeURIComponent(courseCode)}/analytics/insights/refresh`,
    { method: 'POST' },
  )
  if (!res.ok) throw new Error(`${res.status}`)
  return res.json() as Promise<Insights>
}

export async function dismissSignal(
  courseCode: string,
  signalKey: string,
  reason: string,
): Promise<void> {
  const res = await authorizedFetch(
    `/api/v1/courses/${encodeURIComponent(courseCode)}/analytics/insights/dismiss`,
    {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ signalKey, reason }),
    },
  )
  if (!res.ok) throw new Error(`${res.status}`)
}

export async function fetchCrossSection(courseCode: string): Promise<CrossSectionRow[]> {
  const res = await authorizedFetch(
    `/api/v1/courses/${encodeURIComponent(courseCode)}/analytics/cross-section`,
  )
  if (!res.ok) throw new Error(`${res.status}`)
  return res.json() as Promise<CrossSectionRow[]>
}
