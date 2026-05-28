import { useId } from 'react'
import { LocaleTime } from '../ui/locale-time'
import { detectBrowserLocale, detectBrowserTimeZone } from '../../lib/format'
import { LOCALE_OPTIONS, TIMEZONE_OPTIONS } from '../../lib/format/locale-options'

const SAMPLE_ISO = '2026-04-15T10:00:00.000Z'

type Props = {
  locale: string
  timezone: string
  onLocaleChange: (value: string) => void
  onTimezoneChange: (value: string) => void
  disabled?: boolean
}

export function LocaleFormatSettingsPanel({
  locale,
  timezone,
  onLocaleChange,
  onTimezoneChange,
  disabled,
}: Props) {
  const localeId = useId()
  const tzId = useId()
  const browserLocale = detectBrowserLocale()
  const browserTz = detectBrowserTimeZone()

  return (
    <div className="mt-8 space-y-4">
      <div>
        <p className="text-sm font-medium text-slate-700 dark:text-neutral-200">Language & region</p>
        <p className="mt-1 text-sm text-slate-500 dark:text-neutral-400">
          Controls how dates, times, and numbers are formatted across the app. Browser default:{' '}
          <span className="font-mono text-xs">{browserLocale}</span>.
        </p>
        <label htmlFor={localeId} className="sr-only">
          Display locale
        </label>
        <select
          id={localeId}
          value={locale}
          disabled={disabled}
          onChange={(e) => onLocaleChange(e.target.value)}
          className="mt-3 w-full max-w-md rounded-xl border border-slate-200 bg-white px-3 py-2.5 text-sm text-slate-900 outline-none ring-indigo-500/20 focus:border-indigo-400 focus:ring-2 dark:border-neutral-600 dark:bg-neutral-900 dark:text-neutral-100"
          data-testid="settings-locale-select"
        >
          {LOCALE_OPTIONS.map((o) => (
            <option key={o.value} value={o.value}>
              {o.label} ({o.value})
            </option>
          ))}
        </select>
      </div>

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
          className="mt-3 w-full max-w-md rounded-xl border border-slate-200 bg-white px-3 py-2.5 text-sm text-slate-900 outline-none ring-indigo-500/20 focus:border-indigo-400 focus:ring-2 dark:border-neutral-600 dark:bg-neutral-900 dark:text-neutral-100"
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
