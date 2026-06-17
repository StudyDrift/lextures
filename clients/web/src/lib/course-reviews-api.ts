// Course reviews API client (plan 15.7).

import { authorizedFetch } from './api'

export type ReviewSummary = {
  averageRating?: number
  ratingCount: number
  distribution: Record<string, number>
}

export type CourseReview = {
  id: string
  courseId: string
  reviewerId: string
  rating: number
  reviewText?: string
  creatorResponse?: string
  reviewerDisplayName: string
  isFlagged: boolean
  createdAt: string
  updatedAt: string
}

export type ReviewsListResponse = {
  summary: ReviewSummary
  reviews: CourseReview[]
  nextCursor?: string
}

export type ReviewEligibility = {
  eligible: boolean
  progressPercent: number
  hasReview: boolean
  canEdit: boolean
  reviewId?: string
}

export async function fetchPublicCourseReviews(
  slug: string,
  cursor?: string,
): Promise<ReviewsListResponse> {
  const params = new URLSearchParams()
  if (cursor) params.set('cursor', cursor)
  const qs = params.toString()
  const res = await fetch(
    `/api/v1/public/catalog/courses/${encodeURIComponent(slug)}/reviews${qs ? `?${qs}` : ''}`,
  )
  if (!res.ok) throw new Error('Failed to load reviews.')
  return (await res.json()) as ReviewsListResponse
}

export async function fetchCourseReviews(
  courseCode: string,
  cursor?: string,
): Promise<ReviewsListResponse> {
  const params = new URLSearchParams()
  if (cursor) params.set('cursor', cursor)
  const qs = params.toString()
  const res = await authorizedFetch(
    `/api/v1/courses/${encodeURIComponent(courseCode)}/reviews${qs ? `?${qs}` : ''}`,
  )
  if (!res.ok) throw new Error('Failed to load reviews.')
  return (await res.json()) as ReviewsListResponse
}

export async function fetchReviewEligibility(courseCode: string): Promise<ReviewEligibility> {
  const res = await authorizedFetch(
    `/api/v1/courses/${encodeURIComponent(courseCode)}/reviews/eligibility`,
  )
  if (!res.ok) throw new Error('Failed to check review eligibility.')
  return (await res.json()) as ReviewEligibility
}

export async function submitCourseReview(
  courseCode: string,
  body: { rating: number; reviewText?: string },
): Promise<CourseReview> {
  const res = await authorizedFetch(`/api/v1/courses/${encodeURIComponent(courseCode)}/reviews`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  if (!res.ok) {
    const data = (await res.json().catch(() => null)) as { message?: string } | null
    throw new Error(data?.message ?? 'Failed to submit review.')
  }
  return (await res.json()) as CourseReview
}

export async function flagCourseReview(courseCode: string, reviewId: string): Promise<void> {
  const res = await authorizedFetch(
    `/api/v1/courses/${encodeURIComponent(courseCode)}/reviews/${encodeURIComponent(reviewId)}/flag`,
    { method: 'POST' },
  )
  if (!res.ok) throw new Error('Failed to flag review.')
}

export async function respondToCourseReview(
  courseCode: string,
  reviewId: string,
  response: string,
): Promise<void> {
  const res = await authorizedFetch(
    `/api/v1/courses/${encodeURIComponent(courseCode)}/reviews/${encodeURIComponent(reviewId)}/response`,
    {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ response }),
    },
  )
  if (!res.ok) throw new Error('Failed to post response.')
}

export async function fetchFlaggedReviews(): Promise<CourseReview[]> {
  const res = await authorizedFetch('/api/v1/admin/reviews')
  if (!res.ok) throw new Error('Failed to load moderation queue.')
  const data = (await res.json()) as { reviews: CourseReview[] }
  return data.reviews ?? []
}

export async function removeReviewAdmin(reviewId: string): Promise<void> {
  const res = await authorizedFetch(`/api/v1/admin/reviews/${encodeURIComponent(reviewId)}`, {
    method: 'DELETE',
  })
  if (!res.ok) throw new Error('Failed to remove review.')
}

export const STAR_LABELS: Record<number, string> = {
  1: '1 star – Poor',
  2: '2 stars – Fair',
  3: '3 stars – Average',
  4: '4 stars – Good',
  5: '5 stars – Excellent',
}
