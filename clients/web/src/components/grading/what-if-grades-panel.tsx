import { useId, useMemo } from 'react'
import { Calculator, FlaskConical, RotateCcw } from 'lucide-react'
import {
  computeScoreNeededForTarget,
  formatFinalPercent,
  type GradebookColumnForFinal,
  type AssignmentGroupWeight,
} from '../../pages/lms/gradebook/compute-course-final-percent'
import {
  letterTierOptions,
  percentToDisplayGrade,
  type GradingSchemeLike,
} from '../../lib/grading-display'

type WhatIfGradesPanelProps = {
  whatIfMode: boolean
  onToggleMode: () => void
  onReset: () => void
  hasOverrides: boolean
  projectedPercent: number | null
  actualPercent: number | null
  gradingScheme: GradingSchemeLike
  columns: GradebookColumnForFinal[]
  actualGrades: Record<string, string>
  assignmentGroups: AssignmentGroupWeight[]
  excusedByItemId: Record<string, boolean>
  heldItemIds: ReadonlySet<string>
  whatIfOverrides: Record<string, string>
  targetLetter: string
  onTargetLetterChange: (letter: string) => void
}

export function WhatIfGradesPanel({
  whatIfMode,
  onToggleMode,
  onReset,
  hasOverrides,
  projectedPercent,
  actualPercent,
  gradingScheme,
  columns,
  actualGrades,
  assignmentGroups,
  excusedByItemId,
  heldItemIds,
  whatIfOverrides,
  targetLetter,
  onTargetLetterChange,
}: WhatIfGradesPanelProps) {
  const targetPanelId = useId()
  const letterOptions = useMemo(() => letterTierOptions(gradingScheme), [gradingScheme])

  const targetMinPct = useMemo(() => {
    const t = letterOptions.find((o) => o.label === targetLetter)
    return t?.minPct ?? 80
  }, [letterOptions, targetLetter])

  const scoreNeeded = useMemo(
    () =>
      whatIfMode
        ? computeScoreNeededForTarget(
            targetMinPct,
            columns,
            actualGrades,
            assignmentGroups,
            excusedByItemId,
            heldItemIds,
            whatIfOverrides,
          )
        : null,
    [
      whatIfMode,
      targetMinPct,
      columns,
      actualGrades,
      assignmentGroups,
      excusedByItemId,
      heldItemIds,
      whatIfOverrides,
    ],
  )

  const showHypothetical = whatIfMode && (hasOverrides || projectedPercent !== actualPercent)
  const displayPercent = showHypothetical ? projectedPercent : actualPercent
  const projectedLetter = percentToDisplayGrade(displayPercent, gradingScheme)

  return (
    <div className="mt-6 space-y-4">
      <div className="flex flex-wrap items-center gap-3">
        <button
          type="button"
          aria-pressed={whatIfMode}
          className={`inline-flex items-center gap-2 rounded-lg border px-3 py-2 text-sm font-medium transition-[background-color,color,border-color] ${
            whatIfMode
              ? 'border-indigo-300 bg-indigo-50 text-indigo-900 dark:border-indigo-700 dark:bg-indigo-950/60 dark:text-indigo-100'
              : 'border-slate-200 bg-white text-slate-800 hover:bg-slate-50 dark:border-neutral-600 dark:bg-neutral-900 dark:text-neutral-100 dark:hover:bg-neutral-800'
          }`}
          onClick={onToggleMode}
        >
          <FlaskConical className="h-4 w-4" aria-hidden />
          {whatIfMode ? 'What-if mode on' : 'What-if grades'}
        </button>
        {whatIfMode && hasOverrides ? (
          <button
            type="button"
            className="inline-flex items-center gap-1.5 rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm font-medium text-slate-800 hover:bg-slate-50 dark:border-neutral-600 dark:bg-neutral-900 dark:text-neutral-100 dark:hover:bg-neutral-800"
            onClick={onReset}
          >
            <RotateCcw className="h-4 w-4" aria-hidden />
            Reset to actual
          </button>
        ) : null}
      </div>

      {whatIfMode ? (
        <div
          className="rounded-xl border border-indigo-200/80 bg-indigo-50/60 px-4 py-3 text-sm text-indigo-950 dark:border-indigo-800/60 dark:bg-indigo-950/30 dark:text-indigo-100"
          role="status"
        >
          <p className="font-medium">What-if grades are private to you.</p>
          <p className="mt-1 text-indigo-900/90 dark:text-indigo-200/90">
            Hypothetical scores are not saved and your instructor cannot see them. Edit scores below
            to explore scenarios.
          </p>
        </div>
      ) : null}

      <div className="rounded-xl border border-slate-200 bg-white px-4 py-4 shadow-sm dark:border-neutral-700 dark:bg-neutral-900">
        <div className="flex flex-wrap items-baseline gap-2">
          <p className="text-2xl font-semibold tracking-tight text-slate-900 dark:text-neutral-100">
            {showHypothetical ? 'Projected course grade' : 'Course grade'}:{' '}
            {formatFinalPercent(showHypothetical ? projectedPercent : actualPercent)}
          </p>
          {showHypothetical ? (
            <span
              className="inline-flex items-center gap-1 rounded-md border border-indigo-300 bg-indigo-100 px-2 py-0.5 text-xs font-semibold uppercase tracking-wide text-indigo-900 dark:border-indigo-700 dark:bg-indigo-950/70 dark:text-indigo-100"
              aria-label="Hypothetical projected grade"
            >
              <FlaskConical className="h-3 w-3" aria-hidden />
              Hypothetical
            </span>
          ) : null}
          {projectedLetter ? (
            <span className="text-lg font-medium text-slate-700 dark:text-neutral-300">
              ({projectedLetter})
            </span>
          ) : null}
        </div>
        {showHypothetical && actualPercent != null ? (
          <p className="mt-1 text-sm text-slate-600 dark:text-neutral-400">
            Actual course grade: {formatFinalPercent(actualPercent)}
          </p>
        ) : (
          <p className="mt-1 text-sm text-slate-600 dark:text-neutral-400">
            Weighted from assignment groups when your instructor has configured weights; otherwise
            by points earned vs points possible.
          </p>
        )}
      </div>

      {whatIfMode ? (
        <section
          aria-labelledby={targetPanelId}
          className="rounded-xl border border-slate-200 bg-white px-4 py-4 shadow-sm dark:border-neutral-700 dark:bg-neutral-900"
        >
          <div className="flex items-center gap-2">
            <Calculator className="h-4 w-4 text-slate-600 dark:text-neutral-400" aria-hidden />
            <h2 id={targetPanelId} className="text-sm font-semibold text-slate-900 dark:text-neutral-100">
              What score do I need?
            </h2>
          </div>
          <p className="mt-1 text-sm text-slate-600 dark:text-neutral-400">
            Estimate the equal score needed on each remaining ungraded item to reach your target.
          </p>
          <div className="mt-3 flex flex-wrap items-center gap-3">
            <label className="flex items-center gap-2 text-sm text-slate-800 dark:text-neutral-200">
              Target grade
              <select
                className="rounded-md border border-slate-200 bg-white px-2 py-1.5 text-sm dark:border-neutral-600 dark:bg-neutral-800"
                value={targetLetter}
                onChange={(e) => onTargetLetterChange(e.target.value)}
              >
                {letterOptions.map((o) => (
                  <option key={o.label} value={o.label}>
                    {o.label} ({o.minPct}%+)
                  </option>
                ))}
              </select>
            </label>
          </div>
          {scoreNeeded ? (
            <p className="mt-3 text-sm text-slate-800 dark:text-neutral-200" role="status">
              {scoreNeeded.achievable ? (
                <>
                  You need about{' '}
                  <strong>{formatFinalPercent(scoreNeeded.scorePercent)}</strong> on each of{' '}
                  {scoreNeeded.itemIds.length} remaining item
                  {scoreNeeded.itemIds.length === 1 ? '' : 's'} to reach {targetLetter}.
                </>
              ) : (
                scoreNeeded.reason
              )}
            </p>
          ) : null}
        </section>
      ) : null}
    </div>
  )
}
