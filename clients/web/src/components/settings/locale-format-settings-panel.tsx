import { useId } from 'react'
import { LocaleTime } from '../ui/locale-time'
import { detectBrowserTimeZone } from '../../lib/format'
import { TIMEZONE_OPTIONS } from '../../lib/format/locale-options'

const SAMPLE_ISO = '2026-04-15T10:00:00.000Z'

type Props = {
  timezone: string
  onTimezoneChange: (value: string) => void
  disabled?: boolean
  embedded?: boolean
}

export function LocaleFormatSettingsPanel({
  timezone,
  onTimezoneChange,
  disabled,
  embedded = false,
}: Props) {
  const tzId = useId()
  const browserTz = detectBrowserTimeZone()

  return (
    <div className={embedded ? 'space-y-4' : 'mt-8 space-y-4'}>
      <div>
        <p className="text-sm font-medium text-fg-default">Time zone</p>
        <p className="mt-1 text-sm text-fg-muted">
          Due dates and deadlines are shown in this zone. Browser default:{' '}
          <span className="font-mono text-xs">{browserTz}</span>.
        </p>
        <label htmlFor={tzId} className="sr-only">
          Time zone
        </label>
        <select
          id={tzId}
          value={timezone}
          disabled={disabled}
          onChange={(e) => onTimezoneChange(e.target.value)}
          className="mt-3 w-full max-w-md rounded-xl border border-border-default bg-surface-raised px-2 py-1.5 text-sm text-fg-default outline-none ring-indigo-500/20 focus:border-indigo-400 focus:ring-2 dark:border-border-default dark:bg-surface-raised dark:text-fg-default"
          data-testid="settings-timezone-select"
        >
          {TIMEZONE_OPTIONS.map((o) => (
            <option key={o.value} value={o.value}>
              {o.label}
            </option>
          ))}
        </select>
      </div>

      <p className="text-sm text-fg-muted">
        Sample due date:{' '}
        <LocaleTime
          date={SAMPLE_ISO}
          data-testid="settings-locale-sample-date"
          className="font-medium text-fg-default"
        />
      </p>
    </div>
  )
}
