import type { Editor } from '@tiptap/core'
import {
  BetweenHorizonalEnd,
  BetweenVerticalEnd,
  Columns3,
  Minus,
  Plus,
  Rows3,
  TableProperties,
  Trash2,
} from 'lucide-react'
import { useEffect, useState, type MouseEvent } from 'react'
import { createPortal } from 'react-dom'
import { adjustSelectedColumnWidth } from './markdown-table-commands'

export type MarkdownTableControlsProps = {
  editor: Editor
  disabled?: boolean
}

type Anchor = { left: number; top: number }

/**
 * Contextual table controls anchored above the active table while the caret is inside it.
 */
export function MarkdownTableControls({ editor, disabled }: MarkdownTableControlsProps) {
  const [anchor, setAnchor] = useState<Anchor | null>(null)

  useEffect(() => {
    const sync = () => {
      if (disabled || !editor.isEditable || !editor.isActive('table')) {
        setAnchor(null)
        return
      }
      const { view, state } = editor
      const $from = state.selection.$from
      let tablePos = -1
      for (let d = $from.depth; d > 0; d--) {
        if ($from.node(d).type.name === 'table') {
          tablePos = $from.before(d)
          break
        }
      }
      if (tablePos < 0) {
        setAnchor(null)
        return
      }
      const dom = view.nodeDOM(tablePos)
      const el = dom instanceof HTMLElement ? dom : null
      const rect = el?.getBoundingClientRect()
      if (!rect) {
        setAnchor(null)
        return
      }
      setAnchor({
        left: Math.max(8, Math.min(rect.left, window.innerWidth - 360)),
        top: Math.max(8, rect.top - 44),
      })
    }

    sync()
    editor.on('selectionUpdate', sync)
    editor.on('update', sync)
    window.addEventListener('scroll', sync, true)
    window.addEventListener('resize', sync)
    return () => {
      editor.off('selectionUpdate', sync)
      editor.off('update', sync)
      window.removeEventListener('scroll', sync, true)
      window.removeEventListener('resize', sync)
    }
  }, [editor, disabled])

  if (!anchor || disabled) return null

  function preventBlur(e: MouseEvent) {
    e.preventDefault()
  }

  const btn =
    'flex h-7 w-7 shrink-0 items-center justify-center rounded text-slate-600 hover:bg-slate-100 disabled:cursor-not-allowed disabled:opacity-40 dark:text-neutral-300 dark:hover:bg-neutral-700'

  return createPortal(
    <div
      data-toolbar-anchor
      role="toolbar"
      aria-label="Table controls"
      className="pointer-events-auto flex h-9 w-max max-w-[calc(100vw-1rem)] items-center gap-0.5 rounded-lg border border-slate-200 bg-white px-1 py-0.5 shadow-md shadow-slate-900/10 dark:border-neutral-600 dark:bg-neutral-800 dark:shadow-black/40"
      style={{ position: 'fixed', left: anchor.left, top: anchor.top, zIndex: 55 }}
      onMouseDown={(e) => e.stopPropagation()}
    >
      <span className="me-0.5 flex items-center gap-1 px-1 text-[10px] font-semibold uppercase tracking-wide text-slate-400 dark:text-neutral-500">
        <TableProperties className="h-3.5 w-3.5" aria-hidden />
        Table
      </span>
      <button
        type="button"
        className={btn}
        aria-label="Add column"
        title="Add column"
        onMouseDown={preventBlur}
        onClick={() => editor.chain().focus().addColumnAfter().run()}
      >
        <BetweenVerticalEnd className="h-4 w-4" />
      </button>
      <button
        type="button"
        className={btn}
        aria-label="Remove column"
        title="Remove column"
        onMouseDown={preventBlur}
        onClick={() => editor.chain().focus().deleteColumn().run()}
      >
        <Columns3 className="h-4 w-4" />
      </button>
      <button
        type="button"
        className={btn}
        aria-label="Widen column"
        title="Widen column (or drag the column edge)"
        onMouseDown={preventBlur}
        onClick={() => adjustSelectedColumnWidth(editor, 40)}
      >
        <Plus className="h-4 w-4" />
      </button>
      <button
        type="button"
        className={btn}
        aria-label="Narrow column"
        title="Narrow column (or drag the column edge)"
        onMouseDown={preventBlur}
        onClick={() => adjustSelectedColumnWidth(editor, -40)}
      >
        <Minus className="h-4 w-4" />
      </button>
      <span className="mx-0.5 h-5 w-px shrink-0 bg-slate-200 dark:bg-neutral-600" aria-hidden />
      <button
        type="button"
        className={btn}
        aria-label="Add row"
        title="Add row"
        onMouseDown={preventBlur}
        onClick={() => editor.chain().focus().addRowAfter().run()}
      >
        <BetweenHorizonalEnd className="h-4 w-4" />
      </button>
      <button
        type="button"
        className={btn}
        aria-label="Remove row"
        title="Remove row"
        onMouseDown={preventBlur}
        onClick={() => editor.chain().focus().deleteRow().run()}
      >
        <Rows3 className="h-4 w-4" />
      </button>
      <span className="mx-0.5 h-5 w-px shrink-0 bg-slate-200 dark:bg-neutral-600" aria-hidden />
      <button
        type="button"
        className={`${btn} text-rose-600 hover:bg-rose-50 dark:text-rose-300 dark:hover:bg-rose-950/40`}
        aria-label="Delete table"
        title="Delete table"
        onMouseDown={preventBlur}
        onClick={() => editor.chain().focus().deleteTable().run()}
      >
        <Trash2 className="h-4 w-4" />
      </button>
    </div>,
    document.body,
  )
}
