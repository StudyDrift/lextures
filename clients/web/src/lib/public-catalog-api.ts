// Public, unauthenticated course catalog API client (plan 15.1).
// These endpoints are intentionally called with a bare fetch (no auth header) so
// the catalog renders for logged-out visitors.

export type PublicCatalogCourse = {
  id: string
  slug: string
  courseCode: string
  title: string
  description: string
  heroImageUrl: string | null
  category: string | null
  difficultyLevel: string | null
  language: string
  priceCents: number
  enrollmentCount: number
  averageRating: number | null
  instructorName: string | null
  createdAt: string
}

export type PublicCatalogSearchResponse = {
  courses: PublicCatalogCourse[]
  total: number
  nextCursor: string
}

export type PublicCatalogCategory = {
  category: string
  count: number
}

export type PublicCatalogCourseDetail = {
  course: PublicCatalogCourse
  jsonLd: Record<string, unknown>
}

export type PublicCatalogQuery = {
  q?: string
  category?: string
  level?: string
  language?: string
  priceMax?: number
  sort?: string
  cursor?: string
  limit?: number
}

function buildParams(query: PublicCatalogQuery): string {
  const params = new URLSearchParams()
  if (query.q) params.set('q', query.q)
  if (query.category) params.set('category', query.category)
  if (query.level) params.set('level', query.level)
  if (query.language) params.set('language', query.language)
  if (typeof query.priceMax === 'number') params.set('price_max', String(query.priceMax))
  if (query.sort) params.set('sort', query.sort)
  if (query.cursor) params.set('cursor', query.cursor)
  if (typeof query.limit === 'number') params.set('limit', String(query.limit))
  const s = params.toString()
  return s ? `?${s}` : ''
}

async function jsonOrThrow<T>(res: Response): Promise<T> {
  if (!res.ok) {
    if (res.status === 404) {
      throw new Error('Course catalog is not available.')
    }
    throw new Error('Failed to load the course catalog.')
  }
  return (await res.json()) as T
}

export async function searchPublicCatalog(
  query: PublicCatalogQuery,
): Promise<PublicCatalogSearchResponse> {
  const res = await fetch(`/api/v1/public/catalog/courses${buildParams(query)}`)
  return jsonOrThrow<PublicCatalogSearchResponse>(res)
}

export async function fetchPublicCatalogCategories(): Promise<PublicCatalogCategory[]> {
  const res = await fetch('/api/v1/public/catalog/categories')
  const data = await jsonOrThrow<{ categories: PublicCatalogCategory[] }>(res)
  return data.categories ?? []
}

export async function fetchPublicCatalogCourse(slug: string): Promise<PublicCatalogCourseDetail> {
  const res = await fetch(`/api/v1/public/catalog/courses/${encodeURIComponent(slug)}`)
  return jsonOrThrow<PublicCatalogCourseDetail>(res)
}

export function formatPrice(priceCents: number): string {
  if (priceCents <= 0) return 'Free'
  return `$${(priceCents / 100).toFixed(2)}`
}
