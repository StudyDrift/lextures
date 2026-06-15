import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { CalendarClock, GraduationCap } from 'lucide-react'
import { LmsPage } from './lms-page'
import { usePlatformFeatures } from '../../context/platform-features-context'
import { fetchAdvisingNotes, type AdvisingNote } from '../../lib/advising-api'
import { formatDateTime } from '../../lib/format'

export default function AdvisingNotesPage() {
  const { ffAdvisingIntegration, loading: featuresLoading } = usePlatformFeatures()
  const [notes, setNotes] = useState<AdvisingNote[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (featuresLoading || !ffAdvisingIntegration) {
      setLoading(false)
      return
    }
    let cancelled = false
    void fetchAdvisingNotes()
      .then((list) => {
        if (!cancelled) setNotes(list)
      })
      .catch((e) => {
        if (!cancelled) setError(e instanceof Error ? e.message : 'Could not load notes.')
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [ffAdvisingIntegration, featuresLoading])

  if (featuresLoading) {
    return (
      <LmsPage title="Advising notes">
        <p className="text-sm text-slate-500">Loading…</p>
      </LmsPage>
    )
  }

  if (!ffAdvisingIntegration) {
    return (
      <LmsPage title="Advising notes">
        <p className="text-sm text-slate-600 dark:text-neutral-400">
          Advising features are not enabled on this platform.
        </p>
      </LmsPage>
    )
  }

  return (
    <LmsPage title="Advising notes">
      <p className="text-sm text-slate-600 dark:text-neutral-400">
        Notes from your academic advisor about follow-up items and degree planning.
      </p>

      {error && (
        <p role="alert" className="mt-4 text-sm text-red-600 dark:text-red-400">
          {error}
        </p>
      )}

      {loading ? (
        <div className="mt-6 space-y-4" aria-busy="true">
          {[1, 2].map((i) => (
            <div key={i} className="h-24 animate-pulse rounded-2xl bg-slate-100 dark:bg-neutral-800" />
          ))}
        </div>
      ) : notes.length === 0 ? (
        <div className="mt-8 rounded-2xl border border-dashed border-slate-200 px-6 py-12 text-center dark:border-neutral-700">
          <GraduationCap className="mx-auto h-10 w-10 text-slate-300 dark:text-neutral-600" aria-hidden />
          <p className="mt-3 text-sm text-slate-600 dark:text-neutral-400">No advising notes yet.</p>
        </div>
      ) : (
        <ol className="mt-6 space-y-4" aria-label="Advising notes timeline">
          {notes.map((note) => (
            <li
              key={note.id}
              className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm dark:border-neutral-700 dark:bg-neutral-900"
            >
              <div className="flex flex-wrap items-center gap-2 text-xs text-slate-500 dark:text-neutral-400">
                <CalendarClock className="h-4 w-4" aria-hidden />
                <time dateTime={note.createdAt}>{formatDateTime(note.createdAt)}</time>
                <span aria-hidden>·</span>
                <span>
                  {note.advisorDisplayName?.trim() || note.advisorEmail || 'Your advisor'}
                </span>
              </div>
              <p className="mt-3 whitespace-pre-wrap text-sm text-slate-800 dark:text-neutral-100">
                {note.content}
              </p>
            </li>
          ))}
        </ol>
      )}

      <p className="mt-8 text-xs text-slate-500 dark:text-neutral-400">
        Need to meet with your advisor?{' '}
        <Link to="/" className="font-medium text-indigo-600 hover:text-indigo-500 dark:text-indigo-400">
          Return to your dashboard
        </Link>{' '}
        to schedule an appointment.
      </p>
    </LmsPage>
  )
}
