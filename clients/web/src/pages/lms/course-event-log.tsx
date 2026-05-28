import { useCallback, useEffect, useState } from 'react'
import { formatDateTime } from '../../lib/format'
import { useParams } from 'react-router-dom'
import { LmsPage } from './lms-page'
import { fetchCourseXAPIEvents, type XAPIEventRow } from '../../lib/lrs-api'
import { usePlatformFeatures } from '../../context/platform-features-context'

function verbLabel(verb: string): string {
  const parts = verb.split('/')
  return parts[parts.length - 1] || verb
}

export default function CourseEventLogPage() {
  const { courseCode } = useParams<{ courseCode: string }>()
  const { xapiEmissionEnabled } = usePlatformFeatures()
  const [events, setEvents] = useState<XAPIEventRow[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [selected, setSelected] = useState<XAPIEventRow | null>(null)

  const load = useCallback(async () => {
    if (!courseCode) return
    setLoading(true)
    setError(null)
    try {
      setEvents(await fetchCourseXAPIEvents(courseCode))
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Could not load event log.')
      setEvents([])
    } finally {
      setLoading(false)
    }
  }, [courseCode])

  useEffect(() => {
    void load()
  }, [load])

  if (!xapiEmissionEnabled) {
    return (
      <LmsPage title="Event log" description="xAPI / Caliper learning events for this course.">
        <p className="mt-6 text-sm text-slate-600 dark:text-neutral-400">
          xAPI emission is not enabled for this platform.
        </p>
      </LmsPage>
    )
  }

  return (
    <LmsPage
      title="Event log"
      description="xAPI and Caliper statements recorded for the last 7 days."
    >
      {loading && <p className="mt-6 text-sm text-slate-500">Loading…</p>}
      {error && (
        <p className="mt-6 text-sm text-rose-700 dark:text-rose-300" role="alert">
          {error}
        </p>
      )}
      {!loading && !error && (
        <div className="mt-6 overflow-x-auto rounded-2xl border border-slate-200 bg-white shadow-sm dark:border-neutral-800 dark:bg-neutral-950">
          <table className="min-w-full text-left text-sm">
            <thead className="border-b border-slate-200 bg-slate-50 text-xs font-semibold uppercase tracking-wide text-slate-600 dark:border-neutral-700 dark:bg-neutral-800/80 dark:text-neutral-300">
              <tr>
                <th className="px-4 py-3">Timestamp</th>
                <th className="px-4 py-3">Verb</th>
                <th className="px-4 py-3">Object</th>
                <th className="px-4 py-3">Result</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100 dark:divide-neutral-800">
              {events.length === 0 ? (
                <tr>
                  <td colSpan={4} className="px-4 py-6 text-slate-500">
                    No events in the last 7 days.
                  </td>
                </tr>
              ) : (
                events.map((ev) => (
                  <tr
                    key={ev.statementId + ev.storedAt}
                    className="cursor-pointer hover:bg-slate-50/80 dark:hover:bg-neutral-800/80"
                    onClick={() => setSelected(ev)}
                  >
                    <td className="px-4 py-3 tabular-nums text-slate-600 dark:text-neutral-400">
                      {formatDateTime(ev.storedAt)}
                    </td>
                    <td className="px-4 py-3 font-medium text-slate-900 dark:text-neutral-100">
                      {verbLabel(ev.verb)}
                    </td>
                    <td className="px-4 py-3 text-slate-700 dark:text-neutral-300">
                      {ev.objectTitle ?? ev.objectId}
                    </td>
                    <td className="px-4 py-3 text-slate-500">—</td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      )}

      {selected && (
        <div
          role="dialog"
          aria-modal="true"
          aria-label="Statement JSON"
          className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4"
          onClick={() => setSelected(null)}
        >
          <div
            className="max-h-[80vh] w-full max-w-2xl overflow-auto rounded-2xl bg-white p-4 shadow-xl dark:bg-neutral-900"
            onClick={(e) => e.stopPropagation()}
          >
            <h3 className="text-sm font-semibold text-slate-900 dark:text-neutral-100">Statement JSON</h3>
            <pre className="mt-3 whitespace-pre-wrap break-all text-xs text-slate-700 dark:text-neutral-300">
              {JSON.stringify(selected.fullJson, null, 2)}
            </pre>
            <button
              type="button"
              className="mt-4 rounded-lg border border-slate-200 px-3 py-2 text-sm font-semibold dark:border-neutral-600"
              onClick={() => setSelected(null)}
            >
              Close
            </button>
          </div>
        </div>
      )}
    </LmsPage>
  )
}
