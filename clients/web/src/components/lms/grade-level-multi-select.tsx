import { useEffect, useId, useLayoutEffect, useRef, useState } from 'react'
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

type MenuPlacement = 'below' | 'above'

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
  const triggerRef = useRef<HTMLButtonElement>(null)
  const [open, setOpen] = useState(false)
  const [placement, setPlacement] = useState<MenuPlacement>('below')
  const selected = new Set(value)

  useLayoutEffect(() => {
    if (!open || !triggerRef.current) return
    const rect = triggerRef.current.getBoundingClientRect()
    const menuMaxHeight = 288 // max-h-72
    const gap = 6
    const spaceBelow = window.innerHeight - rect.bottom - gap
    const spaceAbove = rect.top - gap
    // Prefer below; flip above when the menu would cover actions under the field
    // (e.g. Continue on course create) and there is more room upward.
    if (spaceBelow < Math.min(menuMaxHeight, 200) && spaceAbove > spaceBelow) {
      setPlacement('above')
    } else {
      setPlacement('below')
    }
  }, [open])

  useEffect(() => {
    if (!open) return
    function onDocPointerDown(e: PointerEvent) {
      if (!rootRef.current?.contains(e.target as Node)) {
        setOpen(false)
      }
    }
    function onKeyDown(e: KeyboardEvent) {
      if (e.key === 'Escape') {
        e.preventDefault()
        setOpen(false)
        triggerRef.current?.focus()
      }
    }
    // Capture so Escape/outside-close still win if a child stops propagation.
    document.addEventListener('pointerdown', onDocPointerDown, true)
    document.addEventListener('keydown', onKeyDown, true)
    return () => {
      document.removeEventListener('pointerdown', onDocPointerDown, true)
      document.removeEventListener('keydown', onKeyDown, true)
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
  const menuPositionClass =
    placement === 'above' ? 'bottom-full mb-1.5' : 'top-full mt-1.5'

  return (
    <div ref={rootRef} className={`relative ${className}`}>
      <button
        ref={triggerRef}
        type="button"
        id={fieldId}
        disabled={disabled}
        aria-haspopup="listbox"
        aria-expanded={open}
        aria-controls={listboxId}
        aria-label={ariaLabel}
        onClick={() => setOpen((o) => !o)}
        className="flex w-full items-center justify-between gap-2 rounded-xl border border-border-default bg-surface-raised px-3 py-2.5 text-left text-sm text-fg-default shadow-sm outline-none ring-indigo-500/0 transition-[background-color,color,border-color,box-shadow] focus:border-indigo-400 focus:ring-2 focus:ring-indigo-500/20 disabled:cursor-not-allowed disabled:opacity-60 dark:border-border-default dark:bg-surface-base dark:focus:border-indigo-500/60"
      >
        <span
          className={
            value.length === 0
              ? 'truncate text-fg-muted'
              : 'truncate'
          }
        >
          {summary}
        </span>
        <ChevronDown
          className={`h-4 w-4 shrink-0 text-fg-subtle transition-transform ${open ? 'rotate-180' : ''}`}
          aria-hidden
        />
      </button>

      {open && (
        <div
          id={listboxId}
          role="listbox"
          aria-multiselectable="true"
          aria-label={ariaLabel}
          className={`absolute z-30 max-h-72 w-full overflow-auto rounded-xl border border-border-default bg-surface-raised py-1 shadow-lg shadow-slate-900/10 dark:border-border-default dark:bg-surface-base dark:shadow-black/40 ${menuPositionClass}`}
        >
          <div className="sticky top-0 z-10 flex items-center justify-between gap-2 border-b border-border-subtle bg-surface-raised px-3 py-2 dark:border-border-subtle dark:bg-surface-base">
            <span className="text-xs font-medium text-fg-muted">
              {value.length === 0
                ? 'None selected'
                : `${value.length} selected`}
            </span>
            <div className="flex items-center gap-2">
              {value.length > 0 && (
                <button
                  type="button"
                  onClick={clearAll}
                  className="text-xs font-medium text-accent-fg hover:text-indigo-500 dark:text-indigo-400 dark:hover:text-indigo-300"
                >
                  Clear
                </button>
              )}
              <button
                type="button"
                onClick={() => {
                  setOpen(false)
                  triggerRef.current?.focus()
                }}
                className="text-xs font-semibold text-fg-default hover:text-indigo-600 dark:hover:text-indigo-300"
              >
                Done
              </button>
            </div>
          </div>
          <ul className="py-1">
            {GRADE_LEVEL_OPTIONS.map((opt) => {
              const checked = selected.has(opt.value)
              return (
                <li key={opt.value} role="option" aria-selected={checked}>
                  <button
                    type="button"
                    onClick={() => toggle(opt.value)}
                    className={`flex w-full items-center gap-2.5 px-3 py-2 text-left text-sm transition-colors hover:bg-surface-base dark:hover:bg-surface-raised ${ checked ? 'bg-indigo-50/70 text-fg-default dark:bg-indigo-950/40' : 'text-fg-default' }`}
                  >
                    <span
                      className={`flex h-4 w-4 shrink-0 items-center justify-center rounded border ${ checked ? 'border-indigo-600 bg-accent-solid text-white dark:border-indigo-500 dark:bg-indigo-500' : 'border-border-strong bg-surface-raised dark:border-border-default dark:bg-surface-raised' }`}
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
