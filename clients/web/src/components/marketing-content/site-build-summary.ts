import type { MarketingBuild } from '../../lib/marketing-content-api'

/** A queued/running site update older than this is treated as stuck (#791). */
export const SITE_BUILD_STALE_MS = 2 * 60 * 60 * 1000

export type SiteBuildSummary = {
  tone: 'info' | 'success' | 'warning' | 'danger'
  title: string
  detail: string
}

const IN_PROGRESS = new Set(['pending', 'dispatched', 'running'])

function formatWhen(iso: string | null | undefined): string | null {
  if (!iso) return null
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return null
  return new Intl.DateTimeFormat(undefined, { dateStyle: 'medium', timeStyle: 'short' }).format(d)
}

/** Plain-language status for the public site's latest update request. */
export function siteBuildSummary(build: MarketingBuild, now: number = Date.now()): SiteBuildSummary {
  const requested = formatWhen(build.createdAt)
  const finished = formatWhen(build.completedAt) ?? requested
  if (build.status === 'succeeded') {
    return {
      tone: 'success',
      title: 'Public site is up to date',
      detail: finished ? `Last updated ${finished}.` : 'Your latest changes are live.',
    }
  }
  if (build.status === 'failed' || build.status === 'timed_out') {
    return {
      tone: 'danger',
      title: 'The last site update didn’t finish',
      detail: `${requested ? `Requested ${requested}. ` : ''}Try Rebuild site. Published changes go live once an update succeeds.`,
    }
  }
  if (IN_PROGRESS.has(build.status)) {
    const created = new Date(build.createdAt).getTime()
    const stale = Number.isFinite(created) && now - created > SITE_BUILD_STALE_MS
    if (stale) {
      return {
        tone: 'warning',
        title: 'Site update is stuck',
        detail: `An update requested ${requested ?? 'earlier'} hasn’t finished. Try Rebuild site; if it stays stuck, site publishing may not be set up yet.`,
      }
    }
    return {
      tone: 'info',
      title: 'Updating the public site',
      detail: 'Published changes go live when the update finishes.',
    }
  }
  return {
    tone: 'info',
    title: 'Public site update',
    detail: requested ? `Requested ${requested}.` : '',
  }
}
