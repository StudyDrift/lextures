import { useCallback, useEffect, useId, useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import {
  fetchAdminCourses,
  patchAdminCourseStatus,
  type AdminCourse,
  type Paginated,
} from '../../lib/admin-console-api'

const STATUSES = ['', 'active', 'archived', 'draft']

export default function AdminCourses() {
  const titleId = useId()
  const [searchParams] = useSearchParams()
  const orgId = searchParams.get('orgId')
  const [q, setQ] = useState('')
  const [status, setStatus] = useState('')
  const [page, setPage] = useState(1)
  const [perPage] = useState(25)
  const [data, setData] = useState<Paginated<AdminCourse> | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      setData(
        await fetchAdminCourses({
          orgId,
          q: q.trim() || undefined,
          status: status || undefined,
          page,
          perPage,
        }),
      )
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load courses.')
    } finally {
      setLoading(false)
    }
  }, [orgId, q, status, page, perPage])

  useEffect(() => {
    void load()
  }, [load])

  async function setCourseStatus(course: AdminCourse, next: 'active' | 'archived' | 'draft') {
    if (!window.confirm(`Set "${course.title}" to ${next}?`)) return
    try {
      await patchAdminCourseStatus(course.id, next)
      await load()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to update course.')
    }
  }

  return (
    <div>
      <h1 id={titleId} className="text-xl font-semibold text-slate-900 dark:text-slate-100">
        Courses
      </h1>

      <div className="mt-4 flex flex-wrap gap-3">
        <label className="flex flex-col text-sm">
          <span className="mb-1 text-slate-600 dark:text-slate-400">Search</span>
          <input
            type="search"
            value={q}
            onChange={(e) => {
              setQ(e.target.value)
              setPage(1)
            }}
            placeholder="Title, code, or instructor"
            className="rounded-lg border border-slate-300 px-3 py-2 dark:border-neutral-700 dark:bg-neutral-900"
          />
        </label>
        <label className="flex flex-col text-sm">
          <span className="mb-1 text-slate-600 dark:text-slate-400">Status</span>
          <select
            value={status}
            onChange={(e) => {
              setStatus(e.target.value)
              setPage(1)
            }}
            className="rounded-lg border border-slate-300 px-3 py-2 dark:border-neutral-700 dark:bg-neutral-900"
          >
            {STATUSES.map((s) => (
              <option key={s || 'all'} value={s}>
                {s || 'All statuses'}
              </option>
            ))}
          </select>
        </label>
      </div>

      {error ? (
        <p role="alert" className="mt-4 text-sm text-red-600 dark:text-red-400">
          {error}
        </p>
      ) : null}

      <div className="mt-4 overflow-x-auto rounded-xl border border-slate-200 dark:border-neutral-800">
        <table className="min-w-full text-left text-sm">
          <caption className="sr-only">Organization courses</caption>
          <thead className="bg-slate-50 text-slate-600 dark:bg-neutral-950 dark:text-slate-400">
            <tr>
              <th scope="col" className="sticky left-0 bg-slate-50 px-4 py-2 font-medium dark:bg-neutral-950">
                Title
              </th>
              <th scope="col" className="px-4 py-2 font-medium">
                Code
              </th>
              <th scope="col" className="px-4 py-2 font-medium">
                Instructor
              </th>
              <th scope="col" className="px-4 py-2 font-medium">
                Term
              </th>
              <th scope="col" className="px-4 py-2 font-medium">
                Status
              </th>
              <th scope="col" className="px-4 py-2 font-medium">
                Enrollments
              </th>
              <th scope="col" className="px-4 py-2 font-medium">
                Actions
              </th>
            </tr>
          </thead>
          <tbody>
            {loading ? (
              <tr>
                <td colSpan={7} className="px-4 py-8 text-center text-slate-500">
                  Loading…
                </td>
              </tr>
            ) : !data?.items.length ? (
              <tr>
                <td colSpan={7} className="px-4 py-8 text-center text-slate-500">
                  No courses found.
                </td>
              </tr>
            ) : (
              data.items.map((course) => (
                <tr key={course.id} className="border-t border-slate-100 dark:border-neutral-800">
                  <td className="sticky left-0 bg-white px-4 py-2 dark:bg-neutral-900">
                    <Link
                      to={`/courses/${encodeURIComponent(course.courseCode)}`}
                      className="text-indigo-600 hover:underline dark:text-indigo-400"
                    >
                      {course.title}
                    </Link>
                  </td>
                  <td className="px-4 py-2 font-mono text-xs">{course.courseCode}</td>
                  <td className="px-4 py-2">{course.instructorName ?? '—'}</td>
                  <td className="px-4 py-2">{course.termName ?? '—'}</td>
                  <td className="px-4 py-2 capitalize">{course.status}</td>
                  <td className="px-4 py-2 tabular-nums">{course.enrollmentCount}</td>
                  <td className="px-4 py-2">
                    {course.status !== 'archived' ? (
                      <button
                        type="button"
                        onClick={() => void setCourseStatus(course, 'archived')}
                        className="text-sm text-slate-600 hover:underline dark:text-slate-400"
                      >
                        Archive
                      </button>
                    ) : (
                      <button
                        type="button"
                        onClick={() => void setCourseStatus(course, 'active')}
                        className="text-sm text-slate-600 hover:underline dark:text-slate-400"
                      >
                        Restore
                      </button>
                    )}
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>

      {data && data.totalPages > 1 ? (
        <nav aria-label="Course table pagination" className="mt-4 flex items-center gap-2">
          <button
            type="button"
            disabled={page <= 1}
            onClick={() => setPage((p) => p - 1)}
            className="rounded border px-3 py-1 text-sm disabled:opacity-50"
          >
            Previous
          </button>
          <span className="text-sm text-slate-600 dark:text-slate-400">
            Page {data.page} of {data.totalPages}
          </span>
          <button
            type="button"
            disabled={page >= data.totalPages}
            onClick={() => setPage((p) => p + 1)}
            className="rounded border px-3 py-1 text-sm disabled:opacity-50"
          >
            Next
          </button>
        </nav>
      ) : null}
    </div>
  )
}
