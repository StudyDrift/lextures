import { useCallback, useEffect, useId, useState } from 'react'
import { Link } from 'react-router-dom'
import { ExternalLink, Loader2, ShoppingBag } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { usePlatformFeatures } from '../../context/platform-features-context'
import {
  fetchMyPurchases,
  formatMoney,
  type CoursePurchase,
} from '../../lib/billing-api'
import { EmptyState } from '../../components/ui/empty-state'
import { LmsPage } from './lms-page'

function sourceLabel(source: string, t: (key: string) => string): string {
  switch (source) {
    case 'free':
      return t('purchases.source.free')
    case 'stripe':
      return t('purchases.source.stripe')
    case 'comp':
      return t('purchases.source.comp')
    default:
      return source
  }
}

export default function MyPurchasesPage() {
  const titleId = useId()
  const { t, i18n } = useTranslation('common')
  const { ffCourseMarketplace, loading: featuresLoading } = usePlatformFeatures()
  const [purchases, setPurchases] = useState<CoursePurchase[] | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const items = await fetchMyPurchases()
      setPurchases(items)
    } catch (e) {
      setError(e instanceof Error ? e.message : t('purchases.error'))
      setPurchases([])
    } finally {
      setLoading(false)
    }
  }, [t])

  useEffect(() => {
    if (featuresLoading || !ffCourseMarketplace) return
    void load()
  }, [featuresLoading, ffCourseMarketplace, load])

  if (featuresLoading) {
    return <p>{t('marketplace.loading')}</p>
  }

  if (!ffCourseMarketplace) {
    return (
      <LmsPage title={t('purchases.title')}>
        <EmptyState
          icon={ShoppingBag}
          title={t('marketplace.notEnabledTitle')}
          body={t('marketplace.notEnabledBody')}
        />
      </LmsPage>
    )
  }

  return (
    <LmsPage title={t('purchases.title')} description={t('purchases.subtitle')}>
      <div className="mx-auto max-w-3xl space-y-6">
        <header>
          <h1 id={titleId} className="text-2xl font-semibold text-slate-900 dark:text-neutral-100">
            {t('purchases.title')}
          </h1>
          <p className="mt-2 text-sm text-slate-600 dark:text-neutral-400">{t('purchases.subtitle')}</p>
        </header>

        {error ? (
          <p
            role="alert"
            className="rounded-xl border border-rose-200 bg-rose-50 px-4 py-3 text-sm text-rose-800 dark:border-rose-900 dark:bg-rose-950/40 dark:text-rose-200"
          >
            {error}
          </p>
        ) : null}

        {loading || purchases === null ? (
          <p className="inline-flex items-center gap-2 text-sm text-slate-600 dark:text-neutral-400">
            <Loader2 className="h-4 w-4 motion-safe:animate-spin" aria-hidden />
            {t('marketplace.loading')}
          </p>
        ) : purchases.length === 0 ? (
          <EmptyState
            icon={ShoppingBag}
            title={t('purchases.empty')}
            body={t('purchases.emptyBody')}
            primaryAction={{ label: t('marketplace.title'), to: '/marketplace' }}
          />
        ) : (
          <ul className="divide-y divide-slate-200 rounded-xl border border-slate-200 bg-white dark:divide-neutral-800 dark:border-neutral-700 dark:bg-neutral-900">
            {purchases.map((p) => {
              const acquired = new Date(p.acquiredAt)
              const dateLabel = Number.isNaN(acquired.getTime())
                ? p.acquiredAt
                : acquired.toLocaleDateString(i18n.language, {
                    year: 'numeric',
                    month: 'short',
                    day: 'numeric',
                  })
              const priceLabel =
                p.priceCents <= 0
                  ? t('marketplace.free')
                  : formatMoney(p.priceCents, p.currency, i18n.language)
              return (
                <li
                  key={p.entitlementId}
                  className="flex flex-col gap-3 px-4 py-4 sm:flex-row sm:items-center sm:justify-between"
                  data-testid="my-purchase-row"
                >
                  <div className="min-w-0">
                    <Link
                      to={`/courses/${encodeURIComponent(p.courseCode)}`}
                      className="font-semibold text-slate-900 hover:text-indigo-600 dark:text-neutral-100 dark:hover:text-indigo-300"
                    >
                      {p.title}
                    </Link>
                    <p className="mt-1 text-sm text-slate-600 dark:text-neutral-400">
                      {sourceLabel(p.source, t)} · {dateLabel} · {priceLabel}
                    </p>
                  </div>
                  <div className="flex shrink-0 flex-wrap items-center gap-2">
                    <Link
                      to={`/courses/${encodeURIComponent(p.courseCode)}`}
                      className="inline-flex items-center rounded-lg border border-slate-200 px-3 py-1.5 text-sm font-medium text-slate-700 hover:bg-slate-50 dark:border-neutral-600 dark:text-neutral-200 dark:hover:bg-neutral-800"
                    >
                      {t('marketplace.goToCourse')}
                    </Link>
                    {p.receiptUrl ? (
                      <Link
                        to="/me/billing"
                        className="inline-flex items-center gap-1 rounded-lg border border-slate-200 px-3 py-1.5 text-sm font-medium text-slate-700 hover:bg-slate-50 dark:border-neutral-600 dark:text-neutral-200 dark:hover:bg-neutral-800"
                      >
                        {t('purchases.receipt')}
                        <ExternalLink className="h-3.5 w-3.5" aria-hidden />
                      </Link>
                    ) : null}
                  </div>
                </li>
              )
            })}
          </ul>
        )}
      </div>
    </LmsPage>
  )
}
