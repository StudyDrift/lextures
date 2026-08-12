import { useEffect, useState } from 'react'
import { ArrowLeft } from 'lucide-react'
import { MarkdownBody } from '../components/markdown-body'
import { CourseHeroPlaceholder, RatingStars } from '../components/courses/course-card'
import { EnrollPanel } from '../components/courses/enroll-panel'
import { ReviewList } from '../components/courses/review-list'
import { WhatsIncluded } from '../components/courses/whats-included'
import { MarketingPageShell } from '../components/marketing-page-shell'
import { resolveApiAssetUrl } from '../lib/api-base'
import { COURSES_COPY } from '../lib/courses-copy'
import { truncateMetaDescription } from '../lib/document-head'
import {
  displayHandoffCoupon,
  fetchPublicMarketplaceCourse,
  fetchPublicMarketplaceReviews,
  MarketplaceApiError,
  type CourseReview,
  type PublicMarketplaceCourseDetail,
} from '../lib/marketplace-api'
import { useDocumentHead } from '../lib/use-document-head'
import { useSsrData } from '../lib/ssr-context'
import { canonicalUrl, SITE_ORIGIN } from '../lib/site-origin'
import { courseDetailGraph } from '../lib/schema/page-graphs'

function readCouponFromWindowSearch(): string {
  if (typeof window === 'undefined') return ''
  try {
    const params = new URLSearchParams(window.location.search)
    return displayHandoffCoupon(params.get('coupon'))
  } catch {
    return ''
  }
}

type CourseDetailPageProps = {
  slug: string
}

