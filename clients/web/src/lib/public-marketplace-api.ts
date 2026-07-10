// Public, unauthenticated marketplace API client (plans MKT7–MKT10).
// Used when the SEO public catalog (plan 15.1) is disabled but marketplace listing is on.

export type PublicMarketplaceCourse = {
  id: string
  slug: string
  courseCode: string
  title: string
  description: string
  heroImageUrl: string | null
  category: string | null
  level: string | null
  language: string
  priceCents: number
  priceCurrency: string
  listPriceCents: number | null
  enrollmentCount: number
  averageRating: number | null
  ratingCount: number
  instructorName: string | null
  createdAt: string
}

export type PublicMarketplaceCategory = {
  category: string
  count: number
}

export type PublicMarketplaceWhatsIncluded = {
  moduleCount: number
  itemCount: number
  estimatedDurationMinutes?: number
}

export type PublicMarketplaceCourseDetail = {
  course: PublicMarketplaceCourse
  whatsIncluded: PublicMarketplaceWhatsIncluded
  jsonLd?: Record<string, unknown>
}

export type PublicMarketplaceSearchResponse = {
  courses: PublicMarketplaceCourse[]
  total: number
  nextCursor: string
}

export type PublicMarketplaceQuery = {
  q?: string
  category?: string
  level?: string
  language?: string
  priceMax?: number
  freeOnly?: boolean
  sort?: string
  cursor?: string
  limit?: number
}

function buildParams(query: PublicMarketplaceQuery): string {
  const params = new URLSearchParams()
  if (query.q) params.set('q', query.q)
  if (query.category) params.set('category', query.category)
  if (query.level) params.set('level', query.level)
  if (query.language) params.set('language', query.language)
  if (typeof query.priceMax === 'number') params.set('price_max', String(query.priceMax))
  if (query.freeOnly) params.set('free_only', 'true')
  if (query.sort) params.set('sort', query.sort)
  if (query.cursor) params.set('cursor', query.cursor)
  if (typeof query.limit === 'number') params.set('limit', String(query.limit))
  const s = params.toString()
  return s ? `?${s}` : ''
}

async function jsonOrThrow<T>(res: Response, notFoundMessage: string): Promise<T> {
  if (!res.ok) {
    if (res.status === 404) {
      throw new Error(notFoundMessage)
    }
    throw new Error('Failed to load the course marketplace.')
  }
  return (await res.json()) as T
}

export async function searchPublicMarketplace(
  query: PublicMarketplaceQuery,
): Promise<PublicMarketplaceSearchResponse> {
  const res = await fetch(`/api/v1/public/marketplace/courses${buildParams(query)}`)
  const data = await jsonOrThrow<PublicMarketplaceSearchResponse>(
    res,
    'Course marketplace is not available.',
  )
  return {
    courses: data.courses ?? [],
    total: data.total ?? 0,
    nextCursor: data.nextCursor ?? '',
  }
}

export async function fetchPublicMarketplaceCategories(): Promise<PublicMarketplaceCategory[]> {
  const res = await fetch('/api/v1/public/marketplace/categories')
  const data = await jsonOrThrow<{ categories: PublicMarketplaceCategory[] }>(
    res,
    'Course marketplace is not available.',
  )
  return data.categories ?? []
}

export async function fetchPublicMarketplaceCourse(
  slug: string,
): Promise<PublicMarketplaceCourseDetail> {
  const res = await fetch(`/api/v1/public/marketplace/courses/${encodeURIComponent(slug)}`)
  return jsonOrThrow<PublicMarketplaceCourseDetail>(res, 'Course not found.')
}
