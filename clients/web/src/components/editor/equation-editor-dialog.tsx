import type { Editor } from '@tiptap/core'
import { useCallback, useEffect, useId, useMemo, useRef, useState } from 'react'
import { equationI18n } from '../../lib/i18n/equation'
import { isEquationEditorEnabled, loadKatex, renderKatexSafe, type KatexModule } from '../../lib/math'
import { postCourseContext } from '../../lib/courses-api'
import {
  caretOffsetAfterSymbolInsert,
  MATH_SYMBOL_CATEGORY_ORDER,
  MATH_SYMBOL_PALETTE,
  type MathSymbolCategory,
} from './math-symbol-palette'
import type { EquationEditTarget } from './equation-editor-context'

export type EquationEditorDialogProps = {
  open: boolean
  onClose: () => void
  editor: Editor | null
  latex: string
  onLatexChange: (next: string) => void
  display: boolean
  onDisplayChange: (display: boolean) => void
  editTarget: EquationEditTarget | null
  courseCode?: string
  structureItemId?: string
  onInserted?: () => void
}

export function EquationEditorDialog({
  open,
  onClose,
  editor,
  latex,
  onLatexChange,
  display,
  onDisplayChange,
  editTarget,
  courseCode,
  structureItemId,
  onInserted,
}: EquationEditorDialogProps) {
  const titleId = useId()
  const inputRef = useRef<HTMLTextAreaElement>(null)
  const [katex, setKatex] = useState<KatexModule | null>(null)
  const [activeCategory, setActiveCategory] = useState<MathSymbolCategory>('general')
  const enabled = isEquationEditorEnabled()

  useEffect(() => {
    if (!open || !enabled) return
    let cancelled = false
    void loadKatex().then((k) => {
      if (!cancelled) setKatex(k)
    })
    return () => {
      cancelled = true
    }
  }, [open, enabled])

  useEffect(() => {
    if (!open) return
    requestAnimationFrame(() => inputRef.current?.focus())
  }, [open])

  useEffect(() => {
    if (!open) return
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape') {
        e.preventDefault()
        onClose()
      }
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [open, onClose])

  const preview = useMemo(() => {
    if (!enabled) {
      return { html: `<code>${latex}</code>`, failed: false }
    }
    if (!katex) {
      return { html: `<span class="text-slate-400">${equationI18n.loadingPreview}</span>`, failed: false }
    }
    return renderKatexSafe(katex, latex, display)
  }, [katex, latex, display, enabled])

  const insertSymbol = useCallback(
    (snippet: string) => {
      const ta = inputRef.current
      if (!ta) {
        onLatexChange(latex + snippet)
        return
      }
      const start = ta.selectionStart
      const end = ta.selectionEnd
      const next = latex.slice(0, start) + snippet + latex.slice(end)
      onLatexChange(next)
      const offset = caretOffsetAfterSymbolInsert(snippet)
      requestAnimationFrame(() => {
        ta.focus()
        const pos = offset != null ? start + offset : start + snippet.length
        ta.setSelectionRange(pos, pos)
      })
    },
    [latex, onLatexChange],
  )

  const confirm = useCallback(() => {
    if (!editor) return
    const t = latex.trim()
    if (!t) return

    if (editTarget) {
      const node =
        display
          ? { type: 'math_block' as const, attrs: { latex: t } }
          : { type: 'math_inline' as const, attrs: { latex: t } }
      const nodeAt = editor.state.doc.nodeAt(editTarget.pos)
      if (nodeAt) {
        editor
          .chain()
          .focus()
          .insertContentAt(
            { from: editTarget.pos, to: editTarget.pos + nodeAt.nodeSize },
            node,
          )
          .run()
      }
    } else {
      const node =
        display
          ? { type: 'math_block' as const, attrs: { latex: t } }
          : { type: 'math_inline' as const, attrs: { latex: t } }
      editor.chain().focus().insertContent(node).run()
    }

    if (courseCode) {
      void postCourseContext(courseCode, {
        kind: 'equation_inserted',
        ...(structureItemId ? { structureItemId } : {}),
      }).catch(() => {
        /* best-effort audit */
      })
    }
    onInserted?.()
    onClose()
  }, [editor, latex, display, editTarget, courseCode, structureItemId, onInserted, onClose])

  if (!open || !enabled) return null

  const isEdit = editTarget != null

  return (
    <div
      className="fixed inset-0 z-[80] flex items-end justify-center bg-slate-900/30 p-4 sm:items-center dark:bg-black/50"
      role="presentation"
      onMouseDown={(e) => {
        if (e.target === e.currentTarget) onClose()
      }}
    >
      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        className="flex max-h-[min(92vh,720px)] w-full max-w-3xl flex-col overflow-hidden rounded-xl border border-slate-200 bg-white shadow-2xl dark:border-neutral-600 dark:bg-neutral-900"
        onMouseDown={(e) => e.stopPropagation()}
      >
        <header className="border-b border-slate-100 px-4 py-3 dark:border-neutral-700">
          <h2 id={titleId} className="text-base font-semibold text-slate-900 dark:text-neutral-100">
            {isEdit ? equationI18n.editMath : equationI18n.editorTitle}
          </h2>
          <p className="mt-0.5 text-xs text-slate-500 dark:text-neutral-400">
            LaTeX with live preview ·{' '}
            <a
              href="https://katex.org/docs/supported.html"
              target="_blank"
              rel="noreferrer noopener"
              className="text-indigo-600 underline dark:text-indigo-400"
            >
              {equationI18n.helpLink}
            </a>
          </p>
        </header>

        <div className="grid min-h-0 flex-1 gap-0 sm:grid-cols-2">
          <div className="flex min-h-0 flex-col border-b border-slate-100 p-4 sm:border-b-0 sm:border-r dark:border-neutral-700">
            <label htmlFor={`${titleId}-latex`} className="text-xs font-medium text-slate-600 dark:text-neutral-300">
              {equationI18n.latexInput}
            </label>
            <textarea
              id={`${titleId}-latex`}
              ref={inputRef}
              value={latex}
              onChange={(e) => onLatexChange(e.target.value)}
              rows={6}
              className="mt-1.5 min-h-[8rem] w-full flex-1 resize-y rounded-lg border border-slate-200 px-2.5 py-2 font-mono text-[13px] text-slate-900 dark:border-neutral-600 dark:bg-neutral-950 dark:text-neutral-100"
              spellCheck={false}
              onKeyDown={(e) => {
                if (e.key === 'Enter' && (e.metaKey || e.ctrlKey)) {
                  e.preventDefault()
                  confirm()
                }
              }}
            />
            <div className="mt-3 flex gap-3">
              <label className="flex cursor-pointer items-center gap-2 text-xs text-slate-700 dark:text-neutral-200">
                <input
                  type="radio"
                  name={`${titleId}-mode`}
                  checked={!display}
                  onChange={() => onDisplayChange(false)}
                />
                {equationI18n.inline}
              </label>
              <label className="flex cursor-pointer items-center gap-2 text-xs text-slate-700 dark:text-neutral-200">
                <input
                  type="radio"
                  name={`${titleId}-mode`}
                  checked={display}
                  onChange={() => onDisplayChange(true)}
                />
                {equationI18n.block}
              </label>
            </div>
          </div>

          <div className="flex min-h-0 flex-col p-4">
            <p className="text-xs font-medium text-slate-600 dark:text-neutral-300">{equationI18n.preview}</p>
            <div
              className={`mt-1.5 min-h-[8rem] flex-1 overflow-x-auto rounded-lg border px-3 py-3 ${
                preview.failed
                  ? 'border-rose-300 bg-rose-50/80 dark:border-rose-800 dark:bg-rose-950/30'
                  : 'border-slate-100 bg-slate-50 dark:border-neutral-700 dark:bg-neutral-950'
              }`}
              aria-live="polite"
              role="math"
              aria-label={latex.trim() || 'Equation preview'}
            >
              <span dangerouslySetInnerHTML={{ __html: preview.html }} />
            </div>
            {preview.failed ? (
              <p className="mt-2 text-xs text-rose-700 dark:text-rose-300" role="alert">
                <span className="font-medium">{equationI18n.syntaxError}</span> — {equationI18n.syntaxErrorHint}
              </p>
            ) : null}
          </div>
        </div>

        <div className="border-t border-slate-100 px-4 py-3 dark:border-neutral-700">
          <p className="mb-2 text-[11px] font-medium uppercase tracking-wide text-slate-500 dark:text-neutral-400">
            {equationI18n.symbols}
          </p>
          <div
            className="flex flex-wrap gap-1 border-b border-slate-100 pb-2 dark:border-neutral-700"
            role="tablist"
            aria-label={equationI18n.symbols}
          >
            {MATH_SYMBOL_CATEGORY_ORDER.map((cat) => (
              <button
                key={cat}
                type="button"
                role="tab"
                aria-selected={activeCategory === cat}
                className={`min-h-[36px] rounded-lg px-2.5 text-xs font-medium ${
                  activeCategory === cat
                    ? 'bg-indigo-600 text-white'
                    : 'bg-slate-100 text-slate-700 dark:bg-neutral-800 dark:text-neutral-200'
                }`}
                onClick={() => setActiveCategory(cat)}
              >
                {equationI18n.categories[cat]}
              </button>
            ))}
          </div>
          <div className="mt-2 flex max-h-32 flex-wrap gap-1 overflow-y-auto" role="tabpanel">
            {MATH_SYMBOL_PALETTE[activeCategory].map((sym) => (
              <button
                key={`${activeCategory}-${sym.latex}`}
                type="button"
                className="min-h-[40px] min-w-[40px] rounded-lg border border-slate-200 bg-white px-2 font-mono text-sm text-slate-800 hover:border-indigo-300 hover:bg-indigo-50 dark:border-neutral-600 dark:bg-neutral-800 dark:text-neutral-100 dark:hover:border-indigo-500"
                aria-label={sym.ariaLabel}
                title={sym.latex}
                onClick={() => insertSymbol(sym.latex)}
              >
                {sym.label}
              </button>
            ))}
          </div>
        </div>

        <footer className="flex justify-end gap-2 border-t border-slate-100 px-4 py-3 dark:border-neutral-700">
          <button
            type="button"
            className="rounded-lg border border-slate-200 px-4 py-2 text-sm text-slate-700 dark:border-neutral-600 dark:text-neutral-200"
            onClick={onClose}
          >
            {equationI18n.cancel}
          </button>
          <button
            type="button"
            className="rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-500"
            onClick={confirm}
          >
            {isEdit ? equationI18n.update : equationI18n.insert}
          </button>
        </footer>
      </div>
    </div>
  )
}
