import { useEffect, useId, useRef, useState, type FormEvent } from 'react'
import { Loader2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { useLocation } from 'react-router-dom'
import { createFocusTrap } from '../../lib/a11y/focus-trap'
import {
  FEEDBACK_MAX_MESSAGE_LEN,
  feedbackMessageValid,
  submitFeedback,
  type FeedbackCategoryOption,
} from '../../lib/feedback-api'
import { toastSaveOk } from '../../lib/lms-toast'

const CATEGORY_OPTIONS: FeedbackCategoryOption[] = ['bug', 'idea', 'question', 'praise', 'other']

export type FeedbackDialogProps = {
  open: boolean
  onClose: () => void
}

function feedbackViewportLabel(): string | undefined {
  if (typeof window === 'undefined') return undefined
  return `${window.innerWidth}x${window.innerHeight}`
}

export function FeedbackDialog({ open, onClose }: FeedbackDialogProps) {
  const { t, i18n } = useTranslation('common')
  const { pathname } = useLocation()
  const titleId = useId()
  const privacyId = useId()
  const errorId = useId()
  const dialogRef = useRef<HTMLFormElement>(null)
  const messageRef = useRef<HTMLTextAreaElement>(null)
  const [message, setMessage] = useState('')
  const [category, setCategory] = useState<FeedbackCategoryOption>('')
  const [submitting, setSubmitting] = useState(false)
  const [errorKey, setErrorKey] = useState<string | null>(null)

  useEffect(() => {
    if (!open) return
    setErrorKey(null)
    const timer = window.setTimeout(() => messageRef.current?.focus(), 0)
    return () => window.clearTimeout(timer)
  }, [open])

  useEffect(() => {
    if (!open) {
      setMessage('')
      setCategory('')
      setSubmitting(false)
      setErrorKey(null)
    }
  }, [open])

  useEffect(() => {
    if (!open) return
    const dialog = dialogRef.current
    if (!dialog) return
    const trap = createFocusTrap(dialog)
    trap.activate()
    return () => trap.deactivate()
  }, [open])

  useEffect(() => {
    if (!open) return
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape' && !submitting) {
        e.preventDefault()
        onClose()
      }
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [open, submitting, onClose])

  if (!open) return null

  const trimmedLen = message.trim().length
  const canSend = feedbackMessageValid(message) && !submitting

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    if (!canSend) return
    setSubmitting(true)
    setErrorKey(null)
    const result = await submitFeedback({
      message,
      category,
      route: pathname,
      locale: i18n.language,
      viewport: feedbackViewportLabel(),
    })
    setSubmitting(false)
    if (result.ok) {
      toastSaveOk(t('feedback.success'))
      onClose()
      return
    }
    if (result.kind === 'rate_limited') {
      setErrorKey('feedback.rateLimited')
      return
    }
    if (result.kind === 'offline') {
      setErrorKey('feedback.offline')
      return
    }
    setErrorKey('feedback.error')
  }

  return (
    <div className="fixed inset-0 z-[400] flex items-center justify-center p-4" role="presentation">
      <button
        type="button"
        aria-label={t('dialogs.close')}
        disabled={submitting}
        className="lex-btn-static absolute inset-0 cursor-default border-0 bg-black/45 p-0 disabled:cursor-not-allowed"
        onClick={() => {
          if (!submitting) onClose()
        }}
      />
      <form
        ref={dialogRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        aria-describedby={privacyId}
        data-testid="feedback-dialog"
        className="relative z-10 flex w-full max-w-lg flex-col rounded-2xl border border-border-default bg-surface-raised p-5 shadow-xl dark:border-border-default dark:bg-surface-raised"
        onSubmit={(e) => void handleSubmit(e)}
      >
        <h2 id={titleId} className="text-lg font-semibold text-slate-950 dark:text-fg-default">
          {t('feedback.dialog.title')}
        </h2>
        <p id={privacyId} className="mt-1 text-sm text-fg-muted">
          {t('feedback.dialog.privacy')}
        </p>

        <div className="mt-4">
          <label htmlFor="feedback-message" className="text-xs font-medium text-fg-default">
            {t('feedback.message.label')}
          </label>
          <textarea
            ref={messageRef}
            id="feedback-message"
            value={message}
            disabled={submitting}
            maxLength={FEEDBACK_MAX_MESSAGE_LEN}
            rows={5}
            placeholder={t('feedback.message.placeholder')}
            onChange={(e) => setMessage(e.target.value)}
            className="mt-1.5 w-full resize-y rounded-xl border border-border-default bg-surface-raised px-3 py-2 text-sm text-fg-default placeholder:text-fg-subtle focus:outline-none focus-visible:ring-2 focus-visible:ring-indigo-500/40 disabled:opacity-60 dark:border-border-default dark:bg-surface-base dark:text-fg-default dark:placeholder:text-neutral-500"
          />
          <p className="mt-1 text-end text-xs text-fg-muted" aria-live="polite">
            {t('feedback.message.counter', { count: trimmedLen, max: FEEDBACK_MAX_MESSAGE_LEN })}
          </p>
        </div>

        <div className="mt-3">
          <label htmlFor="feedback-category" className="text-xs font-medium text-fg-default">
            {t('feedback.category.label')}
          </label>
          <select
            id="feedback-category"
            value={category}
            disabled={submitting}
            onChange={(e) => setCategory(e.target.value as FeedbackCategoryOption)}
            className="mt-1.5 w-full rounded-xl border border-border-default bg-surface-raised px-3 py-2 text-sm text-fg-default focus:outline-none focus-visible:ring-2 focus-visible:ring-indigo-500/40 disabled:opacity-60 dark:border-border-default dark:bg-surface-base dark:text-fg-default"
          >
            <option value="">{t('feedback.category.none')}</option>
            {CATEGORY_OPTIONS.map((value) => (
              <option key={value} value={value}>
                {t(`feedback.category.${value}`)}
              </option>
            ))}
          </select>
        </div>

        {errorKey ? (
          <p
            id={errorId}
            role="alert"
            aria-live="assertive"
            className="mt-3 rounded-lg border border-rose-200 bg-rose-50 px-3 py-2 text-sm text-rose-800 dark:border-rose-900/50 dark:bg-rose-950/40 dark:text-rose-200"
          >
            {t(errorKey)}
          </p>
        ) : null}

        <div className="mt-6 flex flex-wrap justify-end gap-2">
          <button
            type="button"
            disabled={submitting}
            onClick={onClose}
            className="rounded-xl border border-border-default bg-surface-raised px-4 py-2 text-sm font-semibold text-fg-default shadow-sm motion-safe:transition-transform motion-safe:duration-150 motion-safe:ease-out motion-safe:active:scale-[0.96] hover:bg-surface-base disabled:opacity-60 dark:border-border-default dark:bg-surface-raised dark:text-fg-default dark:hover:bg-surface-overlay"
          >
            {t('feedback.cancel')}
          </button>
          <button
            type="submit"
            disabled={!canSend}
            aria-describedby={errorKey ? errorId : undefined}
            className="inline-flex min-h-11 items-center justify-center gap-2 rounded-xl bg-accent-solid px-4 py-2 text-sm font-semibold text-white shadow-sm motion-safe:transition-transform motion-safe:duration-150 motion-safe:ease-out motion-safe:active:scale-[0.96] hover:bg-indigo-500 disabled:cursor-not-allowed disabled:opacity-50"
          >
            {submitting ? (
              <>
                <Loader2 className="h-4 w-4 animate-spin" aria-hidden />
                {t('dialogs.working')}
              </>
            ) : (
              t('feedback.send')
            )}
          </button>
        </div>
      </form>
    </div>
  )
}
