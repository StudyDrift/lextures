import { useId, useState } from 'react'
import { Star, X } from 'lucide-react'
import { STAR_LABELS, submitCourseReview } from '../../lib/course-reviews-api'

type ReviewFormProps = {
  courseCode: string
  initialRating?: number
  initialText?: string
  onClose: () => void
  onSubmitted: () => void
}

export function ReviewForm({
  courseCode,
  initialRating = 0,
  initialText = '',
  onClose,
  onSubmitted,
}: ReviewFormProps) {
  const titleId = useId()
  const errorId = useId()
  const [rating, setRating] = useState(initialRating)
  const [text, setText] = useState(initialText)
  const [error, setError] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)
  const [thanks, setThanks] = useState(false)

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (rating < 1 || rating > 5) {
      setError('Please select a star rating.')
      return
    }
    setSubmitting(true)
    setError(null)
    try {
      await submitCourseReview(courseCode, {
        rating,
        reviewText: text.trim() || undefined,
      })
      setThanks(true)
      onSubmitted()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to submit review.')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-labelledby={titleId}
      className="fixed inset-0 z-50 flex items-end justify-center bg-slate-900/60 p-4 sm:items-center"
    >
      <div className="w-full max-w-lg rounded-2xl border border-slate-200 bg-white p-6 shadow-xl dark:border-neutral-700 dark:bg-neutral-900">
        <div className="mb-4 flex items-start justify-between gap-3">
          <h2 id={titleId} className="text-lg font-semibold text-slate-900 dark:text-neutral-100">
            {thanks ? 'Thanks for your review!' : 'Write a review'}
          </h2>
          <button
            type="button"
            onClick={onClose}
            className="rounded-lg p-1 text-slate-500 hover:bg-slate-100 dark:hover:bg-neutral-800"
            aria-label="Close"
          >
            <X className="h-5 w-5" aria-hidden="true" />
          </button>
        </div>

        {thanks ? (
          <p className="text-sm text-slate-600 dark:text-neutral-300">
            Your feedback helps other learners choose the right course.
          </p>
        ) : (
          <form onSubmit={(e) => void handleSubmit(e)} className="space-y-4">
            <fieldset>
              <legend className="mb-2 text-sm font-medium text-slate-700 dark:text-neutral-200">
                Your rating
              </legend>
              <div role="radiogroup" aria-label="Course rating" className="flex flex-wrap gap-2">
                {[1, 2, 3, 4, 5].map((value) => (
                  <label
                    key={value}
                    className={`flex min-h-11 min-w-11 cursor-pointer items-center justify-center rounded-xl border px-3 py-2 text-sm font-medium transition-[background-color,color,border-color] ${
                      rating === value
                        ? 'border-amber-500 bg-amber-50 text-amber-800 dark:border-amber-400 dark:bg-amber-950 dark:text-amber-200'
                        : 'border-slate-200 text-slate-600 hover:border-slate-300 dark:border-neutral-700 dark:text-neutral-300'
                    }`}
                  >
                    <input
                      type="radio"
                      name="rating"
                      value={value}
                      checked={rating === value}
                      onChange={() => setRating(value)}
                      className="sr-only"
                    />
                    <Star
                      className={`mr-1 h-4 w-4 ${rating >= value ? 'fill-amber-500 text-amber-500' : ''}`}
                      aria-hidden="true"
                    />
                    <span className="sr-only">{STAR_LABELS[value]}</span>
                    <span aria-hidden="true">{value}</span>
                  </label>
                ))}
              </div>
            </fieldset>

            <div>
              <label
                htmlFor="review-text"
                className="mb-1 block text-sm font-medium text-slate-700 dark:text-neutral-200"
              >
                Review (optional)
              </label>
              <textarea
                id="review-text"
                value={text}
                onChange={(e) => setText(e.target.value)}
                maxLength={2000}
                rows={4}
                aria-describedby={error ? errorId : undefined}
                className="w-full rounded-xl border border-slate-200 px-3 py-2 text-sm dark:border-neutral-700 dark:bg-neutral-950"
                placeholder="What did you like or dislike?"
              />
            </div>

            {error ? (
              <p id={errorId} role="alert" className="text-sm text-red-600 dark:text-red-400">
                {error}
              </p>
            ) : null}

            <div className="flex justify-end gap-2">
              <button
                type="button"
                onClick={onClose}
                className="rounded-lg px-4 py-2 text-sm text-slate-600 hover:bg-slate-100 dark:text-neutral-300 dark:hover:bg-neutral-800"
              >
                Cancel
              </button>
              <button
                type="submit"
                disabled={submitting}
                className="rounded-lg bg-indigo-600 px-4 py-2 text-sm font-semibold text-white hover:bg-indigo-500 disabled:opacity-60"
              >
                {submitting ? 'Submitting…' : 'Submit review'}
              </button>
            </div>
          </form>
        )}
      </div>
    </div>
  )
}
