import { MessageSquare } from 'lucide-react'

type ReviewPromptBannerProps = {
  progressPercent: number
  onWriteReview: () => void
  onDismiss: () => void
}

export function ReviewPromptBanner({
  progressPercent,
  onWriteReview,
  onDismiss,
}: ReviewPromptBannerProps) {
  const atCompletion = progressPercent >= 100
  const message = atCompletion
    ? 'You finished the course! Share your experience to help other learners.'
    : 'You are making great progress. Would you leave a quick review?'

  return (
    <div
      role="status"
      className="mb-4 flex flex-col gap-3 rounded-xl border border-indigo-200 bg-indigo-50 p-4 sm:flex-row sm:items-center sm:justify-between dark:border-indigo-900 dark:bg-indigo-950/40"
    >
      <div className="flex items-start gap-3">
        <MessageSquare className="mt-0.5 h-5 w-5 shrink-0 text-indigo-600 dark:text-indigo-300" aria-hidden="true" />
        <div>
          <p className="text-sm font-medium text-indigo-900 dark:text-indigo-100">{message}</p>
          <p className="text-xs text-indigo-700/80 dark:text-indigo-300/80">
            {progressPercent}% complete
          </p>
        </div>
      </div>
      <div className="flex gap-2">
        <button
          type="button"
          onClick={onDismiss}
          className="rounded-lg px-3 py-1.5 text-sm text-indigo-700 hover:bg-indigo-100 dark:text-indigo-200 dark:hover:bg-indigo-900/50"
        >
          Not now
        </button>
        <button
          type="button"
          onClick={onWriteReview}
          className="rounded-lg bg-indigo-600 px-3 py-1.5 text-sm font-semibold text-white hover:bg-indigo-500"
        >
          Write a review
        </button>
      </div>
    </div>
  )
}
