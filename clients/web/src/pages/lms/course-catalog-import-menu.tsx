import { useEffect, useId, useRef, useState } from 'react'
import { ChevronDown, Download } from 'lucide-react'

type Props = {
  onImportCanvas: () => void
}

export function CourseCatalogImportMenu({ onImportCanvas }: Props) {
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
        className="inline-flex w-full items-center justify-center gap-2 rounded-xl border border-slate-200 bg-white px-4 py-2.5 text-sm font-semibold text-slate-800 shadow-sm transition hover:border-slate-300 hover:bg-slate-50 dark:border-neutral-700 dark:bg-neutral-900 dark:text-neutral-100 dark:hover:border-neutral-600 dark:hover:bg-neutral-800 sm:w-auto"
      >
        <Download className="h-4 w-4 shrink-0" aria-hidden />
        <span>Import</span>
        <ChevronDown
          className={`h-4 w-4 shrink-0 transition ${open ? 'rotate-180' : ''}`}
          aria-hidden
        />
      </button>

      {open && (
        <div
          id={menuId}
          role="menu"
          aria-label="Import course from"
          className="absolute start-0 end-0 z-50 mt-1 min-w-0 overflow-hidden rounded-xl border border-slate-200 bg-white py-1 shadow-lg shadow-slate-900/10 sm:left-auto sm:end-0 sm:min-w-[14rem] dark:border-neutral-600 dark:bg-neutral-800 dark:shadow-black/40"
        >
          <button
            type="button"
            role="menuitem"
            onClick={() => {
              onImportCanvas()
              setOpen(false)
            }}
            className="flex w-full flex-col gap-0.5 px-2.5 py-2 text-start text-sm transition hover:bg-slate-50 dark:hover:bg-neutral-700"
          >
            <span className="font-semibold text-slate-950 dark:text-neutral-100">Canvas LMS</span>
            <span className="text-xs text-slate-500 dark:text-neutral-400">
              Import courses with a Canvas API token
            </span>
          </button>
        </div>
      )}
    </div>
  )
}
