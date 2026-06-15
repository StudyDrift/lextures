import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { AlertTriangle, ExternalLink, GraduationCap } from 'lucide-react'
import { fetchDegreeProgress, type DegreeProgress } from '../../lib/advising-api'
import { formatDateTime } from '../../lib/format'

export function DegreeProgressCard() {
  const [progress, setProgress] = useState<DegreeProgress | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(false)

  useEffect(() => {
    let cancelled = false
    void fetchDegreeProgress()
      .then((p) => {
        if (!cancelled) setProgress(p)
      })
      .catch(() => {
        if (!cancelled) setError(true)
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [])

  if (loading) {
    return (
      <section aria-label="Degree progress" aria-busy="true">
        <h2 className="text-sm font-semibold uppercase tracking-wide text-slate-500 dark:text-neutral-400">
          Degree progress
        </h2>
        <div className="mt-3 h-36 animate-pulse rounded-2xl bg-slate-100 dark:bg-neutral-800" />
      </section>
    )
  }

  if (error || !progress) {
    return null
  }

  const hasAudit = progress.configured && progress.completionPercent != null
  const apptUrl = progress.appointmentUrl?.trim()
  const notesCount = progress.recentNotesCount ?? 0

  return (
    <section aria-label="Degree progress and advising">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <h2 className="text-sm font-semibold uppercase tracking-wide text-slate-500 dark:text-neutral-400">
          Degree progress
        </h2>
        {notesCount > 0 ? (
          <Link
            to="/advising-notes"
            className="inline-flex items-center gap-1 rounded-full bg-indigo-600 px-2.5 py-1 text-xs font-semibold text-white"
          >
            {notesCount} advising {notesCount === 1 ? 'note' : 'notes'}
          </Link>
        ) : null}
      </div>

      {progress.atRisk ? (
        <div
          role="alert"
          className="mt-3 flex items-start gap-2 rounded-xl border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-900 dark:border-amber-900/40 dark:bg-amber-950/30 dark:text-amber-100"
        >
          <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0" aria-hidden />
          <p>Your advisor has flagged you as at-risk. Schedule a meeting to review your degree plan.</p>
        </div>
      ) : null}

      <div className="mt-3 rounded-2xl border border-slate-200 bg-white p-5 shadow-sm dark:border-neutral-700 dark:bg-neutral-900">
        {hasAudit ? (
          <div className="flex flex-col gap-4 sm:flex-row sm:items-center">
            <div
              className="relative mx-auto flex h-24 w-24 shrink-0 items-center justify-center sm:mx-0"
              role="progressbar"
              aria-valuemin={0}
              aria-valuemax={100}
              aria-valuenow={progress.completionPercent ?? 0}
              aria-label={`Degree completion ${progress.completionPercent}%`}
            >
              <svg className="h-24 w-24 -rotate-90" viewBox="0 0 36 36" aria-hidden>
                <circle
                  cx="18"
                  cy="18"
                  r="15.5"
                  fill="none"
                  className="stroke-slate-200 dark:stroke-neutral-700"
                  strokeWidth="3"
                />
                <circle
                  cx="18"
                  cy="18"
                  r="15.5"
                  fill="none"
                  className="stroke-indigo-600 dark:stroke-indigo-400"
                  strokeWidth="3"
                  strokeDasharray={`${progress.completionPercent} 100`}
                  strokeLinecap="round"
                />
              </svg>
              <span className="absolute text-lg font-bold text-slate-900 dark:text-neutral-50">
                {progress.completionPercent}%
              </span>
            </div>
            <div className="flex-1 text-sm">
              <p className="font-medium text-slate-900 dark:text-neutral-50">
                {progress.remainingRequiredCount ?? 0} required courses remaining
              </p>
              {progress.remainingRequirements && progress.remainingRequirements.length > 0 ? (
                <ul className="mt-2 space-y-1 text-slate-600 dark:text-neutral-400">
                  {progress.remainingRequirements.slice(0, 3).map((req) => (
                    <li key={req.group}>
                      {req.group}
                      {req.coursesRemaining > 0 ? ` (${req.coursesRemaining} left)` : ''}
                    </li>
                  ))}
                </ul>
              ) : null}
              {progress.stale && progress.lastUpdated ? (
                <p className="mt-2 text-xs text-amber-700 dark:text-amber-300">
                  Data last updated {formatDateTime(progress.lastUpdated)}. Check your SIS for the latest audit.
                </p>
              ) : progress.lastUpdated ? (
                <p className="mt-2 text-xs text-slate-500 dark:text-neutral-500">
                  Updated {formatDateTime(progress.lastUpdated)}
                </p>
              ) : null}
            </div>
          </div>
        ) : (
          <div className="flex items-start gap-3 text-sm text-slate-600 dark:text-neutral-400">
            <GraduationCap className="h-5 w-5 shrink-0 text-indigo-500" aria-hidden />
            <p>
              Degree audit is not configured. View your enrolled courses below, or contact your advisor for degree
              planning help.
            </p>
          </div>
        )}

        {apptUrl ? (
          <a
            href={apptUrl}
            target="_blank"
            rel="noopener noreferrer"
            className="mt-4 inline-flex items-center gap-2 rounded-xl bg-indigo-600 px-4 py-2.5 text-sm font-semibold text-white shadow-sm transition hover:bg-indigo-500"
          >
            Schedule advising appointment
            <ExternalLink className="h-4 w-4" aria-hidden />
          </a>
        ) : null}
      </div>
    </section>
  )
}
