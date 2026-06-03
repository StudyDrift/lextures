import { useCallback, useEffect, useRef, useState } from 'react'
import { Navigate, useParams } from 'react-router-dom'
import { useCourseNavFeatures } from '../../context/course-nav-features-context'
import { usePermissions } from '../../context/use-permissions'
import { courseItemCreatePermission } from '../../lib/courses-api'
import { useViewerEnrollmentRoles } from '../../lib/use-viewer-enrollment-roles'
import {
  ATTENDANCE_STATUS_OPTIONS,
  closeAttendanceSession,
  createAttendanceSession,
  getAttendanceSession,
  listAttendanceSessions,
  saveAttendanceRecords,
  selfReportAttendance,
  type AttendanceRecord,
  type AttendanceSession,
  type AttendanceSessionDetail,
  type AttendanceStatus,
  type CollectionMethod,
} from '../../lib/course-attendance-api'
import { authorizedFetch } from '../../lib/api'
import { toastMutationError, toastSaveOk } from '../../lib/lms-toast'
import { LmsPage } from './lms-page'

type Section = { id: string; sectionCode: string; name?: string | null }

function today(): string {
  return new Date().toISOString().slice(0, 10)
}

function studentLabel(rec: AttendanceRecord): string {
  return rec.displayName?.trim() || rec.studentUserId
}

function collectionMethodLabel(method: CollectionMethod): string {
  return method === 'roll_call' ? 'Roll call' : 'Self report'
}

function sessionStatusBadge(status: AttendanceSession['status']) {
  if (status === 'open') {
    return (
      <span className="inline-flex rounded-full bg-emerald-100 px-2 py-0.5 text-xs font-semibold text-emerald-800 dark:bg-emerald-900/40 dark:text-emerald-300">
        Open
      </span>
    )
  }
  return (
    <span className="inline-flex rounded-full bg-neutral-100 px-2 py-0.5 text-xs font-semibold text-neutral-600 dark:bg-neutral-800 dark:text-neutral-400">
      Closed
    </span>
  )
}

function CollectionMethodCard({
  method,
  selected,
  onSelect,
}: {
  method: CollectionMethod
  selected: boolean
  onSelect: () => void
}) {
  const isRollCall = method === 'roll_call'
  return (
    <button
      type="button"
      role="radio"
      aria-checked={selected}
      onClick={onSelect}
      className={`flex min-w-0 flex-1 flex-col rounded-xl border p-4 text-start transition ${
        selected
          ? 'border-indigo-500 bg-indigo-50/80 ring-2 ring-indigo-500/30 dark:border-indigo-400 dark:bg-indigo-950/40'
          : 'border-slate-200 bg-white hover:border-slate-300 dark:border-neutral-700 dark:bg-neutral-900/40 dark:hover:border-neutral-600'
      }`}
    >
      <span className="text-sm font-semibold text-slate-900 dark:text-neutral-100">
        {isRollCall ? 'Roll call' : 'Self report'}
      </span>
      <span className="mt-1 text-xs leading-relaxed text-slate-500 dark:text-neutral-400">
        {isRollCall
          ? 'Mark each student present, absent, or tardy from the roster.'
          : 'Students check in during an open window; you review and finalize.'}
      </span>
    </button>
  )
}

