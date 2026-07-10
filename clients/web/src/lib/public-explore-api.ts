// Unified public explore client: prefers the SEO catalog (plan 15.1) and falls back to
// the public marketplace API (MKT7) when the catalog feature is disabled.

import type { ReviewsListResponse } from './course-reviews-api'
import {
  fetchPublicCatalogCategories,
  fetchPublicCatalogCourse,
  searchPublicCatalog,
  type PublicCatalogCategory,
  type PublicCatalogCourse,
  type PublicCatalogCourseDetail,
  type PublicCatalogQuery,
  type PublicCatalogSearchResponse,
} from './public-catalog-api'
import {
  fetchPublicMarketplaceCategories,
  fetchPublicMarketplaceCourse,
  searchPublicMarketplace,
  type PublicMarketplaceCourse,
  type PublicMarketplaceQuery,
} from './public-marketplace-api'

function isCatalogUnavailable(error: unknown): boolean {
  return error instanceof Error && error.message === 'Course catalog is not available.'
}

function marketplaceCourseToCatalog(course: PublicMarketplaceCourse): PublicCatalogCourse {
  return {
    id: course.id,
    slug: course.slug,
    courseCode: course.courseCode,
    title: course.title,
    description: course.description,
    heroImageUrl: course.heroImageUrl,
    category: course.category,
    difficultyLevel: course.level,
    language: course.language,
    priceCents: course.priceCents,
    enrollmentCount: course.enrollmentCount,
    averageRating: course.averageRating,
    ratingCount: course.ratingCount,
    instructorName: course.instructorName,
    createdAt: course.createdAt,
  }
}

function toCatalogQuery(query: PublicCatalogQuery): PublicMarketplaceQuery {
  return {
    q: query.q,
    category: query.category,
    level: query.level,
    language: query.language,
    priceMax: query.priceMax,
    freeOnly: query.priceMax === 0,
    sort: query.sort,
    cursor: query.cursor,
    limit: query.limit,
  }
}

export async function searchExploreCatalog(
  query: PublicCatalogQuery,
): Promise<PublicCatalogSearchResponse> {
  try {
    return await searchPublicCatalog(query)
  } catch (error) {
    if (!isCatalogUnavailable(error)) throw error
    const res = await searchPublicMarketplace(toCatalogQuery(query))
    return {
      courses: res.courses.map(marketplaceCourseToCatalog),
      total: res.total,
      nextCursor: res.nextCursor,
    }
  }
}

export async function fetchExploreCatalogCategories(): Promise<PublicCatalogCategory[]> {
  try {
    return await fetchPublicCatalogCategories()
  } catch (error) {
    if (!isCatalogUnavailable(error)) throw error
    return fetchPublicMarketplaceCategories()
  }
}

export async function fetchExploreCourse(slug: string): Promise<PublicCatalogCourseDetail> {
  try {
    return await fetchPublicCatalogCourse(slug)
  } catch (error) {
    if (!isCatalogUnavailable(error)) throw error
    const detail = await fetchPublicMarketplaceCourse(slug)
    return {
      course: marketplaceCourseToCatalog(detail.course),
      jsonLd: detail.jsonLd ?? {},
    }
  }
}

export async function fetchExploreCourseReviews(
  slug: string,
  cursor?: string,
): Promise<ReviewsListResponse> {
  const params = new URLSearchParams()
  if (cursor) params.set('cursor', cursor)
  const qs = params.toString()

  const catalogUrl = `/api/v1/public/catalog/courses/${encodeURIComponent(slug)}/reviews${qs ? `?${qs}` : ''}`
  const catalogRes = await fetch(catalogUrl)
  if (catalogRes.ok) {
    return (await catalogRes.json()) as ReviewsListResponse
  }
  if (catalogRes.status !== 404) {
    throw new Error('Failed to load reviews.')
  }

  const marketplaceUrl = `/api/v1/public/marketplace/courses/${encodeURIComponent(slug)}/reviews${qs ? `?${qs}` : ''}`
  const marketplaceRes = await fetch(marketplaceUrl)
  if (!marketplaceRes.ok) {
    throw new Error('Failed to load reviews.')
  }
  return (await marketplaceRes.json()) as ReviewsListResponse
}
