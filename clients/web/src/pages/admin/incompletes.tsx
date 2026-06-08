import { useCallback, useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { usePlatformFeatures } from '../../context/platform-features-context'
import { fetchAdminIncompletes, type AdminIncompleteRow } from '../../lib/incomplete-grades-api'

export default function IncompletesAdminPage() {
  const { ffIncompleteGradeWorkflow } = usePlatformFeatures()
  const [rows, setRows] = useState<AdminIncompleteRow[] | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [termId, setTermId] = useState('')

  const load = useCallback(() => {
    setLoading(true)
    setError(null)
    void fetchAdminIncompletes({
      status: 'open',
      termId: termId.trim() || undefined,
    })
      .then(setRows)
      .catch((e: unknown) => {
        setError(e instanceof Error ? e.message : 'Failed to load incompletes.')
        setRows(null)
      })
      .finally(() => setLoading(false))
  }, [termId])

  useEffect(() => {
    if (ffIncompleteGradeWorkflow) load()
  }, [ffIncompleteGradeWorkflow, load])

  if (!ffIncompleteGradeWorkflow) {
    return (
      <div className="p-6 max-w-4xl mx-auto">
        <h1 className="text-xl font-semibold mb-2">Incomplete grades</h1>
        <p className="text-sm text-slate-600">Incomplete grade workflow is not enabled for this platform.</p>
      </div>
    )
  }

  return (
    <div className="p-6 max-w-5xl mx-auto">
      <h1 className="text-xl font-semibold mb-1">Open Incomplete grades</h1>
      <p className="text-sm text-slate-600 dark:text-neutral-400 mb-4">
        Registrar report of outstanding Incomplete grades, sorted by nearest deadline.
      </p>

      <div className="flex flex-wrap gap-3 mb-4 items-end">
        <div>
          <label htmlFor="term-filter" className="block text-sm font-medium mb-1">
            Term ID (optional)
          </label>
          <input
            id="term-filter"
            type="text"
            value={termId}
            onChange={(e) => setTermId(e.target.value)}
            placeholder="Filter by term UUID"
            className="border rounded-lg px-3 py-2 text-sm w-72 dark:border-neutral-600 dark:bg-neutral-800"
          />
        </div>
        <button
          type="button"
          onClick={load}
          disabled={loading}
          className="rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-60"
        >
          {loading ? 'Loading…' : 'Refresh'}
        </button>
      </div>

      {error ? (
        <div role="alert" className="mb-3 text-red-600 text-sm">
          {error}
        </div>
      ) : null}

      {rows && rows.length === 0 && !loading ? (
        <p className="text-sm text-slate-500">No open Incomplete grades.</p>
      ) : null}

      {rows && rows.length > 0 ? (
        <div className="overflow-x-auto rounded-xl border border-slate-200 dark:border-neutral-700">
          <table className="min-w-full text-sm">
            <thead className="bg-slate-50 dark:bg-neutral-800">
              <tr>
                <th className="px-4 py-3 text-start font-medium">Student</th>
                <th className="px-4 py-3 text-start font-medium">Course</th>
                <th className="px-4 py-3 text-start font-medium">Deadline</th>
                <th className="px-4 py-3 text-start font-medium">Outstanding</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((r) => (
                <tr key={r.id} className="border-t border-slate-100 dark:border-neutral-700">
                  <td className="px-4 py-3 font-medium">{r.studentName}</td>
                  <td className="px-4 py-3">
                    <Link
                      to={`/courses/${encodeURIComponent(r.courseCode)}/gradebook`}
                      className="text-indigo-700 hover:underline dark:text-indigo-300"
                    >
                      {r.courseTitle}
                    </Link>
                  </td>
                  <td className="px-4 py-3 tabular-nums">{r.extensionDeadline}</td>
                  <td className="px-4 py-3 text-slate-600 dark:text-neutral-400">
                    {r.outstandingTitles.length > 0 ? r.outstandingTitles.join(', ') : '—'}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : null}
    </div>
  )
}
