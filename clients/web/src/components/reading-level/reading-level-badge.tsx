import { BookOpen } from 'lucide-react'
import { formatFkglLabel } from '../../lib/reading-level-api'

export type ReadingLevelBadgeProps = {
  fkgl?: number
  sufficient?: boolean
  aboveThreshold?: boolean
  onClick?: () => void
}

export function ReadingLevelBadge({ fkgl, sufficient, aboveThreshold, onClick }: ReadingLevelBadgeProps) {
  const label = sufficient && fkgl != null ? formatFkglLabel(fkgl) : 'Insufficient text'
  const aria = sufficient && fkgl != null
    ? `Flesch-Kincaid Grade Level: ${fkgl.toFixed(1)}`
    : 'Insufficient text for reading level scoring'

  const base =
    'inline-flex items-center gap-1 rounded-full border px-2 py-0.5 text-xs font-medium'
  const tone = aboveThreshold
    ? 'border-amber-300 bg-amber-50 text-amber-900 dark:border-amber-700 dark:bg-amber-950/40 dark:text-amber-100'
    : 'border-slate-200 bg-slate-50 text-slate-700 dark:border-neutral-600 dark:bg-neutral-800 dark:text-neutral-200'

  if (onClick) {
    return (
      <button
        type="button"
        onClick={onClick}
        className={`${base} ${tone} hover:opacity-90`}
        aria-label={aria}
        title={aria}
      >
        <BookOpen className="h-3.5 w-3.5 shrink-0" aria-hidden />
        {label}
      </button>
    )
  }

  return (
    <span className={`${base} ${tone}`} aria-label={aria} title={aria}>
      <BookOpen className="h-3.5 w-3.5 shrink-0" aria-hidden />
      {label}
    </span>
  )
}
