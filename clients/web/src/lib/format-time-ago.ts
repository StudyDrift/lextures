import { formatRelativeTime } from './format'

/** Locale-aware phrases like "15 minutes ago" / "in 2 days". */
export function formatTimeAgoFromIso(
  iso: string | null | undefined,
  nowMs = Date.now(),
): string {
  if (iso == null || iso === '') return 'Never'
  const then = new Date(iso).getTime()
  if (Number.isNaN(then)) return 'Never'
  return formatRelativeTime(new Date(then), new Date(nowMs))
}
