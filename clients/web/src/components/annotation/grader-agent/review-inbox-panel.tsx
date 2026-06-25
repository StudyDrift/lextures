import { useTranslation } from 'react-i18next'
import type { GraderAgentReviewQueueItem } from '../../../lib/courses-api'
import { HeldReviewQueuePanel } from './held-review-queue-panel'
import { ReviewQueuePanel } from './review-queue-panel'

type ReviewInboxPanelProps = {
  courseCode: string
  itemId: string
  held: GraderAgentReviewQueueItem[]
  flagged: GraderAgentReviewQueueItem[]
  suggestModeEnabled?: boolean
  loading?: boolean
  error?: string | null
  onUpdated?: () => void
  onOpenSubmission?: (submissionId: string) => void
}

export function ReviewInboxPanel({
  courseCode,
  itemId,
  held,
  flagged,
  suggestModeEnabled = false,
  loading = false,
  error = null,
  onUpdated,
  onOpenSubmission,
}: ReviewInboxPanelProps) {
  const { t } = useTranslation('common')

  if (loading && held.length === 0 && flagged.length === 0) {
    return (
      <div
        className="mt-4 rounded-xl border border-slate-200 bg-slate-50/70 p-4 text-sm text-slate-600 dark:border-neutral-700 dark:bg-neutral-900/40 dark:text-neutral-300"
        aria-busy="true"
      >
        {t('gradingAgent.review.inbox.loading')}
      </div>
    )
  }

  if (error && held.length === 0 && flagged.length === 0) {
    return (
      <p className="mt-4 text-sm text-rose-700 dark:text-rose-300" role="alert">
        {error}
      </p>
    )
  }

  if (!loading && held.length === 0 && flagged.length === 0) {
    return (
      <p className="mt-4 text-sm text-slate-600 dark:text-neutral-400" aria-live="polite">
        {t('gradingAgent.review.inbox.empty')}
      </p>
    )
  }

  return (
    <div className="mt-4 space-y-4" aria-live="polite">
      {error ? (
        <p className="text-sm text-rose-700 dark:text-rose-300" role="alert">
          {error}
        </p>
      ) : null}
      <ReviewQueuePanel
        items={flagged}
        courseCode={courseCode}
        itemId={itemId}
        onUpdated={onUpdated}
        onOpenSubmission={onOpenSubmission}
      />
      <HeldReviewQueuePanel
        courseCode={courseCode}
        itemId={itemId}
        items={held}
        suggestModeEnabled={suggestModeEnabled}
        onUpdated={onUpdated}
        onOpenSubmission={onOpenSubmission}
      />
    </div>
  )
}