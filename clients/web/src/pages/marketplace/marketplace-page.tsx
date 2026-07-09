import { useCallback, useEffect, useId, useRef, useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Search, Star, Store } from 'lucide-react'
import { CourseHeroImage } from '../../components/course-hero-image'
import { usePlatformFeatures } from '../../context/platform-features-context'
import {
  fetchMarketplaceCategories,
  marketplaceCardAccessibleName,
  searchMarketplaceCourses,
  type MarketplaceCard,
  type MarketplaceCategory,
} from '../../lib/marketplace-api'
import { EmptyState } from '../../components/ui/empty-state'
import { LmsPage } from '../lms/lms-page'
import { MarketplacePriceBadge } from './marketplace-price-badge'

const LEVELS = [
  { value: '', labelKey: 'marketplace.filters.anyLevel' },
  { value: 'beginner', labelKey: 'marketplace.filters.beginner' },
  { value: 'intermediate', labelKey: 'marketplace.filters.intermediate' },
  { value: 'advanced', labelKey: 'marketplace.filters.advanced' },
] as const

const PRICES = [
  { value: 'any', labelKey: 'marketplace.filters.anyPrice' },
  { value: 'free', labelKey: 'marketplace.filters.freeOnly' },
] as const

const SORTS = [
  { value: 'popular', labelKey: 'marketplace.sort.popular' },
  { value: 'newest', labelKey: 'marketplace.sort.newest' },
  { value: 'price', labelKey: 'marketplace.sort.price' },
  { value: 'rating', labelKey: 'marketplace.sort.rating' },
] as const

const selectClassName =
  'mt-1 w-full rounded-xl border border-slate-200 bg-white px-3 py-2 text-sm text-slate-900 shadow-sm outline-none focus:border-indigo-400 focus:ring-2 focus:ring-indigo-400/30 dark:border-neutral-700 dark:bg-neutral-950 dark:text-neutral-100'

function MarketplaceCourseCard({
  course,
  freeLabel,
  ownedLabel,
  locale,
}: {
  course: MarketplaceCard
  freeLabel: string
  ownedLabel: string
  locale?: string
}) {
  const accessibleName = marketplaceCardAccessibleName(course, freeLabel, ownedLabel, locale)
  return (
    <article className="relative flex flex-col overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-sm motion-safe:transition-[box-shadow,border-color] hover:border-indigo-200 hover:shadow-md dark:border-neutral-700 dark:bg-neutral-900 dark:hover:border-indigo-800">
      {course.heroImageUrl ? (
        <CourseHeroImage
          src={course.heroImageUrl}
          alt=""
          className="h-40 w-full object-cover"
          loading="lazy"
        />
      ) : (
        <div className="h-40 w-full bg-gradient-to-br from-indigo-100 to-sky-100 dark:from-indigo-950 dark:to-sky-950" />
      )}
      <div className="flex flex-1 flex-col gap-2 p-4">
        <div className="flex flex-wrap items-center gap-2 text-xs text-slate-500 dark:text-neutral-400">
          {course.category ? <span>{course.category}</span> : null}
          {course.level ? (
            <span className="rounded-full bg-slate-100 px-2 py-0.5 capitalize dark:bg-neutral-800">
              {course.level}
            </span>
          ) : null}
          {course.owned ? (
            <span
              className="rounded-full bg-emerald-50 px-2 py-0.5 font-medium text-emerald-800 dark:bg-emerald-950/40 dark:text-emerald-200"
              data-testid="marketplace-owned-badge"
            >
              {ownedLabel}
            </span>
          ) : null}
        </div>
        <h2 className="text-base font-semibold text-slate-900 dark:text-neutral-100">
          <Link
            to={`/marketplace/${encodeURIComponent(course.slug || course.courseCode)}`}
            className="after:absolute after:inset-0 focus:outline-none focus-visible:underline"
            aria-label={accessibleName}
            data-testid="marketplace-course-card"
          >
            {course.title}
          </Link>
        </h2>
        {course.instructorName ? (
          <p className="text-sm text-slate-600 dark:text-neutral-300">{course.instructorName}</p>
        ) : null}
        <div className="mt-auto flex items-center justify-between gap-2 pt-2 text-sm">
          <span className="flex items-center gap-1 text-amber-600 dark:text-amber-400">
            {course.averageRating != null ? (
              <>
                <Star className="h-4 w-4 fill-current" aria-hidden="true" />
                <span>{course.averageRating.toFixed(1)}</span>
              </>
            ) : (
              <span className="text-slate-400 dark:text-neutral-500">—</span>
            )}
            <span className="ms-2 text-slate-500 dark:text-neutral-400">
              {course.enrollmentCount.toLocaleString()}
            </span>
          </span>
          <MarketplacePriceBadge
            priceCents={course.priceCents}
            priceCurrency={course.priceCurrency}
            listPriceCents={course.listPriceCents}
            freeLabel={freeLabel}
            locale={locale}
          />
        </div>
      </div>
    </article>
  )
}

