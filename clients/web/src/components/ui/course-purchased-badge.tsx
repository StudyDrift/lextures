import { BadgeCheck } from 'lucide-react'
import { useTranslation } from 'react-i18next'

type Props = {
  /** When true, use compact overlay styling (on hero images). */
  overlay?: boolean
  className?: string
  'data-testid'?: string
}

/**
 * Marketplace acquisition badge for the Courses catalog (plan MKT5).
 * Distinct from {@link CourseCatalogStatusPill} — both may show together.
 */
export function CoursePurchasedBadge({
  overlay = false,
  className = '',
  'data-testid': testId = 'course-purchased-badge',
}: Props) {
  const { t } = useTranslation('common')
  const label = t('courses.badge.purchased')

  const base = overlay
    ? 'inline-flex items-center gap-1 rounded-full bg-emerald-700/90 px-2 py-1 text-[11px] font-medium text-white backdrop-blur-sm'
    : 'inline-flex max-w-full items-center gap-1 rounded-full border border-emerald-300/80 bg-emerald-50 px-2 py-0.5 text-[0.7rem] font-semibold uppercase tracking-wide text-emerald-950 dark:border-emerald-800 dark:bg-emerald-950/50 dark:text-emerald-50'

  return (
    <span
      className={`${base} ${className}`.trim()}
      data-testid={testId}
      title={label}
      aria-label={label}
    >
      <BadgeCheck className="h-3 w-3 shrink-0" strokeWidth={2.5} aria-hidden />
      <span>{label}</span>
    </span>
  )
}
