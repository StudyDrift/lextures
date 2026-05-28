import { useLocaleFormat } from '../../hooks/useLocaleFormat'

type Props = {
  iso: string
  courseTimezone?: string | null
  className?: string
}

/** Locale- and timezone-aware deadline with instructor-timezone tooltip. */
export function DeadlineDateTime({ iso, courseTimezone, className }: Props) {
  const { formatDeadline } = useLocaleFormat(courseTimezone)
  const d = formatDeadline(iso)
  const title = d.instructorHint ? `${d.primary} ${d.abbrev} · ${d.instructorHint}` : `${d.primary} ${d.abbrev}`

  return (
    <time dateTime={d.iso} className={className} title={title} aria-label={d.ariaLabel}>
      {d.primary} {d.abbrev}
    </time>
  )
}