export function CourseDetailPage({ slug }: CourseDetailPageProps) {
  const ssr = useSsrData()
  const ssrDetail =
    ssr.courseDetail?.course &&
    (ssr.courseDetail.course.slug === slug || ssr.courseDetail.course.courseCode === slug)
      ? ssr.courseDetail
      : null

  const [coupon] = useState(() => readCouponFromWindowSearch())
  const [detail, setDetail] = useState<PublicMarketplaceCourseDetail | null>(() => ssrDetail)
  const [reviews, setReviews] = useState<CourseReview[]>([])
  const [reviewsSummary, setReviewsSummary] = useState<{ average: number | null; count: number }>({
    average: null,
    count: 0,
  })
  const [showReviews, setShowReviews] = useState(true)
  const [loading, setLoading] = useState(() => !ssrDetail)
  const [notFound, setNotFound] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [reloadKey, setReloadKey] = useState(0)

  const course = detail?.course
  const courseKind = course ? [course.level, course.category].filter(Boolean).join(' ') : ''
  const title = course
    ? `${course.title} — ${courseKind ? `${courseKind} course` : 'online course'} | Lextures`
    : COURSES_COPY.pageTitle
  const description = course
    ? truncateMetaDescription(course.description || COURSES_COPY.pageDescription)
    : COURSES_COPY.pageDescription
  const canonical = canonicalUrl(`/courses/${encodeURIComponent(slug)}`)
  const heroImageUrl = resolveApiAssetUrl(course?.heroImageUrl)
  const image = heroImageUrl || undefined

  // Override manifest defaults once course payload is available (unique title/JSON-LD).
  const courseJsonLd = detail
    ? courseDetailGraph(
        { path: `/courses/${slug}`, origin: SITE_ORIGIN, params: { slug } },
        detail,
      )
    : null
  useDocumentHead({
    title,
    description,
    canonical,
    image,
    jsonLd: courseJsonLd,
  })

  useEffect(() => {
    const normalizedSlug = slug.trim()
    if (!normalizedSlug) {
      setLoading(false)
      setNotFound(true)
      setError(null)
      return
    }

    // Skip network when SSG already provided matching detail (still refresh reviews).
    const hasSsr = Boolean(ssrDetail)
    if (!hasSsr) {
      setLoading(true)
    }
    setError(null)
    setNotFound(false)
    let cancelled = false
    fetchPublicMarketplaceCourse(normalizedSlug)
      .then(d => {
        if (cancelled) return
        setDetail(d)
        setLoading(false)
        if (typeof window !== 'undefined') {
          window.gtag?.('event', 'course_detail_view', {
            slug,
            hasCoupon: Boolean(readCouponFromWindowSearch()),
          })
        }
      })
      .catch((e: unknown) => {
        if (cancelled) return
        if (hasSsr) {
          // Keep prerendered content when a live refresh fails.
          setLoading(false)
          return
        }
        setLoading(false)
        if (e instanceof MarketplaceApiError && e.status === 404) {
          setNotFound(true)
          return
        }
        setError(e instanceof Error ? e.message : COURSES_COPY.errorBody)
      })

    fetchPublicMarketplaceReviews(normalizedSlug, { limit: 20 })
      .then(r => {
        if (cancelled) return
        setReviews(r.reviews)
        setReviewsSummary({
          average: r.summary?.average ?? null,
          count: r.summary?.count ?? r.reviews.length,
        })
        setShowReviews(true)
      })
      .catch(() => {
        if (!cancelled) setShowReviews(false)
      })

    return () => {
      cancelled = true
    }
  }, [slug, reloadKey, ssrDetail])

  const load = () => setReloadKey(k => k + 1)

  if (loading) {
    return (
      <MarketingPageShell>
        <div className="mx-auto max-w-[1100px] animate-pulse px-5 py-16 md:px-10">
          <div className="h-8 w-48 rounded" style={{ backgroundColor: 'var(--panel)' }} />
          <div className="mt-6 h-12 w-3/4 rounded" style={{ backgroundColor: 'var(--panel)' }} />
          <div className="mt-8 aspect-video rounded" style={{ backgroundColor: 'var(--panel)' }} />
        </div>
      </MarketingPageShell>
    )
  }

  if (notFound) {
    return (
      <MarketingPageShell>
        <div className="mx-auto max-w-[640px] px-5 py-24 text-center md:px-10">
          <h1 className="font-display text-3xl font-semibold" style={{ color: 'var(--ink-nav)' }}>
            {COURSES_COPY.notFoundTitle}
          </h1>
          <p className="mt-3 text-[16px]" style={{ color: 'var(--text-soft)' }}>
            {COURSES_COPY.notFoundBody}
          </p>
          <a href="/courses" className="btn-secondary mt-8 inline-flex gap-2">
            <ArrowLeft className="h-4 w-4" aria-hidden />
            {COURSES_COPY.backToCourses}
          </a>
        </div>
      </MarketingPageShell>
    )
  }

  if (error || !course || !detail) {
    return (
      <MarketingPageShell>
        <div className="mx-auto max-w-[640px] px-5 py-24 text-center md:px-10" role="alert">
          <h1 className="font-display text-3xl font-semibold" style={{ color: 'var(--ink-nav)' }}>
            {COURSES_COPY.errorTitle}
          </h1>
          <p className="mt-3 text-[16px]" style={{ color: 'var(--text-soft)' }}>
            {error || COURSES_COPY.errorBody}
          </p>
          <button type="button" onClick={load} className="btn-primary mt-8">
            {COURSES_COPY.retry}
          </button>
        </div>
      </MarketingPageShell>
    )
  }

  return (
    <MarketingPageShell>
      <div className="mx-auto max-w-[1100px] px-5 pb-28 pt-10 md:px-10 md:pb-20 xl:px-14">
        <nav className="text-[13px]" aria-label="Breadcrumb" style={{ color: 'var(--text-soft)' }}>
          <a href="/courses" className="no-underline hover:underline" style={{ color: 'var(--text-soft)' }}>
            {COURSES_COPY.breadcrumbCourses}
          </a>
          <span aria-hidden> / </span>
          <span style={{ color: 'var(--ink-nav)' }}>{course.title}</span>
        </nav>

        <div className="mt-6 grid gap-10 lg:grid-cols-[1fr_320px]">
          <div>
            <div className="flex flex-wrap gap-2">
              {course.category && (
                <span
                  className="rounded-full px-2.5 py-0.5 text-[12px] font-semibold uppercase"
                  style={{ backgroundColor: 'rgba(106,197,176,0.14)', color: '#4fa894' }}
                >
                  {course.category}
                </span>
              )}
              {course.level && (
                <span
                  className="rounded-full px-2.5 py-0.5 text-[12px] font-medium capitalize"
                  style={{ backgroundColor: 'rgba(38,58,60,0.06)', color: 'var(--text-soft)' }}
                >
                  {course.level}
                </span>
              )}
            </div>

            <h1
              className="font-display mt-4 text-[clamp(28px,4vw,42px)] font-semibold leading-tight"
              style={{ color: 'var(--ink-nav)' }}
            >
              {course.title}
            </h1>

            <div className="mt-3 flex flex-wrap items-center gap-4 text-[14px]" style={{ color: 'var(--text-soft)' }}>
              {course.instructorName && <span>{course.instructorName}</span>}
              <RatingStars average={course.averageRating} count={course.ratingCount} />
              <span>{COURSES_COPY.students(course.enrollmentCount)}</span>
              {course.language && <span className="uppercase">{course.language}</span>}
            </div>

            <div
              className="mt-8 aspect-video overflow-hidden border"
              style={{
                borderColor: 'var(--line-card)',
                borderRadius: 'var(--radius-card)',
              }}
            >
              {heroImageUrl ? (
                <img
                  src={heroImageUrl}
                  alt=""
                  className="h-full w-full object-cover"
                />
              ) : (
                <CourseHeroPlaceholder title={course.title} />
              )}
            </div>

            {course.description && (
              <section className="mt-10" aria-labelledby="about-heading">
                <h2
                  id="about-heading"
                  className="font-display text-[22px] font-semibold"
                  style={{ color: 'var(--ink-nav)' }}
                >
                  About this course
                </h2>
                <MarkdownBody
                  markdown={course.description}
                  className="prose-content mt-4 text-[16px] leading-relaxed"
                />
              </section>
            )}

            <div className="mt-10">
              <WhatsIncluded data={detail.whatsIncluded} />
            </div>

            <section className="mt-10" aria-labelledby="enrollment-heading">
              <h2 id="enrollment-heading" className="font-display text-[22px] font-semibold" style={{ color: 'var(--ink-nav)' }}>
                Enrollment and course access
              </h2>
              <p className="mt-4 text-[16px] leading-relaxed" style={{ color: 'var(--text-soft)' }}>
                Enroll through Lextures to add this course to your learning dashboard. Your progress is saved as you work, and the enrollment panel shows the current price before you continue.
              </p>
              <p className="mt-3 text-[16px] leading-relaxed" style={{ color: 'var(--text-soft)' }}>
                Coupons and refunds follow the terms shown during checkout. Course accessibility depends on the materials supplied by the instructor; contact the instructor before enrolling if you need a specific format or accommodation.
              </p>
              {course.category && (
                <p className="mt-3 text-[16px]">
                  <a href={`/courses?category=${encodeURIComponent(course.category)}`} className="underline">
                    Browse more {course.category} courses
                  </a>
                </p>
              )}
            </section>

            {showReviews && (
              <div className="mt-10">
                <ReviewList
                  reviews={reviews}
                  average={reviewsSummary.average ?? course.averageRating}
                  count={reviewsSummary.count || course.ratingCount}
                />
              </div>
            )}
          </div>

          <aside className="hidden lg:block">
            <div className="sticky top-24">
              <EnrollPanel course={course} coupon={coupon} />
            </div>
          </aside>
        </div>
      </div>

      <EnrollPanel course={course} sticky coupon={coupon} />
    </MarketingPageShell>
  )
}
