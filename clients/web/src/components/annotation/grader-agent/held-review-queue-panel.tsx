import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { patchGraderAgentResult, putSubmissionGrade } from '../../../lib/courses-api'
import type { GraderAgentRunStatus } from '../../../lib/courses-api'

type HeldItem = GraderAgentRunStatus['results'][number]

type HeldReviewQueuePanelProps = {
  courseCode: string
  itemId: string
  runResults: GraderAgentRunStatus['results']
  submissionLabelById?: Record<string, string>
  onUpdated?: () => void
}

export function HeldReviewQueuePanel({
  courseCode,
  itemId,
  runResults,
  submissionLabelById = {},
  onUpdated,
}: HeldReviewQueuePanelProps) {
  const { t } = useTranslation('common')
  const [busyId, setBusyId] = useState<string | null>(null)
  const [editingId, setEditingId] = useState<string | null>(null)
  const [editScore, setEditScore] = useState('')
  const [editComment, setEditComment] = useState('')
  const [error, setError] = useState<string | null>(null)

  const held = useMemo(
    () => runResults.filter((result) => result.status === 'suggested'),
    [runResults],
  )

  if (held.length === 0) return null

  const beginEdit = (item: HeldItem) => {
    if (!item.id) return
    setEditingId(item.id)
    setEditScore(item.suggestedPoints != null ? String(item.suggestedPoints) : '')
    setEditComment(item.comment ?? '')
    setError(null)
  }

  const approve = async (item: HeldItem, edited = false) => {
    if (!item.id) return
    setBusyId(item.id)
    setError(null)
    try {
      const points = edited ? Number(editScore) : item.suggestedPoints
      if (points == null || !Number.isFinite(points)) {
        setError(t('gradingAgent.review.queue.invalidScore'))
        return
      }
      await putSubmissionGrade(courseCode, itemId, item.submissionId, {
        pointsEarned: points,
        instructorComment: edited ? editComment.trim() || null : item.comment ?? null,
        gradedByAi: true,
      })
      const status = edited ? 'overridden' : 'applied'
      await patchGraderAgentResult(courseCode, itemId, item.id, { status })
      setEditingId(null)
      onUpdated?.()
    } catch (err) {
      setError(err instanceof Error ? err.message : t('gradingAgent.review.queue.actionFailed'))
    } finally {
      setBusyId(null)
    }
  }

  const reject = async (item: HeldItem) => {
    if (!item.id) return
    const reason = window.prompt(t('gradingAgent.review.queue.rejectPrompt'))
    if (reason == null) return
    setBusyId(item.id)
    setError(null)
    try {
      await patchGraderAgentResult(courseCode, itemId, item.id, {
        status: 'skipped',
        reason: reason.trim() || t('gradingAgent.review.queue.rejectDefault'),
      })
      onUpdated?.()
    } catch (err) {
      setError(err instanceof Error ? err.message : t('gradingAgent.review.queue.actionFailed'))
    } finally {
      setBusyId(null)
    }
  }

  return (
    <section
      className="mt-4 rounded-xl border border-slate-300 bg-slate-50/70 p-4 dark:border-neutral-600 dark:bg-neutral-900/40"
      aria-label={t('gradingAgent.review.queue.title')}
    >
      <div className="flex items-center justify-between gap-3">
        <h3 className="text-sm font-semibold text-slate-900 dark:text-neutral-100">
          {t('gradingAgent.review.queue.title')}
        </h3>
        <span
          className="rounded-full bg-slate-700 px-2 py-0.5 text-xs font-semibold text-white dark:bg-neutral-200 dark:text-neutral-900"
          aria-live="polite"
        >
          {held.length}
        </span>
      </div>
      {error ? <p className="mt-2 text-sm text-rose-700 dark:text-rose-300">{error}</p> : null}
      <ul className="mt-3 space-y-2">
        {held.map((item) => {
          const label = submissionLabelById[item.submissionId] ?? item.submissionId.slice(0, 8)
          const isEditing = editingId === item.id
          const isBusy = busyId === item.id
          return (
            <li
              key={item.id ?? item.submissionId}
              className="rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm dark:border-neutral-700 dark:bg-neutral-950"
            >
              <p className="font-medium text-slate-900 dark:text-neutral-50">{label}</p>
              <p className="mt-1 text-xs font-semibold uppercase tracking-wide text-slate-600 dark:text-neutral-400">
                {t('gradingAgent.review.queue.badge')}
              </p>
              {item.suggestedPoints != null ? (
                <p className="mt-1 text-slate-700 dark:text-neutral-300">
                  {t('gradingAgent.review.queue.suggestedScore', { score: item.suggestedPoints })}
                </p>
              ) : null}
              {item.confidence != null ? (
                <p className="mt-1 text-xs text-slate-500 dark:text-neutral-400">
                  {t('gradingAgent.review.queue.confidence', {
                    value: Math.round(item.confidence * 100),
                  })}
                </p>
              ) : null}
              {item.heldReason ? (
                <p className="mt-1 text-slate-600 dark:text-neutral-400">{item.heldReason}</p>
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
              ) : item.comment ? (
                <p className="mt-1 text-slate-600 dark:text-neutral-400">{item.comment}</p>
              ) : null}
              <div className="mt-2 flex flex-wrap gap-2">
                {isEditing ? (
                  <>
                    <button
                      type="button"
                      disabled={isBusy}
                      onClick={() => approve(item, true)}
                      className="rounded-md bg-emerald-600 px-2.5 py-1 text-xs font-semibold text-white disabled:opacity-50"
                    >
                      {t('gradingAgent.review.queue.saveApprove')}
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
                      onClick={() => approve(item)}
                      className="rounded-md bg-emerald-600 px-2.5 py-1 text-xs font-semibold text-white disabled:opacity-50"
                    >
                      {t('gradingAgent.review.queue.approve')}
                    </button>
                    <button
                      type="button"
                      disabled={isBusy}
                      onClick={() => beginEdit(item)}
                      className="rounded-md border border-slate-200 px-2.5 py-1 text-xs font-medium dark:border-neutral-700"
                    >
                      {t('gradingAgent.review.queue.editApprove')}
                    </button>
                    <button
                      type="button"
                      disabled={isBusy}
                      onClick={() => reject(item)}
                      className="rounded-md border border-rose-200 px-2.5 py-1 text-xs font-medium text-rose-700 dark:border-rose-900 dark:text-rose-300"
                    >
                      {t('gradingAgent.review.queue.reject')}
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