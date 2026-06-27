import { type FormEvent, useCallback, useEffect, useId, useMemo, useRef, useState } from 'react'
import { Link, Navigate, useParams } from 'react-router-dom'
import { ChevronDown, Mail, Send, X } from 'lucide-react'
import { EnrollmentAvatar } from '../../components/enrollment/enrollment-avatar'
import { useCoursePageTitle } from '../../context/course-document-title-context'
import { usePermission } from '../../context/use-permissions'
import { LmsPage } from './lms-page'
import {
  courseEnrollmentsUpdatePermission,
  courseGradebookViewPermission,
  fetchCourse,
  sendEnrollmentMessage,
} from '../../lib/courses-api'
import { formatTimeAgoFromIso } from '../../lib/format-time-ago'
import { formatAbsolute } from '../../lib/format-datetime'
import { formatDate } from '../../lib/format'
import { toast } from '../../lib/lms-toast'
import {
  createStudentProgressNote,
  deleteStudentProgressNote,
  fetchStudentProgress,
  fetchStudentProgressActivity,
  updateStudentProgressNote,
  type StudentProgressActivityEvent,
  type StudentProgressNote,
  type StudentProgressQuizRow,
  type StudentProgressResponse,
} from '../../lib/student-progress-api'
import { usePlatformFeatures } from '../../context/platform-features-context'
import { studentProgressI18n } from '../../lib/student-progress'

type TabId = 'overview' | 'activity' | 'assignments' | 'quizzes' | 'notes'

const LMS_MODAL_OVERLAY_CLASS =
  'fixed inset-0 z-50 flex items-end justify-center p-4 backdrop-blur-md bg-slate-900/30 dark:bg-black/40 sm:items-center'

function StudentProgressActionsMenu({
  disabled,
  onMessageStudent,
}: {
  disabled: boolean
  onMessageStudent: () => void
}) {
  const [open, setOpen] = useState(false)
  const rootRef = useRef<HTMLDivElement>(null)
  const menuId = useId()

  useEffect(() => {
    if (!open) return
    function onDoc(e: MouseEvent) {
      if (!rootRef.current?.contains(e.target as Node)) setOpen(false)
    }
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape') setOpen(false)
    }
    document.addEventListener('mousedown', onDoc)
    document.addEventListener('keydown', onKey)
    return () => {
      document.removeEventListener('mousedown', onDoc)
      document.removeEventListener('keydown', onKey)
    }
  }, [open])

  return (
    <div ref={rootRef} className="relative">
      <button
        type="button"
        disabled={disabled}
        aria-haspopup="menu"
        aria-expanded={open}
        aria-controls={open ? menuId : undefined}
        onClick={() => setOpen((o) => !o)}
        className="inline-flex items-center gap-2 rounded-xl border border-slate-300 bg-white px-2 py-1.5 text-sm font-semibold text-slate-800 shadow-sm transition-[background-color,color,border-color] hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-60 dark:border-neutral-600 dark:bg-neutral-900 dark:text-neutral-100 dark:hover:bg-neutral-800"
      >
        More
        <ChevronDown
          className={`h-4 w-4 shrink-0 transition-transform ${open ? 'rotate-180' : ''}`}
          aria-hidden
        />
      </button>
      {open && (
        <div
          id={menuId}
          role="menu"
          aria-label="Student progress actions"
          className="absolute end-0 z-50 mt-1 min-w-[12rem] overflow-hidden rounded-xl border border-slate-200 bg-white py-1 shadow-lg shadow-slate-900/10 dark:border-neutral-600 dark:bg-neutral-900"
        >
          <button
            type="button"
            role="menuitem"
            onClick={() => {
              onMessageStudent()
              setOpen(false)
            }}
            className="flex w-full items-center gap-2 px-2.5 py-2 text-start text-sm font-medium text-slate-800 transition-[background-color,color,border-color] hover:bg-slate-50 dark:text-neutral-200 dark:hover:bg-neutral-800"
          >
            <Mail className="h-4 w-4 shrink-0 text-slate-500 dark:text-neutral-400" aria-hidden />
            Message Student
          </button>
        </div>
      )}
    </div>
  )
}

