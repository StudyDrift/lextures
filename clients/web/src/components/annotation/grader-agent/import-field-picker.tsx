import { ChevronDown, Loader2 } from 'lucide-react'
import { useEffect, useId, useMemo, useRef, useState } from 'react'

export type ImportFieldPickerOption = {
  id: string
  label: string
}

type ImportFieldPickerProps = {
  label: string
  value: string
  options: ImportFieldPickerOption[]
  disabled?: boolean
  loading?: boolean
  loadingLabel?: string
  emptyLabel?: string
  placeholder?: string
  searchable?: boolean
  searchPlaceholder?: string
  noMatchLabel?: string
  open: boolean
  onOpenChange: (open: boolean) => void
  onChange: (id: string) => void
}

export function ImportFieldPicker({
  label,
  value,
  options,
  disabled = false,
  loading = false,
  loadingLabel = 'Loading…',
  emptyLabel = 'No options',
  placeholder = 'Choose…',
  searchable = false,
  searchPlaceholder = 'Search…',
  noMatchLabel = 'No matches',
  open,
  onOpenChange,
  onChange,
}: ImportFieldPickerProps) {
  const [query, setQuery] = useState('')
  const [highlightedIndex, setHighlightedIndex] = useState(0)
  const rootRef = useRef<HTMLDivElement>(null)
  const filterRef = useRef<HTMLInputElement>(null)
  const listItemRefs = useRef<Map<number, HTMLButtonElement>>(new Map())
  const buttonId = useId()
  const menuId = useId()
  const filterId = useId()

  const current = options.find((option) => option.id === value) ?? null
  const currentLabel = current?.label ?? (loading ? loadingLabel : options.length === 0 ? emptyLabel : placeholder)

  const visibleOptions = useMemo(() => {
    if (!searchable) return options
    const needle = query.trim().toLowerCase()
    return options.filter((option) => needle === '' || option.label.toLowerCase().includes(needle))
  }, [options, query, searchable])

  useEffect(() => {
    if (!open) {
      setQuery('')
      setHighlightedIndex(0)
      return
    }
    const currentIndex = visibleOptions.findIndex((option) => option.id === value)
    setHighlightedIndex(currentIndex >= 0 ? currentIndex : 0)
    if (searchable) {
      const frame = window.requestAnimationFrame(() => {
        filterRef.current?.focus()
      })
      return () => window.cancelAnimationFrame(frame)
    }
    return undefined
  }, [open, searchable, value, visibleOptions])

  useEffect(() => {
    if (!open) return
    setHighlightedIndex(0)
  }, [open, query])

  useEffect(() => {
    if (!open || visibleOptions.length === 0) return
    const clamped = Math.min(highlightedIndex, visibleOptions.length - 1)
    if (clamped !== highlightedIndex) {
      setHighlightedIndex(clamped)
      return
    }
    listItemRefs.current.get(clamped)?.scrollIntoView({ block: 'nearest' })
  }, [highlightedIndex, open, visibleOptions.length])

  useEffect(() => {
    if (!open) return
    function onPointerDown(e: PointerEvent) {
      if (rootRef.current && !rootRef.current.contains(e.target as Node)) {
        onOpenChange(false)
      }
    }
    function onKeyDown(e: KeyboardEvent) {
      if (e.key === 'Escape') onOpenChange(false)
    }
    document.addEventListener('pointerdown', onPointerDown)
    document.addEventListener('keydown', onKeyDown)
    return () => {
      document.removeEventListener('pointerdown', onPointerDown)
      document.removeEventListener('keydown', onKeyDown)
    }
  }, [onOpenChange, open])

  const triggerDisabled = disabled || loading || options.length === 0

  return (
    <div ref={rootRef} className="relative min-w-0">
      <span id={buttonId} className="mb-1.5 block text-xs font-medium text-slate-600 dark:text-neutral-400">
        {label}
      </span>
      <button
        type="button"
        disabled={triggerDisabled}
        aria-haspopup="menu"
        aria-expanded={open}
        aria-controls={menuId}
        aria-labelledby={buttonId}
        onClick={() => onOpenChange(!open)}
        className="flex w-full min-w-0 items-center gap-2 rounded-xl border border-slate-300 bg-white px-3 py-2 text-start text-sm font-medium text-slate-800 shadow-sm transition-[background-color,color,border-color] hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-60 dark:border-neutral-600 dark:bg-neutral-900 dark:text-neutral-100 dark:hover:bg-neutral-800"
      >
        {loading ? (
          <>
            <Loader2 className="h-4 w-4 shrink-0 motion-safe:animate-spin text-slate-500 dark:text-neutral-400" aria-hidden />
            <span className="min-w-0 flex-1 truncate text-slate-500 dark:text-neutral-400">{loadingLabel}</span>
          </>
        ) : (
          <span className="min-w-0 flex-1 truncate">{currentLabel}</span>
        )}
        <ChevronDown
          className={`h-4 w-4 shrink-0 text-slate-500 transition-transform dark:text-neutral-400 ${
            open ? 'rotate-180' : ''
          }`}
          aria-hidden
        />
      </button>

      {open && !loading ? (
        <div
          id={menuId}
          role="menu"
          aria-labelledby={buttonId}
          className="absolute start-0 top-full z-[60] mt-1 flex max-h-60 w-full min-w-0 flex-col overflow-hidden rounded-xl border border-slate-200 bg-white shadow-lg shadow-slate-900/10 dark:border-neutral-600 dark:bg-neutral-900"
        >
          {searchable ? (
            <div className="shrink-0 border-b border-slate-200 p-2 dark:border-neutral-700">
              <label htmlFor={filterId} className="sr-only">
                {searchPlaceholder}
              </label>
              <input
                ref={filterRef}
                id={filterId}
                type="search"
                value={query}
                placeholder={searchPlaceholder}
                autoComplete="off"
                onChange={(e) => setQuery(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === 'Escape') {
                    e.stopPropagation()
                    onOpenChange(false)
                    return
                  }
                  if (visibleOptions.length === 0) return
                  if (e.key === 'ArrowDown') {
                    e.preventDefault()
                    setHighlightedIndex((prev) => Math.min(prev + 1, visibleOptions.length - 1))
                    return
                  }
                  if (e.key === 'ArrowUp') {
                    e.preventDefault()
                    setHighlightedIndex((prev) => Math.max(prev - 1, 0))
                    return
                  }
                  if (e.key === 'Enter') {
                    e.preventDefault()
                    const entry = visibleOptions[highlightedIndex]
                    if (entry) {
                      onChange(entry.id)
                      onOpenChange(false)
                    }
                  }
                }}
                className="w-full rounded-lg border border-slate-300 bg-white px-2.5 py-1.5 text-xs text-slate-900 outline-none placeholder:text-slate-400 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20 dark:border-neutral-600 dark:bg-neutral-950 dark:text-neutral-100 dark:placeholder:text-neutral-500 dark:focus:border-indigo-400"
              />
            </div>
          ) : null}

          <div className="min-h-0 flex-1 overflow-y-auto py-1">
            {options.length === 0 ? (
              <p className="px-3 py-2 text-sm text-slate-500 dark:text-neutral-400">{emptyLabel}</p>
            ) : visibleOptions.length === 0 ? (
              <p className="px-3 py-2 text-sm text-slate-500 dark:text-neutral-400">{noMatchLabel}</p>
            ) : (
              visibleOptions.map((option, visibleIndex) => {
                const active = option.id === value
                const highlighted = visibleIndex === highlightedIndex
                return (
                  <button
                    key={option.id}
                    ref={(el) => {
                      if (el) listItemRefs.current.set(visibleIndex, el)
                      else listItemRefs.current.delete(visibleIndex)
                    }}
                    type="button"
                    role="menuitemradio"
                    aria-checked={active}
                    onMouseEnter={() => setHighlightedIndex(visibleIndex)}
                    onClick={() => {
                      onChange(option.id)
                      onOpenChange(false)
                    }}
                    className={`flex w-full items-center gap-2 px-2.5 py-2 text-start text-sm transition-[background-color,color,border-color] ${
                      highlighted
                        ? 'bg-indigo-50 font-medium text-indigo-900 dark:bg-indigo-950/50 dark:text-indigo-100'
                        : active
                          ? 'font-semibold text-indigo-800 dark:text-indigo-200'
                          : 'font-medium text-slate-800 hover:bg-slate-50 dark:text-neutral-200 dark:hover:bg-neutral-800'
                    }`}
                  >
                    <span className="min-w-0 flex-1 truncate">{option.label}</span>
                  </button>
                )
              })
            )}
          </div>
        </div>
      ) : null}
    </div>
  )
}