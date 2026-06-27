import { useEffect, useId, useRef, useState } from 'react'
import { Check, ChevronDown, Columns3 } from 'lucide-react'

type StudentTodoViewPickerProps = {
  /** When true, empty weekday columns collapse to a narrow strip. */
  collapseEmpty: boolean
  onCollapseEmptyChange: (collapse: boolean) => void
}

const OPTIONS: { value: boolean; title: string; subtitle: string }[] = [
  { value: false, title: 'Open by default', subtitle: 'Show every day at full width' },
  { value: true, title: 'Collapse empty', subtitle: 'Shrink days with no tasks' },
]

export function StudentTodoViewPicker({ collapseEmpty, onCollapseEmptyChange }: StudentTodoViewPickerProps) {
  const [open, setOpen] = useState(false)
  const rootRef = useRef<HTMLDivElement>(null)
  const menuId = useId()

  useEffect(() => {
    if (!open) return
    function onDoc(e: MouseEvent) {
      if (!rootRef.current?.contains(e.target as Node)) setOpen(false)
    }
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape') setOpen(false)
    }
    document.addEventListener('mousedown', onDoc)
    document.addEventListener('keydown', onKey)
    return () => {
      document.removeEventListener('mousedown', onDoc)
      document.removeEventListener('keydown', onKey)
    }
  }, [open])

  const activeOption = collapseEmpty ? OPTIONS[1] : OPTIONS[0]

  return (
    <div ref={rootRef} className="relative block w-full text-start sm:inline-block sm:w-auto">
      <button
        type="button"
        aria-haspopup="listbox"
        aria-expanded={open}
        aria-controls={open ? menuId : undefined}
        onClick={() => setOpen((prev) => !prev)}
        className="inline-flex w-full min-w-[12rem] items-center justify-between gap-2 rounded-xl border border-slate-200 bg-white px-3 py-2.5 text-start shadow-sm transition-[background-color,border-color,box-shadow,transform] duration-150 ease-out hover:border-slate-300 hover:bg-slate-50 active:scale-[0.98] dark:border-neutral-700 dark:bg-neutral-900 dark:text-neutral-100 dark:hover:border-neutral-600 dark:hover:bg-neutral-800 sm:w-auto"
      >
        <span className="flex min-w-0 items-center gap-2">
          <Columns3 className="h-4 w-4 shrink-0 text-indigo-600 dark:text-indigo-400" aria-hidden />
          <span className="min-w-0">
            <span className="block truncate text-sm font-semibold text-slate-900 dark:text-neutral-100">
              View
            </span>
            <span className="block truncate text-[11px] text-slate-500 dark:text-neutral-400">
              {activeOption.title}
            </span>
          </span>
        </span>
        <ChevronDown
          className={`h-4 w-4 shrink-0 text-slate-500 transition-transform duration-150 dark:text-neutral-400 ${open ? 'rotate-180' : ''}`}
          aria-hidden
        />
      </button>

      {open ? (
        <ul
          id={menuId}
          role="listbox"
          aria-label="Column view"
          className="absolute end-0 top-[calc(100%+0.375rem)] z-50 max-h-72 w-full min-w-[14rem] overflow-y-auto rounded-xl border border-slate-200 bg-white p-1 shadow-lg shadow-slate-900/10 dark:border-neutral-600 dark:bg-neutral-800 dark:shadow-black/40 sm:w-auto"
        >
          {OPTIONS.map((option) => {
            const active = option.value === collapseEmpty
            return (
              <li key={String(option.value)} role="presentation">
                <button
                  type="button"
                  role="option"
                  aria-selected={active}
                  onClick={() => {
                    onCollapseEmptyChange(option.value)
                    setOpen(false)
                  }}
                  className={[
                    'flex w-full items-center justify-between gap-2 rounded-lg px-2.5 py-2 text-start transition-colors duration-150',
                    active
                      ? 'bg-indigo-50 text-indigo-950 dark:bg-indigo-950/50 dark:text-indigo-100'
                      : 'text-slate-800 hover:bg-slate-50 dark:text-neutral-100 dark:hover:bg-neutral-800',
                  ].join(' ')}
                >
                  <span className="min-w-0">
                    <span className="block text-sm font-medium">{option.title}</span>
                    <span className="block text-[11px] text-slate-500 dark:text-neutral-400">
                      {option.subtitle}
                    </span>
                  </span>
                  {active ? <Check className="h-4 w-4 shrink-0 text-indigo-600 dark:text-indigo-400" aria-hidden /> : null}
                </button>
              </li>
            )
          })}
        </ul>
      ) : null}
    </div>
  )
}
