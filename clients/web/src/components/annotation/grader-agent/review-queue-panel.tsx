import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { patchGraderAgentResult, putSubmissionGrade, type GraderAgentReviewQueueItem } from '../../../lib/courses-api'

type ReviewQueuePanelProps = {
  courseCode: string
  itemId: string
  items: GraderAgentReviewQueueItem[]
  onUpdated?: () => void
  onOpenSubmission?: (submissionId: string) => void
}

export function ReviewQueuePanel({
  courseCode,
  itemId,
  items,
  onUpdated,
  onOpenSubmission,
}: ReviewQueuePanelProps) {
  const { t } = useTranslation('common')
  const [busyId, setBusyId] = useState<string | null>(null)
  const [editingId, setEditingId] = useState<string | null>(null)
  const [editScore, setEditScore] = useState('')
  const [editComment, setEditComment] = useState('')
  const [error, setError] = useState<string | null>(null)

  const flagged = useMemo(() => items.filter((result) => result.status === 'flagged'), [items])

  if (flagged.length === 0) return null

  const beginGrade = (item: GraderAgentReviewQueueItem) => {
    setEditingId(item.id)
    setEditScore(item.suggestedPoints != null ? String(item.suggestedPoints) : '')
    setEditComment(item.comment ?? '')
    setError(null)
  }

  const gradeNow = async (item: GraderAgentReviewQueueItem) => {
    setBusyId(item.id)
    setError(null)
    try {
      const points = Number(editScore)
      if (!Number.isFinite(points)) {
        setError(t('gradingAgent.review.queue.invalidScore'))
        return
      }
      await putSubmissionGrade(courseCode, itemId, item.submissionId, {
        pointsEarned: points,
        instructorComment: editComment.trim() || null,
        gradedByAi: true,
      })
      const status =
        item.suggestedPoints != null && points === item.suggestedPoints ? 'applied' : 'overridden'
      await patchGraderAgentResult(courseCode, itemId, item.id, { status })
      setEditingId(null)
      onUpdated?.()
    } catch (err) {
      setError(err instanceof Error ? err.message : t('gradingAgent.review.flagged.actionFailed'))
    } finally {
      setBusyId(null)
    }
  }

  const dismiss = async (item: GraderAgentReviewQueueItem) => {
    const reason = window.prompt(t('gradingAgent.review.flagged.dismissPrompt'))
    if (reason == null) return
    setBusyId(item.id)
    setError(null)
    try {
      await patchGraderAgentResult(courseCode, itemId, item.id, {
        status: 'skipped',
        reason: reason.trim() || t('gradingAgent.review.flagged.dismissDefault'),
      })
      onUpdated?.()
    } catch (err) {
      setError(err instanceof Error ? err.message : t('gradingAgent.review.flagged.actionFailed'))
    } finally {
      setBusyId(null)
    }
  }

  return (
    <section
      className="rounded-xl border border-rose-200 bg-rose-50/50 p-4 dark:border-rose-900/50 dark:bg-rose-950/20"
      aria-label={t('gradingAgent.review.flagged.title')}
    >
      <div className="flex items-center justify-between gap-3">
        <h3 className="text-sm font-semibold text-rose-900 dark:text-rose-100">
          {t('gradingAgent.review.flagged.title')}
        </h3>
        <span
          className="rounded-full bg-rose-600 px-2 py-0.5 text-xs font-semibold text-white"
          aria-live="polite"
        >
          {flagged.length}
        </span>
      </div>
      {error ? <p className="mt-2 text-sm text-rose-700 dark:text-rose-300">{error}</p> : null}
      <ul className="mt-3 space-y-2">
        {flagged.map((item) => {
          const label = item.submissionLabel ?? item.submissionId.slice(0, 8)
          const isEditing = editingId === item.id
          const isBusy = busyId === item.id
          return (
            <li
              key={item.id}
              className="rounded-lg border border-rose-200/80 bg-white px-3 py-2 text-sm dark:border-rose-900/40 dark:bg-neutral-950"
            >
              <p className="font-medium text-slate-900 dark:text-neutral-50">{label}</p>
              <p className="mt-1 text-xs font-semibold uppercase tracking-wide text-rose-700 dark:text-rose-300">
                {t('gradingAgent.review.flagged.badge')}
              </p>
              {item.flagReason ? (
                <p className="mt-1 text-slate-700 dark:text-neutral-300">{item.flagReason}</p>
              ) : null}
              {item.flagPriority ? (
                <p className="mt-1 text-xs text-slate-500 dark:text-neutral-400">
                  {t('gradingAgent.review.flagged.priority')}: {item.flagPriority}
                </p>
              ) : null}
              {isEditing ? (
                <div className="mt-2 space-y-2">
                  <label className="block">
                    <span className="text-xs font-medium text-slate-600 dark:text-neutral-400">
                      {t('gradingAgent.review.queue.editScore')}
                    </span>
                    <input
                      type="number"
                      value={editScore}
                      onChange={(e) => setEditScore(e.target.value)}
                      className="mt-1 w-full rounded-md border border-slate-200 px-2 py-1 text-sm dark:border-neutral-700 dark:bg-neutral-900"
                    />
                  </label>
                  <label className="block">
                    <span className="text-xs font-medium text-slate-600 dark:text-neutral-400">
                      {t('gradingAgent.review.queue.editComment')}
                    </span>
                    <textarea
                      value={editComment}
                      onChange={(e) => setEditComment(e.target.value)}
                      rows={2}
                      className="mt-1 w-full rounded-md border border-slate-200 px-2 py-1 text-sm dark:border-neutral-700 dark:bg-neutral-900"
                    />
                  </label>
                </div>
              ) : null}
              <div className="mt-2 flex flex-wrap gap-2">
                {onOpenSubmission ? (
                  <button
                    type="button"
                    disabled={isBusy}
                    onClick={() => onOpenSubmission(item.submissionId)}
                    className="rounded-md border border-slate-200 px-2.5 py-1 text-xs font-medium dark:border-neutral-700"
                  >
                    {t('gradingAgent.review.flagged.openSubmission')}
                  </button>
                ) : null}
                {isEditing ? (
                  <>
                    <button
                      type="button"
                      disabled={isBusy}
                      onClick={() => gradeNow(item)}
                      className="rounded-md bg-emerald-600 px-2.5 py-1 text-xs font-semibold text-white disabled:opacity-50"
                    >
                      {t('gradingAgent.review.flagged.gradeNow')}
                    </button>
                    <button
                      type="button"
                      disabled={isBusy}
                      onClick={() => setEditingId(null)}
                      className="rounded-md border border-slate-200 px-2.5 py-1 text-xs font-medium dark:border-neutral-700"
                    >
                      {t('gradingAgent.review.queue.cancel')}
                    </button>
                  </>
                ) : (
                  <>
                    <button
                      type="button"
                      disabled={isBusy}
                      onClick={() => beginGrade(item)}
                      className="rounded-md bg-emerald-600 px-2.5 py-1 text-xs font-semibold text-white disabled:opacity-50"
                    >
                      {t('gradingAgent.review.flagged.gradeNow')}
                    </button>
                    <button
                      type="button"
                      disabled={isBusy}
                      onClick={() => dismiss(item)}
                      className="rounded-md border border-rose-200 px-2.5 py-1 text-xs font-medium text-rose-700 dark:border-rose-900 dark:text-rose-300"
                    >
                      {t('gradingAgent.review.flagged.dismiss')}
                    </button>
                  </>
                )}
              </div>
            </li>
          )
        })}
      </ul>
    </section>
  )
}