export default function MarketplacePage() {
  const { t, i18n } = useTranslation('common')
  const { ffCourseMarketplace, loading: featuresLoading } = usePlatformFeatures()
  const [params, setParams] = useSearchParams()
  const searchId = useId()
  const [courses, setCourses] = useState<MarketplaceCard[]>([])
  const [categories, setCategories] = useState<MarketplaceCategory[]>([])
  const [total, setTotal] = useState(0)
  const [nextCursor, setNextCursor] = useState('')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [queryText, setQueryText] = useState(params.get('q') ?? '')
  const [retryToken, setRetryToken] = useState(0)

  const q = params.get('q') ?? ''
  const category = params.get('category') ?? ''
  const level = params.get('level') ?? ''
  const price = params.get('price') ?? 'any'
  const sort = params.get('sort') ?? 'popular'

  const debounceRef = useRef<number | undefined>(undefined)
  useEffect(() => {
    window.clearTimeout(debounceRef.current)
    debounceRef.current = window.setTimeout(() => {
      setParams(
        (prev) => {
          const next = new URLSearchParams(prev)
          if (queryText) next.set('q', queryText)
          else next.delete('q')
          return next
        },
        { replace: true },
      )
    }, 300)
    return () => window.clearTimeout(debounceRef.current)
  }, [queryText, setParams])

  useEffect(() => {
    if (featuresLoading || !ffCourseMarketplace) return
    void fetchMarketplaceCategories()
      .then(setCategories)
      .catch(() => setCategories([]))
  }, [ffCourseMarketplace, featuresLoading])

  const updateParam = useCallback(
    (key: string, value: string) => {
      setParams(
        (prev) => {
          const next = new URLSearchParams(prev)
          if (value) next.set(key, value)
          else next.delete(key)
          return next
        },
        { replace: true },
      )
    },
    [setParams],
  )

  useEffect(() => {
    if (featuresLoading || !ffCourseMarketplace) {
      setLoading(false)
      return
    }
    let cancelled = false
    setLoading(true)
    setError(null)
    searchMarketplaceCourses({
      q,
      category,
      level,
      sort,
      freeOnly: price === 'free',
    })
      .then((res) => {
        if (cancelled) return
        setCourses(res.courses)
        setTotal(res.total)
        setNextCursor(res.nextCursor)
      })
      .catch((e: unknown) => {
        if (cancelled) return
        setError(e instanceof Error ? e.message : t('marketplace.error'))
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [q, category, level, price, sort, ffCourseMarketplace, featuresLoading, retryToken, t])

  const loadMore = useCallback(() => {
    if (!nextCursor) return
    searchMarketplaceCourses({
      q,
      category,
      level,
      sort,
      cursor: nextCursor,
      freeOnly: price === 'free',
    })
      .then((res) => {
        setCourses((prev) => [...prev, ...res.courses])
        setNextCursor(res.nextCursor)
      })
      .catch(() => {
        /* keep existing results on pagination error */
      })
  }, [nextCursor, q, category, level, price, sort])

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

  const freeLabel = t('marketplace.free')
  const ownedLabel = t('marketplace.owned')

  return (
    <LmsPage title={t('marketplace.title')} description={t('marketplace.subtitle')}>
      <form
        role="search"
        aria-label={t('marketplace.filters.label')}
        className="mb-6 rounded-2xl border border-slate-200 bg-white p-4 shadow-sm dark:border-neutral-700 dark:bg-neutral-900"
        onSubmit={(e) => e.preventDefault()}
      >
        <div className="flex flex-wrap items-end gap-3">
          <div className="min-w-[200px] flex-1">
            <label
              htmlFor={searchId}
              className="text-sm font-medium text-slate-700 dark:text-neutral-200"
            >
              {t('marketplace.searchPlaceholder')}
            </label>
            <div className="relative mt-1">
              <Search
                className="absolute start-3 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-400"
                aria-hidden
              />
              <input
                id={searchId}
                type="search"
                value={queryText}
                onChange={(e) => setQueryText(e.target.value)}
                placeholder={t('marketplace.searchPlaceholder')}
                className="w-full rounded-xl border border-slate-200 py-2 ps-9 pe-3 text-sm text-slate-900 outline-none focus:border-indigo-400 focus:ring-2 focus:ring-indigo-400/30 dark:border-neutral-600 dark:bg-neutral-800 dark:text-neutral-100"
                data-testid="marketplace-search"
              />
            </div>
          </div>

          <div className="min-w-36">
            <label
              htmlFor="marketplace-filter-category"
              className="text-sm font-medium text-slate-700 dark:text-neutral-200"
            >
              {t('marketplace.filters.category')}
            </label>
            <select
              id="marketplace-filter-category"
              value={category}
              onChange={(e) => updateParam('category', e.target.value)}
              className={selectClassName}
              data-testid="marketplace-filter-category"
            >
              <option value="">{t('marketplace.filters.allCategories')}</option>
              {categories.map((c) => (
                <option key={c.category} value={c.category}>
                  {c.category} ({c.count})
                </option>
              ))}
            </select>
          </div>

          <div className="min-w-32">
            <label
              htmlFor="marketplace-filter-level"
              className="text-sm font-medium text-slate-700 dark:text-neutral-200"
            >
              {t('marketplace.filters.level')}
            </label>
            <select
              id="marketplace-filter-level"
              value={level}
              onChange={(e) => updateParam('level', e.target.value)}
              className={selectClassName}
              data-testid="marketplace-filter-level"
            >
              {LEVELS.map((l) => (
                <option key={l.value} value={l.value}>
                  {t(l.labelKey)}
                </option>
              ))}
            </select>
          </div>

          <div className="min-w-32">
            <label
              htmlFor="marketplace-filter-price"
              className="text-sm font-medium text-slate-700 dark:text-neutral-200"
            >
              {t('marketplace.filters.price')}
            </label>
            <select
              id="marketplace-filter-price"
              value={price}
              onChange={(e) => updateParam('price', e.target.value === 'any' ? '' : e.target.value)}
              className={selectClassName}
              data-testid="marketplace-filter-price"
            >
              {PRICES.map((p) => (
                <option key={p.value} value={p.value}>
                  {t(p.labelKey)}
                </option>
              ))}
            </select>
          </div>

          <div className="min-w-40">
            <label
              htmlFor="marketplace-sort"
              className="text-sm font-medium text-slate-700 dark:text-neutral-200"
            >
              {t('marketplace.sort.label')}
            </label>
            <select
              id="marketplace-sort"
              value={sort}
              onChange={(e) => updateParam('sort', e.target.value)}
              className={selectClassName}
              data-testid="marketplace-sort"
            >
              {SORTS.map((s) => (
                <option key={s.value} value={s.value}>
                  {t(s.labelKey)}
                </option>
              ))}
            </select>
          </div>
        </div>
      </form>

      <p className="mb-4 text-sm text-slate-600 dark:text-neutral-400" aria-live="polite">
        {loading ? t('marketplace.loading') : t('marketplace.resultsCount', { count: total })}
      </p>

      {error ? (
        <div
          role="alert"
          className="rounded-xl border border-rose-200 bg-rose-50 px-4 py-3 text-sm text-rose-800 dark:border-rose-900/40 dark:bg-rose-950/30 dark:text-rose-100"
          data-testid="marketplace-error"
        >
          <p>{error}</p>
          <button
            type="button"
            className="mt-3 rounded-xl border border-slate-200 bg-white px-3 py-1.5 text-sm font-semibold text-slate-700 shadow-sm hover:bg-slate-50 dark:border-neutral-700 dark:bg-neutral-900 dark:text-neutral-200 dark:hover:bg-neutral-800"
            onClick={() => setRetryToken((n) => n + 1)}
            data-testid="marketplace-retry"
          >
            {t('marketplace.retry')}
          </button>
        </div>
      ) : loading ? (
        <div
          className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3"
          aria-busy="true"
          aria-label={t('marketplace.loading')}
        >
          {Array.from({ length: 6 }).map((_, i) => (
            <div
              key={i}
              className="h-72 motion-safe:animate-pulse rounded-2xl bg-slate-100 dark:bg-neutral-800"
              data-testid="marketplace-skeleton"
            />
          ))}
        </div>
      ) : courses.length === 0 ? (
        <EmptyState
          icon={Store}
          title={t('marketplace.emptyTitle')}
          body={t('marketplace.emptyBody')}
        />
      ) : (
        <>
          <ul className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {courses.map((course) => (
              <li key={course.id}>
                <MarketplaceCourseCard
                  course={course}
                  freeLabel={freeLabel}
                  ownedLabel={ownedLabel}
                  locale={i18n.language}
                />
              </li>
            ))}
          </ul>
          {nextCursor ? (
            <div className="mt-6 flex justify-center">
              <button
                type="button"
                onClick={loadMore}
                className="rounded-xl border border-slate-200 bg-white px-4 py-2 text-sm font-semibold text-slate-700 shadow-sm hover:bg-slate-50 dark:border-neutral-700 dark:bg-neutral-900 dark:text-neutral-200 dark:hover:bg-neutral-800"
                data-testid="marketplace-load-more"
              >
                {t('marketplace.loadMore')}
              </button>
            </div>
          ) : null}
        </>
      )}
    </LmsPage>
  )
}
