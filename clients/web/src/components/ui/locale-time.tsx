import type { ReactNode } from 'react'
import { useLocaleFormat } from '../../hooks/useLocaleFormat'
import { toIsoDateTime } from '../../lib/format'

type LocaleTimeProps = {
  date: string | Date | null | undefined
  children?: ReactNode
  className?: string
  title?: string
  dateStyle?: Intl.DateTimeFormatOptions['dateStyle']
  timeStyle?: Intl.DateTimeFormatOptions['timeStyle']
  /** When set, uses formatDate instead of formatDateTime. */
  dateOnly?: boolean
  'data-testid'?: string
}

export function LocaleTime({
  date,
  children,
  className,
  title,
  dateStyle = 'medium',
  timeStyle = 'short',
  dateOnly = false,
  'data-testid': testId,
}: LocaleTimeProps) {
  const { formatDate, formatDateTime } = useLocaleFormat()
  const iso = toIsoDateTime(date)
  if (!iso) {
    return <span className={className}>—</span>
  }
  const label =
    children ??
    (dateOnly
      ? formatDate(date, { dateStyle })
      : formatDateTime(date, { dateStyle, timeStyle }))
  return (
    <time dateTime={iso} className={className} title={title} data-testid={testId}>
      {label}
    </time>
  )
}
