import { useEffect, useId, useRef, type FormEvent, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'

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

  useEffect(() => {
    if (!open) return
    const timer = window.setTimeout(() => inputRef.current?.focus(), 0)
    return () => window.clearTimeout(timer)
  }, [open])

  useEffect(() => {
    if (!open) return
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape' && !busy) {
        e.preventDefault()
        onClose()
      }
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [open, busy, onClose])

  if (!open) return null

  function handleSubmit(e: FormEvent) {
    e.preventDefault()
    if (busy) return
    onConfirm(value)
  }

  return (
    <div className="fixed inset-0 z-[400] flex items-center justify-center p-4" role="presentation">
      <button
        type="button"
        aria-label={t('dialogs.close')}
        disabled={busy}
        className="lex-btn-static absolute inset-0 cursor-default border-0 bg-black/45 p-0 disabled:cursor-not-allowed"
        onClick={() => {
          if (!busy) onClose()
        }}
      />
      <form
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        aria-describedby={description ? descId : undefined}
        className="relative z-10 w-full max-w-md rounded-2xl border border-slate-200 bg-white p-5 shadow-xl dark:border-neutral-700 dark:bg-neutral-900"
        onSubmit={handleSubmit}
      >
        <h2 id={titleId} className="text-lg font-semibold text-slate-950 dark:text-neutral-100">
          {title}
        </h2>
        {description ? (
          <div id={descId} className="mt-2 text-sm text-slate-600 dark:text-neutral-300">
            {description}
          </div>
        ) : null}
        <div className="mt-4">
          {label ? (
            <label htmlFor={inputId} className="text-xs font-medium text-slate-700 dark:text-neutral-200">
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
            className="mt-1.5 w-full rounded-xl border border-slate-200 bg-white px-3 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-950 dark:text-neutral-100"
          />
        </div>
        <div className="mt-6 flex flex-wrap justify-end gap-2">
          <button
            type="button"
            disabled={busy}
            onClick={onClose}
            className="rounded-xl border border-slate-200 bg-white px-4 py-2 text-sm font-semibold text-slate-800 shadow-sm motion-safe:transition-transform motion-safe:duration-150 motion-safe:ease-out motion-safe:active:scale-[0.96] hover:bg-slate-50 disabled:opacity-60 dark:border-neutral-600 dark:bg-neutral-900 dark:text-neutral-100 dark:hover:bg-neutral-800"
          >
            {cancelLabel ?? t('dialogs.cancel')}
          </button>
          <button
            type="submit"
            disabled={busy}
            className="rounded-xl bg-indigo-600 px-4 py-2 text-sm font-semibold text-white shadow-sm motion-safe:transition-transform motion-safe:duration-150 motion-safe:ease-out motion-safe:active:scale-[0.96] hover:bg-indigo-500 disabled:cursor-not-allowed disabled:opacity-50"
          >
            {busy ? t('dialogs.working') : (confirmLabel ?? t('dialogs.confirm'))}
          </button>
        </div>
      </form>
    </div>
  )
}

