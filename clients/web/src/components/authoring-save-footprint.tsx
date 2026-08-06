import { useEffect, useState } from 'react'
import { formatTimeAgoFromIso } from '../lib/format-time-ago'

type AuthoringSaveFootprintProps = {
  /** Server or client ISO time of last successful save. */
  lastSavedIso: string | null
  saving: boolean
  error: string | null
  onRetry?: () => void
  className?: string
}

export function AuthoringSaveFootprint({
  lastSavedIso,
  saving,
  error,
  onRetry,
  className = '',
}: AuthoringSaveFootprintProps) {
  const [, setTick] = useState(0)
  useEffect(() => {
    const id = window.setInterval(() => setTick((t) => t + 1), 2000)
    return () => window.clearInterval(id)
  }, [])

  const rel = lastSavedIso ? formatTimeAgoFromIso(lastSavedIso) : null

  return (
    <div
      className={`lms-print-hide flex flex-wrap items-center gap-3 rounded-xl border border-border-default bg-surface-raised px-4 py-2.5 text-sm shadow-sm dark:border-border-default dark:bg-surface-raised ${className}`}
      aria-live="polite"
    >
      <div className="min-w-0 flex-1 text-fg-muted">
        {saving ? (
          <span className="font-medium text-fg-default">Saving…</span>
        ) : error ? (
          <span className="text-rose-700 dark:text-rose-300">Could not save — check your connection and try again.</span>
        ) : rel ? (
          <span>
            <span className="font-medium text-emerald-800 dark:text-emerald-200">Saved</span>{' '}
            <span className="text-fg-muted">{rel}</span>
          </span>
        ) : (
          <span className="text-fg-muted">Not saved yet</span>
        )}
      </div>
      {error && onRetry ? (
        <button
          type="button"
          onClick={onRetry}
          className="shrink-0 rounded-lg bg-rose-600 px-3 py-1.5 text-xs font-semibold text-white shadow-sm hover:bg-rose-500 dark:bg-rose-500 dark:hover:bg-rose-400"
        >
          Retry save
        </button>
      ) : null}
    </div>
  )
}
