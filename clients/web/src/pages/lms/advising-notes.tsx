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
        <p className="text-sm text-fg-muted">Loading…</p>
      </LmsPage>
    )
  }

  if (!ffAdvisingIntegration) {
    return (
      <LmsPage title="Advising notes">
        <p className="text-sm text-fg-muted">
          Advising features are not enabled on this platform.
        </p>
      </LmsPage>
    )
  }

  return (
    <LmsPage title="Advising notes">
      <p className="text-sm text-fg-muted">
        Notes from your academic advisor about follow-up items and degree planning.
      </p>

      {error && (
        <p role="alert" className="mt-4 text-sm text-danger-fg">
          {error}
        </p>
      )}

      {loading ? (
        <div className="mt-6 space-y-4" aria-busy="true">
          {[1, 2].map((i) => (
            <div key={i} className="h-24 animate-pulse rounded-2xl bg-surface-sunken" />
          ))}
        </div>
      ) : notes.length === 0 ? (
        <div className="mt-8 rounded-2xl border border-dashed border-border-default px-6 py-12 text-center dark:border-border-default">
          <GraduationCap className="mx-auto h-10 w-10 text-slate-300 dark:text-neutral-600" aria-hidden />
          <p className="mt-3 text-sm text-fg-muted">No advising notes yet.</p>
        </div>
      ) : (
        <ol className="mt-6 space-y-4" aria-label="Advising notes timeline">
          {notes.map((note) => (
            <li
              key={note.id}
              className="rounded-2xl border border-border-default bg-surface-raised p-5 shadow-sm dark:border-border-default dark:bg-surface-raised"
            >
              <div className="flex flex-wrap items-center gap-2 text-xs text-fg-muted">
                <CalendarClock className="h-4 w-4" aria-hidden />
                <time dateTime={note.createdAt}>{formatDateTime(note.createdAt)}</time>
                <span aria-hidden>·</span>
                <span>
                  {note.advisorDisplayName?.trim() || note.advisorEmail || 'Your advisor'}
                </span>
              </div>
              <p className="mt-3 whitespace-pre-wrap text-sm text-fg-default">
                {note.content}
              </p>
            </li>
          ))}
        </ol>
      )}

      <p className="mt-8 text-xs text-fg-muted">
        Need to meet with your advisor?{' '}
        <Link to="/" className="font-medium text-accent-fg hover:text-indigo-500 dark:text-indigo-400">
          Return to your dashboard
        </Link>{' '}
        to schedule an appointment.
      </p>
    </LmsPage>
  )
}
