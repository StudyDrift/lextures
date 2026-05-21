import {
  Bold,
  Braces,
  Code,
  Image as ImageIcon,
  Italic,
  Link as LinkIcon,
  List,
  ListOrdered,
  MoreHorizontal,
  Sigma,
} from 'lucide-react'
import { useEffect, useId, useRef, useState, useSyncExternalStore, type MouseEvent as ReactMouseEvent } from 'react'
import type { MarkdownEditKind } from './markdown-insert'

export type MarkdownFormatToolbarProps = {
  disabled?: boolean
  onApply: (kind: MarkdownEditKind) => void
  /** Insert course image: file picker and drag-and-drop onto the button. */
  courseImage?: {
    onPickClick: () => void
    onFiles: (files: File[]) => void
  }
  /** Open math insert popover (LaTeX + KaTeX preview). */
  mathInsert?: {
    onOpen: () => void
    /** Anchor element for popover positioning (mount/unmount). */
    registerMathAnchor?: (node: HTMLButtonElement | null) => void
  }
}

const iconBtnClass =
  'flex h-11 w-11 shrink-0 items-center justify-center rounded text-slate-600 hover:bg-slate-100 disabled:cursor-not-allowed disabled:opacity-40 dark:text-neutral-300 dark:hover:bg-neutral-700 sm:h-7 sm:w-7'

function subscribeMinSm(onChange: () => void) {
  if (typeof window === 'undefined' || typeof window.matchMedia !== 'function') return () => {}
  const mq = window.matchMedia('(min-width: 640px)')
  mq.addEventListener('change', onChange)
  return () => mq.removeEventListener('change', onChange)
}

function snapshotMinSm(): boolean {
  if (typeof window === 'undefined' || typeof window.matchMedia !== 'function') return true
  return window.matchMedia('(min-width: 640px)').matches
}

/**
 * Markdown formatting buttons for use inside BlockFloatingToolbar children.
 * Uses mousedown preventDefault so the textarea keeps focus while clicking.
 * Below the `sm` (640px) breakpoint, primary actions stay on the bar and the rest move under “More”.
 */
