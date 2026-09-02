import { type FormEvent, type ReactNode, useCallback, useEffect, useId, useRef, useState } from 'react'
import { Link, useNavigate, useSearchParams } from 'react-router-dom'
import {
  Archive,
  ArrowLeft,
  BookOpen,
  BookPlus,
  ChevronDown,
  ExternalLink,
  FileEdit,
  Loader2,
  Search,
  Settings,
  Sparkles,
  Users,
  X,
} from 'lucide-react'
import { formatDateTime } from '../../lib/format'
import { toastMutationError } from '../../lib/lms-toast'
import {
  ensurePlatformCourseAdminAccess,
  fetchCoursesStats,
  fetchPlatformCourseReport,
  searchPlatformCourses,
  type CoursesDashboardStats,
  type CoursesListFilter,
  type PlatformCourseReport,
  type PlatformCourseRow,
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

function formatCount(value: number): string {
  return value.toLocaleString()
}

type StatsTone = 'indigo' | 'emerald' | 'sky' | 'slate' | 'rose'

const TONE_STYLES: Record<
  StatsTone,
  {
    card: string
    cardSelected: string
    iconWrap: string
    icon: string
    value: string
  }
> = {
  indigo: {
    card: 'border-indigo-100/80 bg-gradient-to-br from-indigo-50/90 via-white to-white dark:border-indigo-900/40 dark:from-indigo-950/40 dark:via-neutral-900 dark:to-neutral-900',
    cardSelected:
      'border-indigo-400 ring-2 ring-indigo-500/30 dark:border-indigo-500 dark:ring-indigo-400/30',
    iconWrap: 'bg-indigo-100 text-accent-fg dark:bg-indigo-950/80 dark:text-indigo-300',
    icon: 'text-accent-fg dark:text-indigo-300',
    value: 'text-fg-default',
  },
  emerald: {
    card: 'border-emerald-100/80 bg-gradient-to-br from-emerald-50/90 via-white to-white dark:border-emerald-900/40 dark:from-emerald-950/40 dark:via-neutral-900 dark:to-neutral-900',
    cardSelected:
      'border-emerald-400 ring-2 ring-emerald-500/30 dark:border-emerald-500 dark:ring-emerald-400/30',
    iconWrap: 'bg-emerald-100 text-emerald-600 dark:bg-emerald-950/80 dark:text-emerald-300',
    icon: 'text-emerald-600 dark:text-emerald-300',
    value: 'text-fg-default',
  },
  sky: {
    card: 'border-sky-100/80 bg-gradient-to-br from-sky-50/90 via-white to-white dark:border-sky-900/40 dark:from-sky-950/40 dark:via-neutral-900 dark:to-neutral-900',
    cardSelected:
      'border-sky-400 ring-2 ring-sky-500/30 dark:border-sky-500 dark:ring-sky-400/30',
    iconWrap: 'bg-sky-100 text-sky-600 dark:bg-sky-950/80 dark:text-sky-300',
    icon: 'text-sky-600 dark:text-sky-300',
    value: 'text-fg-default',
  },
  slate: {
    card: 'border-slate-200/90 bg-gradient-to-br from-slate-50/90 via-white to-white dark:border-border-default dark:from-neutral-800/60 dark:via-neutral-900 dark:to-neutral-900',
    cardSelected:
      'border-slate-400 ring-2 ring-slate-400/40 dark:border-neutral-500 dark:ring-neutral-400/30',
    iconWrap: 'bg-surface-sunken text-fg-muted dark:bg-surface-overlay dark:text-fg-muted',
    icon: 'text-fg-muted',
    value: 'text-fg-default',
  },
  rose: {
    card: 'border-rose-100/80 bg-gradient-to-br from-rose-50/90 via-white to-white dark:border-rose-900/40 dark:from-rose-950/40 dark:via-neutral-900 dark:to-neutral-900',
    cardSelected:
      'border-rose-400 ring-2 ring-rose-500/30 dark:border-rose-500 dark:ring-rose-400/30',
    iconWrap: 'bg-rose-100 text-rose-600 dark:bg-rose-950/80 dark:text-rose-300',
    icon: 'text-rose-600 dark:text-rose-300',
    value: 'text-fg-default',
  },
}

const STAT_CARDS: {
  filter: CoursesListFilter
  statsKey: keyof CoursesDashboardStats
  label: string
  hint?: string
  icon: typeof BookOpen
  tone: StatsTone
  tableTitle: string
  tableDescription: string
}[] = [
  {
    filter: 'created_7d',
    statsKey: 'createdLast7Days',
    label: 'New courses',
    hint: 'Past 7 days',
    icon: BookPlus,
    tone: 'indigo',
    tableTitle: 'New courses',
    tableDescription: 'Courses created in the past 7 days.',
  },
  {
    filter: 'active',
    statsKey: 'activeCourses',
    label: 'Active courses',
    hint: 'Published and open',
    icon: BookOpen,
    tone: 'emerald',
    tableTitle: 'Active courses',
    tableDescription: 'Published courses that are not archived.',
  },
  {
    filter: 'draft',
    statsKey: 'draftCourses',
    label: 'Draft courses',
    hint: 'Not yet published',
    icon: FileEdit,
    tone: 'sky',
    tableTitle: 'Draft courses',
    tableDescription: 'Unpublished courses that are not archived.',
  },
  {
    filter: 'total',
    statsKey: 'totalCourses',
    label: 'Total courses',
    icon: BookOpen,
    tone: 'slate',
    tableTitle: 'All courses',
    tableDescription: 'Every course on the platform.',
  },
  {
    filter: 'archived',
    statsKey: 'archivedCourses',
    label: 'Archived',
    hint: 'Hidden from catalogs',
    icon: Archive,
    tone: 'rose',
    tableTitle: 'Archived courses',
    tableDescription: 'Courses hidden from catalogs.',
  },
]

function CoursesStatsCard({
  label,
  value,
  hint,
  icon: Icon,
  tone,
  loading,
  selected,
  onSelect,
}: {
  label: string
  value: number | null
  hint?: string
  icon: typeof BookOpen
  tone: StatsTone
  loading?: boolean
  selected?: boolean
  onSelect: () => void
}) {
  const styles = TONE_STYLES[tone]
  const countLabel = value == null ? 'unknown count' : `${formatCount(value)} ${label}`
  return (
    <button
      type="button"
      onClick={onSelect}
      aria-pressed={selected}
      aria-label={`${selected ? 'Hide' : 'Show'} ${label}: ${countLabel}`}
      className={`relative flex h-full w-full flex-col overflow-hidden rounded-2xl border px-4 py-4 text-left shadow-sm transition-all hover:shadow-md focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-indigo-500/40 ${styles.card} ${ selected ? styles.cardSelected : '' }`}
    >
      <div className="flex items-start justify-between gap-3">
        {/* Fixed 2-line label band so values line up across cards */}
        <p className="min-h-[2.5rem] min-w-0 flex-1 text-[11px] font-semibold uppercase leading-tight tracking-wider text-fg-muted">
          {label}
        </p>
        <span
          className={`flex h-10 w-10 shrink-0 items-center justify-center rounded-xl ${styles.iconWrap}`}
          aria-hidden
        >
          <Icon className={`h-5 w-5 ${styles.icon}`} />
        </span>
      </div>
      {loading ? (
        <div
          className="mt-1.5 h-9 w-16 animate-pulse rounded-md bg-slate-200/80 dark:bg-neutral-700"
          aria-hidden
        />
      ) : (
        <p
          className={`mt-1.5 h-9 text-3xl font-semibold leading-none tracking-tight tabular-nums underline-offset-4 group-hover:underline ${styles.value}`}
        >
          {value == null ? '—' : formatCount(value)}
        </p>
      )}
      {/* Fixed 2-line hint band (empty when no hint) keeps footers aligned */}
      <p className="mt-1.5 min-h-[2rem] text-xs leading-snug text-fg-subtle">
        {hint ?? '\u00a0'}
      </p>
      <p className="mt-auto pt-2 inline-flex items-center gap-1 text-[11px] font-medium text-fg-muted">
        {selected ? 'Hide list' : 'View list'}
        <ChevronDown
          className={`h-3.5 w-3.5 transition-transform ${selected ? 'rotate-180' : ''}`}
          aria-hidden
        />
      </p>
    </button>
  )
}

function CoursesDashboardCards({
  stats,
  loading,
  error,
  selectedFilter,
  onSelectFilter,
}: {
  stats: CoursesDashboardStats | null
  loading: boolean
  error: string | null
  selectedFilter: CoursesListFilter | null
  onSelectFilter: (filter: CoursesListFilter) => void
}) {
  if (error) {
    return (
      <p
        role="alert"
        className="rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-danger-fg dark:border-red-900/50 dark:bg-red-950/30 dark:text-red-300"
      >
        {error}
      </p>
    )
  }

  const value = (key: keyof CoursesDashboardStats): number | null =>
    loading || !stats ? null : stats[key]

  return (
    <section aria-labelledby="courses-dashboard-heading" className="space-y-3">
      <div className="flex items-end justify-between gap-3">
        <h3 id="courses-dashboard-heading" className="sr-only">
          Courses overview
        </h3>
        <p className="text-xs text-fg-subtle">
          Click a metric to inspect matching courses.
        </p>
      </div>
      <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-5">
        {STAT_CARDS.map((card) => (
          <CoursesStatsCard
            key={card.filter}
            label={card.label}
            hint={card.hint}
            value={value(card.statsKey)}
            icon={card.icon}
            tone={card.tone}
            loading={loading}
            selected={selectedFilter === card.filter}
            onSelect={() => onSelectFilter(card.filter)}
          />
        ))}
      </div>
    </section>
  )
}

function courseLaunchPath(courseCode: string): string {
  return `/courses/${encodeURIComponent(courseCode)}`
}

function CourseLaunchLink({
  course,
  launching,
  onLaunch,
  className,
  children,
}: {
  course: PlatformCourseRow
  launching: boolean
  onLaunch: (course: PlatformCourseRow) => void
  className?: string
  children: ReactNode
}) {
  return (
    <Link
      to={courseLaunchPath(course.courseCode)}
      className={`inline-flex min-h-6 min-w-6 items-center ${className ?? ''}`}
      aria-busy={launching || undefined}
      onClick={(e) => {
        e.preventDefault()
        onLaunch(course)
      }}
    >
      {children}
    </Link>
  )
}

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

function CourseStatusBadge({ status }: { status: string }) {
  if (status === 'active') {
    return (
      <span className="inline-flex items-center gap-1.5 rounded-full bg-emerald-50 px-2.5 py-0.5 text-xs font-medium text-emerald-700 ring-1 ring-inset ring-emerald-600/15 dark:bg-emerald-950/40 dark:text-emerald-300 dark:ring-emerald-500/30">
        <span className="h-1.5 w-1.5 rounded-full bg-emerald-500" aria-hidden />
        Active
      </span>
    )
  }
  if (status === 'archived') {
    return (
      <span className="inline-flex items-center gap-1.5 rounded-full bg-rose-50 px-2.5 py-0.5 text-xs font-medium text-rose-700 ring-1 ring-inset ring-rose-600/15 dark:bg-rose-950/40 dark:text-rose-300 dark:ring-rose-500/30">
        <span className="h-1.5 w-1.5 rounded-full bg-rose-500" aria-hidden />
        Archived
      </span>
    )
  }
  if (status === 'draft') {
    return (
      <span className="inline-flex items-center gap-1.5 rounded-full bg-sky-50 px-2.5 py-0.5 text-xs font-medium text-sky-700 ring-1 ring-inset ring-sky-600/15 dark:bg-sky-950/40 dark:text-sky-300 dark:ring-sky-500/30">
        <span className="h-1.5 w-1.5 rounded-full bg-sky-500" aria-hidden />
        Draft
      </span>
    )
  }
  return (
    <span className="inline-flex items-center gap-1.5 rounded-full bg-surface-sunken px-2.5 py-0.5 text-xs font-medium text-fg-muted ring-1 ring-inset ring-slate-500/15 dark:bg-surface-overlay dark:text-fg-muted dark:ring-neutral-600/40">
      {statusLabel(status)}
    </span>
  )
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
    return <p className="mt-6 text-sm text-fg-muted">Loading course…</p>
  }
  if (error) {
    return (
      <p role="alert" className="mt-6 text-sm text-danger-fg">
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
        className="inline-flex items-center gap-2 text-sm font-medium text-fg-muted hover:text-fg-default dark:text-fg-muted dark:hover:text-fg-default"
      >
        <ArrowLeft className="h-4 w-4" aria-hidden />
        Back to search
      </button>

      <div className="rounded-xl border border-border-default bg-surface-raised p-5 dark:border-border-subtle dark:bg-surface-raised">
        <div className="flex flex-wrap items-start justify-between gap-4">
          <div>
            <h3 className="text-lg font-semibold text-fg-default">{report.title}</h3>
            <p className="mt-1 font-mono text-sm text-fg-muted">{report.courseCode}</p>
            {report.description ? (
              <p className="mt-3 text-sm text-fg-muted">{report.description}</p>
            ) : null}
            <dl className="mt-4 grid gap-2 text-sm sm:grid-cols-2">
              <div>
                <dt className="text-fg-subtle">Organization</dt>
                <dd className="text-fg-default">{report.orgName}</dd>
              </div>
              <div>
                <dt className="text-fg-subtle">Status</dt>
                <dd className="mt-0.5">
                  <CourseStatusBadge status={report.status} />
                </dd>
              </div>
              <div>
                <dt className="text-fg-subtle">Instructor</dt>
                <dd className="text-fg-default">{report.instructorName ?? '—'}</dd>
              </div>
              <div>
                <dt className="text-fg-subtle">Term</dt>
                <dd className="text-fg-default">{report.termName ?? '—'}</dd>
              </div>
              <div>
                <dt className="text-fg-subtle">Enrollments</dt>
                <dd className="text-fg-default">{report.enrollmentCount}</dd>
              </div>
              <div>
                <dt className="text-fg-subtle">Last updated</dt>
                <dd className="text-fg-default">{formatDateTime(report.updatedAt)}</dd>
              </div>
            </dl>
          </div>
          <div className="flex flex-wrap gap-2">
            <button
              type="button"
              disabled={busy}
              onClick={() => onOpen(`/courses/${code}`)}
              className="inline-flex items-center gap-2 rounded-lg bg-accent-solid px-3 py-2 text-sm font-medium text-white hover:bg-accent disabled:opacity-50 dark:bg-neutral-100 dark:text-neutral-950 dark:hover:bg-surface-raised"
            >
              {busy ? <Loader2 className="h-4 w-4 animate-spin" aria-hidden /> : <ExternalLink className="h-4 w-4" aria-hidden />}
              Open course
            </button>
            <button
              type="button"
              disabled={busy}
              onClick={() => onOpen(`/courses/${code}/settings`)}
              className="inline-flex items-center gap-2 rounded-lg border border-border-strong px-3 py-2 text-sm font-medium text-fg-muted hover:bg-surface-base disabled:opacity-50 dark:border-border-default dark:text-fg-default dark:hover:bg-surface-overlay"
            >
              <Settings className="h-4 w-4" aria-hidden />
              Settings
            </button>
            <button
              type="button"
              disabled={busy}
              onClick={() => onOpen(`/courses/${code}/enrollments`)}
              className="inline-flex items-center gap-2 rounded-lg border border-border-strong px-3 py-2 text-sm font-medium text-fg-muted hover:bg-surface-base disabled:opacity-50 dark:border-border-default dark:text-fg-default dark:hover:bg-surface-overlay"
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

function CoursesResultsTable({
  data,
  loading,
  emptyTitle,
  emptyHint,
  loadingLabel,
  launchingId,
  onLaunch,
}: {
  data: PaginatedPlatformCourses | null
  loading: boolean
  emptyTitle: string
  emptyHint: string
  loadingLabel: string
  launchingId: string | null
  onLaunch: (course: PlatformCourseRow) => void
}) {
  return (
    <div className="overflow-x-auto">
      <table className="min-w-full text-left text-sm">
        <thead className="bg-slate-50/80 text-fg-muted/60 dark:text-fg-muted">
          <tr>
            <th scope="col" className="px-5 py-2.5 text-xs font-semibold uppercase tracking-wide">
              Title
            </th>
            <th scope="col" className="px-5 py-2.5 text-xs font-semibold uppercase tracking-wide">
              Code
            </th>
            <th scope="col" className="px-5 py-2.5 text-xs font-semibold uppercase tracking-wide">
              Organization
            </th>
            <th scope="col" className="px-5 py-2.5 text-xs font-semibold uppercase tracking-wide">
              Instructor
            </th>
            <th scope="col" className="px-5 py-2.5 text-xs font-semibold uppercase tracking-wide">
              Status
            </th>
            <th scope="col" className="px-5 py-2.5 text-xs font-semibold uppercase tracking-wide">
              Enrollments
            </th>
          </tr>
        </thead>
        <tbody className="divide-y divide-slate-100 dark:divide-neutral-800">
          {loading ? (
            <tr>
              <td colSpan={6} className="px-5 py-12 text-center">
                <Loader2 className="mx-auto h-5 w-5 animate-spin text-indigo-500" aria-hidden />
                <p className="mt-2 text-sm text-fg-muted">{loadingLabel}</p>
              </td>
            </tr>
          ) : !data?.items.length ? (
            <tr>
              <td colSpan={6} className="px-5 py-12 text-center">
                <BookOpen
                  className="mx-auto h-8 w-8 text-slate-300 dark:text-neutral-600"
                  aria-hidden
                />
                <p className="mt-2 text-sm font-medium text-fg-muted">
                  {emptyTitle}
                </p>
                <p className="mt-1 text-sm text-fg-muted">{emptyHint}</p>
              </td>
            </tr>
          ) : (
            data.items.map((course: PlatformCourseRow) => (
              <tr
                key={course.id}
                className="transition-colors hover:bg-slate-50/80 dark:hover:bg-neutral-800/40"
              >
                <td className="px-5 py-3">
                  <CourseLaunchLink
                    course={course}
                    launching={launchingId === course.id}
                    onLaunch={onLaunch}
                    className="font-medium text-accent-fg hover:underline"
                  >
                    {course.title}
                  </CourseLaunchLink>
                </td>
                <td className="px-5 py-3 font-mono text-xs">
                  <CourseLaunchLink
                    course={course}
                    launching={launchingId === course.id}
                    onLaunch={onLaunch}
                    className="text-accent-fg hover:underline"
                  >
                    {course.courseCode}
                  </CourseLaunchLink>
                </td>
                <td className="px-5 py-3 text-fg-muted">{course.orgName}</td>
                <td className="px-5 py-3 text-fg-muted">
                  {course.instructorName ?? '—'}
                </td>
                <td className="px-5 py-3">
                  <CourseStatusBadge status={course.status} />
                </td>
                <td className="px-5 py-3 tabular-nums text-fg-muted">
                  {formatCount(course.enrollmentCount)}
                </td>
              </tr>
            ))
          )}
        </tbody>
      </table>
    </div>
  )
}

function PaginationNav({
  label,
  page,
  totalPages,
  onPrev,
  onNext,
}: {
  label: string
  page: number
  totalPages: number
  onPrev: () => void
  onNext: () => void
}) {
  if (totalPages <= 1) return null
  return (
    <nav
      aria-label={label}
      className="flex items-center justify-between gap-3 border-t border-border-subtle px-5 py-3 dark:border-border-subtle"
    >
      <button
        type="button"
        disabled={page <= 1}
        onClick={onPrev}
        className="rounded-lg border border-border-default px-3 py-1.5 text-sm font-medium text-fg-muted transition-colors hover:bg-surface-base disabled:opacity-50 dark:border-border-default dark:text-fg-default dark:hover:bg-surface-overlay"
      >
        Previous
      </button>
      <span className="text-sm text-fg-muted">
        Page {page} of {totalPages}
      </span>
      <button
        type="button"
        disabled={page >= totalPages}
        onClick={onNext}
        className="rounded-lg border border-border-default px-3 py-1.5 text-sm font-medium text-fg-muted transition-colors hover:bg-surface-base disabled:opacity-50 dark:border-border-default dark:text-fg-default dark:hover:bg-surface-overlay"
      >
        Next
      </button>
    </nav>
  )
}

export function CoursesPanel() {
  const navigate = useNavigate()
  const [searchParams, setSearchParams] = useSearchParams()
  const selectedCourseId = searchParams.get('courseId')
  const filterPanelId = useId()

  const [q, setQ] = useState('')
  const [submittedQ, setSubmittedQ] = useState('')
  const [status, setStatus] = useState<PlatformCourseSearchStatus>('open')
  const [page, setPage] = useState(1)
  const [perPage, setPerPage] = useState(25)
  const [data, setData] = useState<PaginatedPlatformCourses | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [accessBusy, setAccessBusy] = useState(false)
  const [launchingId, setLaunchingId] = useState<string | null>(null)
  const launchingRef = useRef(false)

  const [selectedFilter, setSelectedFilter] = useState<CoursesListFilter | null>(null)
  const [filterPage, setFilterPage] = useState(1)
  const [filterPerPage, setFilterPerPage] = useState(25)
  const [filterData, setFilterData] = useState<PaginatedPlatformCourses | null>(null)
  const [filterLoading, setFilterLoading] = useState(false)
  const [filterError, setFilterError] = useState<string | null>(null)

  const [report, setReport] = useState<PlatformCourseReport | null>(null)
  const [reportLoading, setReportLoading] = useState(false)
  const [reportError, setReportError] = useState<string | null>(null)

  const [stats, setStats] = useState<CoursesDashboardStats | null>(null)
  const [statsLoading, setStatsLoading] = useState(true)
  const [statsError, setStatsError] = useState<string | null>(null)

  const loadStats = useCallback(async () => {
    setStatsLoading(true)
    setStatsError(null)
    try {
      setStats(await fetchCoursesStats())
    } catch (e) {
      setStatsError(e instanceof Error ? e.message : 'Failed to load course stats.')
      setStats(null)
    } finally {
      setStatsLoading(false)
    }
  }, [])

  useEffect(() => {
    void loadStats()
  }, [loadStats])

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

  const loadFilter = useCallback(async () => {
    if (!selectedFilter) {
      setFilterData(null)
      setFilterError(null)
      return
    }
    setFilterLoading(true)
    setFilterError(null)
    try {
      setFilterData(
        await searchPlatformCourses({
          filter: selectedFilter,
          page: filterPage,
          perPage: filterPerPage,
        }),
      )
    } catch (e) {
      setFilterError(e instanceof Error ? e.message : 'Failed to load courses.')
      setFilterData(null)
    } finally {
      setFilterLoading(false)
    }
  }, [selectedFilter, filterPage, filterPerPage])

  useEffect(() => {
    void loadFilter()
  }, [loadFilter])

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

  function toggleFilter(filter: CoursesListFilter) {
    if (selectedFilter === filter) {
      setSelectedFilter(null)
      setFilterData(null)
      setFilterError(null)
      return
    }
    setSelectedFilter(filter)
    setFilterPage(1)
  }

  function closeCourse() {
    const next = new URLSearchParams(searchParams)
    next.delete('courseId')
    setSearchParams(next, { replace: false })
  }

  async function launchCourse(course: PlatformCourseRow) {
    if (launchingRef.current) return
    launchingRef.current = true
    setLaunchingId(course.id)
    try {
      await ensurePlatformCourseAdminAccess(course.id)
      navigate(courseLaunchPath(course.courseCode))
    } catch (e) {
      toastMutationError(e instanceof Error ? e.message : 'Could not grant course access.')
      launchingRef.current = false
      setLaunchingId(null)
    }
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

  const resultCount = data?.total ?? data?.items.length
  const activeStat = STAT_CARDS.find((c) => c.filter === selectedFilter) ?? null
  const filterCount = filterData?.total

  return (
    <div className="mt-6 space-y-6">
      <CoursesDashboardCards
        stats={stats}
        loading={statsLoading}
        error={statsError}
        selectedFilter={selectedFilter}
        onSelectFilter={toggleFilter}
      />

      {selectedFilter && activeStat ? (
        <section
          id={filterPanelId}
          aria-label={activeStat.tableTitle}
          className="overflow-hidden rounded-2xl border border-border-default bg-surface-raised shadow-sm dark:border-border-subtle dark:bg-surface-raised"
        >
          <div className="flex flex-wrap items-start justify-between gap-3 border-b border-border-subtle px-5 py-4 dark:border-border-subtle">
            <div>
              <h3 className="text-sm font-semibold text-fg-default">
                {activeStat.tableTitle}
              </h3>
              <p className="mt-0.5 text-sm text-fg-muted">
                {activeStat.tableDescription}
                {filterCount != null && !filterLoading ? (
                  <>
                    {' '}
                    <span className="font-medium text-fg-muted">
                      {formatCount(filterCount)}
                    </span>{' '}
                    {filterCount === 1 ? 'course' : 'courses'}.
                  </>
                ) : null}
              </p>
            </div>
            <div className="flex flex-wrap items-center gap-2">
              <label className="flex items-center gap-2 text-sm text-fg-muted">
                <span className="text-xs font-medium uppercase tracking-wide">Page size</span>
                <select
                  value={filterPerPage}
                  onChange={(e) => {
                    setFilterPerPage(Number(e.target.value))
                    setFilterPage(1)
                  }}
                  className="rounded-lg border border-border-default bg-surface-raised px-2 py-1.5 text-sm dark:border-border-default dark:bg-surface-base"
                >
                  {PAGE_SIZES.map((n) => (
                    <option key={n} value={n}>
                      {n}
                    </option>
                  ))}
                </select>
              </label>
              <button
                type="button"
                onClick={() => toggleFilter(selectedFilter)}
                className="inline-flex items-center gap-1.5 rounded-lg border border-border-default px-3 py-1.5 text-sm font-medium text-fg-muted transition-colors hover:bg-surface-base dark:border-border-default dark:text-fg-default dark:hover:bg-surface-overlay"
              >
                <X className="h-4 w-4" aria-hidden />
                Close
              </button>
            </div>
          </div>

          {filterError ? (
            <div className="border-b border-border-subtle px-5 py-3 dark:border-border-subtle">
              <p
                role="alert"
                className="rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-danger-fg dark:border-red-900/50 dark:bg-red-950/30 dark:text-red-300"
              >
                {filterError}
              </p>
            </div>
          ) : null}

          <CoursesResultsTable
            data={filterData}
            loading={filterLoading}
            emptyTitle="No courses in this segment"
            emptyHint="Try another metric or search by code or title."
            loadingLabel="Loading courses…"
            launchingId={launchingId}
            onLaunch={(course) => void launchCourse(course)}
          />

          {filterData ? (
            <PaginationNav
              label={`${activeStat.tableTitle} pagination`}
              page={filterData.page}
              totalPages={filterData.totalPages}
              onPrev={() => setFilterPage((p) => p - 1)}
              onNext={() => setFilterPage((p) => p + 1)}
            />
          ) : null}
        </section>
      ) : null}

      <section className="overflow-hidden rounded-2xl border border-border-default bg-surface-raised shadow-sm dark:border-border-subtle dark:bg-surface-raised">
        <div className="flex flex-wrap items-center justify-between gap-3 border-b border-border-subtle px-5 py-4 dark:border-border-subtle">
          <div>
            <h3 className="text-sm font-semibold text-fg-default">Find courses</h3>
            <p className="mt-0.5 text-sm text-fg-muted">
              Search by course code or title. Results appear after you search.
            </p>
          </div>
        </div>

        <form onSubmit={onSearchSubmit} className="flex flex-wrap items-end gap-3 px-5 py-4">
          <label className="flex min-w-[16rem] flex-1 flex-col text-sm">
            <span className="mb-1.5 text-xs font-medium uppercase tracking-wide text-fg-muted">
              Search
            </span>
            <div className="relative">
              <Search
                className="pointer-events-none absolute left-3.5 top-1/2 h-4 w-4 -translate-y-1/2 text-fg-subtle"
                aria-hidden
              />
              <input
                type="search"
                value={q}
                onChange={(e) => setQ(e.target.value)}
                placeholder="Course code or title"
                className="w-full rounded-xl border border-border-default bg-slate-50/50 py-2.5 pl-10 pr-3 text-sm text-fg-default shadow-inner transition-colors placeholder:text-fg-subtle focus:border-indigo-300 focus:bg-surface-raised focus:outline-none focus:ring-2 focus:ring-indigo-500/20 dark:border-border-default dark:bg-surface-base dark:text-fg-default dark:placeholder:text-neutral-500 dark:focus:border-indigo-500/50 dark:focus:bg-surface-base"
              />
            </div>
          </label>
          <label className="flex flex-col text-sm">
            <span className="mb-1.5 text-xs font-medium uppercase tracking-wide text-fg-muted">
              Status
            </span>
            <select
              value={status}
              onChange={(e) => {
                setStatus(e.target.value as PlatformCourseSearchStatus)
                setPage(1)
              }}
              className="rounded-xl border border-border-default bg-surface-raised px-3 py-2.5 text-sm dark:border-border-default dark:bg-surface-base"
            >
              {STATUS_OPTIONS.map((opt) => (
                <option key={opt.value} value={opt.value}>
                  {opt.label}
                </option>
              ))}
            </select>
          </label>
          <label className="flex flex-col text-sm">
            <span className="mb-1.5 text-xs font-medium uppercase tracking-wide text-fg-muted">
              Page size
            </span>
            <select
              value={perPage}
              onChange={(e) => {
                setPerPage(Number(e.target.value))
                setPage(1)
              }}
              className="rounded-xl border border-border-default bg-surface-raised px-3 py-2.5 text-sm dark:border-border-default dark:bg-surface-base"
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
            className="inline-flex items-center gap-2 rounded-xl border border-border-default bg-surface-raised px-4 py-2.5 text-sm font-medium text-fg-muted shadow-sm transition-colors hover:bg-surface-base disabled:opacity-50 dark:border-border-default dark:bg-surface-raised dark:text-fg-default dark:hover:bg-surface-overlay"
          >
            {loading ? (
              <Loader2 className="h-4 w-4 animate-spin" aria-hidden />
            ) : (
              <Search className="h-4 w-4" aria-hidden />
            )}
            {loading ? 'Searching…' : 'Search'}
          </button>
        </form>

        {error ? (
          <div className="border-t border-border-subtle px-5 py-3 dark:border-border-subtle">
            <p
              role="alert"
              className="rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-danger-fg dark:border-red-900/50 dark:bg-red-950/30 dark:text-red-300"
            >
              {error}
            </p>
          </div>
        ) : null}

        {!submittedQ.trim() ? (
          <div className="border-t border-border-subtle px-5 py-12 dark:border-border-subtle">
            <div className="mx-auto flex max-w-sm flex-col items-center text-center">
              <span className="flex h-14 w-14 items-center justify-center rounded-2xl bg-indigo-50 text-accent-fg ring-1 ring-indigo-100 dark:bg-indigo-950/50 dark:text-indigo-300 dark:ring-indigo-900/50">
                <Sparkles className="h-6 w-6" aria-hidden />
              </span>
              <p className="mt-4 text-sm font-medium text-fg-default">
                Search to manage courses
              </p>
              <p className="mt-1.5 text-sm leading-relaxed text-fg-muted">
                Enter a course code or title above, or click a metric card above.
              </p>
            </div>
          </div>
        ) : (
          <div className="border-t border-border-subtle">
            {data && !loading ? (
              <div className="flex items-center justify-between gap-3 border-b border-border-subtle px-5 py-2.5 dark:border-border-subtle">
                <p className="text-xs text-fg-muted">
                  {resultCount != null ? (
                    <>
                      <span className="font-medium text-fg-muted">
                        {formatCount(resultCount)}
                      </span>{' '}
                      {resultCount === 1 ? 'result' : 'results'} for{' '}
                      <span className="font-medium text-fg-muted">
                        “{submittedQ}”
                      </span>
                    </>
                  ) : (
                    <>
                      Results for{' '}
                      <span className="font-medium text-fg-muted">
                        “{submittedQ}”
                      </span>
                    </>
                  )}
                </p>
              </div>
            ) : null}
            <CoursesResultsTable
              data={data}
              loading={loading}
              emptyTitle="No courses matched your search"
              emptyHint="Try a different code, title, or status filter."
              loadingLabel="Searching…"
              launchingId={launchingId}
              onLaunch={(course) => void launchCourse(course)}
            />
          </div>
        )}

        {data ? (
          <PaginationNav
            label="Courses search pagination"
            page={data.page}
            totalPages={data.totalPages}
            onPrev={() => setPage((p) => p - 1)}
            onNext={() => setPage((p) => p + 1)}
          />
        ) : null}
      </section>
    </div>
  )
}