import { useCallback, useEffect, useState } from 'react'
import { Calendar, Copy, RefreshCw } from 'lucide-react'
import { authorizedFetch } from '../../lib/api'
import { readApiErrorMessage } from '../../lib/errors'
import { formatDateTime } from '../../lib/format'
import { toastMutationError, toastSaveOk } from '../../lib/lms-toast'
import { usePlatformFeatures } from '../../context/platform-features-context'

type CourseFeed = {
  courseId: string
  courseCode: string
  title: string
  feedUrl: string
}

type CalendarTokenInfo = {
  hasToken?: boolean
  personalFeedUrl?: string
  expiresAt?: string
  courseFeeds?: CourseFeed[]
  token?: string
  feedUrl?: string
}

function withToken(url: string, token: string): string {
  return url.replace('<token>', encodeURIComponent(token))
}

export function CalendarSubscriptionsPanel() {
  const { ffCalendarFeeds, loading: featuresLoading } = usePlatformFeatures()
  const [loading, setLoading] = useState(true)
  const [token, setToken] = useState<string | null>(null)
  const [expiresAt, setExpiresAt] = useState<string | null>(null)
  const [personalUrl, setPersonalUrl] = useState<string | null>(null)
  const [courseFeeds, setCourseFeeds] = useState<CourseFeed[]>([])
  const [hasActiveToken, setHasActiveToken] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const res = await authorizedFetch('/api/v1/me/calendar-token')
      const raw: unknown = await res.json().catch(() => ({}))
      if (!res.ok) throw new Error(readApiErrorMessage(raw))
      const data = raw as CalendarTokenInfo
      if (data.hasToken === false) {
        setHasActiveToken(false)
        setToken(null)
        setPersonalUrl(null)
        setExpiresAt(null)
        setCourseFeeds([])
        return
      }
      setHasActiveToken(true)
      setExpiresAt(data.expiresAt ?? null)
      setCourseFeeds(data.courseFeeds ?? [])
      setPersonalUrl(data.personalFeedUrl ?? null)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Could not load calendar subscriptions.')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    if (!featuresLoading && ffCalendarFeeds) void load()
  }, [featuresLoading, ffCalendarFeeds, load])

  async function regenerate() {
    if (
      !globalThis.confirm(
        'Regenerate your calendar URL? The old link will stop working immediately.',
      )
    ) {
      return
    }
    setError(null)
    try {
      const res = await authorizedFetch('/api/v1/me/calendar-token', { method: 'POST' })
      const raw: unknown = await res.json().catch(() => ({}))
      if (!res.ok) throw new Error(readApiErrorMessage(raw))
      const data = raw as CalendarTokenInfo
      if (!data.token) throw new Error('Token was missing from the response.')
      setToken(data.token)
      setHasActiveToken(true)
      setExpiresAt(data.expiresAt ?? null)
      setPersonalUrl(data.feedUrl ?? null)
      toastSaveOk('Calendar subscription URL generated.')
      await load()
    } catch (e) {
      toastMutationError(e instanceof Error ? e.message : 'Could not regenerate calendar URL.')
    }
  }

  async function copyText(label: string, text: string) {
    try {
      await navigator.clipboard.writeText(text)
      toastSaveOk(`${label} copied.`)
    } catch {
      toastMutationError('Could not copy to clipboard.')
    }
  }

  if (featuresLoading || !ffCalendarFeeds) return null

  const resolvedPersonal =
    token && personalUrl ? withToken(personalUrl, token) : personalUrl?.replace('?token=<token>', '') ?? null

  return (
    <section className="mt-8 rounded-xl border border-slate-200 bg-white p-5 dark:border-neutral-700 dark:bg-neutral-900">
      <div className="flex items-start gap-3">
        <Calendar className="mt-0.5 h-5 w-5 shrink-0 text-slate-500 dark:text-neutral-400" aria-hidden />
        <div className="min-w-0 flex-1">
          <h3 className="text-sm font-semibold text-slate-900 dark:text-neutral-100">Calendar subscriptions</h3>
          <p className="mt-1 text-sm text-slate-500 dark:text-neutral-400">
            Subscribe to assignment and quiz deadlines in Google Calendar, Apple Calendar, or Outlook using a private
            feed URL.
          </p>

          {loading ? (
            <p className="mt-4 text-sm text-slate-500 dark:text-neutral-400">Loading…</p>
          ) : error ? (
            <p className="mt-4 text-sm text-red-600 dark:text-red-400">{error}</p>
          ) : (
            <div className="mt-4 space-y-4">
              <div>
                <p className="text-xs font-medium uppercase tracking-wide text-slate-500 dark:text-neutral-400">
                  Personal feed (all courses)
                </p>
                {resolvedPersonal ? (
                  <div className="mt-2 flex flex-col gap-2 sm:flex-row sm:items-center">
                    <input
                      readOnly
                      value={resolvedPersonal}
                      className="min-w-0 flex-1 rounded-lg border border-slate-200 bg-slate-50 px-3 py-2 font-mono text-xs text-slate-800 dark:border-neutral-600 dark:bg-neutral-800 dark:text-neutral-100"
                      aria-label="Personal calendar feed URL"
                    />
                    <button
                      type="button"
                      onClick={() => void copyText('Feed URL', resolvedPersonal)}
                      className="inline-flex items-center justify-center gap-1.5 rounded-lg border border-slate-200 px-3 py-2 text-sm font-medium text-slate-700 hover:bg-slate-50 dark:border-neutral-600 dark:text-neutral-200 dark:hover:bg-neutral-800"
                    >
                      <Copy className="h-4 w-4" aria-hidden />
                      Copy
                    </button>
                  </div>
                ) : hasActiveToken ? (
                  <p className="mt-2 text-sm text-slate-600 dark:text-neutral-300">
                    You have an active subscription URL. Regenerate to view and copy the link.
                  </p>
                ) : (
                  <p className="mt-2 text-sm text-slate-600 dark:text-neutral-300">
                    Generate a URL to subscribe in your calendar app.
                  </p>
                )}
                {expiresAt ? (
                  <p className="mt-1 text-xs text-slate-500 dark:text-neutral-400">
                    Expires {formatDateTime(expiresAt)}
                  </p>
                ) : null}
              </div>

              {token && courseFeeds.length > 0 ? (
                <div>
                  <p className="text-xs font-medium uppercase tracking-wide text-slate-500 dark:text-neutral-400">
                    Per-course feeds
                  </p>
                  <ul className="mt-2 divide-y divide-slate-100 dark:divide-neutral-800">
                    {courseFeeds.map((c) => {
                      const url = withToken(c.feedUrl, token)
                      return (
                        <li key={c.courseId} className="flex flex-col gap-2 py-3 sm:flex-row sm:items-center sm:justify-between">
                          <div className="min-w-0">
                            <p className="truncate text-sm font-medium text-slate-900 dark:text-neutral-100">{c.title}</p>
                            <p className="truncate font-mono text-xs text-slate-500 dark:text-neutral-400">{url}</p>
                          </div>
                          <button
                            type="button"
                            onClick={() => void copyText(`${c.title} feed`, url)}
                            className="inline-flex shrink-0 items-center gap-1.5 rounded-lg border border-slate-200 px-3 py-1.5 text-sm text-slate-700 hover:bg-slate-50 dark:border-neutral-600 dark:text-neutral-200 dark:hover:bg-neutral-800"
                          >
                            <Copy className="h-4 w-4" aria-hidden />
                            Copy
                          </button>
                        </li>
                      )
                    })}
                  </ul>
                </div>
              ) : null}

              <div className="rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-sm text-amber-900 dark:border-amber-900/50 dark:bg-amber-950/40 dark:text-amber-100">
                Anyone with your feed URL can see assignment titles and due dates. Do not share it publicly.
              </div>

              <details className="rounded-lg border border-slate-200 p-3 dark:border-neutral-700">
                <summary className="cursor-pointer text-sm font-medium text-slate-800 dark:text-neutral-200">
                  How to subscribe
                </summary>
                <ol className="mt-3 list-decimal space-y-2 pl-5 text-sm text-slate-600 dark:text-neutral-300">
                  <li>
                    <strong>Google Calendar:</strong> Settings → Add calendar → From URL → paste your feed URL.
                  </li>
                  <li>
                    <strong>Apple Calendar:</strong> File → New Calendar Subscription → paste your feed URL.
                  </li>
                  <li>
                    <strong>Outlook:</strong> Add calendar → Subscribe from web → paste your feed URL.
                  </li>
                </ol>
              </details>

              <button
                type="button"
                onClick={() => void regenerate()}
                className="inline-flex items-center gap-1.5 rounded-lg bg-slate-900 px-4 py-2 text-sm font-medium text-white hover:bg-slate-800 dark:bg-neutral-100 dark:text-neutral-900 dark:hover:bg-white"
              >
                <RefreshCw className="h-4 w-4" aria-hidden />
                {token ? 'Regenerate URL' : 'Generate URL'}
              </button>
            </div>
          )}
        </div>
      </div>
    </section>
  )
}
