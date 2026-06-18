import { useCallback, useEffect, useId, useMemo, useState } from 'react'
import { Link, Navigate, useParams } from 'react-router-dom'
import { EnrollmentAvatar } from '../../components/enrollment/enrollment-avatar'
import { useCoursePageTitle } from '../../context/course-document-title-context'
import { LmsPage } from './lms-page'
import { fetchCourse } from '../../lib/courses-api'
import { formatTimeAgoFromIso } from '../../lib/format-time-ago'
import { formatAbsolute } from '../../lib/format-datetime'
import { formatDate } from '../../lib/format'
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

  const isSelf = viewerEnrollmentId != null && enrollmentId === viewerEnrollmentId

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

  return (
    <LmsPage
      title={pageTitle}
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
                    <li key={`${ev.occurredAt}-${i}`} className="flex gap-3 py-3 text-sm">
                      <time
                        className="shrink-0 text-xs text-slate-500 dark:text-neutral-500"
                        dateTime={ev.occurredAt}
                      >
                        {formatAbsolute(new Date(ev.occurredAt))}
                      </time>
                      <div>
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
    </LmsPage>
  )
}
