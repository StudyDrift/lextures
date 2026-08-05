import { useEffect, useId, useRef } from 'react'
import type { RubricCriterion, RubricDefinition } from '../../lib/courses-api'
import { formatPointsCell } from '../../lib/rubric-utils'

export type AssignmentRubricViewerDialogProps = {
  open: boolean
  rubric: RubricDefinition | null
  onClose: () => void
}

function criterionMaxPoints(criterion: RubricCriterion): number {
  if (criterion.levels.length === 0) return 0
  return Math.max(...criterion.levels.map((l) => l.points))
}

function rubricMaxPoints(rubric: RubricDefinition): number {
  return rubric.criteria.reduce((sum, c) => sum + criterionMaxPoints(c), 0)
}

/**
 * Read-only rubric viewer for learners (and anyone viewing assignment details).
 * Shows criteria, descriptions, and rating bands with points.
 */
export function AssignmentRubricViewerDialog({
  open,
  rubric,
  onClose,
}: AssignmentRubricViewerDialogProps) {
  const titleId = useId()
  const closeRef = useRef<HTMLButtonElement>(null)

  useEffect(() => {
    if (!open) return
    const t = window.setTimeout(() => closeRef.current?.focus(), 0)
    return () => window.clearTimeout(t)
  }, [open])

  useEffect(() => {
    if (!open) return
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape') {
        e.preventDefault()
        onClose()
      }
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [open, onClose])

  useEffect(() => {
    if (!open) return
    const prev = document.body.style.overflow
    document.body.style.overflow = 'hidden'
    return () => {
      document.body.style.overflow = prev
    }
  }, [open])

  if (!open || !rubric || rubric.criteria.length === 0) return null

  const heading = rubric.title?.trim() || 'Rubric'
  const totalMax = rubricMaxPoints(rubric)

  return (
    <div
      className="fixed inset-0 z-[300] flex items-end justify-center bg-black/40 p-4 sm:items-center"
      role="dialog"
      aria-modal="true"
      aria-labelledby={titleId}
      onClick={(e) => {
        if (e.target === e.currentTarget) onClose()
      }}
    >
      <div className="flex max-h-[90vh] w-full max-w-lg flex-col overflow-hidden rounded-xl border border-slate-200 bg-white shadow-xl dark:border-neutral-600 dark:bg-neutral-900">
        <div className="flex shrink-0 items-start justify-between gap-3 border-b border-slate-100 px-5 py-4 dark:border-neutral-700">
          <div className="min-w-0">
            <h2
              id={titleId}
              className="text-lg font-semibold text-slate-950 dark:text-neutral-100"
            >
              {heading}
            </h2>
            <p className="mt-0.5 text-sm text-slate-500 dark:text-neutral-400">
              {rubric.criteria.length}{' '}
              {rubric.criteria.length === 1 ? 'criterion' : 'criteria'}
              {totalMax > 0 ? (
                <>
                  {' · '}
                  {formatPointsCell(totalMax)} pts total
                </>
              ) : null}
            </p>
          </div>
          <button
            ref={closeRef}
            type="button"
            onClick={onClose}
            className="shrink-0 rounded-lg border border-slate-200 px-3 py-1.5 text-sm font-medium text-slate-800 hover:bg-slate-50 dark:border-neutral-600 dark:text-neutral-100 dark:hover:bg-neutral-800"
          >
            Close
          </button>
        </div>

        <div className="min-h-0 flex-1 space-y-4 overflow-y-auto px-5 py-4">
          {rubric.criteria.map((criterion, index) => {
            const maxPts = criterionMaxPoints(criterion)
            return (
              <div
                key={criterion.id}
                className="rounded-lg border border-slate-200 dark:border-neutral-700"
              >
                <div className="border-b border-slate-100 px-3.5 py-3 dark:border-neutral-700">
                  <div className="flex items-start justify-between gap-3">
                    <p className="text-sm font-medium leading-snug text-slate-900 dark:text-neutral-100">
                      <span className="me-1.5 font-semibold tabular-nums text-slate-400 dark:text-neutral-500">
                        {index + 1}.
                      </span>
                      {criterion.title}
                    </p>
                    {maxPts > 0 ? (
                      <span className="shrink-0 text-xs font-medium tabular-nums text-slate-500 dark:text-neutral-400">
                        {formatPointsCell(maxPts)} pts
                      </span>
                    ) : null}
                  </div>
                  {criterion.description?.trim() ? (
                    <p className="mt-1.5 text-xs leading-relaxed text-slate-500 dark:text-neutral-400">
                      {criterion.description}
                    </p>
                  ) : null}
                </div>
                {criterion.levels.length > 0 ? (
                  <ul className="divide-y divide-slate-100 dark:divide-neutral-700">
                    {criterion.levels.map((level, li) => (
                      <li
                        key={`${criterion.id}-${li}`}
                        className="flex items-start justify-between gap-3 px-3.5 py-2.5"
                      >
                        <div className="min-w-0">
                          <p className="text-sm font-medium text-slate-800 dark:text-neutral-200">
                            {level.label}
                          </p>
                          {level.description?.trim() ? (
                            <p className="mt-0.5 text-xs leading-relaxed text-slate-500 dark:text-neutral-400">
                              {level.description}
                            </p>
                          ) : null}
                        </div>
                        <span className="shrink-0 text-xs font-semibold tabular-nums text-slate-600 dark:text-neutral-300">
                          {formatPointsCell(level.points)} pts
                        </span>
                      </li>
                    ))}
                  </ul>
                ) : null}
              </div>
            )
          })}
        </div>
      </div>
    </div>
  )
}
