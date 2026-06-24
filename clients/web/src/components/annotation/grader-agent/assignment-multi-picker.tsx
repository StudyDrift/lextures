import { useEffect, useId, useMemo, useState } from 'react'
import type { CourseAssignmentOption } from './activity-node-data'

type AssignmentMultiPickerProps = {
  assignments: CourseAssignmentOption[]
  selectedIds: Set<string>
  disabled?: boolean
  loading?: boolean
  filterPlaceholder: string
  emptyLabel: string
  noMatchLabel: string
  onChange: (next: Set<string>) => void
}

export function AssignmentMultiPicker({
  assignments,
  selectedIds,
  disabled,
  loading,
  filterPlaceholder,
  emptyLabel,
  noMatchLabel,
  onChange,
}: AssignmentMultiPickerProps) {
  const filterId = useId()
  const [query, setQuery] = useState('')

  useEffect(() => {
    if (disabled) setQuery('')
  }, [disabled])

  const visibleEntries = useMemo(() => {
    const needle = query.trim().toLowerCase()
    return assignments.filter(
      (assignment) => needle === '' || assignment.title.toLowerCase().includes(needle),
    )
  }, [assignments, query])

  const toggle = (assignmentId: string) => {
    const next = new Set(selectedIds)
    if (next.has(assignmentId)) next.delete(assignmentId)
    else next.add(assignmentId)
    onChange(next)
  }

  if (loading) {
    return <p className="text-sm text-slate-500 dark:text-neutral-400">{emptyLabel}</p>
  }

  if (assignments.length === 0) {
    return <p className="text-sm text-slate-500 dark:text-neutral-400">{emptyLabel}</p>
  }

  return (
    <div className="space-y-2">
      <label htmlFor={filterId} className="sr-only">
        {filterPlaceholder}
      </label>
      <input
        id={filterId}
        type="search"
        value={query}
        disabled={disabled}
        placeholder={filterPlaceholder}
        autoComplete="off"
        onChange={(e) => setQuery(e.target.value)}
        className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 focus:border-indigo-500 focus:outline-none focus:ring-2 focus:ring-indigo-500/20 disabled:opacity-60 dark:border-neutral-600 dark:bg-neutral-950 dark:text-neutral-100"
      />
      <div className="max-h-56 overflow-y-auto rounded-xl border border-slate-200 dark:border-neutral-700">
        {visibleEntries.length === 0 ? (
          <p className="px-3 py-2 text-sm text-slate-500 dark:text-neutral-400">{noMatchLabel}</p>
        ) : (
          <ul className="divide-y divide-slate-100 dark:divide-neutral-800">
            {visibleEntries.map((assignment) => {
              const checked = selectedIds.has(assignment.id)
              return (
                <li key={assignment.id}>
                  <label className="flex cursor-pointer items-start gap-3 px-3 py-2.5 hover:bg-slate-50 dark:hover:bg-neutral-800/60">
                    <input
                      type="checkbox"
                      checked={checked}
                      disabled={disabled}
                      onChange={() => toggle(assignment.id)}
                      className="mt-0.5"
                    />
                    <span className="min-w-0 text-sm text-slate-800 dark:text-neutral-100">{assignment.title}</span>
                  </label>
                </li>
              )
            })}
          </ul>
        )}
      </div>
    </div>
  )
}
