import { useCallback, useEffect, useId, useRef, useState } from 'react'
import { Link, useNavigate, useParams, useSearchParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { ArrowLeft, Star, Store } from 'lucide-react'
import { CourseHeroImage } from '../../components/course-hero-image'
import { Button } from '../../components/ui'
import { EmptyState } from '../../components/ui/empty-state'
import { usePlatformFeatures } from '../../context/platform-features-context'
import {
  claimMarketplaceCourse,
  checkoutMarketplaceCourse,
  fetchMarketplaceCourse,
  isMarketplaceGrantedFree,
  marketplaceCourseItemPath,
  marketplaceCoursePath,
  MarketplaceApiError,
  type MarketplaceCourseDetail,
} from '../../lib/marketplace-api'
import {
  clearPendingCoupon,
  couponReasonKey,
  readCouponFromLocation,
  readPendingCoupon,
  rememberPendingCoupon,
} from '../../lib/marketplace-coupon'
import { emitLearnerCouponTelemetry } from '../../lib/marketplace-learner-coupon-telemetry'
import { toastSaveOk } from '../../lib/lms-toast'
import { formatMarketplacePrice } from '../../lib/marketplace-price'
import { LmsPage } from '../lms/lms-page'
import { MarketplaceCouponField } from './marketplace-coupon-field'
import { MarketplacePriceBadge } from './marketplace-price-badge'
import { useMarketplaceCoupon } from './use-marketplace-coupon'

export default function MarketplaceCoursePage() {
  const { slug } = useParams<{ slug: string }>()
  const [searchParams] = useSearchParams()
  const navigate = useNavigate()
  const { t, i18n } = useTranslation('common')
  const {
    ffCourseMarketplace,
    ffCourseCoupons,
    loading: featuresLoading,
  } = usePlatformFeatures()
  const [detail, setDetail] = useState<MarketplaceCourseDetail | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)
  const [ctaPending, setCtaPending] = useState(false)
  const [ctaError, setCtaError] = useState<string | null>(null)
  const priceId = useId()
  const statusId = useId()
  const livePriceId = useId()
  const autoAppliedRef = useRef(false)

  const couponsEnabled = Boolean(ffCourseCoupons)
  const coupon = useMarketplaceCoupon(slug, couponsEnabled)

  const formatApplied = useCallback(
    (p: { code: string; chargedCents: number; currency: string }) => {
      const priceText = formatMarketplacePrice(
        p.chargedCents,
        p.currency,
        i18n.language,
        t('marketplace.free'),
      )
      return t('marketplace.coupon.applied', { code: p.code, price: priceText })
    },
    [i18n.language, t],
  )

  const resolveInitialCode = useCallback(
    (courseSlug: string) => {
      if (!couponsEnabled) return null
      const qs = searchParams.toString()
      const fromUrl = readCouponFromLocation(qs ? `?${qs}` : '')
      if (fromUrl) {
        rememberPendingCoupon(courseSlug, fromUrl)
        return fromUrl
      }
      return readPendingCoupon(courseSlug)
    },
    [couponsEnabled, searchParams],
  )

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
    autoAppliedRef.current = false

    const initialCode = resolveInitialCode(slug)
    if (initialCode) coupon.setSeedCode(initialCode)

    fetchMarketplaceCourse(slug, couponsEnabled && initialCode ? { coupon: initialCode } : undefined)
      .then((d) => {
        if (cancelled) return
        setDetail(d)
        if (d.owned) {
          coupon.clearOnOwned()
          return
        }
        if (couponsEnabled && initialCode && d.coupon) {
          coupon.applyPreview(d.coupon, initialCode, true, formatApplied)
          autoAppliedRef.current = true
        }
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
    // eslint-disable-next-line react-hooks/exhaustive-deps -- coupon helpers are stable enough per slug
  }, [slug, ffCourseMarketplace, featuresLoading, couponsEnabled, t, resolveInitialCode, formatApplied])

  useEffect(() => {
    if (!slug || !detail || detail.owned || !couponsEnabled) return
    if (autoAppliedRef.current) return
    if (coupon.status === 'applied' || coupon.status === 'checking') return
    const code = coupon.seed || resolveInitialCode(slug)
    if (!code) return
    if (detail.coupon) {
      coupon.applyPreview(detail.coupon, code, true, formatApplied)
      autoAppliedRef.current = true
      return
    }
    autoAppliedRef.current = true
    void coupon.runPreview(code, true, formatApplied)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [slug, detail, couponsEnabled, coupon.seed])

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
  const listPriceCents = detail?.priceCents ?? course?.priceCents ?? 0
  const priceCurrency = detail?.priceCurrency ?? course?.priceCurrency ?? 'usd'
  const displayPriceCents = coupon.applied ? coupon.preview!.chargedCents : listPriceCents
  const displayListPriceCents = coupon.applied
    ? coupon.preview!.listPriceCents
    : (detail?.listPriceCents ?? course?.listPriceCents ?? null)
  const isFree = displayPriceCents <= 0
  const isFreeAfterCoupon = Boolean(coupon.applied && coupon.preview?.freeAfterDiscount)
  const activeCouponCode = coupon.activeCode

  let ctaLabel = t('marketplace.buy', { price: '' })
  if (owned) {
    ctaLabel = t('marketplace.goToCourse')
  } else if (isFree || isFreeAfterCoupon) {
    ctaLabel = t('marketplace.enrollFree')
  } else {
    const priceText = formatMarketplacePrice(displayPriceCents, priceCurrency, i18n.language, '')
    ctaLabel = t('marketplace.buy', { price: priceText })
  }
  if (ctaPending) ctaLabel = t('marketplace.cta.processing')

  const wasLabel =
    coupon.applied && displayListPriceCents != null && displayListPriceCents > displayPriceCents
      ? t('marketplace.coupon.was', {
          price: formatMarketplacePrice(
            displayListPriceCents,
            priceCurrency,
            i18n.language,
            freeLabel,
          ),
        })
      : null

  async function onCtaClick() {
    if (!slug || !course || ctaPending) return
    setCtaError(null)
    if (owned) {
      navigate(marketplaceCoursePath(course.courseCode))
      return
    }
    setCtaPending(true)
    try {
      if (isFree && !activeCouponCode) {
        const result = await claimMarketplaceCourse(slug)
        clearPendingCoupon(slug)
        navigate(marketplaceCourseItemPath(result.courseCode, result.firstItemId))
        return
      }

      if (isFreeAfterCoupon && activeCouponCode) {
        emitLearnerCouponTelemetry('coupon_checkout_started', { discounted: true })
        let result
        try {
          result = await claimMarketplaceCourse(slug, { couponCode: activeCouponCode })
        } catch {
          result = await checkoutMarketplaceCourse(slug, { couponCode: activeCouponCode })
        }
        if (isMarketplaceGrantedFree(result) || ('enrolled' in result && result.enrolled)) {
          clearPendingCoupon(slug)
          emitLearnerCouponTelemetry('coupon_free_grant')
          toastSaveOk(
            t('marketplace.coupon.applied', { code: activeCouponCode, price: freeLabel }),
          )
          navigate(
            marketplaceCourseItemPath(
              result.courseCode,
              'firstItemId' in result ? result.firstItemId : undefined,
            ),
          )
          return
        }
        if ('alreadyOwned' in result && result.alreadyOwned) {
          clearPendingCoupon(slug)
          navigate(marketplaceCoursePath(result.courseCode))
          return
        }
        setCtaError(t('marketplace.error.retry'))
        return
      }

      emitLearnerCouponTelemetry('coupon_checkout_started', {
        discounted: Boolean(activeCouponCode),
      })
      const result = await checkoutMarketplaceCourse(
        slug,
        activeCouponCode ? { couponCode: activeCouponCode } : undefined,
      )
      if (isMarketplaceGrantedFree(result)) {
        clearPendingCoupon(slug)
        emitLearnerCouponTelemetry('coupon_free_grant')
        toastSaveOk(
          t('marketplace.coupon.applied', {
            code: activeCouponCode ?? result.courseCode,
            price: freeLabel,
          }),
        )
        navigate(marketplaceCourseItemPath(result.courseCode, result.firstItemId))
        return
      }
      if ('alreadyOwned' in result && result.alreadyOwned) {
        clearPendingCoupon(slug)
        navigate(marketplaceCoursePath(result.courseCode))
        return
      }
      if ('checkoutUrl' in result && result.checkoutUrl) {
        window.location.assign(result.checkoutUrl)
        return
      }
      setCtaError(t('marketplace.error.retry'))
    } catch (e: unknown) {
      if (e instanceof MarketplaceApiError && e.status === 402 && e.checkoutHint) {
        navigate(e.checkoutHint)
        return
      }
      if (e instanceof MarketplaceApiError && e.status === 422 && e.reason) {
        coupon.rejectAtCheckout(e.reason)
        setCtaError(t(couponReasonKey(e.reason)))
        return
      }
      setCtaError(e instanceof Error ? e.message : t('marketplace.error.retry'))
    } finally {
      setCtaPending(false)
    }
  }

  const showCouponField =
    couponsEnabled && !owned && listPriceCents > 0 && !loading && Boolean(course)

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
        <div className="h-64 motion-safe:animate-pulse rounded-2xl bg-surface-sunken" aria-hidden />
      ) : error ? (
        <div
          role="alert"
          className="rounded-xl border border-danger-fg/30 bg-danger-surface px-4 py-3 text-sm text-danger-fg"
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
              <div className="h-40 w-full bg-surface-sunken" />
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
                    className="rounded-full bg-success-surface px-2 py-0.5 font-medium text-success-fg"
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
                <span className="flex items-center gap-1 text-warning-fg">
                  {detail?.rating.average != null && detail.rating.count > 0 ? (
                    <>
                      <Star className="h-4 w-4 fill-current" aria-hidden="true" />
                      {detail.rating.average.toFixed(1)}
                      <span className="text-fg-muted">
                        ({detail.rating.count.toLocaleString()})
                      </span>
                    </>
                  ) : (
                    <span className="text-fg-subtle">{t('marketplace.detail.notRated')}</span>
                  )}
                </span>
                <span className="text-fg-muted">
                  {t('marketplace.detail.enrolled', { count: course.enrollmentCount })}
                </span>
              </div>
            </div>
          </header>

          <section className="mt-6 rounded-2xl border border-border-default bg-surface-raised p-6 shadow-sm">
            <h2 className="text-lg font-semibold text-fg-default">{t('marketplace.detail.about')}</h2>
            <p className="mt-2 whitespace-pre-line text-sm leading-relaxed text-fg-muted">
              {course.description || t('marketplace.detail.noDescription')}
            </p>
          </section>

          {detail?.whatsIncluded ? (
            <section className="mt-6 rounded-2xl border border-border-default bg-surface-raised p-6 shadow-sm">
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

          {showCouponField ? (
            <div className="mt-6">
              <MarketplaceCouponField
                slug={slug!}
                defaultOpen={Boolean(coupon.seed)}
                initialCode={coupon.seed}
                status={coupon.status}
                preview={coupon.preview}
                errorReason={coupon.errorReason}
                rateLimited={coupon.status === 'rate_limited'}
                disabled={ctaPending}
                onApply={(code) => coupon.runPreview(code, false, formatApplied)}
                onRemove={coupon.remove}
                locale={i18n.language}
              />
            </div>
          ) : null}

          <div className="mt-6 flex flex-wrap items-center justify-between gap-4 rounded-2xl border border-border-default bg-surface-raised p-6 shadow-sm">
            <span id={priceId} aria-live="polite" aria-atomic="true">
              <MarketplacePriceBadge
                priceCents={displayPriceCents}
                priceCurrency={priceCurrency}
                listPriceCents={displayListPriceCents}
                wasLabel={wasLabel}
                freeLabel={freeLabel}
                locale={i18n.language}
                className="text-2xl"
              />
            </span>
            <span id={livePriceId} className="sr-only" aria-live="polite">
              {coupon.announcement}
            </span>
            <div className="flex flex-col items-end gap-2">
              <Button
                type="button"
                aria-describedby={`${priceId} ${statusId}`}
                aria-busy={ctaPending}
                loading={ctaPending}
                disabled={ctaPending}
                onClick={() => void onCtaClick()}
                data-testid="marketplace-cta"
              >
                {ctaLabel}
              </Button>
              <span id={statusId} className="sr-only" aria-live="polite">
                {ctaPending ? t('marketplace.cta.processing') : (ctaError ?? '')}
              </span>
              {ctaError ? (
                <p role="alert" className="text-sm text-danger-fg" data-testid="marketplace-cta-error">
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