export function MarkdownFormatToolbar({ disabled, onApply, courseImage, mathInsert }: MarkdownFormatToolbarProps) {
  const isSmUp = useSyncExternalStore(subscribeMinSm, snapshotMinSm, () => true)
  const [moreOpen, setMoreOpen] = useState(false)
  const moreRootRef = useRef<HTMLDivElement>(null)
  const moreMenuId = useId()

  useEffect(() => {
    if (!moreOpen) return
    function onDocMouseDown(e: globalThis.MouseEvent) {
      if (!moreRootRef.current?.contains(e.target as Node)) setMoreOpen(false)
    }
    function onKey(e: globalThis.KeyboardEvent) {
      if (e.key === 'Escape') setMoreOpen(false)
    }
    document.addEventListener('mousedown', onDocMouseDown)
    document.addEventListener('keydown', onKey)
    return () => {
      document.removeEventListener('mousedown', onDocMouseDown)
      document.removeEventListener('keydown', onKey)
    }
  }, [moreOpen])

  function preventBlur(e: ReactMouseEvent) {
    e.preventDefault()
  }

  const divider = <span className="mx-0.5 h-5 w-px shrink-0 bg-slate-200 dark:bg-neutral-600" aria-hidden />

  const primaryRow = (
    <>
      {divider}
      <button
        type="button"
        disabled={disabled}
        onMouseDown={preventBlur}
        onClick={() => onApply('bulletList')}
        className={iconBtnClass}
        aria-label="Bullet list"
        title="Bullet list"
      >
        <List className="h-4 w-4" />
      </button>
      <button
        type="button"
        disabled={disabled}
        onMouseDown={preventBlur}
        onClick={() => onApply('orderedList')}
        className={iconBtnClass}
        aria-label="Numbered list"
        title="Numbered list"
      >
        <ListOrdered className="h-4 w-4" />
      </button>
      <button
        type="button"
        disabled={disabled}
        onMouseDown={preventBlur}
        onClick={() => onApply('bold')}
        className={`${iconBtnClass} font-bold text-slate-700 dark:text-neutral-200`}
        aria-label="Bold"
        title="Bold"
      >
        <Bold className="h-4 w-4" />
      </button>
      <button
        type="button"
        disabled={disabled}
        onMouseDown={preventBlur}
        onClick={() => onApply('italic')}
        className={`${iconBtnClass} text-slate-700 dark:text-neutral-200`}
        aria-label="Italic"
        title="Italic"
      >
        <Italic className="h-4 w-4" />
      </button>
    </>
  )

  const extendedRow = (
    <>
      <button
        type="button"
        disabled={disabled}
        onMouseDown={preventBlur}
        onClick={() => onApply('inlineCode')}
        className={iconBtnClass}
        aria-label="Inline code"
        title="Inline code"
      >
        <Code className="h-4 w-4" />
      </button>
      <button
        type="button"
        disabled={disabled}
        onMouseDown={preventBlur}
        onClick={() => onApply('codeBlock')}
        className={iconBtnClass}
        aria-label="Code block"
        title="Code block"
      >
        <Braces className="h-4 w-4" />
      </button>
      <button
        type="button"
        disabled={disabled}
        onMouseDown={preventBlur}
        onClick={() => onApply('link')}
        className={iconBtnClass}
        aria-label="Link"
        title="Link"
      >
        <LinkIcon className="h-4 w-4" />
      </button>
      {mathInsert ? (
        <button
          type="button"
          ref={(node) => {
            mathInsert.registerMathAnchor?.(node)
          }}
          disabled={disabled}
          onMouseDown={preventBlur}
          onClick={() => {
            mathInsert.onOpen()
            setMoreOpen(false)
          }}
          className={iconBtnClass}
          aria-label="Insert math"
          title="Insert math (LaTeX)"
        >
          <Sigma className="h-4 w-4" />
        </button>
      ) : null}
      {courseImage ? (
        <>
          {divider}
          <button
            type="button"
            disabled={disabled}
            onMouseDown={preventBlur}
            onClick={() => {
              courseImage.onPickClick()
              setMoreOpen(false)
            }}
            onDragOver={(e) => {
              if (disabled) return
              e.preventDefault()
              e.dataTransfer.dropEffect = 'copy'
            }}
            onDrop={(e) => {
              if (disabled) return
              e.preventDefault()
              const files = [...e.dataTransfer.files].filter((f) => f.type.startsWith('image/'))
              if (files.length) courseImage.onFiles(files)
            }}
            className={iconBtnClass}
            aria-label="Insert image"
            title="Insert image (drop file here or click)"
          >
            <ImageIcon className="h-4 w-4" />
          </button>
        </>
      ) : null}
    </>
  )

  if (isSmUp) {
    return (
      <>
        {primaryRow}
        {extendedRow}
      </>
    )
  }

  return (
    <>
      {primaryRow}
      <div ref={moreRootRef} className="relative flex shrink-0 items-center">
        <button
          type="button"
          disabled={disabled}
          onMouseDown={preventBlur}
          aria-haspopup="menu"
          aria-expanded={moreOpen}
          aria-controls={moreOpen ? moreMenuId : undefined}
          className={iconBtnClass}
          aria-label="More formatting"
          title="More"
          onClick={() => setMoreOpen((o) => !o)}
        >
          <MoreHorizontal className="h-4 w-4" aria-hidden />
        </button>
        {moreOpen ? (
          <div
            id={moreMenuId}
            role="menu"
            aria-label="More formatting"
            className="absolute right-0 top-full z-[250] mt-1 flex min-w-[10rem] flex-col gap-0.5 rounded-lg border border-slate-200 bg-white p-1 shadow-lg dark:border-neutral-600 dark:bg-neutral-900"
          >
            <button
              type="button"
              role="menuitem"
              disabled={disabled}
              onMouseDown={preventBlur}
              onClick={() => {
                onApply('inlineCode')
                setMoreOpen(false)
              }}
              className="flex w-full items-center gap-2 rounded-md px-3 py-2.5 text-left text-sm text-slate-800 hover:bg-slate-50 disabled:opacity-40 dark:text-neutral-100 dark:hover:bg-neutral-800"
            >
              <Code className="h-4 w-4 shrink-0 opacity-70" aria-hidden />
              Inline code
            </button>
            <button
              type="button"
              role="menuitem"
              disabled={disabled}
              onMouseDown={preventBlur}
              onClick={() => {
                onApply('codeBlock')
                setMoreOpen(false)
              }}
              className="flex w-full items-center gap-2 rounded-md px-3 py-2.5 text-left text-sm text-slate-800 hover:bg-slate-50 disabled:opacity-40 dark:text-neutral-100 dark:hover:bg-neutral-800"
            >
              <Braces className="h-4 w-4 shrink-0 opacity-70" aria-hidden />
              Code block
            </button>
            <button
              type="button"
              role="menuitem"
              disabled={disabled}
              onMouseDown={preventBlur}
              onClick={() => {
                onApply('link')
                setMoreOpen(false)
              }}
              className="flex w-full items-center gap-2 rounded-md px-3 py-2.5 text-left text-sm text-slate-800 hover:bg-slate-50 disabled:opacity-40 dark:text-neutral-100 dark:hover:bg-neutral-800"
            >
              <LinkIcon className="h-4 w-4 shrink-0 opacity-70" aria-hidden />
              Link
            </button>
            {mathInsert ? (
              <button
                type="button"
                role="menuitem"
                ref={(node) => {
                  mathInsert.registerMathAnchor?.(node)
                }}
                disabled={disabled}
                onMouseDown={preventBlur}
                onClick={() => {
                  mathInsert.onOpen()
                  setMoreOpen(false)
                }}
                className="flex w-full items-center gap-2 rounded-md px-3 py-2.5 text-left text-sm text-slate-800 hover:bg-slate-50 disabled:opacity-40 dark:text-neutral-100 dark:hover:bg-neutral-800"
              >
                <Sigma className="h-4 w-4 shrink-0 opacity-70" aria-hidden />
                Math
              </button>
            ) : null}
            {courseImage ? (
              <button
                type="button"
                role="menuitem"
                disabled={disabled}
                onMouseDown={preventBlur}
                onClick={() => {
                  courseImage.onPickClick()
                  setMoreOpen(false)
                }}
                className="flex w-full items-center gap-2 rounded-md px-3 py-2.5 text-left text-sm text-slate-800 hover:bg-slate-50 disabled:opacity-40 dark:text-neutral-100 dark:hover:bg-neutral-800"
              >
                <ImageIcon className="h-4 w-4 shrink-0 opacity-70" aria-hidden />
                Image
              </button>
            ) : null}
          </div>
        ) : null}
      </div>
    </>
  )
}
