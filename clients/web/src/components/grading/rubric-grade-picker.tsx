import type { RubricDefinition } from '../../lib/courses-api'
import { formatPointsCell, rubricGradedCount, rubricTotal, rubricScoresComplete } from '../../lib/rubric-utils'

type RubricGradePickerProps = {
  rubric: RubricDefinition
  scores: Record<string, number>
  onScoresChange: (scores: Record<string, number>) => void
  disabled?: boolean
  compact?: boolean
}

export function RubricGradePicker({
  rubric,
  scores,
  onScoresChange,
  disabled = false,
  compact = false,
}: RubricGradePickerProps) {
  const total = rubricTotal(rubric, scores)
  const gradedCount = rubricGradedCount(rubric, scores)
  const allGraded = rubricScoresComplete(rubric, scores)

  return (
    <div className="space-y-4">
      <div className="rounded-xl border border-slate-200 bg-white p-3 dark:border-neutral-600 dark:bg-neutral-900/50">
        <div className="flex items-center justify-between gap-3">
          <div>
            <p className="text-xs font-medium uppercase tracking-wide text-slate-500 dark:text-neutral-400">
              Rubric score
            </p>
            <p className="mt-0.5 text-2xl font-semibold tabular-nums text-slate-900 dark:text-neutral-50">
              {formatPointsCell(total)}
            </p>
          </div>
          <div className="text-end">
            <p className="text-xs text-slate-500 dark:text-neutral-400">Criteria</p>
            <p className="mt-0.5 text-sm font-medium tabular-nums text-slate-700 dark:text-neutral-200">
              {gradedCount}/{rubric.criteria.length}
            </p>
          </div>
        </div>
        <div className="mt-3 h-1.5 overflow-hidden rounded-full bg-slate-200 dark:bg-neutral-700">
          <div
            className={`h-full rounded-full transition-all ${
              allGraded ? 'bg-emerald-500' : 'bg-indigo-500'
            }`}
            style={{ width: `${rubric.criteria.length ? (gradedCount / rubric.criteria.length) * 100 : 0}%` }}
          />
        </div>
        {!allGraded ? (
          <p className="mt-2 text-xs text-amber-700 dark:text-amber-300">
            Select a rating for each criterion below.
          </p>
        ) : null}
      </div>

      {rubric.title ? (
        <p className="text-sm font-semibold text-slate-900 dark:text-neutral-100">{rubric.title}</p>
      ) : null}

      {rubric.criteria.map((c, index) => {
        const selected = scores[c.id]
        return (
          <fieldset
            key={c.id}
            className="rounded-xl border border-slate-200 bg-white p-3 dark:border-neutral-600 dark:bg-neutral-900/40"
          >
            <legend className="px-1 text-sm font-semibold text-slate-900 dark:text-neutral-100">
              <span className="me-2 inline-flex h-5 min-w-5 items-center justify-center rounded-full bg-slate-100 px-1.5 text-xs font-bold text-slate-600 dark:bg-neutral-800 dark:text-neutral-300">
                {index + 1}
              </span>
              {c.title}
            </legend>
            {c.description ? (
              <p className="mt-1 text-xs leading-relaxed text-slate-500 dark:text-neutral-400">{c.description}</p>
            ) : null}
            <div className={`mt-3 ${compact ? 'space-y-1.5' : 'space-y-2'}`}>
              {c.levels.map((lvl, i) => {
                const active = selected === lvl.points
                return (
                  <button
                    key={`${c.id}-${i}`}
                    type="button"
                    disabled={disabled}
                    aria-pressed={active}
                    onClick={() => onScoresChange({ ...scores, [c.id]: lvl.points })}
                    className={`flex w-full items-start gap-3 rounded-lg border px-3 py-2.5 text-start text-sm transition disabled:opacity-50 ${
                      active
                        ? 'border-indigo-500 bg-indigo-50 ring-1 ring-indigo-500/30 dark:border-indigo-400 dark:bg-indigo-950/50 dark:ring-indigo-400/30'
                        : 'border-slate-200 bg-slate-50 hover:border-slate-300 hover:bg-white dark:border-neutral-600 dark:bg-neutral-900 dark:hover:border-neutral-500 dark:hover:bg-neutral-800'
                    }`}
                  >
                    <span
                      className={`mt-0.5 flex h-4 w-4 shrink-0 items-center justify-center rounded-full border ${
                        active
                          ? 'border-indigo-600 bg-indigo-600 dark:border-indigo-400 dark:bg-indigo-400'
                          : 'border-slate-300 bg-white dark:border-neutral-500 dark:bg-neutral-950'
                      }`}
                      aria-hidden="true"
                    >
                      {active ? <span className="h-1.5 w-1.5 rounded-full bg-white" /> : null}
                    </span>
                    <span className="min-w-0 flex-1">
                      <span className="flex flex-wrap items-baseline gap-x-2 gap-y-0.5">
                        <span className="font-medium text-slate-900 dark:text-neutral-100">{lvl.label}</span>
                        <span className="text-xs font-semibold tabular-nums text-indigo-700 dark:text-indigo-300">
                          {formatPointsCell(lvl.points)} pts
                        </span>
                      </span>
                      {lvl.description ? (
                        <span className="mt-1 block text-xs leading-relaxed text-slate-500 dark:text-neutral-400">
                          {lvl.description}
                        </span>
                      ) : null}
                    </span>
                  </button>
                )
              })}
            </div>
          </fieldset>
        )
      })}
    </div>
  )
}
