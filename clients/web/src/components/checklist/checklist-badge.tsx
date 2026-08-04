import { courseChecklistI18n } from '../../lib/course-checklist-i18n'

type ChecklistBadgeProps = {
  outstandingEssential: number
}

/** Nav badge for outstanding essential checklist items (FR-4 / FR-5). */
export function ChecklistBadge({ outstandingEssential }: ChecklistBadgeProps) {
  if (outstandingEssential <= 0) return null
  const display = outstandingEssential > 99 ? '99+' : String(outstandingEssential)
  return (
    <span
      className="inline-flex min-h-5 min-w-5 shrink-0 items-center justify-center rounded-full bg-amber-600 px-1.5 text-[11px] font-semibold tabular-nums leading-none text-white"
      aria-label={courseChecklistI18n.badgeAria(outstandingEssential)}
    >
      {display}
    </span>
  )
}
