import { CourseCard } from './course-card'
import { COURSES_COPY } from '../../lib/courses-copy'
import type { PublicMarketplaceCourse } from '../../lib/marketplace-api'

type CourseGridProps = {
  courses: PublicMarketplaceCourse[]
  total: number
  loading: boolean
  error: string | null
  unavailable: boolean
  nextCursor: string
  onLoadMore: () => void
  onRetry: () => void
  loadingMore?: boolean
}

export function CourseGrid({
  courses,
  total,
  loading,
  error,
  unavailable,
  nextCursor,
  onLoadMore,
  onRetry,
  loadingMore,
}: CourseGridProps) {
  if (loading && courses.length === 0) {
    return (
      <ul
        className="grid grid-cols-1 gap-5 sm:grid-cols-2 lg:grid-cols-3"
        aria-busy="true"
        aria-label="Loading courses"
      >
        {Array.from({ length: 6 }).map((_, i) => (
          <li
            key={i}
            className="aspect-[4/5] animate-pulse border"
            style={{
              backgroundColor: 'var(--panel)',
              borderColor: 'var(--line-card)',
              borderRadius: 'var(--radius-card)',
            }}
          />
        ))}
      </ul>
    )
  }

  if (error) {
    return (
      <div
        className="rounded-lg border px-6 py-10 text-center"
        style={{ borderColor: 'var(--line-card)', backgroundColor: 'var(--panel)' }}
        role="alert"
      >
        <h2 className="font-display text-xl font-semibold" style={{ color: 'var(--ink-nav)' }}>
          {COURSES_COPY.errorTitle}
        </h2>
        <p className="mt-2 text-[15px]" style={{ color: 'var(--text-soft)' }}>
          {error || COURSES_COPY.errorBody}
        </p>
        <button type="button" onClick={onRetry} className="btn-primary mt-6">
          {COURSES_COPY.retry}
        </button>
      </div>
    )
  }

  if (unavailable) {
    return (
      <div
        className="rounded-lg border px-6 py-10 text-center"
        style={{ borderColor: 'var(--line-card)', backgroundColor: 'var(--panel)' }}
      >
        <h2 className="font-display text-xl font-semibold" style={{ color: 'var(--ink-nav)' }}>
          {COURSES_COPY.unavailableTitle}
        </h2>
        <p className="mt-2 text-[15px]" style={{ color: 'var(--text-soft)' }}>
          {COURSES_COPY.unavailableBody}
        </p>
      </div>
    )
  }

  if (courses.length === 0) {
    return (
      <div
        className="rounded-lg border px-6 py-10 text-center"
        style={{ borderColor: 'var(--line-card)', backgroundColor: 'var(--panel)' }}
      >
        <h2 className="font-display text-xl font-semibold" style={{ color: 'var(--ink-nav)' }}>
          {COURSES_COPY.emptyTitle}
        </h2>
        <p className="mt-2 text-[15px]" style={{ color: 'var(--text-soft)' }}>
          {COURSES_COPY.emptyBody}
        </p>
      </div>
    )
  }

  return (
    <div>
      <p className="mb-4 text-[14px]" style={{ color: 'var(--text-soft)' }} aria-live="polite">
        {COURSES_COPY.resultsCount(total)}
      </p>
      <ul className="grid grid-cols-1 gap-5 sm:grid-cols-2 lg:grid-cols-3">
        {courses.map(c => (
          <CourseCard key={c.id} course={c} />
        ))}
      </ul>
      {nextCursor && (
        <div className="mt-8 flex justify-center">
          <button
            type="button"
            onClick={onLoadMore}
            disabled={loadingMore}
            className="btn-secondary"
          >
            {loadingMore ? 'Loading…' : COURSES_COPY.loadMore}
          </button>
        </div>
      )}
    </div>
  )
}
