import { authorizedFetch } from './api'
import { readApiErrorMessage } from './errors'

export type CourseCatalogListing = {
  isPublic: boolean
  category: string | null
  difficultyLevel: string | null
  language: string
  priceCents: number
  priceCurrency: string
  slug: string
  marketplaceListed: boolean
  publishState: 'draft' | 'published'
  activePurchaseCount: number
}

export type CourseCatalogListingPatch = {
  isPublic?: boolean
  category?: string | null
  difficultyLevel?: string | null
  language?: string
  priceCents?: number
  priceCurrency?: string
  slug?: string
  marketplaceListed?: boolean
}

export async function fetchCourseCatalogListing(courseCode: string): Promise<CourseCatalogListing> {
  const res = await authorizedFetch(`/api/v1/courses/${encodeURIComponent(courseCode)}/catalog-listing`)
  const raw: unknown = await res.json().catch(() => ({}))
  if (!res.ok) throw new Error(readApiErrorMessage(raw))
  const data = raw as { listing?: CourseCatalogListing }
  if (!data.listing) throw new Error('Catalog listing response was missing.')
  return data.listing
}

export async function putCourseCatalogListing(
  courseCode: string,
  patch: CourseCatalogListingPatch,
  existing?: CourseCatalogListing,
): Promise<CourseCatalogListing> {
  const body: Record<string, unknown> = {
    isPublic: patch.isPublic ?? existing?.isPublic ?? false,
    category: patch.category !== undefined ? patch.category : (existing?.category ?? null),
    difficultyLevel:
      patch.difficultyLevel !== undefined ? patch.difficultyLevel : (existing?.difficultyLevel ?? null),
    language: patch.language ?? existing?.language ?? 'en',
    priceCents: patch.priceCents ?? existing?.priceCents ?? 0,
    priceCurrency: patch.priceCurrency ?? existing?.priceCurrency ?? 'usd',
    slug: patch.slug ?? existing?.slug ?? '',
  }
  if (patch.marketplaceListed !== undefined) {
    body.marketplaceListed = patch.marketplaceListed
  }
  const res = await authorizedFetch(`/api/v1/courses/${encodeURIComponent(courseCode)}/catalog-listing`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  const raw: unknown = await res.json().catch(() => ({}))
  if (!res.ok) throw new Error(readApiErrorMessage(raw))
  const data = raw as { listing?: CourseCatalogListing }
  if (!data.listing) throw new Error('Catalog listing response was missing.')
  return data.listing
}
