/** Public www marketplace API client (plans MKT7–MKT10). */

import { API_BASE } from './api-base'
import { APP_ORIGIN } from './site-links'

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

export type MarketplaceCategory = {
  category: string
  count: number
}

export type MarketplaceWhatsIncluded = {
  moduleCount: number
  itemCount: number
  estimatedDurationMinutes?: number
}

export type PublicMarketplaceCourseDetail = {
  course: PublicMarketplaceCourse
  whatsIncluded: MarketplaceWhatsIncluded
  jsonLd?: Record<string, unknown>
}

export type MarketplaceSearchResponse = {
  courses: PublicMarketplaceCourse[]
  total: number
  nextCursor: string
}

export type MarketplaceQuery = {
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

export type CourseReview = {
  id: string
  rating: number
  reviewText?: string
  reviewerDisplayName: string
  createdAt: string
}

export type CourseReviewsResponse = {
  summary?: { average: number | null; count: number }
  reviews: CourseReview[]
  nextCursor?: string
}

export class MarketplaceApiError extends Error {
  status: number
  constructor(status: number, message: string) {
    super(message)
    this.name = 'MarketplaceApiError'
    this.status = status
  }
}

function requireMarketplaceSlug(slug: string): string {
  const trimmed = slug.trim()
  if (!trimmed) {
    throw new MarketplaceApiError(404, 'Course not found')
  }
  return trimmed
}

export function buildMarketplaceParams(query: MarketplaceQuery = {}): string {
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

async function fetchJSON<T>(path: string): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, {
    headers: { Accept: 'application/json' },
  })
  const raw: unknown = await res.json().catch(() => ({}))
  if (!res.ok) {
    const msg =
      typeof raw === 'object' && raw && 'message' in raw && typeof (raw as { message: unknown }).message === 'string'
        ? (raw as { message: string }).message
        : `Request failed (${res.status})`
    throw new MarketplaceApiError(res.status, msg)
  }
  return raw as T
}

export async function searchPublicMarketplaceCourses(
  query: MarketplaceQuery = {},
): Promise<MarketplaceSearchResponse> {
  const data = await fetchJSON<MarketplaceSearchResponse>(
    `/api/v1/public/marketplace/courses${buildMarketplaceParams(query)}`,
  )
  return {
    courses: data.courses ?? [],
    total: data.total ?? 0,
    nextCursor: data.nextCursor ?? '',
  }
}

export async function fetchPublicMarketplaceCategories(): Promise<MarketplaceCategory[]> {
  const data = await fetchJSON<{ categories: MarketplaceCategory[] }>(
    '/api/v1/public/marketplace/categories',
  )
  return data.categories ?? []
}

export async function fetchPublicMarketplaceCourse(
  slug: string,
): Promise<PublicMarketplaceCourseDetail> {
  const normalized = requireMarketplaceSlug(slug)
  return fetchJSON<PublicMarketplaceCourseDetail>(
    `/api/v1/public/marketplace/courses/${encodeURIComponent(normalized)}`,
  )
}

export async function fetchPublicMarketplaceReviews(
  slug: string,
  opts: { cursor?: string; limit?: number } = {},
): Promise<CourseReviewsResponse> {
  const normalized = requireMarketplaceSlug(slug)
  const params = new URLSearchParams()
  if (opts.cursor) params.set('cursor', opts.cursor)
  if (typeof opts.limit === 'number') params.set('limit', String(opts.limit))
  const qs = params.toString()
  const data = await fetchJSON<CourseReviewsResponse>(
    `/api/v1/public/marketplace/courses/${encodeURIComponent(normalized)}/reviews${qs ? `?${qs}` : ''}`,
  )
  return {
    summary: data.summary,
    reviews: data.reviews ?? [],
    nextCursor: data.nextCursor,
  }
}

const ZERO_DECIMAL_CURRENCIES = new Set(['jpy'])

function minorUnitFactor(currency: string): number {
  return ZERO_DECIMAL_CURRENCIES.has(currency.toLowerCase().trim()) ? 1 : 100
}

export function formatMarketplacePrice(
  priceCents: number,
  currency: string,
  freeLabel = 'Free',
): string {
  if (priceCents <= 0) return freeLabel
  const major = priceCents / minorUnitFactor(currency)
  try {
    return new Intl.NumberFormat(undefined, {
      style: 'currency',
      currency: currency.toUpperCase(),
    }).format(major)
  } catch {
    return ZERO_DECIMAL_CURRENCIES.has(currency.toLowerCase())
      ? `${currency.toUpperCase()} ${Math.round(major)}`
      : `${currency.toUpperCase()} ${major.toFixed(2)}`
  }
}

export function enrollHandoffUrl(slug: string): string {
  return `${APP_ORIGIN}/explore/${encodeURIComponent(slug)}?ref=www-courses`
}

export function courseCardAccessibleName(
  course: Pick<PublicMarketplaceCourse, 'title' | 'priceCents' | 'priceCurrency'>,
  freeLabel = 'Free',
): string {
  const price = formatMarketplacePrice(course.priceCents, course.priceCurrency, freeLabel)
  return `${course.title}, ${price}`
}

export function truncateDescription(text: string, maxLen = 160): string {
  const cleaned = text.replace(/\s+/g, ' ').trim()
  if (cleaned.length <= maxLen) return cleaned
  const cut = cleaned.slice(0, maxLen - 1)
  const lastSpace = cut.lastIndexOf(' ')
  return `${(lastSpace > 40 ? cut.slice(0, lastSpace) : cut).trimEnd()}…`
}
