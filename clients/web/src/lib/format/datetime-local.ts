import { isLearnerLocalTimezone } from './timezone'

function pad2(n: number): string {
  return String(n).padStart(2, '0')
}

/**
 * Convert a stored ISO instant into a `datetime-local` wall-clock value.
 * When courseTimezone is LOCAL, the instant's UTC components are the authored wall clock.
 */
export function isoToDatetimeLocalValue(
  iso: string | null | undefined,
  courseTimezone?: string | null,
): string {
  if (!iso) return ''
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return ''
  if (isLearnerLocalTimezone(courseTimezone)) {
    return `${d.getUTCFullYear()}-${pad2(d.getUTCMonth() + 1)}-${pad2(d.getUTCDate())}T${pad2(d.getUTCHours())}:${pad2(d.getUTCMinutes())}`
  }
  return `${d.getFullYear()}-${pad2(d.getMonth() + 1)}-${pad2(d.getDate())}T${pad2(d.getHours())}:${pad2(d.getMinutes())}`
}

/**
 * Convert a `datetime-local` value to an ISO instant for the API.
 * When courseTimezone is LOCAL, the wall clock is stored as UTC components (floating local time).
 */
export function datetimeLocalValueToIso(
  value: string,
  courseTimezone?: string | null,
): string | null {
  const t = value.trim()
  if (!t) return null
  if (isLearnerLocalTimezone(courseTimezone)) {
    // Expect YYYY-MM-DDTHH:mm or YYYY-MM-DDTHH:mm:ss
    const m = t.match(
      /^(\d{4})-(\d{2})-(\d{2})T(\d{2}):(\d{2})(?::(\d{2}))?$/,
    )
    if (!m) {
      // Fallback: still try to treat as floating via Date UTC parts if parseable.
      const d = new Date(t)
      if (Number.isNaN(d.getTime())) return null
      return new Date(
        Date.UTC(
          d.getFullYear(),
          d.getMonth(),
          d.getDate(),
          d.getHours(),
          d.getMinutes(),
          d.getSeconds() || 0,
        ),
      ).toISOString()
    }
    const y = Number(m[1])
    const mo = Number(m[2]) - 1
    const day = Number(m[3])
    const h = Number(m[4])
    const mi = Number(m[5])
    const s = m[6] != null ? Number(m[6]) : 0
    return new Date(Date.UTC(y, mo, day, h, mi, s)).toISOString()
  }
  const d = new Date(t)
  if (Number.isNaN(d.getTime())) return null
  return d.toISOString()
}
