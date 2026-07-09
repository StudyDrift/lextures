import { useEffect, useRef, useState } from 'react'
import { GraduationCap } from 'lucide-react'
import { CourseFilters, type CourseFilterState } from '../components/courses/course-filters'
import { CourseGrid } from '../components/courses/course-grid'
import { MarketingPageShell } from '../components/marketing-page-shell'
import { WindLines } from '../components/home/wind-lines'
import { COURSES_COPY } from '../lib/courses-copy'
import {
  fetchPublicMarketplaceCategories,
  MarketplaceApiError,
  searchPublicMarketplaceCourses,
  type MarketplaceCategory,
  type PublicMarketplaceCourse,
} from '../lib/marketplace-api'
import { useDocumentHead } from '../lib/use-document-head'

const SITE_ORIGIN = 'https://lextures.com'

function readFiltersFromUrl(): CourseFilterState {
  const params = new URLSearchParams(window.location.search)
  return {
    q: params.get('q') ?? '',
    category: params.get('category') ?? '',
    level: params.get('level') ?? '',
    language: params.get('language') ?? '',
    freeOnly: params.get('free_only') === 'true' || params.get('free_only') === '1',
    sort: params.get('sort') || 'popular',
  }
}

function writeFiltersToUrl(filters: CourseFilterState): void {
  const params = new URLSearchParams()
  if (filters.q) params.set('q', filters.q)
  if (filters.category) params.set('category', filters.category)
  if (filters.level) params.set('level', filters.level)
  if (filters.language) params.set('language', filters.language)
  if (filters.freeOnly) params.set('free_only', 'true')
  if (filters.sort && filters.sort !== 'popular') params.set('sort', filters.sort)
  const qs = params.toString()
  const next = qs ? `/courses?${qs}` : '/courses'
  window.history.replaceState(null, '', next)
}

export function CoursesPage() {
  useDocumentHead({
    title: COURSES_COPY.pageTitle,
    description: COURSES_COPY.pageDescription,
    canonical: `${SITE_ORIGIN}/courses`,
  })

  const [filters, setFilters] = useState<CourseFilterState>(() => readFiltersFromUrl())
  const [debouncedQ, setDebouncedQ] = useState(filters.q)
  const [categories, setCategories] = useState<MarketplaceCategory[]>([])
  const [courses, setCourses] = useState<PublicMarketplaceCourse[]>([])
  const [total, setTotal] = useState(0)
  const [nextCursor, setNextCursor] = useState('')
  const [loading, setLoading] = useState(true)
  const [loadingMore, setLoadingMore] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [unavailable, setUnavailable] = useState(false)
  const requestId = useRef(0)

  useEffect(() => {
    const t = window.setTimeout(() => setDebouncedQ(filters.q), 250)
    return () => window.clearTimeout(t)
  }, [filters.q])

  useEffect(() => {
    writeFiltersToUrl(filters)
  }, [filters])

  useEffect(() => {
    fetchPublicMarketplaceCategories()
      .then(setCategories)
      .catch(() => setCategories([]))
  }, [])

  useEffect(() => {
    window.gtag?.('event', 'courses_view')
  }, [])

  useEffect(() => {
    const id = ++requestId.current
    setLoading(true)
    setError(null)
    setUnavailable(false)
    searchPublicMarketplaceCourses({
      q: debouncedQ || undefined,
      category: filters.category || undefined,
      level: filters.level || undefined,
      language: filters.language || undefined,
      freeOnly: filters.freeOnly || undefined,
      sort: filters.sort || undefined,
      limit: 12,
    })
      .then(res => {
        if (id !== requestId.current) return
        setCourses(res.courses)
        setTotal(res.total)
        setNextCursor(res.nextCursor)
        setLoading(false)
        if (debouncedQ) window.gtag?.('event', 'courses_search', { q: debouncedQ })
        if (filters.category || filters.level || filters.freeOnly) {
          window.gtag?.('event', 'courses_filter')
        }
      })
      .catch((e: unknown) => {
        if (id !== requestId.current) return
        setLoading(false)
        if (e instanceof MarketplaceApiError && e.status === 404) {
          setUnavailable(true)
          setCourses([])
          setTotal(0)
          setNextCursor('')
          return
        }
        setError(e instanceof Error ? e.message : COURSES_COPY.errorBody)
      })
  }, [debouncedQ, filters.category, filters.level, filters.language, filters.freeOnly, filters.sort])

  const loadMore = () => {
    if (!nextCursor || loadingMore) return
    setLoadingMore(true)
    searchPublicMarketplaceCourses({
      q: debouncedQ || undefined,
      category: filters.category || undefined,
      level: filters.level || undefined,
      language: filters.language || undefined,
      freeOnly: filters.freeOnly || undefined,
      sort: filters.sort || undefined,
      cursor: nextCursor,
      limit: 12,
    })
      .then(res => {
        setCourses(prev => [...prev, ...res.courses])
        setNextCursor(res.nextCursor)
        setTotal(res.total)
      })
      .catch((e: unknown) => {
        setError(e instanceof Error ? e.message : COURSES_COPY.errorBody)
      })
      .finally(() => setLoadingMore(false))
  }

  return (
    <MarketingPageShell>
      <section className="relative overflow-hidden">
        <WindLines variant="hero" />
        <div
          className="relative z-[2] mx-auto max-w-[960px] px-5 py-14 md:px-10 md:py-16 xl:px-14"
          style={{ animation: 'lx-fade-up 0.7s ease both' }}
        >
          <span
            className="inline-flex items-center gap-2 rounded-full px-3.5 py-[7px] text-[13px] font-semibold uppercase tracking-[0.04em]"
            style={{ color: '#4fa894', backgroundColor: 'rgba(106,197,176,0.14)' }}
          >
            <GraduationCap className="h-3.5 w-3.5" aria-hidden />
            {COURSES_COPY.heroEyebrow}
          </span>
          <h1
            className="font-display mt-5 max-w-[720px] font-semibold leading-[1.05] tracking-[-0.02em]"
            style={{ color: '#22333b', fontSize: 'clamp(32px,4.4vw,48px)' }}
          >
            {COURSES_COPY.heroTitle}
          </h1>
          <p className="mt-4 max-w-[560px] text-[17px] leading-relaxed" style={{ color: '#4a5b5d' }}>
            {COURSES_COPY.heroLead}
          </p>
        </div>
      </section>

      <section className="mx-auto max-w-[1100px] px-5 pb-20 md:px-10 xl:px-14">
        <CourseFilters value={filters} categories={categories} onChange={setFilters} />
        <div className="mt-8">
          <CourseGrid
            courses={courses}
            total={total}
            loading={loading}
            error={error}
            unavailable={unavailable}
            nextCursor={nextCursor}
            onLoadMore={loadMore}
            onRetry={() => setFilters(f => ({ ...f }))}
            loadingMore={loadingMore}
          />
        </div>
      </section>
    </MarketingPageShell>
  )
}
