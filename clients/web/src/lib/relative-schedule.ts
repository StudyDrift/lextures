/**
 * Authoring helpers for enrollment-relative course schedules.
 *
 * Item due/availability timestamps are still stored as absolute instants, shifted
 * at read time from `relativeScheduleAnchorAt` to each student's enrollment start.
 * These helpers convert between those stored instants and amount+unit offsets
 * (same shape as course settings: PnD / PnW / PnM / PnY).
 */

export type RelativeDurationUnit = 'D' | 'W' | 'M' | 'Y'

export type RelativeOffsetParts = {
  amount: string
  unit: RelativeDurationUnit
}

const DAY_MS = 24 * 60 * 60 * 1000

function pad2(n: number): string {
  return String(n).padStart(2, '0')
}

/** `datetime-local` value (local wall clock, no timezone suffix). */
export function dateToDatetimeLocalValue(d: Date): string {
  if (Number.isNaN(d.getTime())) return ''
  return `${d.getFullYear()}-${pad2(d.getMonth() + 1)}-${pad2(d.getDate())}T${pad2(d.getHours())}:${pad2(d.getMinutes())}`
}

export function datetimeLocalValueToDate(value: string): Date | null {
  const t = value.trim()
  if (!t) return null
  const d = new Date(t)
  if (Number.isNaN(d.getTime())) return null
  return d
}

export function extractLocalTime(value: string, fallback = '00:00'): string {
  const d = datetimeLocalValueToDate(value)
  if (!d) return fallback
  return `${pad2(d.getHours())}:${pad2(d.getMinutes())}`
}

function startOfLocalDay(d: Date): Date {
  return new Date(d.getFullYear(), d.getMonth(), d.getDate())
}

export function addRelativeDuration(
  anchor: Date,
  amount: number,
  unit: RelativeDurationUnit,
): Date {
  const d = new Date(anchor.getTime())
  switch (unit) {
    case 'D':
      d.setDate(d.getDate() + amount)
      break
    case 'W':
      d.setDate(d.getDate() + amount * 7)
      break
    case 'M':
      d.setMonth(d.getMonth() + amount)
      break
    case 'Y':
      d.setFullYear(d.getFullYear() + amount)
      break
  }
  return d
}

/** Prefer weeks when divisible by 7; otherwise whole days from the anchor day. */
export function dayDeltaToParts(dayDelta: number): RelativeOffsetParts {
  const days = Math.max(0, Math.round(dayDelta))
  if (days === 0) return { amount: '', unit: 'D' }
  if (days % 7 === 0) {
    return { amount: String(days / 7), unit: 'W' }
  }
  return { amount: String(days), unit: 'D' }
}

/**
 * Derive amount+unit from a datetime-local value relative to the course anchor.
 * Empty input → empty amount. Negative offsets clamp to empty (not supported in UI).
 */
export function partsFromDatetimeLocal(
  datetimeLocal: string,
  anchorIso: string | null | undefined,
): RelativeOffsetParts {
  if (!datetimeLocal.trim() || !anchorIso) return { amount: '', unit: 'D' }
  const target = datetimeLocalValueToDate(datetimeLocal)
  const anchor = new Date(anchorIso)
  if (!target || Number.isNaN(anchor.getTime())) return { amount: '', unit: 'D' }
  const dayDelta =
    (startOfLocalDay(target).getTime() - startOfLocalDay(anchor).getTime()) / DAY_MS
  return dayDeltaToParts(dayDelta)
}

/**
 * Build a datetime-local string from anchor + offset, keeping `timeLocal` (HH:mm)
 * when provided, otherwise the time already on `previousDatetimeLocal`, else `defaultTime`.
 */
export function datetimeLocalFromRelativeParts(
  anchorIso: string,
  amountStr: string,
  unit: RelativeDurationUnit,
  opts?: {
    previousDatetimeLocal?: string
    defaultTime?: string
    timeLocal?: string
  },
): string {
  const amount = parseInt(amountStr, 10)
  if (!Number.isFinite(amount) || amount < 1) return ''
  const anchor = new Date(anchorIso)
  if (Number.isNaN(anchor.getTime())) return ''

  const time =
    opts?.timeLocal?.trim() ||
    (opts?.previousDatetimeLocal
      ? extractLocalTime(opts.previousDatetimeLocal, opts.defaultTime ?? '00:00')
      : (opts?.defaultTime ?? '00:00'))
  const [hhRaw, mmRaw] = time.split(':')
  const hh = Math.min(23, Math.max(0, parseInt(hhRaw ?? '0', 10) || 0))
  const mm = Math.min(59, Math.max(0, parseInt(mmRaw ?? '0', 10) || 0))

  const base = startOfLocalDay(anchor)
  const dated = addRelativeDuration(base, amount, unit)
  dated.setHours(hh, mm, 0, 0)
  return dateToDatetimeLocalValue(dated)
}

export function isRelativeScheduleMode(mode: string | null | undefined): boolean {
  return mode === 'relative'
}
