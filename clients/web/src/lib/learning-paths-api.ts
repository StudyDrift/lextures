import { apiUrl, authorizedFetch } from './api'

export type LearningPathSummary = {
  id: string
  title: string
  description: string
  slug: string
  bundlePriceCents?: number
  courseCount: number
  totalDurationMinutes: number
  individualTotalCents: number
  skillTags: string[]
}

export type LearningPathCourse = {
  courseId: string
  position: number
  courseCode: string
  title: string
  description?: string
  listPriceCents?: number
  durationMinutes: number
  skillTags: string[]
  completed?: boolean
  recommended?: boolean
}

export type LearningPathDetail = {
  path: {
    id: string
    title: string
    description: string
    slug?: string
    bundlePriceCents?: number
    isPublic: boolean
  }
  courses: LearningPathCourse[]
  totalDurationMinutes: number
  individualTotalCents: number
  skillTags: string[]
  slug: string
}

export type PathProgress = {
  pathId: string
  pathTitle: string
  slug?: string
  totalCourses: number
  completedCourses: number
  percent: number
  progressLabel: string
  completedAt?: string
  justCompleted: boolean
  courses: LearningPathCourse[]
}

export type CreatorLearningPath = {
  id: string
  creatorId: string
  title: string
  description: string
  slug?: string
  bundlePriceCents?: number
  isPublic: boolean
  courseIds: string[]
  createdAt: string
  updatedAt: string
}

export async function fetchCatalogPaths(q = '', sort = ''): Promise<LearningPathSummary[]> {
  const params = new URLSearchParams()
  if (q.trim()) params.set('q', q.trim())
  if (sort.trim()) params.set('sort', sort.trim())
  const qs = params.toString()
  const res = await fetch(apiUrl(`/api/v1/catalog/paths${qs ? `?${qs}` : ''}`))
  if (!res.ok) return []
  const data = (await res.json()) as { paths?: LearningPathSummary[] }
  return data.paths ?? []
}

export async function fetchCatalogPathDetail(slug: string): Promise<LearningPathDetail | null> {
  const res = await fetch(apiUrl(`/api/v1/catalog/paths/${encodeURIComponent(slug)}`))
  if (res.status === 404) return null
  if (!res.ok) throw new Error('Failed to load path')
  return (await res.json()) as LearningPathDetail
}

export async function fetchMyPaths(): Promise<PathProgress[]> {
  const res = await authorizedFetch('/api/v1/me/paths')
  if (!res.ok) throw new Error('Failed to load paths')
  const data = (await res.json()) as { paths?: PathProgress[] }
  return data.paths ?? []
}

export async function fetchPathProgress(pathId: string): Promise<PathProgress> {
  const res = await authorizedFetch(`/api/v1/me/paths/${encodeURIComponent(pathId)}/progress`)
  if (!res.ok) throw new Error('Failed to load path progress')
  return (await res.json()) as PathProgress
}

export async function enrollInPath(pathId: string): Promise<{ enrollmentId: string; progress: PathProgress }> {
  const res = await authorizedFetch(`/api/v1/paths/${encodeURIComponent(pathId)}/enroll`, {
    method: 'POST',
  })
  if (!res.ok) {
    const msg = await res.text()
    throw new Error(msg || 'Failed to enroll in path')
  }
  return (await res.json()) as { enrollmentId: string; progress: PathProgress }
}

export async function fetchCreatorLearningPaths(): Promise<CreatorLearningPath[]> {
  const res = await authorizedFetch('/api/v1/creator/learning-paths')
  if (!res.ok) throw new Error('Failed to load learning paths')
  const data = (await res.json()) as { paths?: CreatorLearningPath[] }
  return data.paths ?? []
}

export async function createLearningPath(body: {
  title: string
  description?: string
  courseIds: string[]
  bundlePriceCents?: number
  isPublic?: boolean
}): Promise<CreatorLearningPath> {
  const res = await authorizedFetch('/api/v1/creator/learning-paths', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  if (!res.ok) throw new Error('Failed to create learning path')
  return (await res.json()) as CreatorLearningPath
}

export async function updateLearningPath(
  id: string,
  body: {
    title?: string
    description?: string
    courseIds?: string[]
    bundlePriceCents?: number
    isPublic?: boolean
  },
): Promise<CreatorLearningPath> {
  const res = await authorizedFetch(`/api/v1/creator/learning-paths/${encodeURIComponent(id)}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  if (!res.ok) throw new Error('Failed to update learning path')
  return (await res.json()) as CreatorLearningPath
}

export async function deleteLearningPath(id: string): Promise<void> {
  const res = await authorizedFetch(`/api/v1/creator/learning-paths/${encodeURIComponent(id)}`, {
    method: 'DELETE',
  })
  if (!res.ok) throw new Error('Failed to delete learning path')
}

export function formatCents(cents: number, currency = 'USD'): string {
  return new Intl.NumberFormat(undefined, { style: 'currency', currency }).format(cents / 100)
}

export function formatDurationMinutes(minutes: number): string {
  if (minutes < 60) return `${minutes} min`
  const hours = Math.floor(minutes / 60)
  const rem = minutes % 60
  if (rem === 0) return `${hours} hr`
  return `${hours} hr ${rem} min`
}
