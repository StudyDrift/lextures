import { useEffect, useId, useRef, useState } from 'react'
import { Check, ChevronDown, Layers } from 'lucide-react'
import type { CoursePublic } from '../../lib/courses-api'

type Props = {
  courses: CoursePublic[]
  disabledCourseIds: Set<string>
  structureErrors: Record<string, string>
  onCourseEnabledChange: (courseId: string, enabled: boolean) => void
  onShowAll: () => void
  onHideAll: () => void
}

export function CalendarCoursesViewMenu({
  courses,
  disabledCourseIds,
  structureErrors,
  onCourseEnabledChange,
  onShowAll,
  onHideAll,
}: Props) {
  const [open, setOpen] = useState(false)
  const rootRef = useRef<HTMLDivElement>(null)
  const menuId = useId()

  const enabledCount = courses.length - disabledCourseIds.size
  const allSelected = enabledCount === courses.length
  const noneSelected = enabledCount === 0

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

  const selectionLabel =
    courses.length === 0
      ? 'no courses'
      : allSelected
        ? 'all courses'
        : noneSelected
          ? 'no courses selected'
          : `${enabledCount} of ${courses.length} courses`

  return (
    <div ref={rootRef} className="relative block w-full text-start sm:inline-block sm:w-auto">
      <button
        type="button"
        aria-haspopup="menu"
        aria-expanded={open}
        aria-controls={open ? menuId : undefined}
        aria-label={`View courses on calendar. ${selectionLabel} selected. Open menu to change selection.`}
        onClick={() => setOpen((o) => !o)}
        className="inline-flex w-full items-center justify-center gap-2 rounded-xl border border-border-default bg-surface-raised px-4 py-2.5 text-sm font-semibold text-fg-default shadow-sm transition-[background-color,color,border-color] hover:border-border-strong hover:bg-surface-base dark:border-border-default dark:bg-surface-raised dark:text-fg-default dark:hover:border-border-default dark:hover:bg-surface-overlay sm:w-auto"
      >
        <Layers className="h-4 w-4 shrink-0" aria-hidden />
        <span>View</span>
        <ChevronDown
          className={`h-4 w-4 shrink-0 transition-transform ${open ? 'rotate-180' : ''}`}
          aria-hidden
        />
      </button>

      {open && (
        <div
          id={menuId}
          role="menu"
          aria-label="Calendar courses"
          className="absolute start-0 end-0 z-50 mt-1 min-w-0 overflow-hidden rounded-xl border border-border-default bg-surface-raised py-1 shadow-lg shadow-slate-900/10 sm:left-auto sm:end-0 sm:min-w-[18rem] dark:border-border-default dark:bg-surface-overlay dark:shadow-black/40"
        >
          <div className="border-b border-border-subtle px-2.5 py-2 dark:border-border-default">
            <p className="text-xs font-medium text-fg-muted">
              Show due dates from selected courses
            </p>
            <div className="mt-2 flex flex-wrap gap-2">
              <button
                type="button"
                disabled={allSelected}
                onClick={onShowAll}
                className="rounded-lg border border-border-default bg-surface-base px-2.5 py-1 text-xs font-semibold text-fg-muted transition-[background-color,color,border-color] hover:bg-surface-raised disabled:cursor-not-allowed disabled:opacity-50 dark:border-border-default dark:bg-surface-overlay dark:text-fg-default dark:hover:bg-neutral-700"
              >
                Show all
              </button>
              <button
                type="button"
                disabled={noneSelected}
                onClick={onHideAll}
                className="rounded-lg border border-border-default bg-surface-base px-2.5 py-1 text-xs font-semibold text-fg-muted transition-[background-color,color,border-color] hover:bg-surface-raised disabled:cursor-not-allowed disabled:opacity-50 dark:border-border-default dark:bg-surface-overlay dark:text-fg-default dark:hover:bg-neutral-700"
              >
                Hide all
              </button>
            </div>
          </div>
          <ul className="max-h-64 overflow-y-auto overscroll-contain py-1">
            {courses.map((c) => {
              const enabled = !disabledCourseIds.has(c.id)
              const label = c.title.trim() || c.courseCode
              const err = structureErrors[c.id]
              return (
                <li key={c.id}>
                  <button
                    type="button"
                    role="menuitemcheckbox"
                    aria-checked={enabled}
                    onClick={() => onCourseEnabledChange(c.id, !enabled)}
                    className="flex w-full items-start gap-2.5 px-2.5 py-2 text-start text-sm transition-[background-color,color,border-color] hover:bg-surface-base dark:hover:bg-neutral-700"
                  >
                    <span
                      className={`mt-0.5 flex h-4 w-4 shrink-0 items-center justify-center rounded border ${ enabled ? 'border-indigo-600 bg-accent-solid text-white dark:border-indigo-500 dark:bg-indigo-500' : 'border-border-strong bg-surface-raised dark:border-neutral-500 dark:bg-surface-overlay' }`}
                      aria-hidden
                    >
                      {enabled ? <Check className="h-3 w-3" strokeWidth={3} /> : null}
                    </span>
                    <span className="flex min-w-0 flex-1 flex-col gap-0.5">
                      <span className="font-semibold text-slate-950 dark:text-fg-default">{label}</span>
                      <span className="text-xs text-fg-muted">{c.courseCode}</span>
                      {enabled && err ? (
                        <span className="text-xs text-rose-600 dark:text-rose-400">{err}</span>
                      ) : null}
                    </span>
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