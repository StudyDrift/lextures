import { useEffect, useId, useState } from 'react'
import { Link, useParams, useSearchParams } from 'react-router-dom'
import { ArrowLeft, Star } from 'lucide-react'
import { CourseHeroImage } from '../components/course-hero-image'
import { CourseReviewsSection } from '../components/reviews/course-reviews-section'
import { type ReviewsListResponse } from '../lib/course-reviews-api'
import {
  fetchExploreCourse,
  fetchExploreCourseReviews,
} from '../lib/public-explore-api'
import { formatPrice } from '../lib/public-catalog-api'
import {
  setAffiliateRefCookie,
  trackAffiliateClick,
} from '../lib/revenue-share-api'
import type { PublicCatalogCourseDetail } from '../lib/public-catalog-api'

const JSON_LD_ID = 'catalog-course-jsonld'

function useCourseJsonLd(jsonLd: Record<string, unknown> | null) {
  useEffect(() => {
    if (!jsonLd) return
    const el = document.createElement('script')
    el.type = 'application/ld+json'
    el.id = JSON_LD_ID
    el.text = JSON.stringify(jsonLd)
    document.head.appendChild(el)
    return () => {
      document.getElementById(JSON_LD_ID)?.remove()
    }
  }, [jsonLd])
}

export default function ExploreCoursePage() {
  const { slug } = useParams<{ slug: string }>()
  const [searchParams] = useSearchParams()
  const [detail, setDetail] = useState<PublicCatalogCourseDetail | null>(null)
  const [reviews, setReviews] = useState<ReviewsListResponse | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)
  const priceId = useId()

  useEffect(() => {
    if (!slug) {
      setError('Missing course.')
      setLoading(false)
      return
    }
    let cancelled = false
    setLoading(true)
    Promise.all([
      fetchExploreCourse(slug),
      fetchExploreCourseReviews(slug).catch(() => null),
    ])
      .then(([d, r]) => {
        if (!cancelled) {
          setDetail(d)
          setReviews(r)
        }
      })
      .catch((e: unknown) => {
        if (!cancelled) setError(e instanceof Error ? e.message : 'Course not found.')
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [slug])

  useCourseJsonLd(detail?.jsonLd ?? null)

  useEffect(() => {
    const ref = searchParams.get('ref')?.trim()
    if (!ref) return
    setAffiliateRefCookie(ref)
    void trackAffiliateClick(ref)
  }, [searchParams])

  useEffect(() => {
    if (detail) document.title = `${detail.course.title} — Lextures`
  }, [detail])

  const course = detail?.course

  return (
    <div className="min-h-screen bg-surface-base">
      <div className="mx-auto max-w-4xl px-4 py-8">
        <Link
          to="/explore"
          className="mb-6 inline-flex items-center gap-1 text-sm text-fg-muted hover:text-fg-default dark:text-fg-muted dark:hover:text-fg-default"
        >
          <ArrowLeft className="h-4 w-4" aria-hidden="true" />
          Back to catalog
        </Link>

        {loading ? (
          <p className="text-sm text-fg-muted">Loading…</p>
        ) : error ? (
          <div
            role="alert"
            className="rounded-xl border border-red-200 bg-red-50 p-6 text-sm text-danger-fg dark:border-red-900 dark:bg-red-950 dark:text-red-300"
          >
            {error}
          </div>
        ) : course ? (
          <article>
            <header className="overflow-hidden rounded-2xl border border-border-default bg-surface-raised dark:border-border-subtle dark:bg-surface-raised">
              {course.heroImageUrl ? (
                <CourseHeroImage src={course.heroImageUrl} alt="" className="h-56 w-full object-cover" />
              ) : (
                <div className="h-40 w-full bg-gradient-to-br from-indigo-100 to-sky-100 dark:from-indigo-950 dark:to-sky-950" />
              )}
              <div className="p-6">
                <div className="flex flex-wrap items-center gap-2 text-xs text-fg-muted">
                  {course.category ? <span>{course.category}</span> : null}
                  {course.difficultyLevel ? (
                    <span className="rounded-full bg-surface-sunken px-2 py-0.5 capitalize dark:bg-surface-overlay">
                      {course.difficultyLevel}
                    </span>
                  ) : null}
                  <span className="uppercase">{course.language}</span>
                </div>
                <h1 className="mt-2 text-3xl font-bold text-fg-default">
                  {course.title}
                </h1>
                {course.instructorName ? (
                  <p className="mt-1 text-sm text-fg-muted">
                    Taught by {course.instructorName}
                  </p>
                ) : null}
                <div className="mt-3 flex items-center gap-4 text-sm">
                  <span className="flex items-center gap-1 text-amber-600 dark:text-amber-400">
                    {course.averageRating != null && (course.ratingCount ?? 0) > 0 ? (
                      <>
                        <Star className="h-4 w-4 fill-current" aria-hidden="true" />
                        {course.averageRating.toFixed(1)}
                        <span className="text-fg-muted">
                          ({course.ratingCount?.toLocaleString()} reviews)
                        </span>
                      </>
                    ) : (
                      <span className="text-fg-subtle">Not yet rated</span>
                    )}
                  </span>
                  <span className="text-fg-muted">
                    {course.enrollmentCount.toLocaleString()} learners enrolled
                  </span>
                </div>
              </div>
            </header>

            <section className="mt-6 rounded-2xl border border-border-default bg-surface-raised p-6 dark:border-border-subtle dark:bg-surface-raised">
              <h2 className="text-lg font-semibold text-fg-default">
                About this course
              </h2>
              <p className="mt-2 whitespace-pre-line text-sm leading-relaxed text-fg-muted">
                {course.description || 'No description provided yet.'}
              </p>
            </section>

            {reviews ? (
              <CourseReviewsSection summary={reviews.summary} reviews={reviews.reviews} />
            ) : null}

            <div className="mt-6 flex items-center justify-between rounded-2xl border border-border-default bg-surface-raised p-6 dark:border-border-subtle dark:bg-surface-raised">
              <span id={priceId} className="text-2xl font-bold text-fg-default">
                {formatPrice(course.priceCents)}
              </span>
              <Link
                to={`/marketplace/${encodeURIComponent(course.slug)}`}
                aria-describedby={priceId}
                className="rounded-lg bg-accent-solid px-5 py-2.5 text-sm font-semibold text-white hover:bg-indigo-500"
              >
                Enroll now
              </Link>
            </div>
          </article>
        ) : null}
      </div>
    </div>
  )
}
