import { Star } from 'lucide-react'
import type { CourseReview, ReviewSummary } from '../../lib/course-reviews-api'

function RatingDistribution({ summary }: { summary: ReviewSummary }) {
  const total = summary.ratingCount
  if (total <= 0) return null
  return (
    <div className="space-y-1.5" aria-label="Rating distribution">
      {[5, 4, 3, 2, 1].map((stars) => {
        const count = summary.distribution[String(stars)] ?? 0
        const pct = total > 0 ? Math.round((count / total) * 100) : 0
        return (
          <div key={stars} className="flex items-center gap-2 text-xs text-fg-muted">
            <span className="w-8 tabular-nums">{stars}★</span>
            <div className="h-2 flex-1 overflow-hidden rounded-full bg-surface-sunken">
              <div
                className="h-full rounded-full bg-amber-400"
                style={{ width: `${pct}%` }}
              />
            </div>
            <span className="w-8 text-right tabular-nums">{count}</span>
          </div>
        )
      })}
    </div>
  )
}

function ReviewCard({ review }: { review: CourseReview }) {
  return (
    <article className="rounded-xl border border-border-default p-4 dark:border-border-subtle">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div>
          <p className="font-medium text-fg-default">
            {review.reviewerDisplayName}
          </p>
          <p className="text-xs text-fg-muted">
            {new Date(review.createdAt).toLocaleDateString()}
          </p>
        </div>
        <div className="flex items-center gap-0.5 text-amber-500" aria-label={`${review.rating} out of 5 stars`}>
          {Array.from({ length: 5 }, (_, i) => (
            <Star
              key={i}
              className={`h-4 w-4 ${i < review.rating ? 'fill-current' : 'text-slate-300 dark:text-neutral-600'}`}
              aria-hidden="true"
            />
          ))}
        </div>
      </div>
      {review.reviewText ? (
        <p className="mt-3 text-sm leading-relaxed text-fg-muted">
          {review.reviewText}
        </p>
      ) : null}
      {review.creatorResponse ? (
        <div className="mt-3 rounded-lg bg-surface-base p-3 text-sm dark:bg-surface-overlay">
          <p className="text-xs font-semibold uppercase tracking-wide text-fg-muted">
            Instructor response
          </p>
          <p className="mt-1 text-fg-muted">{review.creatorResponse}</p>
        </div>
      ) : null}
    </article>
  )
}

type CourseReviewsSectionProps = {
  summary: ReviewSummary
  reviews: CourseReview[]
  onWriteReview?: () => void
  showWriteCta?: boolean
}

export function CourseReviewsSection({
  summary,
  reviews,
  onWriteReview,
  showWriteCta = false,
}: CourseReviewsSectionProps) {
  const hasRatings = summary.ratingCount > 0

  return (
    <section className="mt-6 rounded-2xl border border-border-default bg-surface-raised p-6 dark:border-border-subtle dark:bg-surface-raised">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <h2 className="text-lg font-semibold text-fg-default">
            Learner reviews
          </h2>
          {hasRatings && summary.averageRating != null ? (
            <div className="mt-2 flex items-center gap-2">
              <span className="text-3xl font-bold text-fg-default">
                {summary.averageRating.toFixed(1)}
              </span>
              <div>
                <div className="flex text-amber-500">
                  {Array.from({ length: 5 }, (_, i) => (
                    <Star
                      key={i}
                      className={`h-4 w-4 ${ i < Math.round(summary.averageRating ?? 0) ? 'fill-current' : 'text-slate-300' }`}
                      aria-hidden="true"
                    />
                  ))}
                </div>
                <p className="text-xs text-fg-muted">
                  {summary.ratingCount.toLocaleString()} reviews
                </p>
              </div>
            </div>
          ) : (
            <p className="mt-2 text-sm text-fg-muted">
              No reviews yet — be the first!
            </p>
          )}
        </div>
        {showWriteCta && onWriteReview ? (
          <button
            type="button"
            onClick={onWriteReview}
            className="rounded-lg border border-indigo-200 bg-indigo-50 px-4 py-2 text-sm font-semibold text-accent-fg hover:bg-indigo-100 dark:border-indigo-900 dark:bg-indigo-950 dark:text-indigo-200"
          >
            Write a review
          </button>
        ) : null}
      </div>

      {hasRatings ? (
        <div className="mt-6 grid gap-6 md:grid-cols-[minmax(0,12rem)_1fr]">
          <RatingDistribution summary={summary} />
          <div className="space-y-3">
            {reviews.length === 0 ? (
              <p className="text-sm text-fg-muted">No written reviews yet.</p>
            ) : (
              reviews.map((r) => <ReviewCard key={r.id} review={r} />)
            )}
          </div>
        </div>
      ) : null}
    </section>
  )
}
