import { useCallback, useEffect, useMemo, useState } from 'react'
import { AlertTriangle, X } from 'lucide-react'
import {
  fetchStatusSummary,
  STATUS_POLL_INTERVAL_MS,
  type StatusSummary,
} from '../lib/status-api'

const DISMISS_SESSION_KEY = 'lextures.statusIncident.dismissed'

function dismissedIncidentIds(): Set<string> {
  try {
    const raw = sessionStorage.getItem(DISMISS_SESSION_KEY)
    if (!raw) return new Set()
    const parsed = JSON.parse(raw) as unknown
    if (!Array.isArray(parsed)) return new Set()
    return new Set(parsed.filter((v): v is string => typeof v === 'string'))
  } catch {
    return new Set()
  }
}

function persistDismissed(ids: Set<string>) {
  sessionStorage.setItem(DISMISS_SESSION_KEY, JSON.stringify([...ids]))
}

function visibleIncidents(summary: StatusSummary | null): StatusSummary['incidents'] {
  if (!summary || summary.incidents.length === 0) return []
  const dismissed = dismissedIncidentIds()
  return summary.incidents.filter((inc) => !dismissed.has(inc.id))
}

function bannerTone(impact: string): string {
  switch (impact.toLowerCase()) {
    case 'critical':
    case 'major':
      return 'border-red-200 bg-red-50 text-red-950 dark:border-red-900/60 dark:bg-red-950/50 dark:text-red-100'
    default:
      return 'border-amber-200 bg-amber-50 text-amber-950 dark:border-amber-900/60 dark:bg-amber-950/40 dark:text-amber-100'
  }
}

export function IncidentStatusBanner() {
  const [summary, setSummary] = useState<StatusSummary | null>(null)
  const [dismissedVersion, setDismissedVersion] = useState(0)

  const loadSummary = useCallback(async () => {
    try {
      const next = await fetchStatusSummary()
      setSummary(next)
    } catch {
      setSummary(null)
    }
  }, [])

  useEffect(() => {
    void loadSummary()
    const timer = window.setInterval(() => {
      void loadSummary()
    }, STATUS_POLL_INTERVAL_MS)
    return () => window.clearInterval(timer)
  }, [loadSummary])

  const incidents = useMemo(
    () => visibleIncidents(summary),
    [summary, dismissedVersion],
  )
  if (incidents.length === 0 || !summary) {
    return null
  }

  const primary = incidents[0]
  const pageUrl = summary.pageUrl || 'https://status.lextures.io'

  function dismissVisible() {
    const dismissed = dismissedIncidentIds()
    for (const inc of incidents) {
      dismissed.add(inc.id)
    }
    persistDismissed(dismissed)
    setDismissedVersion((v) => v + 1)
  }

  return (
    <div
      role="alert"
      aria-live="polite"
      aria-atomic="true"
      className={`flex items-start gap-3 border-b px-4 py-2 text-sm ${bannerTone(primary.impact)}`}
    >
      <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0" aria-hidden />
      <div className="min-w-0 flex-1">
        <p>
          <strong>{primary.name}</strong>
          {incidents.length > 1 ? ` (+${incidents.length - 1} more)` : null}.{' '}
          <a
            href={pageUrl}
            target="_blank"
            rel="noopener noreferrer"
            className="font-medium underline underline-offset-2"
          >
            View system status
          </a>
        </p>
      </div>
      <button
        type="button"
        onClick={dismissVisible}
        className="rounded p-1 transition-[background-color,color,border-color] hover:bg-black/5 dark:hover:bg-white/10"
        aria-label="Dismiss incident notice for this session"
      >
        <X className="h-4 w-4" aria-hidden />
      </button>
    </div>
  )
}