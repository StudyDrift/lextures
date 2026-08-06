import { useEffect, useId, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import type { JsonSchema, SchemaFieldError } from './schema-form/types'
import { parseAndValidateConfigJSON } from './schema-form/validate'

export type PasteJsonConfigDialogProps = {
  open: boolean
  /** Pretty-printed JSON to seed the textarea when the dialog opens. */
  initialJson: string
  schema: JsonSchema
  busy?: boolean
  /** Server or submit error shown under the textarea. */
  submitError?: string | null
  fieldErrors?: SchemaFieldError[]
  onClose: () => void
  /**
   * Called with a client-validated config object. Parent should PATCH and
   * surface server field errors via `fieldErrors` / `submitError`.
   */
  onApply: (config: Record<string, unknown>) => void | Promise<void>
}

export function PasteJsonConfigDialog({
  open,
  initialJson,
  schema,
  busy = false,
  submitError = null,
  fieldErrors = [],
  onClose,
  onApply,
}: PasteJsonConfigDialogProps) {
  const { t } = useTranslation('contentTools')
  const titleId = useId()
  const descId = useId()
  const textareaId = useId()
  const textareaRef = useRef<HTMLTextAreaElement>(null)
  const [text, setText] = useState(initialJson)
  const [localErrors, setLocalErrors] = useState<SchemaFieldError[]>([])

  useEffect(() => {
    if (!open) return
    setText(initialJson)
    setLocalErrors([])
  }, [open, initialJson])

  useEffect(() => {
    if (!open) return
    const timer = window.setTimeout(() => {
      const el = textareaRef.current
      if (!el) return
      el.focus()
      el.select()
    }, 0)
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

  const displayErrors = localErrors.length > 0 ? localErrors : fieldErrors

  function handleApply() {
    if (busy) return
    const result = parseAndValidateConfigJSON(text, schema)
    if (!result.ok) {
      setLocalErrors(result.errors)
      return
    }
    setLocalErrors([])
    void onApply(result.config)
  }

  return (
    <div
      className="fixed inset-0 z-[80] flex items-center justify-center bg-slate-900/40 p-4"
      role="presentation"
    >
      <button
        type="button"
        aria-label={t('contentTools.authoring.close')}
        disabled={busy}
        className="absolute inset-0 cursor-default border-0 bg-transparent p-0 disabled:cursor-not-allowed"
        onClick={() => {
          if (!busy) onClose()
        }}
      />
      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        aria-describedby={descId}
        className="relative z-10 flex max-h-[min(90vh,40rem)] w-full max-w-xl flex-col rounded-lg border border-border-default bg-surface-raised p-4 shadow-xl dark:border-border-default dark:bg-surface-raised"
      >
        <h2
          id={titleId}
          className="text-sm font-semibold text-fg-default"
        >
          {t('contentTools.authoring.pasteJsonTitle')}
        </h2>
        <p id={descId} className="mt-1 text-xs leading-relaxed text-fg-muted">
          {t('contentTools.authoring.pasteJsonHelp')}
        </p>

        <label htmlFor={textareaId} className="sr-only">
          {t('contentTools.authoring.pasteJsonLabel')}
        </label>
        <textarea
          ref={textareaRef}
          id={textareaId}
          value={text}
          disabled={busy}
          spellCheck={false}
          onChange={(e) => {
            setText(e.target.value)
            if (localErrors.length > 0) setLocalErrors([])
          }}
          className="mt-3 min-h-[14rem] flex-1 resize-y rounded-md border border-border-default bg-surface-raised px-3 py-2 font-mono text-xs leading-relaxed text-fg-default focus:border-slate-400 focus:outline-none focus:ring-2 focus:ring-slate-300 disabled:opacity-60 dark:border-border-default dark:bg-surface-base dark:text-fg-default dark:focus:border-neutral-500 dark:focus:ring-neutral-700"
          placeholder="{}"
          aria-invalid={displayErrors.length > 0 || Boolean(submitError)}
        />

        {displayErrors.length > 0 ? (
          <ul
            className="mt-2 max-h-28 space-y-1 overflow-y-auto text-xs text-rose-600 dark:text-rose-400"
            role="alert"
          >
            {displayErrors.map((err, i) => (
              <li key={`${err.path}-${i}`}>
                {err.path ? (
                  <>
                    <span className="font-medium">{err.path}</span>
                    {': '}
                  </>
                ) : null}
                {err.message}
              </li>
            ))}
          </ul>
        ) : null}

        {submitError ? (
          <p className="mt-2 text-xs text-rose-600 dark:text-rose-400" role="alert">
            {submitError}
          </p>
        ) : null}

        <div className="mt-4 flex flex-wrap justify-end gap-2">
          <button
            type="button"
            disabled={busy}
            onClick={onClose}
            className="rounded-md border border-border-default bg-surface-raised px-3 py-1.5 text-xs font-medium text-fg-muted hover:bg-surface-base disabled:opacity-40 dark:border-border-default dark:bg-surface-raised dark:text-fg-default dark:hover:bg-surface-overlay"
          >
            {t('contentTools.authoring.cancel')}
          </button>
          <button
            type="button"
            disabled={busy || !text.trim()}
            onClick={handleApply}
            className="rounded-md bg-slate-800 px-3 py-1.5 text-xs font-medium text-white hover:bg-slate-700 disabled:opacity-40 dark:bg-neutral-200 dark:text-neutral-900 dark:hover:bg-surface-raised"
          >
            {busy
              ? t('contentTools.authoring.saving')
              : t('contentTools.authoring.pasteJsonApply')}
          </button>
        </div>
      </div>
    </div>
  )
}
