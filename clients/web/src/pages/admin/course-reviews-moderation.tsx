import { useCallback, useEffect, useState } from 'react'
import { Star, Trash2 } from 'lucide-react'
import {
  fetchFlaggedReviews,
  removeReviewAdmin,
  type CourseReview,
} from '../../lib/course-reviews-api'
import { usePlatformFeatures } from '../../context/platform-features-context'

export default function CourseReviewsModerationPage() {
  const { ffCourseReviews, loading: featuresLoading } = usePlatformFeatures()
  const [reviews, setReviews] = useState<CourseReview[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [busyId, setBusyId] = useState<string | null>(null)

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      setReviews(await fetchFlaggedReviews())
    } catch {
      setError('Failed to load flagged reviews.')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    if (!featuresLoading && ffCourseReviews) void load()
  }, [featuresLoading, ffCourseReviews, load])

  async function handleRemove(reviewId: string) {
    setBusyId(reviewId)
    try {
      await removeReviewAdmin(reviewId)
      setReviews((prev) => prev.filter((r) => r.id !== reviewId))
    } catch {
      setError('Failed to remove review.')
    } finally {
      setBusyId(null)
    }
  }

  if (featuresLoading) {
    return <p className="p-8 text-sm text-slate-600">Loading…</p>
  }

  if (!ffCourseReviews) {
    return (
      <p className="p-8 text-sm text-slate-600 dark:text-neutral-300">
        Course reviews are not enabled on this platform.
      </p>
    )
  }

  return (
    <div className="mx-auto max-w-4xl p-6">
      <h1 className="text-2xl font-bold text-slate-900 dark:text-neutral-100">Review moderation</h1>
      <p className="mt-1 text-sm text-slate-600 dark:text-neutral-400">
        Flagged learner reviews awaiting admin action.
      </p>

      {error ? (
        <p role="alert" className="mt-4 text-sm text-red-600 dark:text-red-400">
          {error}
        </p>
      ) : null}

      {loading ? (
        <p className="mt-6 text-sm text-slate-600">Loading queue…</p>
      ) : reviews.length === 0 ? (
        <p className="mt-6 text-sm text-slate-600 dark:text-neutral-400">No flagged reviews.</p>
      ) : (
        <ul className="mt-6 space-y-4">
          {reviews.map((review) => (
            <li
              key={review.id}
              className="rounded-xl border border-slate-200 p-4 dark:border-neutral-800 dark:bg-neutral-900"
            >
              <div className="flex flex-wrap items-start justify-between gap-3">
                <div>
                  <p className="font-medium text-slate-900 dark:text-neutral-100">
                    {review.reviewerDisplayName}
                  </p>
                  <div className="mt-1 flex items-center gap-1 text-amber-500">
                    {Array.from({ length: review.rating }, (_, i) => (
                      <Star key={i} className="h-4 w-4 fill-current" aria-hidden="true" />
                    ))}
                  </div>
                  {review.reviewText ? (
                    <p className="mt-2 text-sm text-slate-700 dark:text-neutral-300">
                      {review.reviewText}
                    </p>
                  ) : null}
                </div>
                <button
                  type="button"
                  disabled={busyId === review.id}
                  onClick={() => void handleRemove(review.id)}
                  className="inline-flex items-center gap-1 rounded-lg border border-red-200 px-3 py-1.5 text-sm text-red-700 hover:bg-red-50 dark:border-red-900 dark:text-red-300 dark:hover:bg-red-950"
                >
                  <Trash2 className="h-4 w-4" aria-hidden="true" />
                  Remove
                </button>
              </div>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}
