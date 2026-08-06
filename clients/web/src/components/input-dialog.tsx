import { useEffect, useId, useRef, type CSSProperties, type FormEvent, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { usePlatformFeatures } from '../context/platform-features-context'
import { overlayClassNames } from '../lib/overlay-motion'
import { useOverlayPresence } from '../lib/use-overlay-presence'

export type InputDialogProps = {
  open: boolean
  title: string
  description?: ReactNode
  label?: string
  value: string
  onValueChange: (value: string) => void
  placeholder?: string
  confirmLabel?: string
  cancelLabel?: string
  busy?: boolean
  onConfirm: (value: string) => void
  onClose: () => void
}

export function InputDialog({
  open,
  title,
  description,
  label,
  value,
  onValueChange,
  placeholder,
  confirmLabel,
  cancelLabel,
  busy,
  onConfirm,
  onClose,
}: InputDialogProps) {
  const { t } = useTranslation('common')
  const titleId = useId()
  const descId = useId()
  const inputId = useId()
  const inputRef = useRef<HTMLInputElement>(null)
  const { ffMotionOverlays } = usePlatformFeatures()
  const presence = useOverlayPresence({
    open,
    kind: 'dialog',
    enabled: ffMotionOverlays !== false,
  })
  const classes = overlayClassNames({
    kind: 'dialog',
    phase: presence.phase,
    enabled: presence.enabled,
    reduceMotion: presence.reducedMotion,
  })

  useEffect(() => {
    if (!presence.entered) return
    const timer = window.setTimeout(() => inputRef.current?.focus(), 0)
    return () => window.clearTimeout(timer)
  }, [presence.entered])

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

  function handleSubmit(e: FormEvent) {
    e.preventDefault()
    if (busy) return
    onConfirm(value)
  }

  const durationStyle = {
    '--lx-overlay-duration': `${classes.durationMs}ms`,
  } as CSSProperties

  // Exit animation keeps the layer mounted at opacity 0 — do not block underlying Links.
  const exiting = presence.phase === 'closing'

  return (
    <div
      className={`fixed inset-0 z-[400] flex items-center justify-center p-4 ${exiting ? 'pointer-events-none' : ''}`}
      role="presentation"
      style={durationStyle}
      data-overlay-phase={presence.phase}
    >
      <button
        type="button"
        aria-label={t('dialogs.close')}
        disabled={busy || exiting}
        className={`lex-btn-static absolute inset-0 cursor-default border-0 bg-black/45 p-0 disabled:cursor-not-allowed ${classes.scrim}`}
        onClick={() => {
          if (!busy && !exiting) onClose()
        }}
      />
      <form
        role="dialog"
        aria-modal={!exiting}
        aria-labelledby={titleId}
        aria-describedby={description ? descId : undefined}
        className={`relative z-10 w-full max-w-md rounded-2xl border border-border-default bg-surface-raised p-5 shadow-xl dark:border-border-default dark:bg-surface-raised ${classes.panel}`}
        onSubmit={handleSubmit}
      >
        <h2 id={titleId} className="text-lg font-semibold text-slate-950 dark:text-fg-default">
          {title}
        </h2>
        {description ? (
          <div id={descId} className="mt-2 text-sm text-fg-muted">
            {description}
          </div>
        ) : null}
        <div className="mt-4">
          {label ? (
            <label htmlFor={inputId} className="text-xs font-medium text-fg-default">
              {label}
            </label>
          ) : null}
          <input
            ref={inputRef}
            id={inputId}
            type="text"
            value={value}
            placeholder={placeholder}
            disabled={busy}
            onChange={(e) => onValueChange(e.target.value)}
            className="mt-1.5 w-full rounded-xl border border-border-default bg-surface-raised px-3 py-2 text-sm dark:border-border-default dark:bg-surface-base dark:text-fg-default"
          />
        </div>
        <div className="mt-6 flex flex-wrap justify-end gap-2">
          <button
            type="button"
            disabled={busy}
            onClick={onClose}
            className="rounded-xl border border-border-default bg-surface-raised px-4 py-2 text-sm font-semibold text-fg-default shadow-sm motion-safe:transition-transform motion-safe:duration-150 motion-safe:ease-out motion-safe:active:scale-[0.96] hover:bg-surface-base disabled:opacity-60 dark:border-border-default dark:bg-surface-raised dark:text-fg-default dark:hover:bg-surface-overlay"
          >
            {cancelLabel ?? t('dialogs.cancel')}
          </button>
          <button
            type="submit"
            disabled={busy}
            className="rounded-xl bg-accent-solid px-4 py-2 text-sm font-semibold text-white shadow-sm motion-safe:transition-transform motion-safe:duration-150 motion-safe:ease-out motion-safe:active:scale-[0.96] hover:bg-indigo-500 disabled:cursor-not-allowed disabled:opacity-50"
          >
            {busy ? t('dialogs.working') : (confirmLabel ?? t('dialogs.confirm'))}
          </button>
        </div>
      </form>
    </div>
  )
}
