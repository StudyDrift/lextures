import { useCallback, useEffect, useId, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { applyMarkdownEdit, type MarkdownEditKind } from '../editor/block-editor/markdown-insert'
import { MarkdownFormatToolbar } from '../editor/block-editor/markdown-format-toolbar'
import { InputDialog } from '../input-dialog'

export type MarkdownEmailEditorProps = {
  value: string
  onChange: (markdown: string) => void
  onInsertReady?: (insert: (token: string) => void) => void
  disabled?: boolean
  placeholder?: string
  'aria-label'?: string
}

/**
 * Shared Markdown email body editor (ET-3). Uses the app's markdown toolbar
 * helpers and a plain textarea so merge tokens like {{link}} stay literal.
 */
export function MarkdownEmailEditor({
  value,
  onChange,
  onInsertReady,
  disabled = false,
  placeholder,
  'aria-label': ariaLabel,
}: MarkdownEmailEditorProps) {
  const { t } = useTranslation('common')
  const taRef = useRef<HTMLTextAreaElement>(null)
  const areaId = useId()
  const [linkDialogOpen, setLinkDialogOpen] = useState(false)
  const [linkUrl, setLinkUrl] = useState('')
  const pendingLinkSel = useRef<{ start: number; end: number } | null>(null)

  const insertAtCursor = useCallback(
    (token: string) => {
      const el = taRef.current
      if (!el || disabled) return
      const start = el.selectionStart ?? value.length
      const end = el.selectionEnd ?? value.length
      const next = value.slice(0, start) + token + value.slice(end)
      onChange(next)
      requestAnimationFrame(() => {
        el.focus()
        const pos = start + token.length
        el.setSelectionRange(pos, pos)
      })
    },
    [disabled, onChange, value],
  )

  useEffect(() => {
    if (!onInsertReady) return
    onInsertReady(insertAtCursor)
  }, [insertAtCursor, onInsertReady])

  const onApply = useCallback(
    (kind: MarkdownEditKind) => {
      const el = taRef.current
      if (!el || disabled) return
      const start = el.selectionStart ?? 0
      const end = el.selectionEnd ?? 0
      if (kind === 'link') {
        pendingLinkSel.current = { start, end }
        setLinkUrl('https://')
        setLinkDialogOpen(true)
        return
      }
      const result = applyMarkdownEdit(value, start, end, kind)
      onChange(result.value)
      requestAnimationFrame(() => {
        el.focus()
        el.setSelectionRange(result.selStart, result.selEnd)
      })
    },
    [disabled, onChange, value],
  )

  return (
    <>
    <div className="rounded-xl border border-slate-200 dark:border-neutral-700">
      <div className="flex flex-wrap items-center gap-1 border-b border-slate-200 p-2 dark:border-neutral-700">
        <button
          type="button"
          disabled={disabled}
          onMouseDown={(e) => e.preventDefault()}
          onClick={() => {
            const el = taRef.current
            if (!el || disabled) return
            const start = el.selectionStart ?? 0
            const end = el.selectionEnd ?? 0
            // Prefix selection / current line with heading.
            const lineStart = value.lastIndexOf('\n', Math.max(0, start - 1)) + 1
            const next = value.slice(0, lineStart) + '## ' + value.slice(lineStart)
            onChange(next)
            requestAnimationFrame(() => {
              el.focus()
              el.setSelectionRange(start + 3, end + 3)
            })
          }}
          className="rounded px-2 py-1 text-xs font-semibold text-slate-600 hover:bg-slate-100 disabled:opacity-50 dark:text-neutral-300 dark:hover:bg-neutral-800"
          aria-label={t('emailTemplates.toolbar.heading', { defaultValue: 'H2' })}
        >
          {t('emailTemplates.toolbar.heading', { defaultValue: 'H2' })}
        </button>
        <MarkdownFormatToolbar disabled={disabled} onApply={onApply} />
      </div>
      <label htmlFor={areaId} className="sr-only">
        {ariaLabel ?? t('emailTemplates.editorLabel', { defaultValue: 'Email template markdown' })}
      </label>
      <textarea
        id={areaId}
        ref={taRef}
        value={value}
        disabled={disabled}
        placeholder={
          placeholder ??
          t('emailTemplates.editorPlaceholder', {
            defaultValue: 'Write the email body in Markdown. Use {{tokens}} for merge fields.',
          })
        }
        onChange={(e) => onChange(e.target.value)}
        rows={14}
        className="block w-full resize-y border-0 bg-transparent px-3 py-2 font-mono text-sm text-slate-900 outline-none focus:ring-0 disabled:opacity-50 dark:text-neutral-100 min-h-[220px]"
        spellCheck
      />
      <div className="border-t border-slate-200 p-2 text-xs text-slate-500 dark:border-neutral-700 dark:text-neutral-400">
        {t('emailTemplates.editorHint', {
          defaultValue: 'Markdown with merge fields. Type {{ or use the field buttons to insert tokens.',
        })}
      </div>
    </div>
    <InputDialog
      open={linkDialogOpen}
      title={t('dialogs.linkUrl.title')}
      label={t('dialogs.linkUrl.label')}
      placeholder={t('dialogs.linkUrl.placeholder')}
      value={linkUrl}
      onValueChange={setLinkUrl}
      onConfirm={(url) => {
        const el = taRef.current
        const sel = pendingLinkSel.current
        setLinkDialogOpen(false)
        pendingLinkSel.current = null
        if (!el || !sel) return
        const trimmed = url.trim() || 'https://'
        const result = applyMarkdownEdit(value, sel.start, sel.end, 'link', trimmed)
        onChange(result.value)
        requestAnimationFrame(() => {
          el.focus()
          el.setSelectionRange(result.selStart, result.selEnd)
        })
      }}
      onClose={() => {
        setLinkDialogOpen(false)
        pendingLinkSel.current = null
      }}
    />
    </>
  )
}

export function MergeFieldChip({
  label,
  token,
  onInsert,
  disabled,
}: {
  label: string
  token: string
  onInsert: (token: string) => void
  disabled?: boolean
}) {
  return (
    <button
      type="button"
      disabled={disabled}
      onClick={() => onInsert(token)}
      className="rounded-full border border-slate-200 bg-white px-2 py-0.5 text-xs text-slate-700 hover:bg-slate-50 disabled:opacity-50 dark:border-neutral-600 dark:bg-neutral-900 dark:text-neutral-200 dark:hover:bg-neutral-800"
      title={token}
    >
      {label}
    </button>
  )
}
