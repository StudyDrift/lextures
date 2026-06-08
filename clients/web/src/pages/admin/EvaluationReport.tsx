import { useCallback, useEffect, useState } from 'react'
import { usePlatformFeatures } from '../../context/platform-features-context'
import {
  fetchAdminEvaluationReport,
  type AdminReportRow,
} from '../../lib/course-evaluations-api'

function formatDate(iso: string) {
  return new Date(iso).toLocaleDateString(undefined, { dateStyle: 'short' })
}

function CompletionBadge({ pct }: { pct: number }) {
  const rounded = Math.round(pct)
  const color =
    rounded >= 75
      ? 'bg-emerald-100 text-emerald-800 dark:bg-emerald-900/40 dark:text-emerald-300'
      : rounded >= 40
        ? 'bg-amber-100 text-amber-800 dark:bg-amber-900/40 dark:text-amber-300'
        : 'bg-red-100 text-red-800 dark:bg-red-900/40 dark:text-red-300'
  return (
    <span className={`inline-flex rounded-full px-2 py-0.5 text-xs font-semibold ${color}`}>
      {rounded}%
    </span>
  )
}

export default function EvaluationReport() {
  const { ffCourseEvaluations, loading: featuresLoading } = usePlatformFeatures()
  const [rows, setRows] = useState<AdminReportRow[]>([])
  const [loading, setLoading] = useState(true)
  const [closedOnly, setClosedOnly] = useState(true)
  const [sortField, setSortField] = useState<keyof AdminReportRow>('completionPct')
  const [sortAsc, setSortAsc] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      setRows(await fetchAdminEvaluationReport(closedOnly))
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load report.')
    } finally {
      setLoading(false)
    }
  }, [closedOnly])

  useEffect(() => {
    if (!featuresLoading && ffCourseEvaluations) {
      void load()
    }
  }, [featuresLoading, ffCourseEvaluations, load])

  if (featuresLoading) return <p>Loading…</p>
  if (!ffCourseEvaluations) {
    return (
      <div role="alert">
        <p>Course evaluations are not enabled for this institution.</p>
      </div>
    )
  }

  const handleSort = (field: keyof AdminReportRow) => {
    if (sortField === field) {
      setSortAsc((a) => !a)
    } else {
      setSortField(field)
      setSortAsc(true)
    }
  }

  const sorted = [...rows].sort((a, b) => {
    const av = a[sortField]
    const bv = b[sortField]
    if (av == null && bv == null) return 0
    if (av == null) return 1
    if (bv == null) return -1
    if (typeof av === 'number' && typeof bv === 'number') {
      return sortAsc ? av - bv : bv - av
    }
    return sortAsc
      ? String(av).localeCompare(String(bv))
      : String(bv).localeCompare(String(av))
  })

  const SortHeader = ({
    field,
    children,
  }: {
    field: keyof AdminReportRow
    children: React.ReactNode
  }) => (
    <th
      className="cursor-pointer select-none px-4 py-2 text-left text-xs font-semibold text-slate-500 hover:text-slate-900 dark:text-neutral-400 dark:hover:text-neutral-100"
      onClick={() => handleSort(field)}
    >
      {children}
      {sortField === field && (
        <span className="ml-1">{sortAsc ? '↑' : '↓'}</span>
      )}
    </th>
  )

  return (
    <main className="mx-auto max-w-5xl px-4 py-8">
      <div className="mb-6 flex flex-wrap items-center justify-between gap-4">
        <h1 className="text-xl font-bold text-slate-900 dark:text-neutral-100">
          Evaluation Report
        </h1>
        <div className="flex items-center gap-4">
          <label className="flex items-center gap-2 text-sm text-slate-700 dark:text-neutral-300">
            <input
              type="checkbox"
              checked={closedOnly}
              onChange={(e) => setClosedOnly(e.target.checked)}
              className="accent-indigo-500"
            />
            Closed windows only
          </label>
          <button
            type="button"
            onClick={load}
            disabled={loading}
            className="rounded-lg bg-indigo-600 px-4 py-1.5 text-sm font-semibold text-white hover:bg-indigo-700 disabled:opacity-60 dark:bg-indigo-500"
          >
            {loading ? 'Loading…' : 'Refresh'}
          </button>
        </div>
      </div>

      {error && (
        <p className="mb-4 text-sm text-red-600 dark:text-red-400">{error}</p>
      )}

      {!loading && rows.length === 0 ? (
        <p className="py-16 text-center text-slate-500 dark:text-neutral-400">
          No evaluation windows found.
        </p>
      ) : (
        <div className="overflow-x-auto rounded-xl border border-slate-200 dark:border-neutral-700">
          <table className="min-w-full divide-y divide-slate-200 dark:divide-neutral-700">
            <thead className="bg-slate-50 dark:bg-neutral-800/50">
              <tr>
                <SortHeader field="courseTitle">Course</SortHeader>
                <SortHeader field="closesAt">Window</SortHeader>
                <SortHeader field="responseCount">Responses</SortHeader>
                <SortHeader field="enrolledCount">Enrolled</SortHeader>
                <SortHeader field="completionPct">Completion</SortHeader>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100 bg-white dark:divide-neutral-800 dark:bg-neutral-900">
              {sorted.map((row) => (
                <tr key={row.windowId} className="hover:bg-slate-50 dark:hover:bg-neutral-800/40">
                  <td className="px-4 py-3">
                    <p className="text-sm font-medium text-slate-900 dark:text-neutral-100">
                      {row.courseTitle}
                    </p>
                    <p className="text-xs text-slate-500 dark:text-neutral-400">{row.courseCode}</p>
                  </td>
                  <td className="px-4 py-3 text-xs text-slate-600 dark:text-neutral-400">
                    {formatDate(row.opensAt)} – {formatDate(row.closesAt)}
                  </td>
                  <td className="px-4 py-3 text-sm text-slate-700 dark:text-neutral-300">
                    {row.responseCount}
                  </td>
                  <td className="px-4 py-3 text-sm text-slate-700 dark:text-neutral-300">
                    {row.enrolledCount}
                  </td>
                  <td className="px-4 py-3">
                    <CompletionBadge pct={row.completionPct} />
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </main>
  )
}
