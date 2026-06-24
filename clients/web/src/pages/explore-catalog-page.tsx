import { useCallback, useEffect, useId, useRef, useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import { Search, Star } from 'lucide-react'
import {
  fetchPublicCatalogCategories,
  formatPrice,
  searchPublicCatalog,
  type PublicCatalogCategory,
  type PublicCatalogCourse,
} from '../lib/public-catalog-api'

const LEVELS = [
  { value: '', label: 'Any level' },
  { value: 'beginner', label: 'Beginner' },
  { value: 'intermediate', label: 'Intermediate' },
  { value: 'advanced', label: 'Advanced' },
] as const

const PRICES = [
  { value: 'any', label: 'Any price' },
  { value: 'free', label: 'Free' },
  { value: 'paid', label: 'Paid' },
] as const

const SORTS = [
  { value: 'popular', label: 'Most popular' },
  { value: 'rating', label: 'Highest rated' },
  { value: 'newest', label: 'Newest' },
  { value: 'relevance', label: 'Relevance' },
] as const

function priceParams(price: string): { priceMax?: number } {
  if (price === 'free') return { priceMax: 0 }
  return {}
}

function CourseCard({ course }: { course: PublicCatalogCourse }) {
  const priceId = useId()
  return (
    <article className="flex flex-col overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-sm transition-[box-shadow,background-color,color,border-color] hover:shadow-md dark:border-neutral-800 dark:bg-neutral-900">
      {course.heroImageUrl ? (
        <img
          src={course.heroImageUrl}
          alt=""
          className="lex-content-img h-40 w-full object-cover"
          loading="lazy"
        />
      ) : (
        <div className="h-40 w-full bg-gradient-to-br from-indigo-100 to-sky-100 dark:from-indigo-950 dark:to-sky-950" />
      )}
      <div className="flex flex-1 flex-col gap-2 p-4">
        <div className="flex items-center gap-2 text-xs text-slate-500 dark:text-neutral-400">
          {course.category ? <span>{course.category}</span> : null}
          {course.difficultyLevel ? (
            <span className="rounded-full bg-slate-100 px-2 py-0.5 capitalize dark:bg-neutral-800">
              {course.difficultyLevel}
            </span>
          ) : null}
        </div>
        <h2 className="text-base font-semibold text-slate-900 dark:text-neutral-100">
          <Link
            to={`/explore/${course.slug}`}
            className="after:absolute after:inset-0 focus:outline-none focus-visible:underline"
          >
            {course.title}
          </Link>
        </h2>
        {course.instructorName ? (
          <p className="text-sm text-slate-600 dark:text-neutral-300">{course.instructorName}</p>
        ) : null}
        <div className="mt-auto flex items-center justify-between pt-2 text-sm">
          <span className="flex items-center gap-1 text-amber-600 dark:text-amber-400">
            {course.averageRating != null ? (
              <>
                <Star className="h-4 w-4 fill-current" aria-hidden="true" />
                <span>{course.averageRating.toFixed(1)}</span>
              </>
            ) : (
              <span className="text-slate-400">Not yet rated</span>
            )}
            <span className="ms-2 text-slate-500 dark:text-neutral-400">
              {course.enrollmentCount.toLocaleString()} enrolled
            </span>
          </span>
          <span
            id={priceId}
            className="font-semibold text-slate-900 dark:text-neutral-100"
            data-testid="course-price"
          >
            {formatPrice(course.priceCents)}
          </span>
        </div>
      </div>
    </article>
  )
}

export default function ExploreCatalogPage() {
  const [params, setParams] = useSearchParams()
  const searchId = useId()
  const [courses, setCourses] = useState<PublicCatalogCourse[]>([])
  const [categories, setCategories] = useState<PublicCatalogCategory[]>([])
  const [total, setTotal] = useState(0)
  const [nextCursor, setNextCursor] = useState('')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [queryText, setQueryText] = useState(params.get('q') ?? '')

  const q = params.get('q') ?? ''
  const category = params.get('category') ?? ''
  const level = params.get('level') ?? ''
  const price = params.get('price') ?? 'any'
  const sort = params.get('sort') ?? 'popular'

  // Debounced search box → URL query param.
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
    void fetchPublicCatalogCategories()
      .then(setCategories)
      .catch(() => setCategories([]))
  }, [])

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
    let cancelled = false
    setLoading(true)
    setError(null)
    searchPublicCatalog({
      q,
      category,
      level,
      sort,
      ...priceParams(price),
    })
      .then((res) => {
        if (cancelled) return
        setCourses(res.courses)
        setTotal(res.total)
        setNextCursor(res.nextCursor)
      })
      .catch((e: unknown) => {
        if (cancelled) return
        setError(e instanceof Error ? e.message : 'Failed to load courses.')
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [q, category, level, price, sort])

  const loadMore = useCallback(() => {
    if (!nextCursor) return
    searchPublicCatalog({ q, category, level, sort, cursor: nextCursor, ...priceParams(price) })
      .then((res) => {
        setCourses((prev) => [...prev, ...res.courses])
        setNextCursor(res.nextCursor)
      })
      .catch(() => {
        /* keep existing results on pagination error */
      })
  }, [nextCursor, q, category, level, price, sort])

  return (
    <div className="min-h-screen bg-slate-50 dark:bg-neutral-950">
      <header className="border-b border-slate-200 bg-white px-4 py-6 dark:border-neutral-800 dark:bg-neutral-900">
        <div className="mx-auto max-w-6xl">
          <h1 className="text-2xl font-bold text-slate-900 dark:text-neutral-100">
            Explore courses
          </h1>
          <p className="mt-1 text-sm text-slate-600 dark:text-neutral-300">
            Browse the full Lextures catalog — no account required.
          </p>
          <div className="mt-4 flex items-center gap-2 rounded-xl border border-slate-300 bg-white px-3 py-2 dark:border-neutral-700 dark:bg-neutral-800">
            <Search className="h-5 w-5 text-slate-400" aria-hidden="true" />
            <label htmlFor={searchId} className="sr-only">
              Search courses
            </label>
            <input
              id={searchId}
              type="search"
              value={queryText}
              onChange={(e) => setQueryText(e.target.value)}
              placeholder="Search courses, e.g. Python programming"
              className="w-full bg-transparent text-sm text-slate-900 outline-none dark:text-neutral-100"
            />
          </div>
        </div>
      </header>

      <div className="mx-auto flex max-w-6xl flex-col gap-6 px-4 py-6 md:flex-row">
        <aside
          className="w-full shrink-0 md:w-60"
          aria-label="Catalog filters"
        >
          <fieldset className="mb-4">
            <legend className="mb-1 text-sm font-semibold text-slate-900 dark:text-neutral-100">
              Category
            </legend>
            <select
              value={category}
              onChange={(e) => updateParam('category', e.target.value)}
              className="w-full rounded-lg border border-slate-300 bg-white px-2 py-1.5 text-sm dark:border-neutral-700 dark:bg-neutral-800"
            >
              <option value="">All categories</option>
              {categories.map((c) => (
                <option key={c.category} value={c.category}>
                  {c.category} ({c.count})
                </option>
              ))}
            </select>
          </fieldset>

          <fieldset className="mb-4">
            <legend className="mb-1 text-sm font-semibold text-slate-900 dark:text-neutral-100">
              Level
            </legend>
            <select
              value={level}
              onChange={(e) => updateParam('level', e.target.value)}
              className="w-full rounded-lg border border-slate-300 bg-white px-2 py-1.5 text-sm dark:border-neutral-700 dark:bg-neutral-800"
            >
              {LEVELS.map((l) => (
                <option key={l.value} value={l.value}>
                  {l.label}
                </option>
              ))}
            </select>
          </fieldset>

          <fieldset className="mb-4">
            <legend className="mb-1 text-sm font-semibold text-slate-900 dark:text-neutral-100">
              Price
            </legend>
            <select
              value={price}
              onChange={(e) => updateParam('price', e.target.value === 'any' ? '' : e.target.value)}
              className="w-full rounded-lg border border-slate-300 bg-white px-2 py-1.5 text-sm dark:border-neutral-700 dark:bg-neutral-800"
            >
              {PRICES.map((p) => (
                <option key={p.value} value={p.value}>
                  {p.label}
                </option>
              ))}
            </select>
          </fieldset>
        </aside>

        <main className="flex-1">
          <div className="mb-4 flex items-center justify-between">
            <p className="text-sm text-slate-600 dark:text-neutral-300" aria-live="polite">
              {loading ? 'Loading…' : `${total.toLocaleString()} courses found`}
            </p>
            <div className="flex items-center gap-2">
              <label htmlFor="catalog-sort" className="text-sm text-slate-600 dark:text-neutral-300">
                Sort
              </label>
              <select
                id="catalog-sort"
                value={sort}
                onChange={(e) => updateParam('sort', e.target.value)}
                className="rounded-lg border border-slate-300 bg-white px-2 py-1.5 text-sm dark:border-neutral-700 dark:bg-neutral-800"
              >
                {SORTS.map((s) => (
                  <option key={s.value} value={s.value}>
                    {s.label}
                  </option>
                ))}
              </select>
            </div>
          </div>

          {error ? (
            <div role="alert" className="rounded-xl border border-red-200 bg-red-50 p-6 text-sm text-red-700 dark:border-red-900 dark:bg-red-950 dark:text-red-300">
              {error}
            </div>
          ) : loading ? (
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
              {Array.from({ length: 6 }).map((_, i) => (
                <div
                  key={i}
                  className="h-72 motion-safe:animate-pulse rounded-2xl border border-slate-200 bg-white dark:border-neutral-800 dark:bg-neutral-900"
                  data-testid="skeleton-card"
                />
              ))}
            </div>
          ) : courses.length === 0 ? (
            <div className="rounded-xl border border-slate-200 bg-white p-10 text-center dark:border-neutral-800 dark:bg-neutral-900">
              <p className="text-base font-medium text-slate-900 dark:text-neutral-100">
                No courses found
              </p>
              <p className="mt-1 text-sm text-slate-600 dark:text-neutral-300">
                Try a different search term or clear your filters.
              </p>
            </div>
          ) : (
            <>
              <ul className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
                {courses.map((course) => (
                  <li key={course.id} className="relative">
                    <CourseCard course={course} />
                  </li>
                ))}
              </ul>
              {nextCursor ? (
                <div className="mt-6 flex justify-center">
                  <button
                    type="button"
                    onClick={loadMore}
                    className="rounded-lg border border-slate-300 bg-white px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-50 dark:border-neutral-700 dark:bg-neutral-800 dark:text-neutral-200"
                  >
                    Load more
                  </button>
                </div>
              ) : null}
            </>
          )}
        </main>
      </div>
    </div>
  )
}
