import { courseChecklistI18n } from '../../lib/course-checklist-i18n'

type ChecklistProgressProps = {
  done: number
  total: number
  outstandingTotal?: number
  checkedLabel?: string
  className?: string
}

export function ChecklistProgressBar({
  done,
  total,
  outstandingTotal,
  checkedLabel,
  className = '',
}: ChecklistProgressProps) {
  const safeTotal = Math.max(total, 0)
  const safeDone = Math.min(Math.max(done, 0), safeTotal || done)
  const pct = safeTotal > 0 ? Math.round((safeDone / safeTotal) * 100) : 0
  const label = courseChecklistI18n.progressDoneOfTotal(safeDone, safeTotal)

  return (
    <div className={className}>
      <div className="flex flex-wrap items-baseline gap-x-2 gap-y-1 text-sm text-slate-600 dark:text-neutral-400">
        <span className="font-medium text-slate-800 dark:text-neutral-200">{label}</span>
        {outstandingTotal != null && outstandingTotal > 0 ? (
          <span>· {courseChecklistI18n.needAttention(outstandingTotal)}</span>
        ) : null}
        {checkedLabel ? <span>· {checkedLabel}</span> : null}
      </div>
      <div
        className="mt-2 h-2 w-full overflow-hidden rounded-full bg-slate-200 dark:bg-neutral-700"
        role="progressbar"
        aria-valuemin={0}
        aria-valuemax={safeTotal || 100}
        aria-valuenow={safeDone}
        aria-label={label}
      >
        <div
          className="h-full rounded-full bg-emerald-600 transition-[width] duration-300 motion-reduce:transition-none dark:bg-emerald-500"
          style={{ width: `${pct}%` }}
        />
      </div>
    </div>
  )
}
