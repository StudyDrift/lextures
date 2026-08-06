import { useEffect, useId, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { ArrowLeft, Star, Store } from 'lucide-react'
import { CourseHeroImage } from '../../components/course-hero-image'
import { usePlatformFeatures } from '../../context/platform-features-context'
import {
  claimMarketplaceCourse,
  checkoutMarketplaceCourse,
  fetchMarketplaceCourse,
  marketplaceCourseItemPath,
  marketplaceCoursePath,
  MarketplaceApiError,
  type MarketplaceCourseDetail,
} from '../../lib/marketplace-api'
import { EmptyState } from '../../components/ui/empty-state'
import { LmsPage } from '../lms/lms-page'
import { formatMarketplacePrice } from '../../lib/marketplace-price'
import { MarketplacePriceBadge } from './marketplace-price-badge'

export default function MarketplaceCoursePage() {
  const { slug } = useParams<{ slug: string }>()
  const navigate = useNavigate()
  const { t, i18n } = useTranslation('common')
  const { ffCourseMarketplace, loading: featuresLoading } = usePlatformFeatures()
  const [detail, setDetail] = useState<MarketplaceCourseDetail | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)
  const [ctaPending, setCtaPending] = useState(false)
  const [ctaError, setCtaError] = useState<string | null>(null)
  const priceId = useId()
  const statusId = useId()

  useEffect(() => {
    if (featuresLoading || !ffCourseMarketplace) {
      setLoading(false)
      return
    }
    if (!slug) {
      setError(t('marketplace.detail.missing'))
      setLoading(false)
      return
    }
    let cancelled = false
    setLoading(true)
    setError(null)
    fetchMarketplaceCourse(slug)
      .then((d) => {
        if (!cancelled) setDetail(d)
      })
      .catch((e: unknown) => {
        if (!cancelled) setError(e instanceof Error ? e.message : t('marketplace.detail.notFound'))
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [slug, ffCourseMarketplace, featuresLoading, t])

  if (!ffCourseMarketplace && !featuresLoading) {
    return (
      <LmsPage title={t('marketplace.title')} description={t('marketplace.subtitle')}>
        <EmptyState
          icon={Store}
          title={t('marketplace.notEnabledTitle')}
          body={t('marketplace.notEnabledBody')}
        />
      </LmsPage>
    )
  }

  const course = detail?.course
  const freeLabel = t('marketplace.free')
  const owned = detail?.owned ?? course?.owned ?? false
  const priceCents = detail?.priceCents ?? course?.priceCents ?? 0
  const priceCurrency = detail?.priceCurrency ?? course?.priceCurrency ?? 'usd'
  const isFree = priceCents <= 0

  let ctaLabel = t('marketplace.buy', { price: '' })
  if (owned) {
    ctaLabel = t('marketplace.goToCourse')
  } else if (isFree) {
    ctaLabel = t('marketplace.enrollFree')
  } else {
    const priceText = formatMarketplacePrice(priceCents, priceCurrency, i18n.language, '')
    ctaLabel = t('marketplace.buy', { price: priceText })
  }
  if (ctaPending) {
    ctaLabel = t('marketplace.cta.processing')
  }

  async function onCtaClick() {
    if (!slug || !course || ctaPending) return
    setCtaError(null)
    if (owned) {
      navigate(marketplaceCoursePath(course.courseCode))
      return
    }
    setCtaPending(true)
    try {
      if (isFree) {
        const result = await claimMarketplaceCourse(slug)
        navigate(marketplaceCourseItemPath(result.courseCode, result.firstItemId))
        return
      }
      const result = await checkoutMarketplaceCourse(slug)
      if (result.alreadyOwned) {
        navigate(marketplaceCoursePath(result.courseCode))
        return
      }
      if (result.checkoutUrl) {
        window.location.assign(result.checkoutUrl)
        return
      }
      setCtaError(t('marketplace.error.retry'))
    } catch (e: unknown) {
      if (e instanceof MarketplaceApiError && e.status === 402 && e.checkoutHint) {
        navigate(e.checkoutHint)
        return
      }
      setCtaError(e instanceof Error ? e.message : t('marketplace.error.retry'))
    } finally {
      setCtaPending(false)
    }
  }

  return (
    <LmsPage title={course?.title ?? t('marketplace.title')}>
      <Link
        to="/marketplace"
        className="mb-6 inline-flex items-center gap-1 text-sm text-fg-muted hover:text-fg-default dark:text-fg-muted dark:hover:text-fg-default"
      >
        <ArrowLeft className="h-4 w-4" aria-hidden="true" />
        {t('marketplace.back')}
      </Link>

      {loading ? (
        <div
          className="h-64 motion-safe:animate-pulse rounded-2xl bg-surface-sunken"
          aria-hidden
        />
      ) : error ? (
        <div
          role="alert"
          className="rounded-xl border border-rose-200 bg-rose-50 px-4 py-3 text-sm text-rose-800 dark:border-rose-900/40 dark:bg-rose-950/30 dark:text-rose-100"
          data-testid="marketplace-detail-error"
        >
          {error}
        </div>
      ) : course ? (
        <article data-testid="marketplace-course-detail" className="max-w-3xl">
          <header className="overflow-hidden rounded-2xl border border-border-default bg-surface-raised shadow-sm dark:border-border-default dark:bg-surface-raised">
            {course.heroImageUrl ? (
              <CourseHeroImage
                src={course.heroImageUrl}
                alt=""
                className="h-56 w-full object-cover"
              />
            ) : (
              <div className="h-40 w-full bg-gradient-to-br from-indigo-100 to-sky-100 dark:from-indigo-950 dark:to-sky-950" />
            )}
            <div className="p-6">
              <div className="flex flex-wrap items-center gap-2 text-xs text-fg-muted">
                {course.category ? <span>{course.category}</span> : null}
                {course.level ? (
                  <span className="rounded-full bg-surface-sunken px-2 py-0.5 capitalize dark:bg-surface-overlay">
                    {course.level}
                  </span>
                ) : null}
                <span className="uppercase tracking-wide">{course.language}</span>
                {owned ? (
                  <span
                    className="rounded-full bg-emerald-50 px-2 py-0.5 font-medium text-emerald-800 dark:bg-emerald-950/40 dark:text-emerald-200"
                    data-testid="marketplace-owned-badge"
                  >
                    {t('marketplace.owned')}
                  </span>
                ) : null}
              </div>
              <h1 className="mt-2 text-2xl font-semibold tracking-tight text-fg-default sm:text-3xl">
                {course.title}
              </h1>
              {course.instructorName ? (
                <p className="mt-1 text-sm text-fg-muted">
                  {t('marketplace.detail.taughtBy', { name: course.instructorName })}
                </p>
              ) : null}
              <div className="mt-3 flex flex-wrap items-center gap-4 text-sm">
                <span className="flex items-center gap-1 text-amber-600 dark:text-amber-400">
                  {detail?.rating.average != null && detail.rating.count > 0 ? (
                    <>
                      <Star className="h-4 w-4 fill-current" aria-hidden="true" />
                      {detail.rating.average.toFixed(1)}
                      <span className="text-fg-muted">
                        ({detail.rating.count.toLocaleString()})
                      </span>
                    </>
                  ) : (
                    <span className="text-fg-subtle">
                      {t('marketplace.detail.notRated')}
                    </span>
                  )}
                </span>
                <span className="text-fg-muted">
                  {t('marketplace.detail.enrolled', { count: course.enrollmentCount })}
                </span>
              </div>
            </div>
          </header>

          <section className="mt-6 rounded-2xl border border-border-default bg-surface-raised p-6 shadow-sm dark:border-border-default dark:bg-surface-raised">
            <h2 className="text-lg font-semibold text-fg-default">
              {t('marketplace.detail.about')}
            </h2>
            <p className="mt-2 whitespace-pre-line text-sm leading-relaxed text-fg-muted">
              {course.description || t('marketplace.detail.noDescription')}
            </p>
          </section>

          {detail?.whatsIncluded ? (
            <section className="mt-6 rounded-2xl border border-border-default bg-surface-raised p-6 shadow-sm dark:border-border-default dark:bg-surface-raised">
              <h2 className="text-lg font-semibold text-fg-default">
                {t('marketplace.detail.whatsIncluded')}
              </h2>
              <ul className="mt-3 list-disc space-y-1 ps-5 text-sm text-fg-muted">
                <li>
                  {t('marketplace.detail.modules', { count: detail.whatsIncluded.moduleCount })}
                </li>
                <li>{t('marketplace.detail.items', { count: detail.whatsIncluded.itemCount })}</li>
                {detail.whatsIncluded.estimatedDurationMinutes != null ? (
                  <li>
                    {t('marketplace.detail.duration', {
                      minutes: detail.whatsIncluded.estimatedDurationMinutes,
                    })}
                  </li>
                ) : null}
              </ul>
            </section>
          ) : null}

          <div className="mt-6 flex flex-wrap items-center justify-between gap-4 rounded-2xl border border-border-default bg-surface-raised p-6 shadow-sm dark:border-border-default dark:bg-surface-raised">
            <span id={priceId}>
              <MarketplacePriceBadge
                priceCents={priceCents}
                priceCurrency={priceCurrency}
                listPriceCents={detail?.listPriceCents ?? course.listPriceCents}
                freeLabel={freeLabel}
                locale={i18n.language}
                className="text-2xl"
              />
            </span>
            <div className="flex flex-col items-end gap-2">
              <button
                type="button"
                aria-describedby={`${priceId} ${statusId}`}
                aria-busy={ctaPending}
                disabled={ctaPending}
                onClick={() => void onCtaClick()}
                className="inline-flex items-center justify-center rounded-xl bg-accent-solid px-5 py-2.5 text-sm font-semibold text-white shadow-sm hover:bg-indigo-500 disabled:cursor-wait disabled:opacity-70"
                data-testid="marketplace-cta"
              >
                {ctaLabel}
              </button>
              <span id={statusId} className="sr-only" aria-live="polite">
                {ctaPending ? t('marketplace.cta.processing') : ctaError ?? ''}
              </span>
              {ctaError ? (
                <p role="alert" className="text-sm text-rose-700 dark:text-rose-300" data-testid="marketplace-cta-error">
                  {ctaError}
                </p>
              ) : null}
            </div>
          </div>
        </article>
      ) : null}
    </LmsPage>
  )
}
