import { useEffect, useId, useRef, useState, type CSSProperties, type FormEvent } from 'react'
import { usePlatformFeatures } from '../../../context/platform-features-context'
import {
  dismissReasonSchema,
  type DismissReason,
} from '../../../lib/course-checklist-api-schemas'
import { courseChecklistI18n } from '../../../lib/course-checklist-i18n'
import { overlayClassNames } from '../../../lib/overlay-motion'
import { useOverlayPresence } from '../../../lib/use-overlay-presence'

const REASONS = dismissReasonSchema.options

type ChecklistDismissDialogProps = {
  open: boolean
  itemTitle: string
  busy?: boolean
  error?: string | null
  onConfirm: (body: { reason: DismissReason; note?: string }) => void
  onClose: () => void
}

export function ChecklistDismissDialog({
  open,
  itemTitle,
  busy,
  error,
  onConfirm,
  onClose,
}: ChecklistDismissDialogProps) {
  const titleId = useId()
  const descId = useId()
  const reasonId = useId()
  const noteId = useId()
  const firstFieldRef = useRef<HTMLSelectElement>(null)
  const triggerReturnRef = useRef<Element | null>(null)
  const { ffMotionOverlays } = usePlatformFeatures()
  const presence = useOverlayPresence({
    open,
    kind: 'dialog',
    enabled: ffMotionOverlays !== false,
  })
  const [reason, setReason] = useState<DismissReason>('not_applicable')
  const [note, setNote] = useState('')

  useEffect(() => {
    if (open) {
      triggerReturnRef.current = document.activeElement
      setReason('not_applicable')
      setNote('')
    }
  }, [open])

  useEffect(() => {
    if (!presence.entered) return
    const t = window.setTimeout(() => firstFieldRef.current?.focus(), 0)
    return () => window.clearTimeout(t)
  }, [presence.entered])

  useEffect(() => {
    if (presence.phase === 'closed' && !open && triggerReturnRef.current instanceof HTMLElement) {
      triggerReturnRef.current.focus()
      triggerReturnRef.current = null
    }
  }, [presence.phase, open])

  useEffect(() => {
    if (!presence.mounted) return
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape' && !busy) {
        e.preventDefault()
        onClose()
      }
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [presence.mounted, busy, onClose])

  if (!presence.mounted) return null

  const classes = overlayClassNames({
    kind: 'dialog',
    phase: presence.phase,
    enabled: presence.enabled,
    reduceMotion: presence.reducedMotion,
  })
  const durationStyle = {
    '--lx-overlay-duration': `${classes.durationMs}ms`,
  } as CSSProperties

  const submit = (e: FormEvent) => {
    e.preventDefault()
    const trimmed = note.trim()
    onConfirm({
      reason,
      note: trimmed ? trimmed.slice(0, 500) : undefined,
    })
  }

  return (
    <div
      className="fixed inset-0 z-[400] flex items-center justify-center p-4"
      role="presentation"
      style={durationStyle}
    >
      <button
        type="button"
        className={`absolute inset-0 bg-slate-950/50 ${classes.scrim}`}
        aria-label={courseChecklistI18n.dismissCancel}
        disabled={busy}
        onClick={() => {
          if (!busy) onClose()
        }}
      />
      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        aria-describedby={descId}
        className={`relative z-10 w-full max-w-md rounded-xl border border-slate-200 bg-white p-5 shadow-xl dark:border-neutral-700 dark:bg-neutral-900 ${classes.panel}`}
      >
        <h2 id={titleId} className="text-lg font-semibold text-slate-900 dark:text-neutral-50">
          {courseChecklistI18n.dismissDialogTitle}
        </h2>
        <p id={descId} className="mt-1 text-sm text-slate-600 dark:text-neutral-400">
          {courseChecklistI18n.dismissDialogHelp}
        </p>
        <p className="mt-2 text-sm font-medium text-slate-800 dark:text-neutral-200">{itemTitle}</p>
        <form className="mt-4 space-y-4" onSubmit={submit}>
          <div>
            <label htmlFor={reasonId} className="block text-sm font-medium text-slate-700 dark:text-neutral-300">
              {courseChecklistI18n.dismissReasonLabel}
            </label>
            <select
              ref={firstFieldRef}
              id={reasonId}
              value={reason}
              disabled={busy}
              onChange={(e) => setReason(e.target.value as DismissReason)}
              className="mt-1 w-full min-h-11 rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-950 dark:text-neutral-100"
            >
              {REASONS.map((r) => (
                <option key={r} value={r}>
                  {courseChecklistI18n.dismissReasons[r]}
                </option>
              ))}
            </select>
          </div>
          <div>
            <label htmlFor={noteId} className="block text-sm font-medium text-slate-700 dark:text-neutral-300">
              {courseChecklistI18n.dismissNoteLabel}
            </label>
            <textarea
              id={noteId}
              value={note}
              maxLength={500}
              disabled={busy}
              rows={3}
              placeholder={courseChecklistI18n.dismissNotePlaceholder}
              onChange={(e) => setNote(e.target.value)}
              className="mt-1 w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-950 dark:text-neutral-100"
            />
            <p className="mt-1 text-xs text-slate-500">{note.length}/500</p>
          </div>
          {error ? (
            <p className="text-sm text-red-600 dark:text-red-400" role="alert">
              {error}
            </p>
          ) : null}
          <div className="flex justify-end gap-2">
            <button
              type="button"
              disabled={busy}
              onClick={onClose}
              className="inline-flex min-h-11 items-center rounded-lg border border-slate-300 px-4 text-sm font-semibold text-slate-800 dark:border-neutral-600 dark:text-neutral-100"
            >
              {courseChecklistI18n.dismissCancel}
            </button>
            <button
              type="submit"
              disabled={busy}
              className="inline-flex min-h-11 items-center rounded-lg bg-amber-700 px-4 text-sm font-semibold text-white hover:bg-amber-600 disabled:opacity-60"
            >
              {busy ? courseChecklistI18n.rechecking : courseChecklistI18n.dismissConfirm}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}
