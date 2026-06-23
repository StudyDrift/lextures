import { useEffect, useId, useState } from 'react'
import { fetchMySeatTime, type SeatTimeProgress } from '../../lib/seat-time-api'

type Props = {
  courseId: string | undefined
  compact?: boolean
}

function formatHours(minutes: number): string {
  return (minutes / 60).toFixed(1)
}

export function SeatTimeProgressBar({ courseId, compact = false }: Props) {
  const labelId = useId()
  const [progress, setProgress] = useState<SeatTimeProgress | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!courseId) return
    let cancelled = false
    setLoading(true)
    setError(null)
    void fetchMySeatTime(courseId)
      .then((data) => {
        if (!cancelled) setProgress(data)
      })
      .catch((e: unknown) => {
        if (!cancelled) {
          setError(e instanceof Error ? e.message : 'Failed to load CE progress.')
        }
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [courseId])

  if (!courseId || loading) {
    return (
      <div
        className="animate-pulse rounded-lg border border-slate-200 bg-slate-50 px-3 py-2 dark:border-neutral-700 dark:bg-neutral-900/50"
        aria-hidden
      >
        <div className="h-3 w-32 rounded bg-slate-200 dark:bg-neutral-700" />
        <div className="mt-2 h-2 w-full rounded bg-slate-200 dark:bg-neutral-700" />
      </div>
    )
  }

  if (error || !progress || progress.requiredHours <= 0) {
    return null
  }

  const contactHours = formatHours(progress.totalMinutes)
  const required = progress.requiredHours.toFixed(1)
  const pct = Math.round(progress.progressPct)
  const earnedLabel = progress.ceuEarned.toFixed(2)

  if (compact) {
    return (
      <p className="text-xs text-slate-600 dark:text-neutral-400" aria-live="polite">
        CE: <span className="lex-num">{contactHours}</span>/<span className="lex-num">{required}</span>h
      </p>
    )
  }

  return (
    <section
      className="rounded-xl border border-teal-100 bg-teal-50/60 px-4 py-3 dark:border-teal-900/40 dark:bg-teal-950/20"
      aria-labelledby={labelId}
    >
      <p id={labelId} className="text-sm font-medium text-slate-900 dark:text-neutral-100">
        Continuing education progress
      </p>
      <p className="mt-1 text-xs text-slate-600 dark:text-neutral-400">
        Contact hours: <span className="lex-num">{contactHours}</span> / <span className="lex-num">{required}</span>
      </p>
      <div
        className="mt-2 h-2 overflow-hidden rounded-full bg-teal-100 dark:bg-teal-900/50"
        role="progressbar"
        aria-valuemin={0}
        aria-valuemax={100}
        aria-valuenow={pct}
        aria-label={`CE contact hours ${pct} percent complete`}
      >
        <div
          className="h-full rounded-full bg-teal-600 motion-safe:transition-[width] motion-safe:duration-300 dark:bg-teal-400"
          style={{ width: `${pct}%` }}
        />
      </div>
      <p className="mt-2 text-xs font-medium text-teal-800 dark:text-teal-200" aria-live="polite">
        {progress.awarded ? (
          <>
            CEU earned! <span className="lex-num">{earnedLabel}</span> CEU awarded.
          </>
        ) : (
          <>
            CEU credit: <span className="lex-num">{earnedLabel}</span> earned
          </>
        )}
      </p>
    </section>
  )
}
