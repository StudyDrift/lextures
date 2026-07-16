import { useEffect, useId, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { reportBoardContent } from '../../lib/boards-api'
import { toastMutationError } from '../../lib/lms-toast'

const REASONS = ['hurtful', 'inappropriate', 'spam', 'other'] as const

type Props = {
  open: boolean
  onClose: () => void
  courseCode: string
  boardId: string
  postId?: string
  commentId?: string
}

export function BoardReportDialog({ open, onClose, courseCode, boardId, postId, commentId }: Props) {
  const { t } = useTranslation('common')
  const titleId = useId()
  const firstRef = useRef<HTMLSelectElement>(null)
  const [reasonKey, setReasonKey] = useState<(typeof REASONS)[number]>('hurtful')
  const [details, setDetails] = useState('')
  const [submitting, setSubmitting] = useState(false)

  useEffect(() => {
    if (open) {
      setReasonKey('hurtful')
      setDetails('')
      queueMicrotask(() => firstRef.current?.focus())
    }
  }, [open])

  if (!open) return null

  async function submit() {
    setSubmitting(true)
    try {
      const reason = [t(`boards.report.reason.${reasonKey}`), details.trim()].filter(Boolean).join(' — ')
      await reportBoardContent(courseCode, boardId, { postId, commentId, reason })
      onClose()
    } catch (err) {
      toastMutationError(err instanceof Error ? err.message : String(err))
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4"
      role="dialog"
      aria-modal="true"
      aria-labelledby={titleId}
      onClick={(e) => {
        if (e.target === e.currentTarget) onClose()
      }}
      onKeyDown={(e) => {
        if (e.key === 'Escape') onClose()
      }}
    >
      <div className="w-full max-w-md rounded-lg border border-slate-200 bg-white p-4 shadow-xl dark:border-neutral-700 dark:bg-neutral-900">
        <h2 id={titleId} className="text-lg font-semibold text-slate-900 dark:text-neutral-100">
          {t('boards.report.title')}
        </h2>
        <p className="mt-1 text-sm text-slate-600 dark:text-neutral-300">{t('boards.report.subtitle')}</p>
        <label className="mt-4 flex flex-col gap-1 text-sm">
          <span className="font-medium">{t('boards.report.reasonLabel')}</span>
          <select
            ref={firstRef}
            value={reasonKey}
            onChange={(e) => setReasonKey(e.target.value as (typeof REASONS)[number])}
            className="rounded-md border border-slate-300 px-2 py-1.5 dark:border-neutral-600 dark:bg-neutral-800"
          >
            {REASONS.map((key) => (
              <option key={key} value={key}>
                {t(`boards.report.reason.${key}`)}
              </option>
            ))}
          </select>
        </label>
        <label className="mt-3 flex flex-col gap-1 text-sm">
          <span className="font-medium">{t('boards.report.detailsLabel')}</span>
          <textarea
            value={details}
            onChange={(e) => setDetails(e.target.value)}
            rows={3}
            className="rounded-md border border-slate-300 px-2 py-1.5 dark:border-neutral-600 dark:bg-neutral-800"
          />
        </label>
        <div className="mt-4 flex justify-end gap-2">
          <button
            type="button"
            onClick={onClose}
            className="rounded-md px-3 py-1.5 text-sm text-slate-600 dark:text-neutral-300"
          >
            {t('dialogs.cancel')}
          </button>
          <button
            type="button"
            disabled={submitting}
            onClick={() => void submit()}
            className="rounded-md bg-indigo-600 px-3 py-1.5 text-sm font-medium text-white disabled:opacity-50"
          >
            {t('boards.report.submit')}
          </button>
        </div>
      </div>
    </div>
  )
}
