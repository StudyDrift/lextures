import { type FormEvent, useCallback, useEffect, useState } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import {
  ArrowLeft,
  ExternalLink,
  Loader2,
  Search,
  Settings,
  Users,
} from 'lucide-react'
import { formatDateTime } from '../../lib/format'
import { toastMutationError } from '../../lib/lms-toast'
import {
  ensurePlatformCourseAdminAccess,
  fetchPlatformCourseReport,
  searchPlatformCourses,
  type PlatformCourseReport,
  type PlatformCourseSearchStatus,
  type PaginatedPlatformCourses,
} from '../../lib/platform-courses-api'

const PAGE_SIZES = [25, 50, 100]

const STATUS_OPTIONS: { value: PlatformCourseSearchStatus; label: string }[] = [
  { value: 'open', label: 'Open courses' },
  { value: 'active', label: 'Active' },
  { value: 'draft', label: 'Draft' },
  { value: 'archived', label: 'Archived' },
  { value: 'all', label: 'All statuses' },
]

function statusLabel(status: string): string {
  switch (status) {
    case 'active':
      return 'Active'
    case 'archived':
      return 'Archived'
    case 'draft':
      return 'Draft'
    default:
      return status
  }
}

function CourseReportView({
  report,
  loading,
  error,
  busy,
  onBack,
  onOpen,
}: {
  report: PlatformCourseReport | null
  loading: boolean
  error: string | null
  busy: boolean
  onBack: () => void
  onOpen: (path: string) => void
}) {
  if (loading) {
    return <p className="mt-6 text-sm text-slate-500 dark:text-neutral-400">Loading course…</p>
  }
  if (error) {
    return (
      <p role="alert" className="mt-6 text-sm text-red-600 dark:text-red-400">
        {error}
      </p>
    )
  }
  if (!report) return null

  const code = encodeURIComponent(report.courseCode)

  return (
    <div className="mt-6 space-y-6">
      <button
        type="button"
        onClick={onBack}
        className="inline-flex items-center gap-2 text-sm font-medium text-slate-600 hover:text-slate-900 dark:text-neutral-400 dark:hover:text-neutral-100"
      >
        <ArrowLeft className="h-4 w-4" aria-hidden />
        Back to search
      </button>

      <div className="rounded-xl border border-slate-200 bg-white p-5 dark:border-neutral-800 dark:bg-neutral-900">
        <div className="flex flex-wrap items-start justify-between gap-4">
          <div>
            <h3 className="text-lg font-semibold text-slate-900 dark:text-neutral-100">{report.title}</h3>
            <p className="mt-1 font-mono text-sm text-slate-600 dark:text-neutral-400">{report.courseCode}</p>
            {report.description ? (
              <p className="mt-3 text-sm text-slate-600 dark:text-neutral-400">{report.description}</p>
            ) : null}
            <dl className="mt-4 grid gap-2 text-sm sm:grid-cols-2">
              <div>
                <dt className="text-slate-500 dark:text-neutral-500">Organization</dt>
                <dd className="text-slate-900 dark:text-neutral-100">{report.orgName}</dd>
              </div>
              <div>
                <dt className="text-slate-500 dark:text-neutral-500">Status</dt>
                <dd className="text-slate-900 dark:text-neutral-100">{statusLabel(report.status)}</dd>
              </div>
              <div>
                <dt className="text-slate-500 dark:text-neutral-500">Instructor</dt>
                <dd className="text-slate-900 dark:text-neutral-100">{report.instructorName ?? '—'}</dd>
              </div>
              <div>
                <dt className="text-slate-500 dark:text-neutral-500">Term</dt>
                <dd className="text-slate-900 dark:text-neutral-100">{report.termName ?? '—'}</dd>
              </div>
              <div>
                <dt className="text-slate-500 dark:text-neutral-500">Enrollments</dt>
                <dd className="text-slate-900 dark:text-neutral-100">{report.enrollmentCount}</dd>
              </div>
              <div>
                <dt className="text-slate-500 dark:text-neutral-500">Last updated</dt>
                <dd className="text-slate-900 dark:text-neutral-100">{formatDateTime(report.updatedAt)}</dd>
              </div>
            </dl>
          </div>
          <div className="flex flex-wrap gap-2">
            <button
              type="button"
              disabled={busy}
              onClick={() => onOpen(`/courses/${code}`)}
              className="inline-flex items-center gap-2 rounded-lg bg-indigo-600 px-3 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50 dark:bg-neutral-100 dark:text-neutral-950 dark:hover:bg-white"
            >
              {busy ? <Loader2 className="h-4 w-4 animate-spin" aria-hidden /> : <ExternalLink className="h-4 w-4" aria-hidden />}
              Open course
            </button>
            <button
              type="button"
              disabled={busy}
              onClick={() => onOpen(`/courses/${code}/settings`)}
              className="inline-flex items-center gap-2 rounded-lg border border-slate-300 px-3 py-2 text-sm font-medium text-slate-700 hover:bg-slate-50 disabled:opacity-50 dark:border-neutral-700 dark:text-neutral-200 dark:hover:bg-neutral-800"
            >
              <Settings className="h-4 w-4" aria-hidden />
              Settings
            </button>
            <button
              type="button"
              disabled={busy}
              onClick={() => onOpen(`/courses/${code}/enrollments`)}
              className="inline-flex items-center gap-2 rounded-lg border border-slate-300 px-3 py-2 text-sm font-medium text-slate-700 hover:bg-slate-50 disabled:opacity-50 dark:border-neutral-700 dark:text-neutral-200 dark:hover:bg-neutral-800"
            >
              <Users className="h-4 w-4" aria-hidden />
              Enrollments
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}

export function CoursesPanel() {
  const navigate = useNavigate()
  const [searchParams, setSearchParams] = useSearchParams()
  const selectedCourseId = searchParams.get('courseId')

  const [q, setQ] = useState('')
  const [submittedQ, setSubmittedQ] = useState('')
  const [status, setStatus] = useState<PlatformCourseSearchStatus>('open')
  const [page, setPage] = useState(1)
  const [perPage, setPerPage] = useState(25)
  const [data, setData] = useState<PaginatedPlatformCourses | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [accessBusy, setAccessBusy] = useState(false)

  const [report, setReport] = useState<PlatformCourseReport | null>(null)
  const [reportLoading, setReportLoading] = useState(false)
  const [reportError, setReportError] = useState<string | null>(null)

  const loadSearch = useCallback(async () => {
    const query = submittedQ.trim()
    if (!query) {
      setData(null)
      return
    }
    setLoading(true)
    setError(null)
    try {
      setData(await searchPlatformCourses({ q: query, status, page, perPage }))
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Search failed.')
      setData(null)
    } finally {
      setLoading(false)
    }
  }, [submittedQ, status, page, perPage])

  useEffect(() => {
    void loadSearch()
  }, [loadSearch])

  const loadReport = useCallback(async (courseId: string) => {
    setReportLoading(true)
    setReportError(null)
    try {
      setReport(await fetchPlatformCourseReport(courseId))
    } catch (e) {
      setReportError(e instanceof Error ? e.message : 'Failed to load course.')
      setReport(null)
    } finally {
      setReportLoading(false)
    }
  }, [])

  useEffect(() => {
    if (!selectedCourseId) {
      setReport(null)
      setReportError(null)
      return
    }
    void loadReport(selectedCourseId)
  }, [selectedCourseId, loadReport])

  function onSearchSubmit(e: FormEvent) {
    e.preventDefault()
    setSubmittedQ(q.trim())
    setPage(1)
  }

  function openCourse(courseId: string) {
    const next = new URLSearchParams(searchParams)
    next.set('courseId', courseId)
    setSearchParams(next, { replace: false })
  }

  function closeCourse() {
    const next = new URLSearchParams(searchParams)
    next.delete('courseId')
    setSearchParams(next, { replace: false })
  }

  async function openWithAccess(path: string) {
    if (!selectedCourseId) return
    setAccessBusy(true)
    try {
      await ensurePlatformCourseAdminAccess(selectedCourseId)
      navigate(path)
    } catch (e) {
      toastMutationError(e instanceof Error ? e.message : 'Could not grant course access.')
    } finally {
      setAccessBusy(false)
    }
  }

  if (selectedCourseId) {
    return (
      <CourseReportView
        report={report}
        loading={reportLoading}
        error={reportError}
        busy={accessBusy}
        onBack={closeCourse}
        onOpen={(path) => void openWithAccess(path)}
      />
    )
  }

  return (
    <div className="mt-6 space-y-6">
      <p className="text-sm text-slate-600 dark:text-neutral-400">
        Search courses by course code or title. Results are not shown until you search.
      </p>

      <form onSubmit={onSearchSubmit} className="flex flex-wrap items-end gap-3">
        <label className="flex min-w-[16rem] flex-1 flex-col text-sm">
          <span className="mb-1 text-slate-600 dark:text-neutral-400">Search</span>
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-400" aria-hidden />
            <input
              type="search"
              value={q}
              onChange={(e) => setQ(e.target.value)}
              placeholder="Course code or title"
              className="w-full rounded-lg border border-slate-300 py-2 pl-9 pr-3 dark:border-neutral-700 dark:bg-neutral-900"
            />
          </div>
        </label>
        <label className="flex flex-col text-sm">
          <span className="mb-1 text-slate-600 dark:text-neutral-400">Status</span>
          <select
            value={status}
            onChange={(e) => {
              setStatus(e.target.value as PlatformCourseSearchStatus)
              setPage(1)
            }}
            className="rounded-lg border border-slate-300 px-3 py-2 dark:border-neutral-700 dark:bg-neutral-900"
          >
            {STATUS_OPTIONS.map((opt) => (
              <option key={opt.value} value={opt.value}>
                {opt.label}
              </option>
            ))}
          </select>
        </label>
        <label className="flex flex-col text-sm">
          <span className="mb-1 text-slate-600 dark:text-neutral-400">Page size</span>
          <select
            value={perPage}
            onChange={(e) => {
              setPerPage(Number(e.target.value))
              setPage(1)
            }}
            className="rounded-lg border border-slate-300 px-3 py-2 dark:border-neutral-700 dark:bg-neutral-900"
          >
            {PAGE_SIZES.map((n) => (
              <option key={n} value={n}>
                {n}
              </option>
            ))}
          </select>
        </label>
        <button
          type="submit"
          disabled={!q.trim() || loading}
          className="rounded-lg border border-slate-300 px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-50 disabled:opacity-50 dark:border-neutral-700 dark:text-neutral-200 dark:hover:bg-neutral-800"
        >
          {loading ? 'Searching…' : 'Search'}
        </button>
      </form>

      {error ? (
        <p role="alert" className="text-sm text-red-600 dark:text-red-400">
          {error}
        </p>
      ) : null}

      {!submittedQ.trim() ? (
        <p className="rounded-xl border border-dashed border-slate-200 px-4 py-8 text-center text-sm text-slate-500 dark:border-neutral-700 dark:text-neutral-400">
          Enter a course code or title above to find courses.
        </p>
      ) : (
        <div className="overflow-x-auto rounded-xl border border-slate-200 dark:border-neutral-800">
          <table className="min-w-full text-left text-sm">
            <thead className="bg-slate-50 text-slate-600 dark:bg-neutral-950 dark:text-neutral-400">
              <tr>
                <th scope="col" className="px-4 py-2 font-medium">Title</th>
                <th scope="col" className="px-4 py-2 font-medium">Code</th>
                <th scope="col" className="px-4 py-2 font-medium">Organization</th>
                <th scope="col" className="px-4 py-2 font-medium">Instructor</th>
                <th scope="col" className="px-4 py-2 font-medium">Status</th>
                <th scope="col" className="px-4 py-2 font-medium">Enrollments</th>
              </tr>
            </thead>
            <tbody>
              {loading ? (
                <tr>
                  <td colSpan={6} className="px-4 py-8 text-center text-slate-500">
                    <Loader2 className="mx-auto h-5 w-5 animate-spin" aria-hidden />
                  </td>
                </tr>
              ) : !data?.items.length ? (
                <tr>
                  <td colSpan={6} className="px-4 py-8 text-center text-slate-500">
                    No courses matched your search.
                  </td>
                </tr>
              ) : (
                data.items.map((course) => (
                  <tr key={course.id} className="border-t border-slate-100 dark:border-neutral-800">
                    <td className="px-4 py-2">
                      <button
                        type="button"
                        onClick={() => openCourse(course.id)}
                        className="font-medium text-indigo-600 hover:underline dark:text-indigo-400"
                      >
                        {course.title}
                      </button>
                    </td>
                    <td className="px-4 py-2 font-mono text-xs">{course.courseCode}</td>
                    <td className="px-4 py-2">{course.orgName}</td>
                    <td className="px-4 py-2">{course.instructorName ?? '—'}</td>
                    <td className="px-4 py-2">{statusLabel(course.status)}</td>
                    <td className="px-4 py-2 tabular-nums">{course.enrollmentCount}</td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      )}

      {data && data.totalPages > 1 ? (
        <nav aria-label="Courses search pagination" className="flex items-center gap-2">
          <button
            type="button"
            disabled={page <= 1}
            onClick={() => setPage((p) => p - 1)}
            className="rounded border px-3 py-1 text-sm disabled:opacity-50"
          >
            Previous
          </button>
          <span className="text-sm text-slate-600 dark:text-neutral-400">
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