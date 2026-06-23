import type { RubricDefinition, RubricCriterion } from '../../lib/courses-api'
import { formatPointsCell, rubricGradedCount, rubricTotal, rubricScoresComplete } from '../../lib/rubric-utils'

type RubricGradePickerProps = {
  rubric: RubricDefinition
  scores: Record<string, number>
  onScoresChange: (scores: Record<string, number>) => void
  disabled?: boolean
  compact?: boolean
}

const SEGMENTED_MAX_LABEL_LENGTH = 32

function criterionUsesSegmentedControl(c: RubricCriterion): boolean {
  if (c.levels.length !== 2) return false
  return c.levels.every(
    (lvl) => !lvl.description?.trim() && lvl.label.length <= SEGMENTED_MAX_LABEL_LENGTH,
  )
}

type CriterionPickerProps = {
  criterion: RubricCriterion
  index: number
  selected: number | undefined
  disabled: boolean
  compact: boolean
  onSelect: (points: number) => void
}

function CriterionPicker({
  criterion: c,
  index,
  selected,
  disabled,
  compact,
  onSelect,
}: CriterionPickerProps) {
  const segmented = criterionUsesSegmentedControl(c)

  return (
    <div className="overflow-hidden rounded-lg border border-slate-200 dark:border-neutral-700">
      <div className={`${compact ? 'px-3 py-2.5' : 'px-4 py-3'}`}>
        <p className="text-sm leading-snug text-slate-900 dark:text-neutral-100">
          <span className="me-1.5 font-semibold tabular-nums text-slate-400 dark:text-neutral-500">
            {index + 1}.
          </span>
          {c.title}
        </p>
        {c.description ? (
          <p className="mt-1 text-xs leading-relaxed text-slate-500 dark:text-neutral-400">{c.description}</p>
        ) : null}
      </div>

      {segmented ? (
        <div
          className="grid border-t border-slate-200 dark:border-neutral-700"
          style={{ gridTemplateColumns: `repeat(${c.levels.length}, minmax(0, 1fr))` }}
          role="radiogroup"
          aria-label={c.title}
        >
          {c.levels.map((lvl, i) => {
            const active = selected === lvl.points
            return (
              <button
                key={`${c.id}-${i}`}
                type="button"
                role="radio"
                aria-checked={active}
                disabled={disabled}
                onClick={() => onSelect(lvl.points)}
                className={`border-slate-200 px-2 py-2.5 text-center transition disabled:opacity-50 dark:border-neutral-700 ${
                  i > 0 ? 'border-s' : ''
                } ${
                  active
                    ? 'bg-indigo-600 text-white dark:bg-indigo-500'
                    : 'bg-slate-50 text-slate-700 hover:bg-slate-100 dark:bg-neutral-900 dark:text-neutral-200 dark:hover:bg-neutral-800'
                }`}
              >
                <span className="block text-xs font-medium leading-tight">{lvl.label}</span>
                <span
                  className={`mt-0.5 block text-[11px] tabular-nums ${
                    active ? 'text-indigo-100' : 'text-slate-500 dark:text-neutral-400'
                  }`}
                >
                  {formatPointsCell(lvl.points)} pts
                </span>
              </button>
            )
          })}
        </div>
      ) : (
        <div
          className={`border-t border-slate-200 dark:border-neutral-700 ${compact ? 'space-y-2 p-2' : 'space-y-2.5 p-3'}`}
          role="radiogroup"
          aria-label={c.title}
        >
          {c.levels.map((lvl, i) => {
            const active = selected === lvl.points
            return (
              <button
                key={`${c.id}-${i}`}
                type="button"
                role="radio"
                aria-checked={active}
                disabled={disabled}
                onClick={() => onSelect(lvl.points)}
                className={`w-full rounded-lg border px-3 py-2.5 text-start transition disabled:opacity-50 ${
                  active
                    ? 'border-indigo-500 bg-indigo-50 ring-1 ring-indigo-500/25 dark:border-indigo-400 dark:bg-indigo-950/50 dark:ring-indigo-400/25'
                    : 'border-slate-200 bg-slate-50 hover:border-slate-300 hover:bg-white dark:border-neutral-600 dark:bg-neutral-900 dark:hover:border-neutral-500 dark:hover:bg-neutral-800'
                }`}
              >
                <div className="flex items-start justify-between gap-3">
                  <span
                    className={`text-sm font-medium leading-snug ${
                      active ? 'text-indigo-900 dark:text-indigo-100' : 'text-slate-900 dark:text-neutral-100'
                    }`}
                  >
                    {lvl.label}
                  </span>
                  <span
                    className={`shrink-0 text-xs font-semibold tabular-nums ${
                      active ? 'text-indigo-700 dark:text-indigo-300' : 'text-slate-500 dark:text-neutral-400'
                    }`}
                  >
                    {formatPointsCell(lvl.points)} pts
                  </span>
                </div>
                {lvl.description ? (
                  <p className="mt-1.5 text-xs leading-relaxed text-slate-600 dark:text-neutral-400">
                    {lvl.description}
                  </p>
                ) : null}
              </button>
            )
          })}
        </div>
      )}
    </div>
  )
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
    <div className={compact ? 'space-y-3' : 'space-y-4'}>
      {!compact ? (
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
              className={`h-full rounded-full motion-safe:transition-[width] motion-safe:duration-300 ${
                allGraded ? 'bg-emerald-500' : 'bg-indigo-500'
              }`}
              style={{ width: `${rubric.criteria.length ? (gradedCount / rubric.criteria.length) * 100 : 0}%` }}
            />
          </div>
          {!allGraded ? (
            <p className="mt-2 text-xs text-slate-500 dark:text-neutral-400">
              Select a rating for each criterion below.
            </p>
          ) : null}
        </div>
      ) : !allGraded ? (
        <p className="text-xs text-slate-500 dark:text-neutral-400">
          {gradedCount} of {rubric.criteria.length} criteria rated
        </p>
      ) : null}

      {rubric.title ? (
        <p className="text-sm font-medium text-slate-800 dark:text-neutral-200">{rubric.title}</p>
      ) : null}

      <div className={compact ? 'space-y-3' : 'space-y-4'}>
        {rubric.criteria.map((c, index) => (
          <CriterionPicker
            key={c.id}
            criterion={c}
            index={index}
            selected={scores[c.id]}
            disabled={disabled}
            compact={compact}
            onSelect={(points) => onScoresChange({ ...scores, [c.id]: points })}
          />
        ))}
      </div>
    </div>
  )
}
