import { useEffect, useId, useRef, useState } from 'react'
import { Check, ChevronDown } from 'lucide-react'
import {
  formatGradeLevelsSummary,
  GRADE_LEVEL_OPTIONS,
  sortGradeLevels,
} from '../../lib/grade-levels'

type GradeLevelMultiSelectProps = {
  id?: string
  value: string[]
  onChange: (next: string[]) => void
  disabled?: boolean
  className?: string
  'aria-label'?: string
}

export function GradeLevelMultiSelect({
  id,
  value,
  onChange,
  disabled,
  className = '',
  'aria-label': ariaLabel = 'Grade levels',
}: GradeLevelMultiSelectProps) {
  const autoId = useId()
  const fieldId = id ?? autoId
  const listboxId = `${fieldId}-listbox`
  const rootRef = useRef<HTMLDivElement>(null)
  const [open, setOpen] = useState(false)
  const selected = new Set(value)

  useEffect(() => {
    if (!open) return
    function onDocPointerDown(e: MouseEvent) {
      if (!rootRef.current?.contains(e.target as Node)) {
        setOpen(false)
      }
    }
    function onKeyDown(e: KeyboardEvent) {
      if (e.key === 'Escape') setOpen(false)
    }
    document.addEventListener('mousedown', onDocPointerDown)
    document.addEventListener('keydown', onKeyDown)
    return () => {
      document.removeEventListener('mousedown', onDocPointerDown)
      document.removeEventListener('keydown', onKeyDown)
    }
  }, [open])

  function toggle(token: string) {
    const next = new Set(selected)
    if (next.has(token)) next.delete(token)
    else next.add(token)
    onChange(sortGradeLevels([...next]))
  }

  function clearAll() {
    onChange([])
  }

  const summary = formatGradeLevelsSummary(value)

  return (
    <div ref={rootRef} className={`relative ${className}`}>
      <button
        type="button"
        id={fieldId}
        disabled={disabled}
        aria-haspopup="listbox"
        aria-expanded={open}
        aria-controls={listboxId}
        aria-label={ariaLabel}
        onClick={() => setOpen((o) => !o)}
        className="flex w-full items-center justify-between gap-2 rounded-xl border border-slate-200 bg-white px-3 py-2.5 text-left text-sm text-slate-900 shadow-sm outline-none ring-indigo-500/0 transition-[background-color,color,border-color,box-shadow] focus:border-indigo-400 focus:ring-2 focus:ring-indigo-500/20 disabled:cursor-not-allowed disabled:opacity-60 dark:border-neutral-700 dark:bg-neutral-950 dark:text-neutral-50 dark:focus:border-indigo-500/60"
      >
        <span
          className={
            value.length === 0
              ? 'truncate text-slate-500 dark:text-neutral-400'
              : 'truncate'
          }
        >
          {summary}
        </span>
        <ChevronDown
          className={`h-4 w-4 shrink-0 text-slate-400 transition-transform dark:text-neutral-500 ${open ? 'rotate-180' : ''}`}
          aria-hidden
        />
      </button>

      {open && (
        <div
          id={listboxId}
          role="listbox"
          aria-multiselectable="true"
          aria-label={ariaLabel}
          className="absolute z-30 mt-1.5 max-h-72 w-full overflow-auto rounded-xl border border-slate-200 bg-white py-1 shadow-lg shadow-slate-900/10 dark:border-neutral-700 dark:bg-neutral-950 dark:shadow-black/40"
        >
          <div className="sticky top-0 z-10 flex items-center justify-between gap-2 border-b border-slate-100 bg-white px-3 py-2 dark:border-neutral-800 dark:bg-neutral-950">
            <span className="text-xs font-medium text-slate-500 dark:text-neutral-400">
              {value.length === 0
                ? 'None selected'
                : `${value.length} selected`}
            </span>
            {value.length > 0 && (
              <button
                type="button"
                onClick={clearAll}
                className="text-xs font-medium text-indigo-600 hover:text-indigo-500 dark:text-indigo-400 dark:hover:text-indigo-300"
              >
                Clear
              </button>
            )}
          </div>
          <ul className="py-1">
            {GRADE_LEVEL_OPTIONS.map((opt) => {
              const checked = selected.has(opt.value)
              return (
                <li key={opt.value} role="option" aria-selected={checked}>
                  <button
                    type="button"
                    onClick={() => toggle(opt.value)}
                    className={`flex w-full items-center gap-2.5 px-3 py-2 text-left text-sm transition-colors hover:bg-slate-50 dark:hover:bg-neutral-900 ${
                      checked
                        ? 'bg-indigo-50/70 text-slate-900 dark:bg-indigo-950/40 dark:text-neutral-50'
                        : 'text-slate-800 dark:text-neutral-100'
                    }`}
                  >
                    <span
                      className={`flex h-4 w-4 shrink-0 items-center justify-center rounded border ${
                        checked
                          ? 'border-indigo-600 bg-indigo-600 text-white dark:border-indigo-500 dark:bg-indigo-500'
                          : 'border-slate-300 bg-white dark:border-neutral-600 dark:bg-neutral-900'
                      }`}
                      aria-hidden
                    >
                      {checked ? <Check className="h-3 w-3" strokeWidth={3} /> : null}
                    </span>
                    {opt.label}
                  </button>
                </li>
              )
            })}
          </ul>
        </div>
      )}
    </div>
  )
}