function statusBadgeClass(status: string): string {
  switch (status) {
    case 'submitted':
      return 'bg-emerald-100 text-emerald-900 dark:bg-emerald-950/50 dark:text-emerald-200'
    case 'late':
    case 'missing':
      return 'bg-red-100 text-red-900 dark:bg-red-950/50 dark:text-red-200'
    default:
      return 'bg-slate-100 text-slate-800 dark:bg-neutral-800 dark:text-neutral-200'
  }
}

function formatQuizAxisDate(iso: string): string {
  return formatDate(iso, { month: 'short', day: 'numeric' })
}

function formatQuizScorePercent(score: number | null | undefined): string {
  if (score == null) return '—'
  return `${Math.round(score * 10) / 10}%`
}

function sortedQuizAttempts(quizzes: StudentProgressQuizRow[]): StudentProgressQuizRow[] {
  return [...quizzes].sort(
    (a, b) => new Date(a.submittedAt).getTime() - new Date(b.submittedAt).getTime(),
  )
}

function QuizScoreChart({
  quizzes,
  captionId,
}: {
  quizzes: StudentProgressResponse['quizzes']
  captionId: string
}) {
  if (quizzes.length === 0) {
    return <p className="text-sm text-slate-500 dark:text-neutral-400">No quiz attempts yet.</p>
  }

  const sorted = sortedQuizAttempts(quizzes)
  const scores = sorted.map((q) => q.scorePercent ?? 0)
  const minScore = Math.min(...scores)
  const maxScore = Math.max(...scores)
  const roundedMin = Math.round(minScore * 10) / 10
  const roundedMax = Math.round(maxScore * 10) / 10

  if (sorted.length === 1) {
    const only = sorted[0]
    return (
      <p className="text-sm text-slate-600 dark:text-neutral-400">
        One quiz attempt so far ({formatQuizAxisDate(only.submittedAt)}):{' '}
        <strong className="text-slate-900 dark:text-neutral-100">
          {formatQuizScorePercent(only.scorePercent)}
        </strong>
        . A trend chart appears after a second attempt.
      </p>
    )
  }

  const width = 520
  const height = 220
  const marginLeft = 44
  const marginRight = 12
  const marginTop = 12
  const marginBottom = 52
  const innerW = width - marginLeft - marginRight
  const innerH = height - marginTop - marginBottom
  const yTicks = [0, 25, 50, 75, 100]

  const points = sorted.map((quiz, i) => {
    const x =
      marginLeft + (sorted.length === 1 ? innerW / 2 : (i / (sorted.length - 1)) * innerW)
    const y = marginTop + innerH - (scores[i] / 100) * innerH
    return { x, y, quiz, score: scores[i] }
  })

  const labelStep = sorted.length <= 6 ? 1 : Math.ceil(sorted.length / 6)

  return (
    <figure className="mt-2">
      <figcaption className="text-sm font-semibold text-slate-900 dark:text-neutral-100">
        Quiz scores over time
      </figcaption>
      <p className="mt-1 text-sm text-slate-600 dark:text-neutral-400">
        Each dot is one submitted quiz, ordered oldest to newest (left to right). The vertical axis
        is the score percentage.
        {roundedMin === roundedMax
          ? ` Every attempt so far scored ${formatQuizScorePercent(roundedMin)}.`
          : ` Scores range from ${formatQuizScorePercent(roundedMin)} to ${formatQuizScorePercent(roundedMax)}.`}
      </p>
      <svg
        role="img"
        aria-labelledby={captionId}
        viewBox={`0 0 ${width} ${height}`}
        className="mt-4 max-w-full rounded-lg border border-slate-200 bg-white dark:border-neutral-700 dark:bg-neutral-900"
      >
        <title id={captionId}>Quiz scores over time</title>
        {yTicks.map((tick) => {
          const y = marginTop + innerH - (tick / 100) * innerH
          return (
            <g key={tick}>
              <line
                x1={marginLeft}
                y1={y}
                x2={width - marginRight}
                y2={y}
                className="stroke-slate-200 dark:stroke-neutral-700"
                strokeWidth="1"
              />
              <text
                x={marginLeft - 8}
                y={y + 4}
                textAnchor="end"
                className="fill-slate-500 text-[10px] dark:fill-neutral-400"
              >
                {tick}%
              </text>
            </g>
          )
        })}
        <line
          x1={marginLeft}
          y1={marginTop + innerH}
          x2={width - marginRight}
          y2={marginTop + innerH}
          className="stroke-slate-300 dark:stroke-neutral-600"
          strokeWidth="1"
        />
        <text
          x={marginLeft + innerW / 2}
          y={height - 6}
          textAnchor="middle"
          className="fill-slate-500 text-[10px] dark:fill-neutral-400"
        >
          Submission date (oldest → newest)
        </text>
        <polyline
          fill="none"
          strokeWidth="2"
          className="stroke-indigo-600 dark:stroke-indigo-400"
          points={points.map((p) => `${p.x},${p.y}`).join(' ')}
        />
        {points.map((p) => (
          <g key={p.quiz.attemptId}>
            <circle
              cx={p.x}
              cy={p.y}
              r={5}
              className="fill-indigo-600 dark:fill-indigo-400"
            >
              <title>
                {p.quiz.title}: {formatQuizScorePercent(p.score)} on{' '}
                {formatAbsolute(new Date(p.quiz.submittedAt))}
              </title>
            </circle>
          </g>
        ))}
        {points.map((p, i) =>
          i % labelStep === 0 || i === points.length - 1 ? (
            <text
              key={`${p.quiz.attemptId}-label`}
              x={p.x}
              y={marginTop + innerH + 16}
              textAnchor="middle"
              className="fill-slate-600 text-[9px] dark:fill-neutral-400"
            >
              {formatQuizAxisDate(p.quiz.submittedAt)}
            </text>
          ) : null,
        )}
      </svg>
      <p id={captionId} className="sr-only">
        {sorted
          .map(
            (q) =>
              `${q.title}: ${formatQuizScorePercent(q.scorePercent)} on ${formatAbsolute(new Date(q.submittedAt))}`,
          )
          .join('; ')}
      </p>
    </figure>
  )
}

