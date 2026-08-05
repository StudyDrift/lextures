import { Check, Circle, CircleDot, HelpCircle } from 'lucide-react'
import {
  normalizeChecklistStatus,
  type ChecklistStatus,
} from '../../lib/course-checklist-api-schemas'
import { courseChecklistI18n } from '../../lib/course-checklist-i18n'

const iconClass = 'h-4 w-4 shrink-0'

export function ChecklistStatusAffordance({ status }: { status: string }) {
  const s: ChecklistStatus = normalizeChecklistStatus(status)
  switch (s) {
    case 'done':
      return (
        <span className="inline-flex items-center text-emerald-600 dark:text-emerald-400">
          <Check className={iconClass} aria-hidden />
          <span className="sr-only">{courseChecklistI18n.completedLabel}</span>
        </span>
      )
    case 'in_progress':
      return (
        <span className="inline-flex items-center text-amber-600 dark:text-amber-400">
          <CircleDot className={iconClass} aria-hidden />
          <span className="sr-only">In progress</span>
        </span>
      )
    case 'unknown':
      return (
        <span className="inline-flex items-center text-slate-400 dark:text-neutral-500">
          <HelpCircle className={iconClass} aria-hidden />
          <span className="sr-only">Unknown</span>
        </span>
      )
    case 'not_applicable':
      return (
        <span className="inline-flex items-center text-slate-400 dark:text-neutral-500">
          <Circle className={`${iconClass} opacity-40`} aria-hidden />
          <span className="sr-only">Not applicable</span>
        </span>
      )
    case 'todo':
      return (
        <span className="inline-flex items-center text-slate-500 dark:text-neutral-400">
          <Circle className={iconClass} aria-hidden />
          <span className="sr-only">To do</span>
        </span>
      )
    default: {
      const _exhaustive: never = s
      return _exhaustive
    }
  }
}
