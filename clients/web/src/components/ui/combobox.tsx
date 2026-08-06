import {
  useEffect,
  useId,
  useMemo,
  useRef,
  useState,
  type KeyboardEvent,
} from 'react'
import { cx, focusRingClass, sizeClasses, type ControlSize } from './utils'

export type ComboboxOption = {
  value: string
  label: string
  disabled?: boolean
}

export type ComboboxProps = {
  options: ComboboxOption[]
  value?: string
  onChange?: (value: string) => void
  placeholder?: string
  disabled?: boolean
  invalid?: boolean
  size?: ControlSize
  className?: string
  id?: string
  emptyLabel?: string
  /** Accessible name when no external label is wired. */
  'aria-label'?: string
  'aria-labelledby'?: string
}

/**
 * Accessible combobox with typeahead filter (WAI-ARIA APG combobox pattern, simplified).
 */
export function Combobox({
  options,
  value = '',
  onChange,
  placeholder,
  disabled,
  invalid,
  size = 'md',
  className = '',
  id,
  emptyLabel = 'No results',
  'aria-label': ariaLabel,
  'aria-labelledby': ariaLabelledby,
}: ComboboxProps) {
  const autoId = useId()
  const listboxId = `${autoId}-listbox`
  const inputId = id ?? autoId
  const [open, setOpen] = useState(false)
  const [query, setQuery] = useState('')
  const [activeIndex, setActiveIndex] = useState(0)
  const rootRef = useRef<HTMLDivElement>(null)

  const selected = options.find((o) => o.value === value)
  const display = open ? query : (selected?.label ?? '')

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase()
    if (!q) return options
    return options.filter((o) => o.label.toLowerCase().includes(q))
  }, [options, query])

  useEffect(() => {
    if (!open) return
    function onDoc(e: MouseEvent) {
      if (!rootRef.current?.contains(e.target as Node)) setOpen(false)
    }
    document.addEventListener('mousedown', onDoc)
    return () => document.removeEventListener('mousedown', onDoc)
  }, [open])

  function select(opt: ComboboxOption) {
    if (opt.disabled) return
    onChange?.(opt.value)
    setQuery(opt.label)
    setOpen(false)
  }

  function onKeyDown(e: KeyboardEvent<HTMLInputElement>) {
    if (e.key === 'ArrowDown') {
      e.preventDefault()
      setOpen(true)
      setActiveIndex((i) => Math.min(i + 1, Math.max(0, filtered.length - 1)))
    } else if (e.key === 'ArrowUp') {
      e.preventDefault()
      setActiveIndex((i) => Math.max(i - 1, 0))
    } else if (e.key === 'Home' && open) {
      e.preventDefault()
      setActiveIndex(0)
    } else if (e.key === 'End' && open) {
      e.preventDefault()
      setActiveIndex(Math.max(0, filtered.length - 1))
    } else if (e.key === 'Enter' && open) {
      e.preventDefault()
      const opt = filtered[activeIndex]
      if (opt) select(opt)
    } else if (e.key === 'Escape') {
      e.preventDefault()
      setOpen(false)
    }
  }

  return (
    <div ref={rootRef} className={cx('relative', className)}>
      <input
        id={inputId}
        role="combobox"
        aria-expanded={open}
        aria-controls={listboxId}
        aria-autocomplete="list"
        aria-activedescendant={open && filtered[activeIndex] ? `${listboxId}-${activeIndex}` : undefined}
        aria-invalid={invalid || undefined}
        aria-label={ariaLabel}
        aria-labelledby={ariaLabelledby}
        disabled={disabled}
        placeholder={placeholder}
        value={display}
        autoComplete="off"
        className={cx(
          'w-full rounded-xl border bg-surface-raised text-fg-default placeholder:text-fg-subtle shadow-sm disabled:opacity-50',
          focusRingClass,
          sizeClasses[size],
          invalid ? 'border-danger-fg' : 'border-border-default',
        )}
        onChange={(e) => {
          setQuery(e.target.value)
          setOpen(true)
          setActiveIndex(0)
        }}
        onFocus={() => {
          setQuery(selected?.label ?? '')
          setOpen(true)
        }}
        onKeyDown={onKeyDown}
      />
      {open ? (
        <ul
          id={listboxId}
          role="listbox"
          className="absolute z-50 mt-1 max-h-60 w-full overflow-auto rounded-xl border border-border-default bg-surface-raised py-1 shadow-lg"
        >
          {filtered.length === 0 ? (
            <li className="px-3 py-2 text-sm text-fg-muted" role="presentation">
              {emptyLabel}
            </li>
          ) : (
            filtered.map((opt, i) => (
              <li
                key={opt.value}
                id={`${listboxId}-${i}`}
                role="option"
                aria-selected={opt.value === value}
                aria-disabled={opt.disabled || undefined}
                className={cx(
                  'cursor-pointer px-3 py-2 text-sm text-fg-default',
                  i === activeIndex && 'bg-accent-surface text-accent-fg',
                  opt.disabled && 'cursor-not-allowed opacity-50',
                )}
                onMouseEnter={() => setActiveIndex(i)}
                onMouseDown={(e) => {
                  e.preventDefault()
                  select(opt)
                }}
              >
                {opt.label}
              </li>
            ))
          )}
        </ul>
      ) : null}
    </div>
  )
}
