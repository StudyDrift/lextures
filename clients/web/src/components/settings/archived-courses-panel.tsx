import { useCallback, useEffect, useState } from 'react'
import { Archive, RefreshCw, RotateCcw, Trash2 } from 'lucide-react'
import {
  deleteArchivedCoursePermanently,
  fetchArchivedCourses,
  restoreArchivedCourse,
  type ArchivedCourseRow,
} from '../../lib/archived-courses-api'
import { formatDateTime } from '../../lib/format'
import { toastMutationError, toastSaveOk } from '../../lib/lms-toast'

function archivedByLabel(row: ArchivedCourseRow): string {
  const name = row.archivedByName?.trim()
  if (name) return name
  const email = row.archivedByEmail?.trim()
  if (email) return email
  return '—'
}

export function ArchivedCoursesPanel() {
  const [courses, setCourses] = useState<ArchivedCourseRow[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [busyCode, setBusyCode] = useState<string | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<ArchivedCourseRow | null>(null)

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const rows = await fetchArchivedCourses()
      setCourses(rows)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load archived courses.')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    void load()
  }, [load])

  async function onRestore(row: ArchivedCourseRow) {
    setBusyCode(row.courseCode)
    try {
      await restoreArchivedCourse(row.courseCode)
      toastSaveOk(`Restored ${row.title || row.courseCode}.`)
      await load()
    } catch (e) {
      toastMutationError(e instanceof Error ? e.message : 'Could not restore course.')
    } finally {
      setBusyCode(null)
    }
  }

  async function confirmDelete() {
    if (!deleteTarget) return
    const row = deleteTarget
    setBusyCode(row.courseCode)
    try {
      await deleteArchivedCoursePermanently(row.courseCode)
      toastSaveOk(`Permanently deleted ${row.title || row.courseCode}.`)
      setDeleteTarget(null)
      await load()
    } catch (e) {
      toastMutationError(e instanceof Error ? e.message : 'Could not delete course.')
    } finally {
      setBusyCode(null)
    }
  }

  return (
    <div className="mt-6 space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <p className="text-sm text-slate-600 dark:text-neutral-400">
          Archived courses are hidden from catalogs and search. Restore a course to bring it back, or
          permanently delete it to remove all content, files, submissions, and enrollments.
        </p>
        <button
          type="button"
          onClick={() => void load()}
          disabled={loading}
          className="inline-flex items-center gap-2 rounded-xl border border-slate-200 bg-white px-3 py-2 text-sm font-medium text-slate-800 shadow-sm transition-[background-color,color,border-color] hover:bg-slate-50 disabled:opacity-50 dark:border-neutral-600 dark:bg-neutral-900 dark:text-neutral-100 dark:hover:bg-neutral-800"
        >
          <RefreshCw className={`h-4 w-4 ${loading ? 'animate-spin' : ''}`} aria-hidden />
          Refresh
        </button>
      </div>

      {error && (
        <p
          className="rounded-lg border border-rose-200 bg-rose-50 px-3 py-2 text-sm text-rose-800 dark:border-rose-900/50 dark:bg-rose-950/40 dark:text-rose-100"
          role="alert"
        >
          {error}
        </p>
      )}

      {loading && courses.length === 0 ? (
        <div className="space-y-2" aria-busy="true" aria-label="Loading archived courses">
          {[0, 1, 2].map((i) => (
            <div key={i} className="h-12 motion-safe:animate-pulse rounded-xl bg-slate-100 dark:bg-neutral-800" />
          ))}
        </div>
      ) : courses.length === 0 ? (
        <p className="rounded-xl border border-dashed border-slate-200 px-4 py-8 text-center text-sm text-slate-600 dark:border-neutral-600 dark:text-neutral-400">
          <Archive className="mx-auto mb-2 h-8 w-8 text-slate-300 dark:text-neutral-600" aria-hidden />
          No archived courses in this organization.
        </p>
      ) : (
        <div className="overflow-x-auto rounded-xl border border-slate-200 dark:border-neutral-600">
          <table
            className="min-w-full divide-y divide-slate-200 text-start text-sm dark:divide-neutral-600"
            aria-label="Archived courses"
          >
            <thead className="bg-slate-50 dark:bg-neutral-800/80">
              <tr>
                <th scope="col" className="px-3 py-2 font-medium text-slate-700 dark:text-neutral-200">
                  Name
                </th>
                <th scope="col" className="px-3 py-2 font-medium text-slate-700 dark:text-neutral-200">
                  Course code
                </th>
                <th scope="col" className="px-3 py-2 font-medium text-slate-700 dark:text-neutral-200">
                  Archived by
                </th>
                <th scope="col" className="px-3 py-2 font-medium text-slate-700 dark:text-neutral-200">
                  Archived
                </th>
                <th scope="col" className="px-3 py-2 font-medium text-slate-700 dark:text-neutral-200">
                  Actions
                </th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-200 bg-white dark:divide-neutral-600 dark:bg-neutral-900">
              {courses.map((row) => {
                const busy = busyCode === row.courseCode
                return (
                  <tr key={row.id} className="hover:bg-slate-50 dark:hover:bg-neutral-800/60">
                    <th
                      scope="row"
                      className="max-w-[16rem] truncate px-3 py-2.5 font-normal text-slate-900 dark:text-neutral-100"
                    >
                      {row.title || '—'}
                    </th>
                    <td className="whitespace-nowrap px-3 py-2.5 font-mono text-xs text-slate-600 dark:text-neutral-300">
                      {row.courseCode}
                    </td>
                    <td className="whitespace-nowrap px-3 py-2.5 text-slate-600 dark:text-neutral-300">
                      {archivedByLabel(row)}
                    </td>
                    <td className="whitespace-nowrap px-3 py-2.5 text-slate-600 dark:text-neutral-300">
                      {row.archivedAt ? formatDateTime(row.archivedAt) : '—'}
                    </td>
                    <td className="px-3 py-2.5">
                      <div className="flex flex-wrap gap-2">
                        <button
                          type="button"
                          onClick={() => void onRestore(row)}
                          disabled={busy}
                          className="inline-flex items-center gap-1.5 rounded-lg border border-slate-200 bg-white px-2.5 py-1.5 text-xs font-medium text-slate-800 hover:border-indigo-200 hover:bg-indigo-50 disabled:cursor-not-allowed disabled:opacity-50 dark:border-neutral-600 dark:bg-neutral-900 dark:text-neutral-100 dark:hover:border-indigo-400 dark:hover:bg-indigo-950/40"
                        >
                          <RotateCcw className="h-3.5 w-3.5" aria-hidden />
                          {busy && !deleteTarget ? 'Restoring…' : 'Restore'}
                        </button>
                        <button
                          type="button"
                          onClick={() => setDeleteTarget(row)}
                          disabled={busy}
                          className="inline-flex items-center gap-1.5 rounded-lg border border-rose-200 bg-white px-2.5 py-1.5 text-xs font-medium text-rose-700 hover:bg-rose-50 disabled:cursor-not-allowed disabled:opacity-50 dark:border-rose-900/60 dark:bg-neutral-900 dark:text-rose-300 dark:hover:bg-rose-950/40"
                        >
                          <Trash2 className="h-3.5 w-3.5" aria-hidden />
                          Delete permanently
                        </button>
                      </div>
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        </div>
      )}

      {deleteTarget ? (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/40 p-4 dark:bg-black/50"
          role="presentation"
          onMouseDown={(e) => {
            if (e.target === e.currentTarget && busyCode !== deleteTarget.courseCode) {
              setDeleteTarget(null)
            }
          }}
        >
          <div
            role="dialog"
            aria-modal="true"
            aria-labelledby="delete-archived-course-title"
            className="w-full max-w-md overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-xl dark:border-neutral-700 dark:bg-neutral-900"
          >
            <div className="border-b border-slate-200 px-4 py-3 dark:border-neutral-700">
              <h3
                id="delete-archived-course-title"
                className="text-sm font-semibold text-slate-900 dark:text-neutral-100"
              >
                Permanently delete course
              </h3>
            </div>
            <div className="space-y-3 p-4 text-sm text-slate-600 dark:text-neutral-300">
              <p>
                Delete <span className="font-medium text-slate-900 dark:text-neutral-100">{deleteTarget.title}</span>{' '}
                (<code className="font-mono text-xs">{deleteTarget.courseCode}</code>) permanently?
              </p>
              <p>
                This removes all modules, assignments, quizzes, submissions, grades, enrollments, uploaded
                files, and related data. This cannot be undone.
              </p>
            </div>
            <div className="flex flex-wrap justify-end gap-2 border-t border-slate-200 px-4 py-3 dark:border-neutral-700">
              <button
                type="button"
                onClick={() => setDeleteTarget(null)}
                disabled={busyCode === deleteTarget.courseCode}
                className="rounded-xl border border-slate-200 bg-white px-4 py-2 text-sm font-medium text-slate-700 shadow-sm hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-50 dark:border-neutral-600 dark:bg-neutral-800 dark:text-neutral-200"
              >
                Cancel
              </button>
              <button
                type="button"
                onClick={() => void confirmDelete()}
                disabled={busyCode === deleteTarget.courseCode}
                className="rounded-xl bg-rose-600 px-4 py-2 text-sm font-semibold text-white shadow-sm hover:bg-rose-500 disabled:cursor-not-allowed disabled:opacity-60"
              >
                {busyCode === deleteTarget.courseCode ? 'Deleting…' : 'Delete permanently'}
              </button>
            </div>
          </div>
        </div>
      ) : null}
    </div>
  )
}