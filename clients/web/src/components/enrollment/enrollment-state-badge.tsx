import type { EnrollmentState } from '../../lib/enrollment-state-api'

const STATE_LABELS: Record<EnrollmentState, string> = {
  active: 'Active',
  waitlist: 'Waitlisted',
  dropped: 'Dropped',
  withdrawn: 'Withdrawn',
  audit: 'Audit',
  no_credit: 'No Credit',
  incomplete: 'Incomplete',
}

const STATE_ABBREV: Record<EnrollmentState, string> = {
  active: 'Active',
  waitlist: 'WL',
  dropped: 'DR',
  withdrawn: 'W',
  audit: 'AU',
  no_credit: 'NC',
  incomplete: 'I',
}

const STATE_COLORS: Record<EnrollmentState, string> = {
  active: 'bg-emerald-100 text-emerald-800 dark:bg-emerald-950/60 dark:text-emerald-200',
  waitlist: 'bg-amber-100 text-amber-900 dark:bg-amber-950/50 dark:text-amber-200',
  dropped: 'bg-slate-200 text-slate-700 dark:bg-neutral-700 dark:text-neutral-200',
  withdrawn: 'bg-rose-100 text-rose-900 dark:bg-rose-950/50 dark:text-rose-200',
  audit: 'bg-sky-100 text-sky-900 dark:bg-sky-950/50 dark:text-sky-200',
  no_credit: 'bg-orange-100 text-orange-900 dark:bg-orange-950/50 dark:text-orange-200',
  incomplete: 'bg-violet-100 text-violet-900 dark:bg-violet-950/50 dark:text-violet-200',
}

type Props = {
  state: EnrollmentState
  changedAt?: string | null
  className?: string
}

export function EnrollmentStateBadge({ state, changedAt, className = '' }: Props) {
  const label = STATE_LABELS[state] ?? state
  const abbrev = STATE_ABBREV[state] ?? label
  const colors = STATE_COLORS[state] ?? STATE_COLORS.active
  const aria = changedAt ? `${label} — changed ${changedAt}` : label
  return (
    <span
      className={`inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium ${colors} ${className}`}
      aria-label={aria}
      title={label}
    >
      <span className="sm:hidden">{abbrev}</span>
      <span className="hidden sm:inline">{label}</span>
    </span>
  )
}
