import { useState } from 'react'
import { RatingStars } from './course-card'
import { COURSES_COPY } from '../../lib/courses-copy'
import { formatDate } from '../../lib/format-date'
import type { CourseReview } from '../../lib/marketplace-api'

const INITIAL = 5

type ReviewListProps = {
  reviews: CourseReview[]
  average: number | null
  count: number
}

export function ReviewList({ reviews, average, count }: ReviewListProps) {
  const [expanded, setExpanded] = useState(false)
  const visible = expanded ? reviews : reviews.slice(0, INITIAL)

  return (
    <section aria-labelledby="reviews-heading">
      <div className="flex flex-wrap items-baseline justify-between gap-3">
        <h2
          id="reviews-heading"
          className="font-display text-[22px] font-semibold"
          style={{ color: 'var(--ink-nav)' }}
        >
          {COURSES_COPY.reviews}
        </h2>
        <RatingStars average={average} count={count} />
      </div>

      {reviews.length === 0 ? (
        <p className="mt-4 text-[15px]" style={{ color: 'var(--text-soft)' }}>
          {COURSES_COPY.noReviews}
        </p>
      ) : (
        <>
          <ul className="mt-4 flex flex-col gap-4">
            {visible.map(r => (
              <li
                key={r.id}
                className="border p-4"
                style={{
                  borderColor: 'var(--line-card)',
                  borderRadius: 'var(--radius-card)',
                  backgroundColor: 'var(--panel)',
                }}
              >
                <div className="flex flex-wrap items-center justify-between gap-2">
                  <span className="text-[14px] font-semibold" style={{ color: 'var(--ink-nav)' }}>
                    {r.reviewerDisplayName}
                  </span>
                  <time
                    dateTime={r.createdAt}
                    className="text-[12px]"
                    style={{ color: 'var(--text-soft)' }}
                  >
                    {formatDate(r.createdAt, { year: 'numeric', month: 'short', day: 'numeric' })}
                  </time>
                </div>
                <div className="mt-1">
                  <RatingStars average={r.rating} count={0} />
                </div>
                {r.reviewText && (
                  <p className="mt-2 text-[14px] leading-relaxed" style={{ color: 'var(--text)' }}>
                    {r.reviewText}
                  </p>
                )}
              </li>
            ))}
          </ul>
          {!expanded && reviews.length > INITIAL && (
            <button
              type="button"
              onClick={() => setExpanded(true)}
              className="btn-secondary mt-4"
            >
              {COURSES_COPY.showMoreReviews}
            </button>
          )}
        </>
      )}
    </section>
  )
}