export default function StudentProgressPage() {
  const { courseCode, enrollmentId } = useParams<{ courseCode: string; enrollmentId: string }>()
  const { studentProgressEnabled, loading: featuresLoading } = usePlatformFeatures()
  const canUpdateEnrollments = usePermission(
    courseCode ? courseEnrollmentsUpdatePermission(courseCode) : 'global:app:noop:noop',
  )
  const canViewGradebook = usePermission(
    courseCode ? courseGradebookViewPermission(courseCode) : 'global:app:noop:noop',
  )
  const canMessageStudent = canUpdateEnrollments || canViewGradebook
  const tabsId = useId()
  const [tab, setTab] = useState<TabId>('overview')
  const [loadState, setLoadState] = useState<'idle' | 'loading' | 'ok' | 'error'>('idle')
  const [loadError, setLoadError] = useState<string | null>(null)
  const [data, setData] = useState<StudentProgressResponse | null>(null)
  const [viewerEnrollmentId, setViewerEnrollmentId] = useState<string | null>(null)
  const [activity, setActivity] = useState<StudentProgressActivityEvent[]>([])
  const [activityCursor, setActivityCursor] = useState<string | null>(null)
  const [activityLoading, setActivityLoading] = useState(false)
  const [noteDraft, setNoteDraft] = useState('')
  const [editingNote, setEditingNote] = useState<StudentProgressNote | null>(null)
  const [noteBusy, setNoteBusy] = useState(false)
  const [messageOpen, setMessageOpen] = useState(false)
  const [messageSubject, setMessageSubject] = useState('')
  const [messageBody, setMessageBody] = useState('')
  const [messageStatus, setMessageStatus] = useState<'idle' | 'loading' | 'error'>('idle')
  const [messageError, setMessageError] = useState<string | null>(null)

  const isSelf = viewerEnrollmentId != null && enrollmentId === viewerEnrollmentId
  const showProgressActions = !isSelf && canMessageStudent

  const load = useCallback(async () => {
    if (!courseCode || !enrollmentId) return
    setLoadState('loading')
    setLoadError(null)
    try {
      const [progress, course] = await Promise.all([
        fetchStudentProgress(courseCode, enrollmentId),
        fetchCourse(courseCode),
      ])
      setData(progress)
      setViewerEnrollmentId(course.viewerStudentEnrollmentId ?? null)
      setLoadState('ok')
    } catch (e: unknown) {
      setLoadState('error')
      setLoadError(e instanceof Error ? e.message : 'Could not load progress.')
    }
  }, [courseCode, enrollmentId])

  useEffect(() => {
    if (featuresLoading || !studentProgressEnabled) return
    void load()
  }, [load, featuresLoading, studentProgressEnabled])

  const loadActivity = useCallback(
    async (reset: boolean) => {
      if (!courseCode || !enrollmentId) return
      setActivityLoading(true)
      try {
        const page = await fetchStudentProgressActivity(
          courseCode,
          enrollmentId,
          reset ? undefined : activityCursor ?? undefined,
        )
        setActivity((prev) => (reset ? page.events : [...prev, ...page.events]))
        setActivityCursor(page.nextCursor ?? null)
      } finally {
        setActivityLoading(false)
      }
    },
    [courseCode, enrollmentId, activityCursor],
  )

  useEffect(() => {
    if (tab === 'activity' && activity.length === 0 && loadState === 'ok') {
      void loadActivity(true)
    }
  }, [tab, activity.length, loadState, loadActivity])

  const closeMessageModal = useCallback(() => {
    setMessageOpen(false)
    setMessageSubject('')
    setMessageBody('')
    setMessageStatus('idle')
    setMessageError(null)
  }, [])

  const openMessageModal = useCallback(() => {
    setMessageOpen(true)
    setMessageSubject('')
    setMessageBody('')
    setMessageStatus('idle')
    setMessageError(null)
  }, [])

  useEffect(() => {
    if (!messageOpen) return
    function onKey(e: KeyboardEvent) {
      if (e.key !== 'Escape') return
      e.preventDefault()
      closeMessageModal()
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [messageOpen, closeMessageModal])

  const tabs = useMemo(() => {
    const base: { id: TabId; label: string }[] = [
      { id: 'overview', label: studentProgressI18n.tabOverview },
      { id: 'activity', label: studentProgressI18n.tabActivity },
      { id: 'assignments', label: studentProgressI18n.tabAssignments },
      { id: 'quizzes', label: studentProgressI18n.tabQuizzes },
    ]
    if (data?.summary.canManageNotes) {
      base.push({ id: 'notes', label: studentProgressI18n.tabNotes })
    }
    return base
  }, [data?.summary.canManageNotes])

  const chartCaptionId = `${tabsId}-quiz-chart`
  const progressPageTitle =
    data?.summary.studentDisplayName ??
    (isSelf ? studentProgressI18n.myProgressTitle : studentProgressI18n.progressTitle)

  useCoursePageTitle(loadState === 'ok' ? progressPageTitle : null)

  if (!courseCode || !enrollmentId) {
    return <Navigate to="/courses" replace />
  }

  const title = isSelf ? studentProgressI18n.myProgressTitle : studentProgressI18n.progressTitle
  const pageTitle = data?.summary.studentDisplayName ?? title

  if (featuresLoading) {
    return (
      <LmsPage title={title}>
        <p className="mt-6 text-sm text-slate-500 dark:text-neutral-400">Loading…</p>
      </LmsPage>
    )
  }

  if (!studentProgressEnabled) {
    return <Navigate to={`/courses/${encodeURIComponent(courseCode)}`} replace />
  }

  async function onSaveNote() {
    if (!courseCode || !enrollmentId || !noteDraft.trim()) return
    setNoteBusy(true)
    try {
      if (editingNote) {
        await updateStudentProgressNote(courseCode, enrollmentId, editingNote.id, noteDraft.trim())
      } else {
        await createStudentProgressNote(courseCode, enrollmentId, noteDraft.trim())
      }
      setNoteDraft('')
      setEditingNote(null)
      await load()
    } catch (e: unknown) {
      setLoadError(e instanceof Error ? e.message : 'Could not save note.')
    } finally {
      setNoteBusy(false)
    }
  }

  async function onDeleteNote(noteId: string) {
    if (!courseCode || !enrollmentId) return
    setNoteBusy(true)
    try {
      await deleteStudentProgressNote(courseCode, enrollmentId, noteId)
      await load()
    } catch (e: unknown) {
      setLoadError(e instanceof Error ? e.message : 'Could not delete note.')
    } finally {
      setNoteBusy(false)
    }
  }

  async function onSubmitStudentMessage(ev: FormEvent) {
    ev.preventDefault()
    if (!courseCode || !enrollmentId || !data) return
    const body = messageBody.trim()
    if (!body) {
      setMessageError('Enter a message.')
      setMessageStatus('error')
      return
    }
    setMessageStatus('loading')
    setMessageError(null)
    try {
      await sendEnrollmentMessage(courseCode, enrollmentId, {
        subject: messageSubject.trim(),
        body,
      })
      const name = data.summary.studentDisplayName?.trim() || 'this student'
      toast.success('Message sent', { description: `Delivered to ${name}'s inbox.` })
      closeMessageModal()
    } catch (e: unknown) {
      setMessageStatus('error')
      setMessageError(e instanceof Error ? e.message : 'Could not send message.')
    }
  }

  return (
    <LmsPage
      title={pageTitle}
      actions={
        showProgressActions ? (
          <StudentProgressActionsMenu
            disabled={loadState !== 'ok'}
            onMessageStudent={openMessageModal}
          />
        ) : undefined
      }
      titleContent={
        data ? (
          <div className="flex items-center gap-3">
            <EnrollmentAvatar
              userId={data.summary.studentUserId}
              name={data.summary.studentDisplayName}
              avatarUrl={data.summary.studentAvatarUrl}
              size="md"
            />
            <h1 className="text-2xl font-semibold tracking-tight text-slate-900 dark:text-neutral-100">
              {pageTitle}
            </h1>
          </div>
        ) : undefined
      }
      description={
        isSelf
          ? 'Track your assignment completion, quiz performance, and recent activity in this course.'
          : 'Consolidated view of assignment completion, quiz scores, engagement, and missing work.'
      }
    >
      {loadState === 'loading' && (
        <p className="mt-6 text-sm text-slate-500 dark:text-neutral-400">Loading progress…</p>
      )}
      {loadState === 'error' && loadError && (
        <p className="mt-6 text-sm text-red-600 dark:text-red-400" role="alert">
          {loadError}
        </p>
      )}
      {loadState === 'ok' && data && (
        <>
          <p className="mt-4 text-sm text-slate-600 dark:text-neutral-400">
            {studentProgressI18n.lastUpdated}{' '}
            {data.summary.staleMinutes > 0
              ? `${data.summary.staleMinutes} min ago`
              : 'just now'}
            <span className="text-slate-400 dark:text-neutral-500">
              {' '}
              ({formatAbsolute(new Date(data.summary.dataAsOf))})
            </span>
          </p>

          <div
            role="status"
            className="mt-6 grid gap-4 rounded-xl border border-slate-200 bg-white p-4 shadow-sm sm:grid-cols-2 lg:grid-cols-4 dark:border-neutral-700 dark:bg-neutral-900"
          >
            <dl>
              <dt className="text-xs font-medium uppercase tracking-wide text-slate-500 dark:text-neutral-400">
                {studentProgressI18n.submitted}
              </dt>
              <dd className="mt-1 text-2xl font-semibold tabular-nums text-slate-950 dark:text-neutral-50">
                {Math.round(data.summary.assignmentsSubmittedPct)}%
                <span className="sr-only"> assignments submitted</span>
              </dd>
            </dl>
            <dl>
              <dt className="text-xs font-medium uppercase tracking-wide text-slate-500 dark:text-neutral-400">
                {studentProgressI18n.modulesViewed}
              </dt>
              <dd className="mt-1 text-2xl font-semibold tabular-nums text-slate-950 dark:text-neutral-50">
                {Math.round(data.summary.modulesViewedPct)}%
                <span className="sr-only"> modules viewed</span>
              </dd>
            </dl>
            <dl>
              <dt className="text-xs font-medium uppercase tracking-wide text-slate-500 dark:text-neutral-400">
                {studentProgressI18n.avgScore}
              </dt>
              <dd className="mt-1 text-2xl font-semibold tabular-nums text-slate-950 dark:text-neutral-50">
                {data.summary.avgGradePercent != null
                  ? `${Math.round(data.summary.avgGradePercent * 10) / 10}%`
                  : '—'}
              </dd>
            </dl>
            <dl>
              <dt className="text-xs font-medium uppercase tracking-wide text-slate-500 dark:text-neutral-400">
                {studentProgressI18n.lastActive}
              </dt>
              <dd className="mt-1 text-lg font-semibold text-slate-950 dark:text-neutral-50">
                {data.summary.lastActiveAt
                  ? formatTimeAgoFromIso(data.summary.lastActiveAt)
                  : 'Never'}
              </dd>
            </dl>
          </div>

          {data.summary.missingCount > 0 && tab === 'overview' && (
            <section className="mt-6 rounded-xl border border-amber-200 bg-amber-50/80 p-4 dark:border-amber-900/50 dark:bg-amber-950/30">
              <h2 className="text-sm font-semibold text-amber-950 dark:text-amber-100">
                {studentProgressI18n.missing} ({data.summary.missingCount})
              </h2>
              <ul className="mt-2 space-y-2 text-sm">
                {data.missing.map((m) => (
                  <li key={m.itemId} className="flex flex-wrap items-baseline justify-between gap-2">
                    <span className="font-medium text-slate-900 dark:text-neutral-100">{m.title}</span>
                    <span className="text-amber-900 dark:text-amber-200">
                      {m.daysOverdue} day{m.daysOverdue === 1 ? '' : 's'} overdue
                    </span>
                  </li>
                ))}
              </ul>
            </section>
          )}

          <div className="mt-8">
            <div role="tablist" aria-label="Progress sections" className="flex flex-wrap gap-1 border-b border-slate-200 dark:border-neutral-700">
              {tabs.map((t) => (
                <button
                  key={t.id}
                  type="button"
                  role="tab"
                  id={`${tabsId}-tab-${t.id}`}
                  aria-selected={tab === t.id}
                  aria-controls={`${tabsId}-panel-${t.id}`}
                  tabIndex={tab === t.id ? 0 : -1}
                  onClick={() => setTab(t.id)}
                  className={`rounded-t-lg px-4 py-2 text-sm font-medium ${
                    tab === t.id
                      ? 'border border-b-0 border-slate-200 bg-white text-indigo-700 dark:border-neutral-600 dark:bg-neutral-900 dark:text-indigo-300'
                      : 'text-slate-600 hover:text-slate-900 dark:text-neutral-400 dark:hover:text-neutral-100'
                  }`}
                >
                  {t.label}
                </button>
              ))}
            </div>

            <div
              role="tabpanel"
              id={`${tabsId}-panel-${tab}`}
              aria-labelledby={`${tabsId}-tab-${tab}`}
              className="rounded-b-xl border border-t-0 border-slate-200 bg-white p-4 dark:border-neutral-700 dark:bg-neutral-900"
            >
              {tab === 'overview' && (
                <div className="space-y-4 text-sm text-slate-700 dark:text-neutral-300">
                  <p>
                    Average quiz score:{' '}
                    <strong>
                      {data.summary.avgQuizScore != null
                        ? `${Math.round(data.summary.avgQuizScore * 10) / 10}%`
                        : '—'}
                    </strong>
                  </p>
                  {!isSelf && (
                    <p>
                      <Link
                        to={`/courses/${encodeURIComponent(courseCode)}/gradebook?student=${encodeURIComponent(data.summary.studentUserId)}`}
                        className="font-medium text-indigo-600 hover:underline dark:text-indigo-400"
                      >
                        Open gradebook for this student
                      </Link>
                    </p>
                  )}
                </div>
              )}

              {tab === 'activity' && (
                <ul className="divide-y divide-slate-100 dark:divide-neutral-800">
                  {activity.map((ev, i) => (
                    <li
                      key={`${ev.occurredAt}-${i}`}
                      className="grid grid-cols-1 gap-1 py-3 text-sm sm:grid-cols-[minmax(11rem,13rem)_minmax(0,1fr)] sm:items-start sm:gap-x-4"
                    >
                      <time
                        className="shrink-0 text-xs tabular-nums text-slate-500 sm:text-end dark:text-neutral-500"
                        dateTime={ev.occurredAt}
                      >
                        {formatAbsolute(new Date(ev.occurredAt))}
                      </time>
                      <div className="min-w-0">
                        <p className="font-medium text-slate-900 dark:text-neutral-100">{ev.label}</p>
                        {ev.detail ? (
                          <p className="text-slate-600 dark:text-neutral-400">{ev.detail}</p>
                        ) : null}
                      </div>
                    </li>
                  ))}
                </ul>
              )}
              {tab === 'activity' && activityCursor && (
                <button
                  type="button"
                  disabled={activityLoading}
                  className="mt-4 text-sm font-medium text-indigo-600 hover:underline disabled:opacity-50 dark:text-indigo-400"
                  onClick={() => void loadActivity(false)}
                >
                  Load more
                </button>
              )}

              {tab === 'assignments' && (
                <div className="overflow-x-auto">
                  <table className="min-w-full text-sm">
                    <thead>
                      <tr className="border-b border-slate-200 text-start dark:border-neutral-700">
                        <th scope="col" className="py-2 pe-4 font-medium">
                          Item
                        </th>
                        <th scope="col" className="py-2 pe-4 font-medium">
                          Due
                        </th>
                        <th scope="col" className="py-2 pe-4 font-medium">
                          Submitted
                        </th>
                        <th scope="col" className="py-2 pe-4 font-medium">
                          Grade
                        </th>
                        <th scope="col" className="py-2 font-medium">
                          Status
                        </th>
                      </tr>
                    </thead>
                    <tbody>
                      {data.assignments.map((a) => (
                        <tr key={a.itemId} className="border-b border-slate-100 dark:border-neutral-800">
                          <td className="py-2 pe-4">{a.title}</td>
                          <td className="py-2 pe-4">
                            {a.dueAt ? formatAbsolute(new Date(a.dueAt)) : '—'}
                          </td>
                          <td className="py-2 pe-4">
                            {a.submittedAt ? formatAbsolute(new Date(a.submittedAt)) : '—'}
                          </td>
                          <td className="py-2 pe-4 tabular-nums">{a.grade}</td>
                          <td className="py-2">
                            <span
                              className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${statusBadgeClass(a.status)}`}
                            >
                              {a.status}
                            </span>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}

              {tab === 'quizzes' && (
                <>
                  <QuizScoreChart quizzes={data.quizzes} captionId={chartCaptionId} />
                  <h3 className="mt-8 text-sm font-semibold text-slate-900 dark:text-neutral-100">
                    All quiz attempts
                  </h3>
                  <div className="mt-3 overflow-x-auto">
                    <table className="min-w-full text-sm">
                      <thead>
                        <tr className="border-b border-slate-200 text-start dark:border-neutral-700">
                          <th scope="col" className="py-2 pe-4 font-medium">
                            Quiz
                          </th>
                          <th scope="col" className="py-2 pe-4 font-medium">
                            Submitted
                          </th>
                          <th scope="col" className="py-2 font-medium">
                            Score
                          </th>
                        </tr>
                      </thead>
                      <tbody>
                        {data.quizzes.map((q) => (
                          <tr key={q.attemptId} className="border-b border-slate-100 dark:border-neutral-800">
                            <td className="py-2 pe-4">{q.title}</td>
                            <td className="py-2 pe-4">{formatAbsolute(new Date(q.submittedAt))}</td>
                            <td className="py-2 tabular-nums">
                              {q.scorePercent != null ? `${Math.round(q.scorePercent * 10) / 10}%` : '—'}
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                </>
              )}

              {tab === 'notes' && data.summary.canManageNotes && (
                <div className="space-y-4">
                  <div className="rounded-lg border border-slate-200 p-3 dark:border-neutral-700">
                    <label htmlFor="progress-note" className="text-sm font-medium">
                      {editingNote ? 'Edit note' : 'Add private note'}
                    </label>
                    <textarea
                      id="progress-note"
                      rows={3}
                      value={noteDraft}
                      onChange={(e) => setNoteDraft(e.target.value)}
                      className="mt-2 w-full rounded-lg border border-slate-200 px-3 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-950"
                    />
                    <div className="mt-2 flex gap-2">
                      <button
                        type="button"
                        disabled={noteBusy || !noteDraft.trim()}
                        onClick={() => void onSaveNote()}
                        className="rounded-lg bg-indigo-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-indigo-500 disabled:opacity-50"
                      >
                        Save
                      </button>
                      {editingNote && (
                        <button
                          type="button"
                          onClick={() => {
                            setEditingNote(null)
                            setNoteDraft('')
                          }}
                          className="rounded-lg border border-slate-200 px-3 py-1.5 text-sm dark:border-neutral-600"
                        >
                          Cancel
                        </button>
                      )}
                    </div>
                  </div>
                  {(data.notes ?? []).map((n) => (
                    <article
                      key={n.id}
                      className="rounded-lg border border-slate-200 p-3 dark:border-neutral-700"
                    >
                      <p className="whitespace-pre-wrap text-sm text-slate-800 dark:text-neutral-200">
                        {n.noteText}
                      </p>
                      <p className="mt-2 text-xs text-slate-500 dark:text-neutral-500">
                        Updated {formatTimeAgoFromIso(n.updatedAt)}
                      </p>
                      <div className="mt-2 flex gap-2">
                        <button
                          type="button"
                          className="text-xs font-medium text-indigo-600 dark:text-indigo-400"
                          onClick={() => {
                            setEditingNote(n)
                            setNoteDraft(n.noteText)
                          }}
                        >
                          Edit
                        </button>
                        <button
                          type="button"
                          className="text-xs font-medium text-red-600 dark:text-red-400"
                          disabled={noteBusy}
                          onClick={() => void onDeleteNote(n.id)}
                        >
                          Delete
                        </button>
                      </div>
                    </article>
                  ))}
                </div>
              )}
            </div>
          </div>
        </>
      )}

      {messageOpen && data ? (
        <div
          className={LMS_MODAL_OVERLAY_CLASS}
          role="dialog"
          aria-modal="true"
          aria-labelledby="student-progress-message-title"
          onClick={(ev) => {
            if (ev.target === ev.currentTarget) closeMessageModal()
          }}
        >
          <div className="w-full max-w-lg overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-xl dark:border-neutral-700 dark:bg-neutral-900">
            <div className="flex items-center justify-between border-b border-slate-200 px-4 py-3 dark:border-neutral-700">
              <h3
                id="student-progress-message-title"
                className="text-sm font-semibold text-slate-900 dark:text-neutral-100"
              >
                Send message
              </h3>
              <button
                type="button"
                onClick={() => closeMessageModal()}
                className="rounded-lg p-1.5 text-slate-500 hover:bg-slate-100 hover:text-slate-800 dark:text-neutral-400 dark:hover:bg-neutral-800"
                aria-label="Close"
              >
                <X className="h-5 w-5" />
              </button>
            </div>
            <form
              onSubmit={(ev) => void onSubmitStudentMessage(ev)}
              className="space-y-3 px-4 py-4 text-sm text-slate-700 dark:text-neutral-300"
            >
              <p className="text-slate-600 dark:text-neutral-400">
                To{' '}
                <span className="font-medium text-slate-900 dark:text-neutral-100">
                  {data.summary.studentDisplayName?.trim() || '—'}
                </span>
              </p>
              {messageError ? (
                <p className="text-sm text-rose-700 dark:text-rose-300" role="alert">
                  {messageError}
                </p>
              ) : null}
              <label className="block">
                <span className="text-xs font-medium text-slate-600 dark:text-neutral-400">Subject</span>
                <input
                  value={messageSubject}
                  onChange={(ev) => setMessageSubject(ev.target.value)}
                  className="mt-1 w-full rounded-xl border border-slate-200 bg-white px-3 py-2 text-sm outline-none focus:border-indigo-300 focus:ring-2 focus:ring-indigo-500/20 dark:border-neutral-600 dark:bg-neutral-950 dark:text-neutral-100"
                  placeholder="Optional subject"
                  disabled={messageStatus === 'loading'}
                />
              </label>
              <label className="block">
                <span className="text-xs font-medium text-slate-600 dark:text-neutral-400">Message</span>
                <textarea
                  value={messageBody}
                  onChange={(ev) => setMessageBody(ev.target.value)}
                  rows={6}
                  className="mt-1 w-full resize-y rounded-xl border border-slate-200 bg-white px-3 py-2 text-sm outline-none focus:border-indigo-300 focus:ring-2 focus:ring-indigo-500/20 dark:border-neutral-600 dark:bg-neutral-950 dark:text-neutral-100"
                  placeholder="Write your message…"
                  disabled={messageStatus === 'loading'}
                  required
                />
              </label>
              <div className="flex justify-end gap-2 border-t border-slate-200 pt-3 dark:border-neutral-700">
                <button
                  type="button"
                  onClick={() => closeMessageModal()}
                  className="rounded-xl px-3 py-2 text-sm font-medium text-slate-600 hover:bg-slate-100 dark:text-neutral-300 dark:hover:bg-neutral-800"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  disabled={messageStatus === 'loading' || !messageBody.trim()}
                  className="inline-flex items-center gap-2 rounded-xl bg-indigo-600 px-4 py-2 text-sm font-semibold text-white hover:bg-indigo-500 disabled:cursor-not-allowed disabled:opacity-50"
                >
                  <Send className="h-4 w-4" aria-hidden />
                  {messageStatus === 'loading' ? 'Sending…' : 'Send'}
                </button>
              </div>
            </form>
          </div>
        </div>
      ) : null}
    </LmsPage>
  )
}
