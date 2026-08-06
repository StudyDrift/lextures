import { useEffect, useId, useRef, useState } from 'react'
import { ChevronDown, Download } from 'lucide-react'

type Props = {
  onImportCanvas: () => void
  onImportFromCourse: () => void
  onImportFromJson: () => void
}

export function CourseCatalogImportMenu({
  onImportCanvas,
  onImportFromCourse,
  onImportFromJson,
}: Props) {
  const [open, setOpen] = useState(false)
  const rootRef = useRef<HTMLDivElement>(null)
  const menuId = useId()

  useEffect(() => {
    if (!open) return
    function onDoc(e: MouseEvent) {
      if (!rootRef.current?.contains(e.target as Node)) setOpen(false)
    }
    document.addEventListener('mousedown', onDoc)
    return () => document.removeEventListener('mousedown', onDoc)
  }, [open])

  return (
    <div ref={rootRef} className="relative block w-full text-start sm:inline-block sm:w-auto">
      <button
        type="button"
        aria-haspopup="menu"
        aria-expanded={open}
        aria-controls={open ? menuId : undefined}
        onClick={() => setOpen((o) => !o)}
        className="inline-flex w-full items-center justify-center gap-2 rounded-xl border border-border-default bg-surface-raised px-4 py-2.5 text-sm font-semibold text-fg-default shadow-sm transition-[background-color,color,border-color] hover:border-border-strong hover:bg-surface-base dark:border-border-default dark:bg-surface-raised dark:text-fg-default dark:hover:border-border-default dark:hover:bg-surface-overlay sm:w-auto"
      >
        <Download className="h-4 w-4 shrink-0" aria-hidden />
        <span>Import</span>
        <ChevronDown
          className={`h-4 w-4 shrink-0 transition-transform ${open ? 'rotate-180' : ''}`}
          aria-hidden
        />
      </button>

      {open && (
        <div
          id={menuId}
          role="menu"
          aria-label="Import course from"
          className="absolute start-0 end-0 z-50 mt-1 min-w-0 overflow-hidden rounded-xl border border-border-default bg-surface-raised py-1 shadow-lg shadow-slate-900/10 sm:left-auto sm:end-0 sm:min-w-[14rem] dark:border-border-default dark:bg-surface-overlay dark:shadow-black/40"
        >
          <button
            type="button"
            role="menuitem"
            onClick={() => {
              onImportFromCourse()
              setOpen(false)
            }}
            className="flex w-full flex-col gap-0.5 px-2.5 py-2 text-start text-sm transition-[background-color,color,border-color] hover:bg-surface-base dark:hover:bg-neutral-700"
          >
            <span className="font-semibold text-slate-950 dark:text-fg-default">From another course</span>
            <span className="text-xs text-fg-muted">
              Copy content from a course you already teach in Lextures
            </span>
          </button>
          <button
            type="button"
            role="menuitem"
            onClick={() => {
              onImportCanvas()
              setOpen(false)
            }}
            className="flex w-full flex-col gap-0.5 px-2.5 py-2 text-start text-sm transition-[background-color,color,border-color] hover:bg-surface-base dark:hover:bg-neutral-700"
          >
            <span className="font-semibold text-slate-950 dark:text-fg-default">Canvas LMS</span>
            <span className="text-xs text-fg-muted">
              Import courses with a Canvas API token
            </span>
          </button>
          <button
            type="button"
            role="menuitem"
            onClick={() => {
              onImportFromJson()
              setOpen(false)
            }}
            className="flex w-full flex-col gap-0.5 px-2.5 py-2 text-start text-sm transition-[background-color,color,border-color] hover:bg-surface-base dark:hover:bg-neutral-700"
          >
            <span className="font-semibold text-slate-950 dark:text-fg-default">From JSON</span>
            <span className="text-xs text-fg-muted">
              Create a course from a Lextures JSON export file
            </span>
          </button>
        </div>
      )}
    </div>
  )
}
