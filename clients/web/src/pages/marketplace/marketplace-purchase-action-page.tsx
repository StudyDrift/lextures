import { useEffect, useState } from 'react'
import { Link, useLocation, useNavigate, useParams, useSearchParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Loader2, Store } from 'lucide-react'
import { usePlatformFeatures } from '../../context/platform-features-context'
import {
  claimMarketplaceCourse,
  checkoutMarketplaceCourse,
  isMarketplaceGrantedFree,
  marketplaceCourseItemPath,
  marketplaceCoursePath,
  MarketplaceApiError,
} from '../../lib/marketplace-api'
import {
  clearPendingCoupon,
  normalizeCouponInput,
  readCouponFromLocation,
  readPendingCoupon,
  rememberPendingCoupon,
} from '../../lib/marketplace-coupon'
import { emitLearnerCouponTelemetry } from '../../lib/marketplace-learner-coupon-telemetry'
import { toastSaveOk } from '../../lib/lms-toast'
import { EmptyState } from '../../components/ui/empty-state'
import { LmsPage } from '../lms/lms-page'

/**
 * Runs free claim or paid checkout for a marketplace slug (plan MKT4 / MKTC.5).
 * Detail-page CTAs also call the API directly; these routes support deep links with ?coupon=.
 */
export default function MarketplacePurchaseActionPage() {
  const { slug } = useParams<{ slug: string }>()
  const location = useLocation()
  const [searchParams] = useSearchParams()
  const navigate = useNavigate()
  const { t } = useTranslation('common')
  const { ffCourseMarketplace, ffCourseCoupons, loading: featuresLoading } = usePlatformFeatures()
  const isClaim = location.pathname.endsWith('/claim')
  const [error, setError] = useState<string | null>(null)
  const [status, setStatus] = useState<'idle' | 'working' | 'done'>('idle')
  const [retryKey, setRetryKey] = useState(0)

  useEffect(() => {
    if (featuresLoading || !ffCourseMarketplace || !slug) return
    let cancelled = false

    function resolveCoupon(): string | undefined {
      if (!ffCourseCoupons) return undefined
      const qs = searchParams.toString()
      const fromUrl = readCouponFromLocation(qs ? `?${qs}` : '')
      if (fromUrl) {
        rememberPendingCoupon(slug!, fromUrl)
        return fromUrl
      }
      return readPendingCoupon(slug!) ?? undefined
    }

    async function runPurchase() {
      setStatus('working')
      setError(null)
      const couponCode = resolveCoupon()
      const opts = couponCode ? { couponCode: normalizeCouponInput(couponCode) } : undefined
      try {
        if (isClaim) {
          if (opts?.couponCode) {
            emitLearnerCouponTelemetry('coupon_checkout_started', { discounted: true })
          }
          const result = await claimMarketplaceCourse(slug!, opts)
          if (cancelled) return
          setStatus('done')
          clearPendingCoupon(slug!)
          if (isMarketplaceGrantedFree(result) || result.grantedFree) {
            emitLearnerCouponTelemetry('coupon_free_grant')
            toastSaveOk(t('marketplace.coupon.reason.ok'))
          }
          navigate(marketplaceCourseItemPath(result.courseCode, result.firstItemId), {
            replace: true,
          })
          return
        }
        if (opts?.couponCode) {
          emitLearnerCouponTelemetry('coupon_checkout_started', { discounted: true })
        }
        const result = await checkoutMarketplaceCourse(slug!, opts)
        if (cancelled) return
        if (isMarketplaceGrantedFree(result)) {
          setStatus('done')
          clearPendingCoupon(slug!)
          emitLearnerCouponTelemetry('coupon_free_grant')
          toastSaveOk(t('marketplace.coupon.reason.ok'))
          navigate(marketplaceCourseItemPath(result.courseCode, result.firstItemId), {
            replace: true,
          })
          return
        }
        if ('alreadyOwned' in result && result.alreadyOwned) {
          setStatus('done')
          clearPendingCoupon(slug!)
          navigate(marketplaceCoursePath(result.courseCode), { replace: true })
          return
        }
        if ('checkoutUrl' in result && result.checkoutUrl) {
          setStatus('done')
          window.location.assign(result.checkoutUrl)
          return
        }
        throw new Error(t('marketplace.error.retry'))
      } catch (e: unknown) {
        if (cancelled) return
        setStatus('idle')
        if (e instanceof MarketplaceApiError && e.status === 402 && e.checkoutHint) {
          navigate(e.checkoutHint, { replace: true })
          return
        }
        setError(e instanceof Error ? e.message : t('marketplace.error.retry'))
      }
    }

    void runPurchase()
    return () => {
      cancelled = true
    }
  }, [
    slug,
    isClaim,
    ffCourseMarketplace,
    ffCourseCoupons,
    featuresLoading,
    navigate,
    t,
    retryKey,
    searchParams,
  ])

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

  const backHref = slug
    ? (() => {
        const code = readCouponFromLocation(`?${searchParams.toString()}`) || readPendingCoupon(slug)
        const base = `/marketplace/${encodeURIComponent(slug)}`
        return code ? `${base}?coupon=${encodeURIComponent(code)}` : base
      })()
    : '/marketplace'

  return (
    <LmsPage title={isClaim ? t('marketplace.enrollFree') : t('marketplace.checkoutTitle')}>
      <div
        className="max-w-xl rounded-2xl border border-border-default bg-surface-raised p-6 shadow-sm dark:border-border-default dark:bg-surface-raised"
        data-testid="marketplace-purchase-action"
        data-flow={isClaim ? 'claim' : 'checkout'}
        aria-live="polite"
      >
        {status === 'working' || status === 'done' ? (
          <div className="flex flex-col items-center gap-3 py-4 text-center">
            <Loader2 className="h-8 w-8 motion-safe:animate-spin text-accent-fg" aria-hidden />
            <p className="text-sm text-fg-muted">{t('marketplace.cta.processing')}</p>
          </div>
        ) : null}
        {error ? (
          <div role="alert" className="space-y-3">
            <p className="text-sm text-danger-fg">{error}</p>
            <div className="flex flex-wrap gap-3">
              <button
                type="button"
                className="rounded-lg bg-accent-solid px-3 py-1.5 text-sm font-medium text-fg-on-accent hover:opacity-90"
                onClick={() => setRetryKey((k) => k + 1)}
              >
                {t('marketplace.error.retry')}
              </button>
              <Link to={backHref} className="text-sm font-semibold text-accent-fg hover:opacity-90">
                {t('marketplace.back')}
              </Link>
            </div>
          </div>
        ) : null}
      </div>
    </LmsPage>
  )
}
