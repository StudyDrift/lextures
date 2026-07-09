/** Authenticated in-app course marketplace API client (plan MKT3). */

import { authorizedFetch } from './api'
import { readApiErrorMessage } from './errors'
import { formatMarketplacePrice } from './marketplace-price'

export type MarketplaceCard = {
  id: string
  slug: string
  courseCode: string
  title: string
  description?: string
  heroImageUrl: string | null
  category: string | null
  level: string | null
  language: string
  priceCents: number
  priceCurrency: string
  listPriceCents: number | null
  enrollmentCount: number
  averageRating: number | null
  ratingCount?: number
  instructorName?: string | null
  createdAt?: string
  owned: boolean
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

export type MarketplaceCourseDetail = {
  course: MarketplaceCard
  owned: boolean
  priceCents: number
  priceCurrency: string
  listPriceCents?: number | null
  whatsIncluded: MarketplaceWhatsIncluded
  rating: { average: number | null; count: number }
}

export type MarketplaceSearchResponse = {
  courses: MarketplaceCard[]
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

export function buildMarketplaceParams(query: MarketplaceQuery): string {
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

async function jsonOrThrow<T>(res: Response): Promise<T> {
  const raw: unknown = await res.json().catch(() => ({}))
  if (!res.ok) {
    if (res.status === 404) {
      throw new Error(readApiErrorMessage(raw) || 'Marketplace is not available.')
    }
    if (res.status === 401) {
      throw new Error('Sign in required.')
    }
    throw new Error(readApiErrorMessage(raw) || 'Failed to load the marketplace.')
  }
  return raw as T
}

export async function searchMarketplaceCourses(
  query: MarketplaceQuery = {},
): Promise<MarketplaceSearchResponse> {
  const res = await authorizedFetch(`/api/v1/marketplace/courses${buildMarketplaceParams(query)}`)
  const data = await jsonOrThrow<MarketplaceSearchResponse>(res)
  return {
    courses: data.courses ?? [],
    total: data.total ?? 0,
    nextCursor: data.nextCursor ?? '',
  }
}

export async function fetchMarketplaceCategories(): Promise<MarketplaceCategory[]> {
  const res = await authorizedFetch('/api/v1/marketplace/categories')
  const data = await jsonOrThrow<{ categories: MarketplaceCategory[] }>(res)
  return data.categories ?? []
}

export async function fetchMarketplaceCourse(slug: string): Promise<MarketplaceCourseDetail> {
  const res = await authorizedFetch(`/api/v1/marketplace/courses/${encodeURIComponent(slug)}`)
  return jsonOrThrow<MarketplaceCourseDetail>(res)
}

export type MarketplaceClaimResult = {
  enrolled: boolean
  entitlementId: string
  alreadyOwned?: boolean
  firstItemId?: string
  courseCode: string
}

export type MarketplaceCheckoutResult =
  | { checkoutUrl: string; sessionId: string; alreadyOwned?: false }
  | { alreadyOwned: true; courseCode: string; courseId: string; checkoutUrl?: never }

export class MarketplaceApiError extends Error {
  readonly status: number
  readonly code?: string
  readonly checkoutHint?: string

  constructor(status: number, message: string, code?: string, checkoutHint?: string) {
    super(message)
    this.name = 'MarketplaceApiError'
    this.status = status
    this.code = code
    this.checkoutHint = checkoutHint
  }
}

async function marketplaceMutationOrThrow<T>(res: Response): Promise<T> {
  const raw: unknown = await res.json().catch(() => ({}))
  if (!res.ok) {
    const message = readApiErrorMessage(raw) || 'Request failed'
    const code =
      raw && typeof raw === 'object' && 'error' in raw
        ? (raw as { error?: { code?: string } }).error?.code
        : undefined
    const checkoutHint =
      raw && typeof raw === 'object' && 'checkoutHint' in raw
        ? String((raw as { checkoutHint?: unknown }).checkoutHint ?? '')
        : undefined
    throw new MarketplaceApiError(res.status, message, code, checkoutHint || undefined)
  }
  return raw as T
}

/** Free claim: entitlement + enrollment (plan MKT4). */
export async function claimMarketplaceCourse(slug: string): Promise<MarketplaceClaimResult> {
  const res = await authorizedFetch(`/api/v1/marketplace/courses/${encodeURIComponent(slug)}/claim`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: '{}',
  })
  return marketplaceMutationOrThrow<MarketplaceClaimResult>(res)
}

/** Paid checkout: returns Stripe Checkout URL (plan MKT4). */
export async function checkoutMarketplaceCourse(slug: string): Promise<MarketplaceCheckoutResult> {
  const res = await authorizedFetch(
    `/api/v1/marketplace/courses/${encodeURIComponent(slug)}/checkout`,
    {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: '{}',
    },
  )
  return marketplaceMutationOrThrow<MarketplaceCheckoutResult>(res)
}

/** Accessible name for a marketplace card link (title + price/Free + owned). */
export function marketplaceCardAccessibleName(
  course: Pick<MarketplaceCard, 'title' | 'priceCents' | 'priceCurrency' | 'owned'>,
  freeLabel = 'Free',
  ownedLabel = 'Owned',
  locale?: string,
): string {
  const price = formatMarketplacePrice(course.priceCents, course.priceCurrency, locale, freeLabel)
  if (course.owned) return `${course.title}, ${ownedLabel}, ${price}`
  return `${course.title}, ${price}`
}

/** Client routes that run claim / checkout (plan MKT4). */
export function marketplaceClaimPath(slug: string): string {
  return `/marketplace/${encodeURIComponent(slug)}/claim`
}

export function marketplaceCheckoutPath(slug: string): string {
  return `/marketplace/${encodeURIComponent(slug)}/checkout`
}

export function marketplaceCoursePath(courseCode: string): string {
  return `/courses/${encodeURIComponent(courseCode)}`
}

/** After claim/purchase, land on the course home (item deep-links vary by kind). */
export function marketplaceCourseItemPath(courseCode: string, _itemId?: string | null): string {
  return marketplaceCoursePath(courseCode)
}