export default function CourseAttendance() {
  const { courseCode } = useParams<{ courseCode: string }>()
  const { attendanceEnabled, sectionsEnabled, loading: featuresLoading } = useCourseNavFeatures()
  const { allows, loading: permLoading } = usePermissions()
  const viewerRoles = useViewerEnrollmentRoles(courseCode ?? '')

  const [sessions, setSessions] = useState<AttendanceSession[]>([])
  const [activeSessionId, setActiveSessionId] = useState<string | null>(null)
  const [sessionDetail, setSessionDetail] = useState<AttendanceSessionDetail | null>(null)
  const [draft, setDraft] = useState<Record<string, AttendanceStatus>>({})
  const [sections, setSections] = useState<Section[]>([])
  const [selectedSection, setSelectedSection] = useState<string>('')
  const [collectionMethod, setCollectionMethod] = useState<CollectionMethod>('roll_call')
  const [gradebookEnabled, setGradebookEnabled] = useState(false)
  const [pointsPossible, setPointsPossible] = useState(10)
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [creating, setCreating] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const announceRef = useRef<HTMLDivElement>(null)

  const isStaff =
    !permLoading &&
    courseCode != null &&
    allows(courseItemCreatePermission(courseCode))
  const isStudent = viewerRoles?.includes('student') === true && !isStaff

  const announce = useCallback((msg: string) => {
    if (announceRef.current) {
      announceRef.current.textContent = msg
    }
  }, [])

  const refreshSessions = useCallback(async () => {
    if (!courseCode) return
    const list = await listAttendanceSessions(courseCode)
    setSessions(list)
  }, [courseCode])

  const loadSession = useCallback(
    async (sessionId: string) => {
      if (!courseCode) return
      const detail = await getAttendanceSession(courseCode, sessionId)
      setSessionDetail(detail)
      if (detail.records) {
        const map: Record<string, AttendanceStatus> = {}
        for (const r of detail.records) {
          map[r.studentUserId] = r.status
        }
        setDraft(map)
      }
    },
    [courseCode],
  )

  useEffect(() => {
    if (!courseCode || !attendanceEnabled) return
    void (async () => {
      setLoading(true)
      setError(null)
      try {
        await refreshSessions()
        if (sectionsEnabled) {
          const res = await authorizedFetch(
            `/api/v1/courses/${encodeURIComponent(courseCode)}/sections`,
          )
          if (res.ok) {
            const body = (await res.json()) as { sections?: Section[] }
            const list = body.sections ?? []
            setSections(list)
            if (list.length > 0) setSelectedSection(list[0].id)
          }
        }
      } catch (e) {
        setError(e instanceof Error ? e.message : 'Failed to load attendance.')
      } finally {
        setLoading(false)
      }
    })()
  }, [courseCode, attendanceEnabled, sectionsEnabled, refreshSessions])

  useEffect(() => {
    if (!isStudent || activeSessionId || sessions.length === 0) return
    const open = sessions.find(
      (s) => s.collectionMethod === 'self_report' && s.status === 'open',
    )
    if (open) setActiveSessionId(open.id)
  }, [isStudent, activeSessionId, sessions])

  useEffect(() => {
    if (!activeSessionId) {
      setSessionDetail(null)
      setDraft({})
      return
    }
    void loadSession(activeSessionId).catch((e: unknown) => {
      setError(e instanceof Error ? e.message : 'Failed to load session.')
    })
  }, [activeSessionId, loadSession])

  const handleCreateSession = async () => {
    if (!courseCode) return
    setCreating(true)
    setError(null)
    try {
      const sess = await createAttendanceSession(courseCode, {
        collectionMethod,
        sessionDate: today(),
        title: collectionMethod === 'roll_call' ? `Roll call — ${today()}` : `Self report — ${today()}`,
        sectionId: selectedSection || null,
        gradebookEnabled,
        pointsPossible: gradebookEnabled ? pointsPossible : undefined,
      })
      await refreshSessions()
      setActiveSessionId(sess.id)
      toastSaveOk('Session started.')
    } catch (e) {
      toastMutationError(e instanceof Error ? e.message : 'Failed to create session.')
      setError(e instanceof Error ? e.message : 'Failed to create session.')
    } finally {
      setCreating(false)
    }
  }

  const handleMarkAllPresent = () => {
    if (!sessionDetail?.records) return
    const next: Record<string, AttendanceStatus> = {}
    for (const r of sessionDetail.records) {
      next[r.studentUserId] = 'present'
    }
    setDraft(next)
    announce('All students marked present.')
  }

  const handleSave = async () => {
    if (!courseCode || !activeSessionId || !sessionDetail?.records) return
    setSaving(true)
    setError(null)
    try {
      const records = sessionDetail.records.map((r) => ({
        studentUserId: r.studentUserId,
        status: draft[r.studentUserId] ?? 'not_recorded',
        source: 'instructor' as const,
      }))
      const result = await saveAttendanceRecords(courseCode, activeSessionId, records)
      toastSaveOk(result.message)
      announce(result.message)
      await loadSession(activeSessionId)
    } catch (e) {
      toastMutationError(e instanceof Error ? e.message : 'Save failed.')
      setError(e instanceof Error ? e.message : 'Save failed.')
    } finally {
      setSaving(false)
    }
  }

  const handleClose = async () => {
    if (!courseCode || !activeSessionId) return
    setSaving(true)
    try {
      await closeAttendanceSession(courseCode, activeSessionId, true)
      await refreshSessions()
      await loadSession(activeSessionId)
      toastSaveOk('Session closed.')
    } catch (e) {
      toastMutationError(e instanceof Error ? e.message : 'Failed to close session.')
    } finally {
      setSaving(false)
    }
  }

  const handleSelfReport = async (status: 'present' | 'tardy') => {
    if (!courseCode || !activeSessionId) return
    setSaving(true)
    try {
      await selfReportAttendance(courseCode, activeSessionId, status)
      toastSaveOk('Check-in recorded.')
      await loadSession(activeSessionId)
    } catch (e) {
      toastMutationError(e instanceof Error ? e.message : 'Check-in failed.')
    } finally {
      setSaving(false)
    }
  }

  if (!courseCode) {
    return <p className="p-4 text-red-600">No course selected.</p>
  }

  if (featuresLoading || permLoading) {
    return (
      <LmsPage title="Attendance">
        <p className="text-sm text-slate-500" aria-busy="true">
          Loading…
        </p>
      </LmsPage>
    )
  }

  if (!attendanceEnabled) {
    return <Navigate to={`/courses/${encodeURIComponent(courseCode)}`} replace />
  }

  const openSelfReport =
    sessionDetail?.collectionMethod === 'self_report' &&
    sessionDetail.status === 'open' &&
    sessionDetail.canSelfReport

  return (
    <LmsPage
      title="Attendance"
      description={
        isStaff
          ? 'Take roll call or run a self-report check-in for your class.'
          : 'Check in when your instructor opens a self-report session.'
      }
    >
      <div
        ref={announceRef}
        role="status"
        aria-live="polite"
        aria-atomic="true"
        className="sr-only"
      />

      {error && (
        <div role="alert" className="mb-4 rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700 dark:border-red-900 dark:bg-red-950/40 dark:text-red-300">
          {error}
        </div>
      )}

      {loading ? (
        <p className="text-sm text-slate-500" aria-busy="true">
          Loading…
        </p>
      ) : (
        <>
          {isStaff && !activeSessionId && (
            <section className="mb-8 rounded-2xl border border-slate-200 bg-white p-5 shadow-sm shadow-slate-900/5 dark:border-neutral-800 dark:bg-neutral-950">
              <div className="border-b border-slate-100 pb-4 dark:border-neutral-800">
                <h2 className="text-base font-semibold text-slate-900 dark:text-neutral-100">New session</h2>
                <p className="mt-1 text-sm text-slate-500 dark:text-neutral-400">
                  Choose how to collect attendance, then start a session for today.
                </p>
              </div>

              <div className="mt-5 space-y-5">
                {sectionsEnabled && sections.length > 0 && (
                  <div className="max-w-xs">
                    <label
                      htmlFor="attendance-section"
                      className="block text-sm font-medium text-slate-700 dark:text-neutral-300"
                    >
                      Section
                    </label>
                    <select
                      id="attendance-section"
                      value={selectedSection}
                      onChange={(e) => setSelectedSection(e.target.value)}
                      className="mt-1.5 w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-900"
                    >
                      {sections.map((s) => (
                        <option key={s.id} value={s.id}>
                          {s.name ?? s.sectionCode}
                        </option>
                      ))}
                    </select>
                  </div>
                )}

                <fieldset>
                  <legend className="text-sm font-medium text-slate-700 dark:text-neutral-300">
                    Collection method
                  </legend>
                  <div
                    className="mt-2 grid gap-3 sm:grid-cols-2"
                    role="radiogroup"
                    aria-label="Collection method"
                  >
                    <CollectionMethodCard
                      method="roll_call"
                      selected={collectionMethod === 'roll_call'}
                      onSelect={() => setCollectionMethod('roll_call')}
                    />
                    <CollectionMethodCard
                      method="self_report"
                      selected={collectionMethod === 'self_report'}
                      onSelect={() => setCollectionMethod('self_report')}
                    />
                  </div>
                </fieldset>

                <div className="rounded-xl border border-slate-200 bg-slate-50/80 p-4 dark:border-neutral-800 dark:bg-neutral-900/50">
                  <label className="flex cursor-pointer items-start gap-3">
                    <input
                      type="checkbox"
                      checked={gradebookEnabled}
                      onChange={(e) => setGradebookEnabled(e.target.checked)}
                      className="mt-0.5"
                    />
                    <span>
                      <span className="block text-sm font-medium text-slate-900 dark:text-neutral-100">
                        Add to gradebook
                      </span>
                      <span className="mt-0.5 block text-xs text-slate-500 dark:text-neutral-400">
                        Creates a gradebook column when you close the session.
                      </span>
                    </span>
                  </label>

                  {gradebookEnabled && (
                    <div className="mt-4 border-t border-slate-200 pt-4 dark:border-neutral-700">
                      <label
                        htmlFor="attendance-points"
                        className="block text-sm font-medium text-slate-700 dark:text-neutral-300"
                      >
                        Points possible
                      </label>
                      <input
                        id="attendance-points"
                        type="number"
                        min={1}
                        value={pointsPossible}
                        onChange={(e) => setPointsPossible(Number(e.target.value) || 1)}
                        className="mt-1.5 w-28 rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-900"
                      />
                    </div>
                  )}
                </div>
              </div>

              <div className="mt-6 flex justify-end border-t border-slate-100 pt-4 dark:border-neutral-800">
                <button
                  type="button"
                  onClick={() => void handleCreateSession()}
                  disabled={creating}
                  className="rounded-lg bg-indigo-600 px-5 py-2.5 text-sm font-semibold text-white hover:bg-indigo-700 disabled:opacity-50"
                >
                  {creating ? 'Starting…' : 'Start session'}
                </button>
              </div>
            </section>
          )}

          {isStudent && openSelfReport && (
            <section className="mb-6 rounded-xl border border-emerald-200 bg-emerald-50 p-4 dark:border-emerald-900 dark:bg-emerald-950/30">
              <h2 className="text-sm font-semibold text-emerald-900 dark:text-emerald-200">Check in</h2>
              <p className="mt-1 text-sm text-emerald-800 dark:text-emerald-300">
                {sessionDetail?.title ?? 'Self-report session'} is open.
              </p>
              <div className="mt-3 flex gap-2">
                <button
                  type="button"
                  disabled={saving}
                  onClick={() => void handleSelfReport('present')}
                  className="rounded-lg bg-emerald-600 px-4 py-2 text-sm font-semibold text-white hover:bg-emerald-700 disabled:opacity-50"
                >
                  I&apos;m here
                </button>
                <button
                  type="button"
                  disabled={saving}
                  onClick={() => void handleSelfReport('tardy')}
                  className="rounded-lg border border-emerald-600 px-4 py-2 text-sm font-semibold text-emerald-800 hover:bg-emerald-100 disabled:opacity-50 dark:text-emerald-200"
                >
                  I&apos;m late
                </button>
              </div>
            </section>
          )}

          {sessions.length > 0 && (
            <section className="mb-8">
              <h2 className="mb-3 text-sm font-semibold text-slate-900 dark:text-neutral-100">Recent sessions</h2>
              <ul className="divide-y divide-slate-200 overflow-hidden rounded-2xl border border-slate-200 dark:divide-neutral-800 dark:border-neutral-800">
                {sessions.map((s) => (
                  <li key={s.id}>
                    <button
                      type="button"
                      onClick={() => setActiveSessionId(s.id)}
                      className={`flex w-full items-center justify-between gap-3 px-4 py-3.5 text-start transition hover:bg-slate-50 dark:hover:bg-neutral-900/60 ${
                        activeSessionId === s.id ? 'bg-indigo-50 dark:bg-indigo-950/30' : 'bg-white dark:bg-neutral-950'
                      }`}
                    >
                      <div className="min-w-0">
                        <p className="truncate text-sm font-medium text-slate-900 dark:text-neutral-100">{s.title}</p>
                        <p className="mt-0.5 text-xs text-slate-500 dark:text-neutral-400">
                          {s.sessionDate} · {collectionMethodLabel(s.collectionMethod)}
                          {s.gradebookEnabled ? ' · Gradebook' : ''}
                        </p>
                      </div>
                      {sessionStatusBadge(s.status)}
                    </button>
                  </li>
                ))}
              </ul>
            </section>
          )}

          {sessions.length === 0 && !isStaff && (
            <p className="text-sm text-slate-500">No open attendance sessions.</p>
          )}

          {isStaff && activeSessionId && sessionDetail?.records && sessionDetail.records.length > 0 && (
            <section className="rounded-2xl border border-slate-200 bg-white dark:border-neutral-800 dark:bg-neutral-950">
              <div className="flex flex-wrap items-center gap-3 border-b border-slate-100 px-4 py-3 dark:border-neutral-800">
                <div className="min-w-0 flex-1">
                  <h2 className="truncate text-sm font-semibold text-slate-900 dark:text-neutral-100">
                    {sessionDetail.title}
                  </h2>
                  <p className="mt-0.5 text-xs text-slate-500 dark:text-neutral-400">
                    {sessionDetail.records.length} students · {collectionMethodLabel(sessionDetail.collectionMethod)}
                  </p>
                </div>
                <button
                  type="button"
                  onClick={() => setActiveSessionId(null)}
                  className="text-xs font-medium text-slate-500 hover:text-slate-700 dark:text-neutral-400 dark:hover:text-neutral-200"
                >
                  Back to list
                </button>
              </div>

              {sessionDetail.status === 'open' && (
                <div className="flex flex-wrap gap-2 border-b border-slate-100 px-4 py-3 dark:border-neutral-800">
                  <button
                    type="button"
                    onClick={handleMarkAllPresent}
                    className="rounded-lg border border-emerald-600 px-3 py-1.5 text-xs font-semibold text-emerald-700 hover:bg-emerald-50 dark:text-emerald-300 dark:hover:bg-emerald-950/30"
                  >
                    Mark all present
                  </button>
                  <button
                    type="button"
                    onClick={() => void handleSave()}
                    disabled={saving}
                    className="rounded-lg bg-indigo-600 px-3 py-1.5 text-xs font-semibold text-white hover:bg-indigo-700 disabled:opacity-50"
                  >
                    {saving ? 'Saving…' : 'Save'}
                  </button>
                  <button
                    type="button"
                    onClick={() => void handleClose()}
                    disabled={saving}
                    className="rounded-lg border border-slate-300 px-3 py-1.5 text-xs font-semibold text-slate-700 hover:bg-slate-50 dark:border-neutral-600 dark:text-neutral-300 dark:hover:bg-neutral-900"
                  >
                    Close session
                  </button>
                </div>
              )}

              <div
                role="grid"
                aria-label="Attendance roll"
                aria-rowcount={sessionDetail.records.length + 1}
              >
                <div
                  role="row"
                  className="grid grid-cols-[1fr_180px] border-b bg-slate-50 text-sm font-medium dark:bg-neutral-800"
                >
                  <div role="columnheader" className="px-3 py-2">
                    Student
                  </div>
                  <div role="columnheader" className="px-3 py-2">
                    Status
                  </div>
                </div>
                {sessionDetail.records.map((student, idx) => {
                  const status = draft[student.studentUserId] ?? student.status
                  const rowClass =
                    status === 'absent'
                      ? 'bg-red-50 dark:bg-red-950/20'
                      : status === 'tardy'
                        ? 'bg-amber-50 dark:bg-amber-950/20'
                        : status === 'present' || status === 'excused'
                          ? 'bg-emerald-50 dark:bg-emerald-950/20'
                          : ''
                  return (
                    <div
                      key={student.studentUserId}
                      role="row"
                      aria-rowindex={idx + 2}
                      className={`grid grid-cols-[1fr_180px] border-b border-slate-100 text-sm last:border-0 dark:border-neutral-800 ${rowClass}`}
                    >
                      <div role="gridcell" className="px-3 py-2">
                        {studentLabel(student)}
                      </div>
                      <div role="gridcell" className="px-3 py-2">
                        <select
                          value={status}
                          disabled={sessionDetail.status === 'closed'}
                          onChange={(e) => {
                            const v = e.target.value as AttendanceStatus
                            setDraft((prev) => ({ ...prev, [student.studentUserId]: v }))
                            announce(`Status changed to ${v}.`)
                          }}
                          aria-label={`Attendance status for ${studentLabel(student)}`}
                          className="w-full rounded border border-slate-300 px-2 py-0.5 text-sm dark:border-neutral-600 dark:bg-neutral-800"
                        >
                          {ATTENDANCE_STATUS_OPTIONS.map((opt) => (
                            <option key={opt.value} value={opt.value}>
                              {opt.label}
                            </option>
                          ))}
                        </select>
                      </div>
                    </div>
                  )
                })}
              </div>
            </section>
          )}

          {isStaff && activeSessionId && sessionDetail?.records?.length === 0 && (
            <p className="text-sm text-slate-500">No students enrolled for this session scope.</p>
          )}
        </>
      )}
    </LmsPage>
  )
}
