import type { RelativeDurationUnit } from '../../lib/relative-schedule'
import {
  datetimeLocalFromRelativeParts,
  extractLocalTime,
  isRelativeScheduleMode,
  partsFromDatetimeLocal,
} from '../../lib/relative-schedule'

export type ScheduleDatetimeFieldProps = {
  id: string
  /** Accessible name; used as the field label. */
  label: string
  /** Absolute (fixed) mode label override — defaults to `label`. */
  fixedLabel?: string
  /** Relative mode label (e.g. "Due after enrollment"). Defaults to `label`. */
  relativeLabel?: string
  hint?: string
  /** Fixed-mode hint override. */
  fixedHint?: string
  /** Relative-mode hint override. */
  relativeHint?: string
  /** `datetime-local` value (local wall clock). Empty clears the date. */
  value: string
  onChange: (value: string) => void
  disabled?: boolean
  /** Course `scheduleMode`. When `relative` + anchor, shows amount+unit. */
  scheduleMode?: string | null
  /** Course `relativeScheduleAnchorAt` ISO string. */
  relativeAnchorAt?: string | null
  /** Default local time (HH:mm) when setting a new relative offset. */
  defaultTime?: string
  /** Optional class for the primary control. */
  className?: string
  /** When true, hide the outer label (caller supplies its own). */
  hideLabel?: boolean
}

const defaultInputClass =
  'w-full rounded-lg border border-border-default bg-surface-raised px-2 py-1.5 text-sm text-fg-default focus:border-indigo-400 focus:outline-none focus:ring-1 focus:ring-indigo-400 disabled:opacity-60 dark:border-border-default dark:bg-surface-base dark:text-fg-default dark:focus:border-indigo-500 dark:focus:ring-indigo-500'

/**
 * Date control that switches between calendar `datetime-local` (fixed courses)
 * and enrollment-relative amount+unit (relative courses), matching course settings.
 */
export function ScheduleDatetimeField({
  id,
  label,
  fixedLabel,
  relativeLabel,
  hint,
  fixedHint,
  relativeHint,
  value,
  onChange,
  disabled,
  scheduleMode,
  relativeAnchorAt,
  defaultTime = '00:00',
  className = defaultInputClass,
  hideLabel = false,
}: ScheduleDatetimeFieldProps) {
  const relative = isRelativeScheduleMode(scheduleMode) && Boolean(relativeAnchorAt?.trim())
  const displayLabel = relative ? (relativeLabel ?? label) : (fixedLabel ?? label)
  const displayHint = relative ? (relativeHint ?? hint) : (fixedHint ?? hint)
  const parts = relative
    ? partsFromDatetimeLocal(value, relativeAnchorAt)
    : { amount: '', unit: 'D' as RelativeDurationUnit }
  const timeLocal = extractLocalTime(value, defaultTime)

  return (
    <div>
      {!hideLabel ? (
        <div className="mb-1 flex items-center justify-between gap-2">
          <label htmlFor={id} className="text-xs font-medium text-fg-muted">
            {displayLabel}
          </label>
          {value ? (
            <button
              type="button"
              onClick={() => onChange('')}
              disabled={disabled}
              className="text-[11px] font-medium text-accent-fg hover:text-indigo-500 disabled:opacity-50 dark:text-indigo-400"
            >
              Clear
            </button>
          ) : null}
        </div>
      ) : value ? (
        <div className="mb-1 flex justify-end">
          <button
            type="button"
            onClick={() => onChange('')}
            disabled={disabled}
            className="text-[11px] font-medium text-accent-fg hover:text-indigo-500 disabled:opacity-50 dark:text-indigo-400"
          >
            Clear
          </button>
        </div>
      ) : null}

      {relative ? (
        <div className="space-y-2">
          <div className="flex gap-2">
            <input
              id={id}
              type="number"
              min={1}
              step={1}
              inputMode="numeric"
              placeholder="e.g. 7"
              value={parts.amount}
              disabled={disabled}
              onChange={(e) => {
                const amount = e.target.value
                onChange(
                  datetimeLocalFromRelativeParts(relativeAnchorAt!, amount, parts.unit, {
                    previousDatetimeLocal: value,
                    defaultTime,
                    timeLocal,
                  }),
                )
              }}
              className={`min-w-0 flex-1 ${className}`}
            />
            <select
              aria-label={`${displayLabel} unit`}
              value={parts.unit}
              disabled={disabled}
              onChange={(e) => {
                const unit = e.target.value as RelativeDurationUnit
                const amount = parts.amount || '1'
                onChange(
                  datetimeLocalFromRelativeParts(relativeAnchorAt!, amount, unit, {
                    previousDatetimeLocal: value,
                    defaultTime,
                    timeLocal,
                  }),
                )
              }}
              className={`w-28 shrink-0 ${className}`}
            >
              <option value="D">Days</option>
              <option value="W">Weeks</option>
              <option value="M">Months</option>
              <option value="Y">Years</option>
            </select>
          </div>
          <div className="flex items-center gap-2">
            <label
              htmlFor={`${id}-time`}
              className="shrink-0 text-[11px] font-medium text-fg-muted"
            >
              Time
            </label>
            <input
              id={`${id}-time`}
              type="time"
              value={value ? timeLocal : defaultTime}
              disabled={disabled || !value}
              onChange={(e) => {
                if (!parts.amount) return
                onChange(
                  datetimeLocalFromRelativeParts(
                    relativeAnchorAt!,
                    parts.amount,
                    parts.unit,
                    {
                      previousDatetimeLocal: value,
                      defaultTime,
                      timeLocal: e.target.value,
                    },
                  ),
                )
              }}
              className={`min-w-0 flex-1 ${className}`}
            />
          </div>
        </div>
      ) : (
        <input
          id={id}
          type="datetime-local"
          value={value}
          onChange={(e) => onChange(e.target.value)}
          disabled={disabled}
          className={className}
        />
      )}

      {displayHint ? (
        <p className="mt-1 text-[11px] leading-snug text-fg-muted">
          {displayHint}
        </p>
      ) : null}
    </div>
  )
}

export type ScheduleModeBannerProps = {
  scheduleMode?: string | null
  relativeAnchorAt?: string | null
  className?: string
}

/** Short note shown above scheduling fields when the course uses relative dates. */
export function RelativeScheduleBanner({
  scheduleMode,
  relativeAnchorAt,
  className = '',
}: ScheduleModeBannerProps) {
  if (!isRelativeScheduleMode(scheduleMode) || !relativeAnchorAt?.trim()) return null
  return (
    <p
      className={`rounded-lg border border-indigo-200/80 bg-indigo-50/80 px-2.5 py-2 text-[11px] leading-snug text-indigo-950 dark:border-indigo-500/30 dark:bg-indigo-950/40 dark:text-indigo-100 ${className}`}
    >
      This course uses <strong className="font-semibold">relative dates</strong>. Offsets are from
      each student&apos;s enrollment (course timeline anchor), not a fixed calendar day.
    </p>
  )
}
