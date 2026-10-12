/**
 * Scheduled publishing for repository-native guides and blog articles.
 *
 * A page with a `publishDate` (YYYY-MM-DD) is live on and after that calendar
 * date in America/Boise. The static build drops future-dated pages from
 * routes, the sitemap, and internal listings; the daily scheduled deploy
 * (see www/docs/scheduled-publishing.md) rebuilds the site so they go live.
 */
export const PUBLISH_TIME_ZONE = 'America/Boise'

const ISO_DATE = /^\d{4}-\d{2}-\d{2}$/

/** Calendar date (YYYY-MM-DD) for `now` in America/Boise. */
export function boiseDate(now: Date = new Date()): string {
  const parts = new Intl.DateTimeFormat('en-CA', {
    timeZone: PUBLISH_TIME_ZONE, year: 'numeric', month: '2-digit', day: '2-digit',
  }).formatToParts(now)
  const get = (type: string) => parts.find(p => p.type === type)?.value ?? ''
  return `${get('year')}-${get('month')}-${get('day')}`
}

/**
 * Today's publishing date. `PUBLISH_DATE_OVERRIDE=YYYY-MM-DD` (build env)
 * previews the site as of another day, e.g. to check a scheduled page locally.
 */
export function publishToday(now: Date = new Date()): string {
  const env = (globalThis as { process?: { env?: Record<string, string | undefined> } }).process?.env
  const override = env?.PUBLISH_DATE_OVERRIDE?.trim()
  if (override) {
    if (!ISO_DATE.test(override)) throw new Error(`PUBLISH_DATE_OVERRIDE must be YYYY-MM-DD, got "${override}"`)
    return override
  }
  return boiseDate(now)
}

/** True when a page with this publishDate is live on `today`. No date = live. */
export function isPublishedOn(publishDate: string | undefined | null, today: string = publishToday()): boolean {
  if (!publishDate) return true
  const date = String(publishDate).slice(0, 10)
  if (!ISO_DATE.test(date)) throw new Error(`publishDate must be YYYY-MM-DD, got "${publishDate}"`)
  return date <= today
}

export function filterPublished<T extends { publishDate?: string | null }>(items: T[], today: string = publishToday()): T[] {
  return items.filter(item => isPublishedOn(item.publishDate, today))
}
