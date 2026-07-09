import { useEffect, useState } from 'react'
import { Link, useLocation, useNavigate, useParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Loader2, Store } from 'lucide-react'
import { usePlatformFeatures } from '../../context/platform-features-context'
import {
  claimMarketplaceCourse,
  checkoutMarketplaceCourse,
  marketplaceCourseItemPath,
  marketplaceCoursePath,
  MarketplaceApiError,
} from '../../lib/marketplace-api'
import { EmptyState } from '../../components/ui/empty-state'
import { LmsPage } from '../lms/lms-page'

/**
 * Runs free claim or paid checkout for a marketplace slug (plan MKT4).
 * Detail-page CTAs also call the API directly; these routes support deep links.
 */
export default function MarketplacePurchaseActionPage() {
  const { slug } = useParams<{ slug: string }>()
  const location = useLocation()
  const navigate = useNavigate()
  const { t } = useTranslation('common')
  const { ffCourseMarketplace, loading: featuresLoading } = usePlatformFeatures()
  const isClaim = location.pathname.endsWith('/claim')
  const [error, setError] = useState<string | null>(null)
  const [status, setStatus] = useState<'idle' | 'working' | 'done'>('idle')
  const [retryKey, setRetryKey] = useState(0)

  useEffect(() => {
    if (featuresLoading || !ffCourseMarketplace || !slug) return
    let cancelled = false

    async function runPurchase() {
      setStatus('working')
      setError(null)
      try {
        if (isClaim) {
          const result = await claimMarketplaceCourse(slug!)
          if (cancelled) return
          setStatus('done')
          navigate(marketplaceCourseItemPath(result.courseCode, result.firstItemId), {
            replace: true,
          })
          return
        }
        const result = await checkoutMarketplaceCourse(slug!)
        if (cancelled) return
        if (result.alreadyOwned) {
          setStatus('done')
          navigate(marketplaceCoursePath(result.courseCode), { replace: true })
          return
        }
        if (result.checkoutUrl) {
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
  }, [slug, isClaim, ffCourseMarketplace, featuresLoading, navigate, t, retryKey])

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

  return (
    <LmsPage title={isClaim ? t('marketplace.enrollFree') : t('marketplace.checkoutTitle')}>
      <div
        className="max-w-xl rounded-2xl border border-slate-200 bg-white p-6 shadow-sm dark:border-neutral-700 dark:bg-neutral-900"
        data-testid="marketplace-purchase-action"
        data-flow={isClaim ? 'claim' : 'checkout'}
        aria-live="polite"
      >
        {status === 'working' || status === 'done' ? (
          <div className="flex flex-col items-center gap-3 py-4 text-center">
            <Loader2 className="h-8 w-8 motion-safe:animate-spin text-indigo-600" aria-hidden />
            <p className="text-sm text-slate-600 dark:text-neutral-300">
              {t('marketplace.cta.processing')}
            </p>
          </div>
        ) : null}
        {error ? (
          <div role="alert" className="space-y-3">
            <p className="text-sm text-rose-700 dark:text-rose-300">{error}</p>
            <div className="flex flex-wrap gap-3">
              <button
                type="button"
                className="rounded-lg bg-indigo-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-indigo-500"
                onClick={() => setRetryKey((k) => k + 1)}
              >
                {t('marketplace.error.retry')}
              </button>
              <Link
                to={slug ? `/marketplace/${encodeURIComponent(slug)}` : '/marketplace'}
                className="text-sm font-semibold text-indigo-600 hover:text-indigo-500 dark:text-indigo-400"
              >
                {t('marketplace.back')}
              </Link>
            </div>
          </div>
        ) : null}
      </div>
    </LmsPage>
  )
}
