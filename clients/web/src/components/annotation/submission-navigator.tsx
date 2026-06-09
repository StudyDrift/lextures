import { ChevronDown } from 'lucide-react'
import { useEffect, useId, useMemo, useRef, useState } from 'react'
import type { ModuleAssignmentSubmissionApi } from '../../lib/courses-api'
import {
  submissionStudentLabel,
  type GradedFilter,
} from './submission-navigator-utils'

type SubmissionStudentPickerProps = {
  submissions: ModuleAssignmentSubmissionApi[]
  index: number
  disabled?: boolean
  onIndexChange: (i: number) => void
}

function SubmissionStudentPicker({
  submissions,
  index,
  disabled,
  onIndexChange,
}: SubmissionStudentPickerProps) {
  const [open, setOpen] = useState(false)
  const [query, setQuery] = useState('')
  const rootRef = useRef<HTMLDivElement>(null)
  const filterRef = useRef<HTMLInputElement>(null)
  const buttonId = useId()
  const menuId = useId()
  const filterId = useId()
  const current = submissions[index] ?? null
  const currentLabel = submissionStudentLabel(current, index) ?? 'No submissions'

  const visibleEntries = useMemo(() => {
    const needle = query.trim().toLowerCase()
    return submissions
      .map((submission, i) => ({
        submission,
        i,
        label: submissionStudentLabel(submission, i) ?? `Submission ${i + 1}`,
      }))
      .filter((entry) => needle === '' || entry.label.toLowerCase().includes(needle))
  }, [query, submissions])

  useEffect(() => {
    if (!open) {
      setQuery('')
      return
    }
    const frame = window.requestAnimationFrame(() => {
      filterRef.current?.focus()
    })
    return () => window.cancelAnimationFrame(frame)
  }, [open])

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
    <div ref={rootRef} className="relative min-w-0 flex-1">
      <button
        id={buttonId}
        type="button"
        disabled={disabled || submissions.length === 0}
        aria-haspopup="menu"
        aria-expanded={open}
        aria-controls={menuId}
        onClick={() => setOpen((prev) => !prev)}
        className="flex w-full min-w-0 items-center gap-1.5 rounded-lg border border-slate-300 bg-white px-2.5 py-1.5 text-start text-xs font-semibold text-slate-800 hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-50 dark:border-neutral-600 dark:bg-neutral-950 dark:text-neutral-100 dark:hover:bg-neutral-900"
      >
        <span className="min-w-0 flex-1 truncate">{currentLabel}</span>
        <ChevronDown
          className={`h-3.5 w-3.5 shrink-0 text-slate-500 transition-transform dark:text-neutral-400 ${
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
              Filter students
            </label>
            <input
              ref={filterRef}
              id={filterId}
              type="search"
              value={query}
              placeholder="Filter students…"
              autoComplete="off"
              onChange={(e) => setQuery(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Escape') {
                  e.stopPropagation()
                  setOpen(false)
                }
              }}
              className="w-full rounded-lg border border-slate-300 bg-white px-2.5 py-1.5 text-xs text-slate-900 outline-none ring-indigo-500/0 placeholder:text-slate-400 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20 dark:border-neutral-600 dark:bg-neutral-950 dark:text-neutral-100 dark:placeholder:text-neutral-500 dark:focus:border-indigo-400"
            />
          </div>

          <div className="min-h-0 flex-1 overflow-y-auto py-1">
            {submissions.length === 0 ? (
              <p className="px-3 py-2 text-xs text-slate-500 dark:text-neutral-400">No submissions.</p>
            ) : visibleEntries.length === 0 ? (
              <p className="px-3 py-2 text-xs text-slate-500 dark:text-neutral-400">No matching students.</p>
            ) : (
              visibleEntries.map(({ submission, i, label }) => {
                const active = i === index
                return (
                  <button
                    key={submission.id}
                    type="button"
                    role="menuitemradio"
                    aria-checked={active}
                    onClick={() => {
                      onIndexChange(i)
                      setOpen(false)
                    }}
                    className={`flex w-full items-center gap-2 px-3 py-2 text-start text-xs transition ${
                      active
                        ? 'bg-indigo-50 font-semibold text-indigo-900 dark:bg-indigo-950/50 dark:text-indigo-100'
                        : 'text-slate-700 hover:bg-slate-50 dark:text-neutral-200 dark:hover:bg-neutral-800'
                    }`}
                  >
                    <span className="w-5 shrink-0 tabular-nums text-slate-400 dark:text-neutral-500">
                      {i + 1}.
                    </span>
                    <span className="min-w-0 flex-1 truncate">{label}</span>
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

export type SubmissionNavigatorProps = {
  submissions: ModuleAssignmentSubmissionApi[]
  index: number
  onIndexChange: (i: number) => void
  gradedFilter: GradedFilter
  onGradedFilterChange: (f: GradedFilter) => void
  disabled?: boolean
  /** When blind grading is active, exposes WCAG label for the current submission. */
  anonymisedAriaLabel?: string
}

export function SubmissionNavigator({
  submissions,
  index,
  onIndexChange,
  gradedFilter,
  onGradedFilterChange,
  disabled,
  anonymisedAriaLabel,
}: SubmissionNavigatorProps) {
  const prev = () => onIndexChange(Math.max(0, index - 1))
  const next = () => onIndexChange(Math.min(Math.max(submissions.length - 1, 0), index + 1))

  return (
    <div
      className="flex min-w-0 flex-wrap items-center gap-3 rounded-xl border border-slate-200 bg-slate-50/80 px-3 py-2 text-sm dark:border-neutral-700 dark:bg-neutral-900/80"
      aria-label={anonymisedAriaLabel}
    >
      <label className="inline-flex shrink-0 items-center gap-2 font-medium text-slate-700 dark:text-neutral-200">
        <span className="sr-only">Filter submissions</span>
        <select
          className="rounded-lg border border-slate-300 bg-white px-2 py-1 text-sm dark:border-neutral-600 dark:bg-neutral-950"
          value={gradedFilter}
          disabled={disabled}
          onChange={(e) => onGradedFilterChange(e.target.value as GradedFilter)}
        >
          <option value="all">All</option>
          <option value="graded">Graded</option>
          <option value="ungraded">Ungraded</option>
        </select>
      </label>

      <div className="grid min-w-[18rem] flex-1 grid-cols-[4.5rem_minmax(0,1fr)_4.5rem] items-center gap-2">
        <button
          type="button"
          className="w-full rounded-lg border border-slate-300 bg-white px-3 py-1.5 text-xs font-semibold hover:bg-slate-50 disabled:opacity-50 dark:border-neutral-600 dark:bg-neutral-950 dark:hover:bg-neutral-900"
          disabled={disabled || index <= 0}
          onClick={prev}
          aria-label="Previous submission"
        >
          Prev
        </button>

        <div className="flex min-w-0 items-center gap-2">
          <SubmissionStudentPicker
            submissions={submissions}
            index={index}
            disabled={disabled}
            onIndexChange={onIndexChange}
          />
          <span className="w-10 shrink-0 text-end text-xs tabular-nums text-slate-500 dark:text-neutral-400">
            {submissions.length === 0 ? '0/0' : `${index + 1}/${submissions.length}`}
          </span>
        </div>

        <button
          type="button"
          className="w-full rounded-lg border border-slate-300 bg-white px-3 py-1.5 text-xs font-semibold hover:bg-slate-50 disabled:opacity-50 dark:border-neutral-600 dark:bg-neutral-950 dark:hover:bg-neutral-900"
          disabled={disabled || submissions.length === 0 || index >= submissions.length - 1}
          onClick={next}
          aria-label="Next submission"
        >
          Next
        </button>
      </div>
    </div>
  )
}
