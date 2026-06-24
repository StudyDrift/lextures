import { ChevronDown } from 'lucide-react'
import { useEffect, useId, useMemo, useRef, useState } from 'react'
import type { CourseAssignmentOption } from './activity-node-data'

type AssignmentPickerProps = {
  assignments: CourseAssignmentOption[]
  value: string
  disabled?: boolean
  loading?: boolean
  filterPlaceholder: string
  emptyLabel: string
  noMatchLabel: string
  onChange: (assignmentId: string) => void
}

export function AssignmentPicker({
  assignments,
  value,
  disabled,
  loading,
  filterPlaceholder,
  emptyLabel,
  noMatchLabel,
  onChange,
}: AssignmentPickerProps) {
  const [open, setOpen] = useState(false)
  const [query, setQuery] = useState('')
  const [highlightedIndex, setHighlightedIndex] = useState(0)
  const rootRef = useRef<HTMLDivElement>(null)
  const filterRef = useRef<HTMLInputElement>(null)
  const listItemRefs = useRef<Map<number, HTMLButtonElement>>(new Map())
  const openedRef = useRef(false)
  const buttonId = useId()
  const menuId = useId()
  const filterId = useId()

  const current = assignments.find((assignment) => assignment.id === value) ?? null
  const currentLabel = current?.title ?? (loading ? '…' : emptyLabel)

  const visibleEntries = useMemo(() => {
    const needle = query.trim().toLowerCase()
    return assignments.filter(
      (assignment) => needle === '' || assignment.title.toLowerCase().includes(needle),
    )
  }, [assignments, query])

  useEffect(() => {
    if (!open) {
      openedRef.current = false
      setQuery('')
      setHighlightedIndex(0)
      return
    }
    if (openedRef.current) return
    openedRef.current = true
    const currentEntryIndex = visibleEntries.findIndex((assignment) => assignment.id === value)
    setHighlightedIndex(currentEntryIndex >= 0 ? currentEntryIndex : 0)
    const frame = window.requestAnimationFrame(() => {
      filterRef.current?.focus()
    })
    return () => window.cancelAnimationFrame(frame)
  }, [open, value, visibleEntries])

  useEffect(() => {
    if (!open) return
    setHighlightedIndex(0)
  }, [query, open])

  useEffect(() => {
    if (!open || visibleEntries.length === 0) return
    const clamped = Math.min(highlightedIndex, visibleEntries.length - 1)
    if (clamped !== highlightedIndex) {
      setHighlightedIndex(clamped)
      return
    }
    listItemRefs.current.get(clamped)?.scrollIntoView({ block: 'nearest' })
  }, [highlightedIndex, open, visibleEntries.length])

  useEffect(() => {
    if (!open) return
    function onPointerDown(e: PointerEvent) {
      if (rootRef.current && !rootRef.current.contains(e.target as Node)) {
        setOpen(false)
      }
    }
    function onKeyDown(e: KeyboardEvent) {
      if (e.key === 'Escape') setOpen(false)
    }
    document.addEventListener('pointerdown', onPointerDown)
    document.addEventListener('keydown', onKeyDown)
    return () => {
      document.removeEventListener('pointerdown', onPointerDown)
      document.removeEventListener('keydown', onKeyDown)
    }
  }, [open])

  return (
    <div ref={rootRef} className="relative min-w-0">
      <button
        id={buttonId}
        type="button"
        disabled={disabled || loading || assignments.length === 0}
        aria-haspopup="menu"
        aria-expanded={open}
        aria-controls={menuId}
        onClick={() => setOpen((prev) => !prev)}
        className="flex w-full min-w-0 items-center gap-1.5 rounded-lg border border-slate-300 bg-white px-2.5 py-2 text-start text-sm font-medium text-slate-800 hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-50 dark:border-neutral-600 dark:bg-neutral-950 dark:text-neutral-100 dark:hover:bg-neutral-900"
      >
        <span className="min-w-0 flex-1 truncate">{currentLabel}</span>
        <ChevronDown
          className={`h-4 w-4 shrink-0 text-slate-500 transition-transform dark:text-neutral-400 ${
            open ? 'rotate-180' : ''
          }`}
          aria-hidden="true"
        />
      </button>

      {open ? (
        <div
          id={menuId}
          role="menu"
          aria-labelledby={buttonId}
          className="absolute start-0 top-full z-50 mt-1 flex max-h-72 w-full min-w-[14rem] flex-col overflow-hidden rounded-xl border border-slate-200 bg-white shadow-lg dark:border-neutral-600 dark:bg-neutral-900"
        >
          <div className="shrink-0 border-b border-slate-200 p-2 dark:border-neutral-700">
            <label htmlFor={filterId} className="sr-only">
              {filterPlaceholder}
            </label>
            <input
              ref={filterRef}
              id={filterId}
              type="search"
              value={query}
              placeholder={filterPlaceholder}
              autoComplete="off"
              onChange={(e) => setQuery(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Escape') {
                  e.stopPropagation()
                  setOpen(false)
                  return
                }
                if (visibleEntries.length === 0) return

                if (e.key === 'ArrowDown') {
                  e.preventDefault()
                  setHighlightedIndex((prev) => Math.min(prev + 1, visibleEntries.length - 1))
                  return
                }
                if (e.key === 'ArrowUp') {
                  e.preventDefault()
                  setHighlightedIndex((prev) => Math.max(prev - 1, 0))
                  return
                }
                if (e.key === 'Enter') {
                  e.preventDefault()
                  const entry = visibleEntries[highlightedIndex]
                  if (entry) {
                    onChange(entry.id)
                    setOpen(false)
                  }
                }
              }}
              className="w-full rounded-lg border border-slate-300 bg-white px-2.5 py-1.5 text-xs text-slate-900 outline-none ring-indigo-500/0 placeholder:text-slate-400 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20 dark:border-neutral-600 dark:bg-neutral-950 dark:text-neutral-100 dark:placeholder:text-neutral-500 dark:focus:border-indigo-400"
            />
          </div>

          <div className="min-h-0 flex-1 overflow-y-auto py-1">
            {assignments.length === 0 ? (
              <p className="px-3 py-2 text-xs text-slate-500 dark:text-neutral-400">{emptyLabel}</p>
            ) : visibleEntries.length === 0 ? (
              <p className="px-3 py-2 text-xs text-slate-500 dark:text-neutral-400">{noMatchLabel}</p>
            ) : (
              visibleEntries.map((assignment, visibleIndex) => {
                const active = assignment.id === value
                const highlighted = visibleIndex === highlightedIndex
                return (
                  <button
                    key={assignment.id}
                    ref={(el) => {
                      if (el) listItemRefs.current.set(visibleIndex, el)
                      else listItemRefs.current.delete(visibleIndex)
                    }}
                    type="button"
                    role="menuitemradio"
                    aria-checked={active}
                    onMouseEnter={() => setHighlightedIndex(visibleIndex)}
                    onClick={() => {
                      onChange(assignment.id)
                      setOpen(false)
                    }}
                    className={`flex w-full items-center gap-2 px-3 py-2 text-start text-xs transition-[background-color,color,border-color] ${
                      highlighted
                        ? 'bg-indigo-50 font-semibold text-indigo-900 dark:bg-indigo-950/50 dark:text-indigo-100'
                        : active
                          ? 'font-semibold text-indigo-800 dark:text-indigo-200'
                          : 'text-slate-700 hover:bg-slate-50 dark:text-neutral-200 dark:hover:bg-neutral-800'
                    }`}
                  >
                    <span className="min-w-0 flex-1 truncate">{assignment.title}</span>
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