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
        <p className="text-sm font-medium text-slate-700 dark:text-neutral-200">Time zone</p>
        <p className="mt-1 text-sm text-slate-500 dark:text-neutral-400">
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
          className="mt-3 w-full max-w-md rounded-xl border border-slate-200 bg-white px-2 py-1.5 text-sm text-slate-900 outline-none ring-indigo-500/20 focus:border-indigo-400 focus:ring-2 dark:border-neutral-600 dark:bg-neutral-900 dark:text-neutral-100"
          data-testid="settings-timezone-select"
        >
          {TIMEZONE_OPTIONS.map((o) => (
            <option key={o.value} value={o.value}>
              {o.label}
            </option>
          ))}
        </select>
      </div>

      <p className="text-sm text-slate-600 dark:text-neutral-400">
        Sample due date:{' '}
        <LocaleTime
          date={SAMPLE_ISO}
          data-testid="settings-locale-sample-date"
          className="font-medium text-slate-900 dark:text-neutral-100"
        />
      </p>
    </div>
  )
}
