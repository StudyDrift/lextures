import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  patchGraderAgentResult,
  postGraderAgentReviewBulk,
  putSubmissionGrade,
  type GraderAgentReviewQueueItem,
} from '../../../lib/courses-api'
import { usePrompt } from '../../use-prompt'
import { useConfirm } from '../../use-confirm'

type HeldReviewQueuePanelProps = {
  courseCode: string
  itemId: string
  items: GraderAgentReviewQueueItem[]
  suggestModeEnabled?: boolean
  onUpdated?: () => void
  onOpenSubmission?: (submissionId: string) => void
}

export function HeldReviewQueuePanel({
  courseCode,
  itemId,
  items,
  suggestModeEnabled = false,
  onUpdated,
  onOpenSubmission,
}: HeldReviewQueuePanelProps) {
  const { t } = useTranslation('common')
  const { prompt, InputDialogHost } = usePrompt()
  const { confirm, ConfirmDialogHost } = useConfirm()
  const [busyId, setBusyId] = useState<string | null>(null)
  const [bulkBusy, setBulkBusy] = useState(false)
  const [editingId, setEditingId] = useState<string | null>(null)
  const [editScore, setEditScore] = useState('')
  const [editComment, setEditComment] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [selectedIds, setSelectedIds] = useState<Set<string>>(() => new Set())
  const [confidenceThreshold, setConfidenceThreshold] = useState('80')

  const held = useMemo(() => items.filter((result) => result.status === 'suggested'), [items])
  const allSelected = held.length > 0 && held.every((item) => selectedIds.has(item.id))

  if (held.length === 0) return null

  const toggleSelected = (id: string) => {
    setSelectedIds((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }

  const toggleSelectAll = () => {
    if (allSelected) {
      setSelectedIds(new Set())
      return
    }
    setSelectedIds(new Set(held.map((item) => item.id)))
  }

  const beginEdit = (item: GraderAgentReviewQueueItem) => {
    setEditingId(item.id)
    setEditScore(item.suggestedPoints != null ? String(item.suggestedPoints) : '')
    setEditComment(item.comment ?? '')
    setError(null)
  }

  const approveLegacy = async (item: GraderAgentReviewQueueItem, edited = false) => {
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
  }

  const approveViaBulk = async (
    item: GraderAgentReviewQueueItem,
    edited = false,
    extra?: { resultIds?: string[]; minConfidence?: number; action?: 'approve' | 'approve_all' },
  ) => {
    const points = edited ? Number(editScore) : item.suggestedPoints
    if (edited && (points == null || !Number.isFinite(points))) {
      setError(t('gradingAgent.review.queue.invalidScore'))
      return
    }
    await postGraderAgentReviewBulk(courseCode, itemId, {
      action: extra?.action ?? 'approve',
      resultIds: extra?.resultIds,
      minConfidence: extra?.minConfidence,
      items: edited
        ? [
            {
              resultId: item.id,
              pointsEarned: points ?? undefined,
              comment: editComment.trim() || null,
            },
          ]
        : undefined,
    })
  }

  const approve = async (item: GraderAgentReviewQueueItem, edited = false) => {
    setBusyId(item.id)
    setError(null)
    try {
      if (suggestModeEnabled) {
        await approveViaBulk(item, edited, { resultIds: [item.id] })
      } else {
        await approveLegacy(item, edited)
      }
      setEditingId(null)
      onUpdated?.()
    } catch (err) {
      setError(err instanceof Error ? err.message : t('gradingAgent.review.queue.actionFailed'))
    } finally {
      setBusyId(null)
    }
  }

  const reject = async (item: GraderAgentReviewQueueItem) => {
    const reason = await prompt({
      title: t('gradingAgent.review.queue.rejectPrompt'),
    })
    if (reason == null) return
    setBusyId(item.id)
    setError(null)
    try {
      if (suggestModeEnabled) {
        await postGraderAgentReviewBulk(courseCode, itemId, {
          action: 'reject',
          resultIds: [item.id],
        })
      } else {
        await patchGraderAgentResult(courseCode, itemId, item.id, {
          status: 'skipped',
          reason: reason.trim() || t('gradingAgent.review.queue.rejectDefault'),
        })
      }
      onUpdated?.()
    } catch (err) {
      setError(err instanceof Error ? err.message : t('gradingAgent.review.queue.actionFailed'))
    } finally {
      setBusyId(null)
    }
  }

  const runBulk = async (action: 'approve_all' | 'approve' | 'reject', opts?: { minConfidence?: number }) => {
    if (action === 'approve_all') {
      const confirmed = await confirm({
        title: t('gradingAgent.review.bulk.confirmApproveAll', { count: held.length }),
      })
      if (!confirmed) return
    }
    setBulkBusy(true)
    setError(null)
    try {
      if (suggestModeEnabled) {
        const resultIds =
          action === 'approve_all' ? undefined : action === 'approve' || action === 'reject' ? [...selectedIds] : undefined
        await postGraderAgentReviewBulk(courseCode, itemId, {
          action,
          resultIds: resultIds && resultIds.length > 0 ? resultIds : undefined,
          minConfidence: opts?.minConfidence,
        })
      } else if (action === 'reject') {
        for (const item of held) {
          if (!selectedIds.has(item.id)) continue
          await patchGraderAgentResult(courseCode, itemId, item.id, {
            status: 'skipped',
            reason: t('gradingAgent.review.queue.rejectDefault'),
          })
        }
      } else {
        for (const item of held) {
          if (action === 'approve' && !selectedIds.has(item.id)) continue
          if (opts?.minConfidence != null) {
            const floor = opts.minConfidence
            if (item.confidence == null || item.confidence < floor) continue
          }
          await approveLegacy(item)
        }
      }
      setSelectedIds(new Set())
      onUpdated?.()
    } catch (err) {
      setError(err instanceof Error ? err.message : t('gradingAgent.review.bulk.failed'))
    } finally {
      setBulkBusy(false)
    }
  }

  const approveAboveThreshold = async () => {
    const parsed = Number(confidenceThreshold)
    if (!Number.isFinite(parsed) || parsed < 0 || parsed > 100) {
      setError(t('gradingAgent.review.bulk.invalidThreshold'))
      return
    }
    await runBulk('approve', { minConfidence: parsed / 100 })
  }

  return (
    <>
    {InputDialogHost}
    {ConfirmDialogHost}
    <section
      className="rounded-xl border border-slate-300 bg-slate-50/70 p-4 dark:border-neutral-600 dark:bg-neutral-900/40"
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

      {suggestModeEnabled ? (
        <div className="mt-3 flex flex-wrap items-center gap-2 rounded-lg border border-slate-200 bg-white p-2 dark:border-neutral-700 dark:bg-neutral-950">
          <label className="flex items-center gap-2 text-xs font-medium text-slate-700 dark:text-neutral-300">
            <input
              type="checkbox"
              className="size-4"
              checked={allSelected}
              onChange={toggleSelectAll}
              aria-label={t('gradingAgent.review.bulk.selectAll')}
            />
            {t('gradingAgent.review.bulk.selectAll')}
          </label>
          <label className="flex items-center gap-1 text-xs text-slate-600 dark:text-neutral-400">
            <span>{t('gradingAgent.review.bulk.thresholdLabel')}</span>
            <input
              type="number"
              min={0}
              max={100}
              value={confidenceThreshold}
              onChange={(e) => setConfidenceThreshold(e.target.value)}
              className="w-14 rounded-md border border-slate-200 px-1.5 py-0.5 text-xs dark:border-neutral-700 dark:bg-neutral-900"
              aria-label={t('gradingAgent.review.bulk.thresholdLabel')}
            />
            <span>%</span>
          </label>
          <button
            type="button"
            disabled={bulkBusy}
            onClick={() => void approveAboveThreshold()}
            className="rounded-md border border-slate-200 px-2 py-1 text-xs font-medium dark:border-neutral-700"
          >
            {t('gradingAgent.review.bulk.approveThreshold')}
          </button>
          <button
            type="button"
            disabled={bulkBusy || selectedIds.size === 0}
            onClick={() => void runBulk('approve')}
            className="rounded-md bg-emerald-600 px-2 py-1 text-xs font-semibold text-white disabled:opacity-50"
          >
            {t('gradingAgent.review.bulk.approveSelected', { count: selectedIds.size })}
          </button>
          <button
            type="button"
            disabled={bulkBusy || selectedIds.size === 0}
            onClick={() => void runBulk('reject')}
            className="rounded-md border border-rose-200 px-2 py-1 text-xs font-medium text-rose-700 dark:border-rose-900 dark:text-rose-300 disabled:opacity-50"
          >
            {t('gradingAgent.review.bulk.rejectSelected', { count: selectedIds.size })}
          </button>
          <button
            type="button"
            disabled={bulkBusy || held.length === 0}
            onClick={() => void runBulk('approve_all')}
            className="rounded-md border border-emerald-300 px-2 py-1 text-xs font-semibold text-emerald-800 dark:border-emerald-900 dark:text-emerald-300 disabled:opacity-50"
          >
            {t('gradingAgent.review.bulk.approveAll')}
          </button>
        </div>
      ) : null}

      {error ? <p className="mt-2 text-sm text-rose-700 dark:text-rose-300">{error}</p> : null}
      <ul className="mt-3 space-y-2">
        {held.map((item) => {
          const label = item.submissionLabel ?? item.submissionId.slice(0, 8)
          const isEditing = editingId === item.id
          const isBusy = busyId === item.id || bulkBusy
          const isSelected = selectedIds.has(item.id)
          return (
            <li
              key={item.id}
              className="rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm dark:border-neutral-700 dark:bg-neutral-950"
              aria-selected={suggestModeEnabled ? isSelected : undefined}
            >
              <div className="flex items-start gap-2">
                {suggestModeEnabled ? (
                  <input
                    type="checkbox"
                    className="mt-1 size-4"
                    checked={isSelected}
                    onChange={() => toggleSelected(item.id)}
                    aria-label={t('gradingAgent.review.bulk.selectItem', { label })}
                  />
                ) : null}
                <div className="min-w-0 flex-1">
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
                </div>
              </div>
            </li>
          )
        })}
      </ul>
    </section>
    </>
  )
}