import { useCallback, useEffect, useMemo, useState } from 'react'
import { Link, Navigate, useParams } from 'react-router-dom'
import { FlaskConical } from 'lucide-react'
import {
  fetchCourseMyGrades,
  fetchMyGradeItemHistory,
  type AssignmentGroup,
  type CourseGradebookGridColumn,
  type GradeHistoryEvent,
  type GradingSchemeSummary,
  viewerShouldShowMyGradesNav,
} from '../../lib/courses-api'
import { useCourseViewAs } from '../../lib/course-view-as'
import { useViewerEnrollmentRoles } from '../../lib/use-viewer-enrollment-roles'
import {
  computeCourseFinalPercent,
  computeDroppedGrades,
  computeWhatIfFinalPercent,
  formatFinalPercent,
  recordWhatIfSession,
  type AssignmentGroupWeight,
  type GradebookColumnForFinal,
} from './gradebook/compute-course-final-percent'
import { GradeHistoryPanel } from '../../components/grading/grade-history-panel'
import { WhatIfGradesPanel } from '../../components/grading/what-if-grades-panel'
import { usePlatformFeatures } from '../../context/platform-features-context'
import { FolderOpen } from 'lucide-react'
import { LmsPage } from './lms-page'

function parseEarned(raw: string | undefined): number {
  const t = (raw ?? '').trim()
  if (!t) return 0
  const n = Number.parseFloat(t.replace(/,/g, ''))
  return Number.isFinite(n) ? n : 0
}

function formatRowPercent(earned: number, max: number | null): string {
  if (max == null || max <= 0) return '—'
  return `${Math.round((earned / max) * 1000) / 10}%`
}

function whatIfStorageKey(courseCode: string): string {
  return `whatif:${courseCode}`
}

export default function CourseMyGrades() {
  const { courseCode } = useParams<{ courseCode: string }>()
  const { ffEportfolio, ffWhatifGrades } = usePlatformFeatures()
  const viewerEnrollmentRoles = useViewerEnrollmentRoles(courseCode)
  const courseViewPreview = useCourseViewAs(courseCode)
  const [loadState, setLoadState] = useState<'idle' | 'loading' | 'ok' | 'error'>('idle')
  const [loadError, setLoadError] = useState<string | null>(null)
  const [columns, setColumns] = useState<CourseGradebookGridColumn[]>([])
  const [grades, setGrades] = useState<Record<string, string>>({})
  const [displayGrades, setDisplayGrades] = useState<Record<string, string>>({})
  const [assignmentGroups, setAssignmentGroups] = useState<AssignmentGroup[]>([])
  const [gradingScheme, setGradingScheme] = useState<GradingSchemeSummary | null>(null)
  const [heldGradeItemIds, setHeldGradeItemIds] = useState<string[]>([])
  const [droppedGrades, setDroppedGrades] = useState<Record<string, boolean>>({})
  const [gradeStatuses, setGradeStatuses] = useState<Record<string, string>>({})
  const [whatIfMode, setWhatIfMode] = useState(false)
  const [whatIfOverrides, setWhatIfOverrides] = useState<Record<string, string>>({})
  const [targetLetter, setTargetLetter] = useState('B')
  const [historyItem, setHistoryItem] = useState<{
    id: string
    title: string
  } | null>(null)
  const [historyLoad, setHistoryLoad] = useState<'idle' | 'loading' | 'ok' | 'error'>('idle')
  const [historyErr, setHistoryErr] = useState<string | null>(null)
  const [historyEvents, setHistoryEvents] = useState<GradeHistoryEvent[] | null>(null)

  const canView = useMemo(() => {
    if (!courseCode) return false
    if (viewerEnrollmentRoles === null && courseViewPreview !== 'student') return false
    return viewerShouldShowMyGradesNav(viewerEnrollmentRoles, courseViewPreview)
  }, [courseCode, viewerEnrollmentRoles, courseViewPreview])

  const load = useCallback(async () => {
    if (!courseCode) return
    setLoadState('loading')
    setLoadError(null)
    try {
      const data = await fetchCourseMyGrades(courseCode)
      setColumns(data.columns)
      setGrades(data.grades)
      setDisplayGrades(data.displayGrades ?? {})
      setAssignmentGroups(data.assignmentGroups)
      setGradingScheme(data.gradingScheme ?? null)
      setHeldGradeItemIds(data.heldGradeItemIds ?? [])
      setDroppedGrades(data.droppedGrades ?? {})
      setGradeStatuses(data.gradeStatuses ?? {})
      setLoadState('ok')
    } catch (e: unknown) {
      setLoadState('error')
      setLoadError(e instanceof Error ? e.message : 'Could not load grades.')
    }
  }, [courseCode])

  useEffect(() => {
    if (!courseCode || !canView) return
    let cancelled = false
    queueMicrotask(() => {
      if (cancelled) return
      void load()
    })
    return () => {
      cancelled = true
    }
  }, [courseCode, canView, load])

  useEffect(() => {
    if (!courseCode || !historyItem) return
    let cancelled = false
    void (async () => {
      try {
        const d = await fetchMyGradeItemHistory(courseCode, historyItem.id)
        if (cancelled) return
        setHistoryEvents(d.events)
        setHistoryLoad('ok')
      } catch (e: unknown) {
        if (cancelled) return
        setHistoryErr(e instanceof Error ? e.message : 'Could not load history')
        setHistoryLoad('error')
      }
    })()
    return () => {
      cancelled = true
    }
  }, [courseCode, historyItem])

  useEffect(() => {
    if (!courseCode || !ffWhatifGrades) return
    try {
      const raw = sessionStorage.getItem(whatIfStorageKey(courseCode))
      if (!raw) return
      const parsed = JSON.parse(raw) as { overrides?: Record<string, string>; mode?: boolean }
      if (parsed.overrides && typeof parsed.overrides === 'object') {
        setWhatIfOverrides(parsed.overrides)
      }
      if (parsed.mode === true) setWhatIfMode(true)
    } catch {
      // ignore corrupt session data
    }
  }, [courseCode, ffWhatifGrades])

  useEffect(() => {
    if (!courseCode || !ffWhatifGrades) return
    try {
      sessionStorage.setItem(
        whatIfStorageKey(courseCode),
        JSON.stringify({ overrides: whatIfOverrides, mode: whatIfMode }),
      )
    } catch {
      // ignore storage errors
    }
  }, [courseCode, ffWhatifGrades, whatIfOverrides, whatIfMode])

  const openGradeHistory = useCallback((id: string, title: string) => {
    setHistoryErr(null)
    setHistoryEvents(null)
    setHistoryLoad('loading')
    setHistoryItem({ id, title })
  }, [])

  const closeGradeHistory = useCallback(() => {
    setHistoryItem(null)
    setHistoryLoad('idle')
    setHistoryErr(null)
    setHistoryEvents(null)
  }, [])

  const toggleWhatIfMode = useCallback(() => {
    setWhatIfMode((prev) => {
      const next = !prev
      if (next) recordWhatIfSession()
      return next
    })
  }, [])

  const resetWhatIf = useCallback(() => {
    setWhatIfOverrides({})
  }, [])

  const setOverride = useCallback((itemId: string, value: string) => {
    setWhatIfOverrides((prev) => {
      const t = value.trim()
      if (t === '') {
        const next = { ...prev }
        delete next[itemId]
        return next
      }
      return { ...prev, [itemId]: t }
    })
  }, [])

  const heldSet = useMemo(() => new Set(heldGradeItemIds), [heldGradeItemIds])

  const finalCols: GradebookColumnForFinal[] = useMemo(
    () =>
      columns.map((c) => ({
        id: c.id,
        maxPoints: c.maxPoints,
        assignmentGroupId: c.assignmentGroupId ?? null,
        neverDrop: c.neverDrop === true,
        replaceWithFinal: c.replaceWithFinal === true,
        dueAt: c.dueAt ?? null,
      })),
    [columns],
  )

  const calcCols: GradebookColumnForFinal[] = useMemo(
    () => finalCols.filter((c) => !heldSet.has(c.id)),
    [finalCols, heldSet],
  )

  const groupsForFinal: AssignmentGroupWeight[] = useMemo(
    () =>
      assignmentGroups.map((g) => ({
        id: g.id,
        weightPercent: g.weightPercent,
        dropLowest: g.dropLowest,
        dropHighest: g.dropHighest,
        replaceLowestWithFinal: g.replaceLowestWithFinal,
      })),
    [assignmentGroups],
  )

  const excusedByItem = useMemo(() => {
    const o: Record<string, boolean> = {}
    for (const [k, v] of Object.entries(gradeStatuses)) {
      if (v === 'excused') o[k] = true
    }
    return o
  }, [gradeStatuses])

  const actualPct = useMemo(
    () => computeCourseFinalPercent(calcCols, grades, groupsForFinal, excusedByItem),
    [calcCols, grades, groupsForFinal, excusedByItem],
  )

  const projectedPct = useMemo(
    () =>
      computeWhatIfFinalPercent(
        calcCols,
        grades,
        groupsForFinal,
        excusedByItem,
        whatIfOverrides,
        heldSet,
      ),
    [calcCols, grades, groupsForFinal, excusedByItem, whatIfOverrides, heldSet],
  )

  const activeDropped = useMemo(() => {
    if (!whatIfMode || Object.keys(whatIfOverrides).length === 0) return droppedGrades
    return computeDroppedGrades(calcCols, grades, groupsForFinal, excusedByItem, {
      mode: 'whatIf',
      whatIfOverrides,
      heldItemIds: heldSet,
    })
  }, [
    whatIfMode,
    whatIfOverrides,
    droppedGrades,
    calcCols,
    grades,
    groupsForFinal,
    excusedByItem,
    heldSet,
  ])

  const hasOverrides = Object.keys(whatIfOverrides).length > 0
  const showWhatIf = ffWhatifGrades === true

  const base = `/courses/${encodeURIComponent(courseCode ?? '')}`

  if (!courseCode) {
    return <Navigate to="/courses" replace />
  }

  if (viewerEnrollmentRoles === null && courseViewPreview !== 'student') {
    return null
  }

  if (!canView) {
    return <Navigate to={`/courses/${encodeURIComponent(courseCode)}`} replace />
  }

  return (
    <LmsPage
      title="My grades"
      description="Your earned points and course average from the gradebook. Contact your instructor if something looks wrong."
      actions={
        ffEportfolio ? (
          <Link
            to="/portfolios"
            className="inline-flex items-center gap-1.5 rounded-md border px-3 py-1.5 text-sm font-medium hover:bg-accent"
          >
            <FolderOpen className="h-4 w-4" aria-hidden />
            Add to Portfolio
          </Link>
        ) : undefined
      }
    >
      {loadState === 'loading' && (
        <p className="mt-6 text-sm text-slate-600 dark:text-neutral-400">Loading grades…</p>
      )}
      {loadState === 'error' && loadError && (
        <p className="mt-6 text-sm text-red-600 dark:text-red-400" role="alert">
          {loadError}
        </p>
      )}
      {loadState === 'ok' && (
        <>
          {showWhatIf ? (
            <WhatIfGradesPanel
              whatIfMode={whatIfMode}
              onToggleMode={toggleWhatIfMode}
              onReset={resetWhatIf}
              hasOverrides={hasOverrides}
              projectedPercent={projectedPct}
              actualPercent={actualPct}
              gradingScheme={gradingScheme}
              columns={calcCols}
              actualGrades={grades}
              assignmentGroups={groupsForFinal}
              excusedByItemId={excusedByItem}
              heldItemIds={heldSet}
              whatIfOverrides={whatIfOverrides}
              targetLetter={targetLetter}
              onTargetLetterChange={setTargetLetter}
            />
          ) : (
            <div className="mt-6 rounded-xl border border-slate-200 bg-white px-4 py-4 shadow-sm dark:border-neutral-700 dark:bg-neutral-900">
              <p className="text-2xl font-semibold tracking-tight text-slate-900 dark:text-neutral-100">
                Course grade: {formatFinalPercent(actualPct)}
              </p>
              <p className="mt-1 text-sm text-slate-600 dark:text-neutral-400">
                Weighted from assignment groups when your instructor has configured weights; otherwise
                by points earned vs points possible.
              </p>
            </div>
          )}
          {columns.length === 0 ? (
            <p className="mt-6 text-sm text-slate-600 dark:text-neutral-400">
              No graded assignments or quizzes are listed in this course yet.
            </p>
          ) : (
            <div className="mt-6 overflow-x-auto rounded-xl border border-slate-200 bg-white shadow-sm dark:border-neutral-700 dark:bg-neutral-900">
              <table className="min-w-full divide-y divide-slate-200 text-start text-sm dark:divide-neutral-700">
                <thead>
                  <tr className="bg-slate-50 dark:bg-neutral-800/80">
                    <th className="px-4 py-3 font-semibold text-slate-900 dark:text-neutral-100">
                      Assignment
                    </th>
                    <th className="px-4 py-3 font-semibold text-slate-900 dark:text-neutral-100">
                      Type
                    </th>
                    <th className="px-4 py-3 font-semibold text-slate-900 dark:text-neutral-100">
                      Earned
                    </th>
                    <th className="px-4 py-3 font-semibold text-slate-900 dark:text-neutral-100">
                      Possible
                    </th>
                    <th className="px-4 py-3 font-semibold text-slate-900 dark:text-neutral-100">
                      Item %
                    </th>
                    <th className="px-4 py-3 font-semibold text-slate-900 dark:text-neutral-100">Policy</th>
                    <th className="px-4 py-3 font-semibold text-slate-900 dark:text-neutral-100">History</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-200 dark:divide-neutral-700">
                  {columns.map((col) => {
                    const held = heldSet.has(col.id)
                    const excused = !held && gradeStatuses[col.id] === 'excused'
                    const dropped = !held && !excused && activeDropped[col.id] === true
                    const hasOverride = (whatIfOverrides[col.id] ?? '').trim() !== ''
                    const isHypothetical = whatIfMode && hasOverride
                    const earned = held ? undefined : grades[col.id]
                    const display = held ? undefined : displayGrades[col.id]
                    const effectiveEarned = isHypothetical
                      ? parseEarned(whatIfOverrides[col.id])
                      : held || excused
                        ? 0
                        : parseEarned(earned)
                    const max = col.maxPoints
                    const editable =
                      showWhatIf && whatIfMode && !excused && max != null && max > 0
                    const href =
                      col.kind === 'quiz'
                        ? `${base}/modules/quiz/${encodeURIComponent(col.id)}`
                        : `${base}/modules/assignment/${encodeURIComponent(col.id)}`
                    return (
                      <tr
                        key={col.id}
                        className={`hover:bg-slate-50/80 dark:hover:bg-neutral-800/80 ${dropped ? 'text-slate-500 dark:text-neutral-500' : ''}`}
                      >
                        <td className="px-4 py-3 font-medium text-slate-900 dark:text-neutral-100">
                          <Link
                            to={href}
                            className="text-indigo-600 hover:underline dark:text-indigo-400"
                          >
                            {col.title}
                          </Link>
                        </td>
                        <td className="px-4 py-3 capitalize text-slate-600 dark:text-neutral-400">
                          {col.kind === 'quiz' ? 'Quiz' : 'Assignment'}
                        </td>
                        <td
                          className={`px-4 py-3 text-slate-800 dark:text-neutral-200 ${dropped ? 'line-through decoration-slate-400' : ''}`}
                          aria-label={
                            isHypothetical
                              ? `Hypothetical score ${whatIfOverrides[col.id]}`
                              : excused
                                ? 'Excused — not counted toward your grade for this item'
                                : dropped
                                  ? 'Score shown but dropped from course total by group policy'
                                  : undefined
                          }
                        >
                          {editable ? (
                            <div className="flex items-center gap-2">
                              <input
                                type="number"
                                min={0}
                                max={max ?? undefined}
                                step="any"
                                className="w-24 rounded-md border border-indigo-300 bg-white px-2 py-1 text-sm dark:border-indigo-700 dark:bg-neutral-900"
                                aria-label={`Hypothetical score for ${col.title}`}
                                placeholder={held ? 'Hypothetical' : (earned ?? '').trim() || '—'}
                                value={whatIfOverrides[col.id] ?? ''}
                                onChange={(e) => setOverride(col.id, e.target.value)}
                              />
                              {isHypothetical ? (
                                <span
                                  className="inline-flex items-center gap-0.5 rounded border border-indigo-300 bg-indigo-50 px-1.5 py-0.5 text-xs font-semibold text-indigo-900 dark:border-indigo-700 dark:bg-indigo-950/60 dark:text-indigo-100"
                                  title="Hypothetical score"
                                >
                                  <FlaskConical className="h-3 w-3" aria-hidden />
                                  <span className="sr-only">Hypothetical</span>
                                </span>
                              ) : null}
                            </div>
                          ) : held ? (
                            <span className="text-amber-800 dark:text-amber-200/90" title="Grades not yet released">
                              Grades pending
                            </span>
                          ) : excused ? (
                            <span
                              className="inline-flex rounded-md border border-slate-200 bg-slate-100 px-2 py-0.5 text-xs font-semibold text-slate-800 dark:border-neutral-600 dark:bg-neutral-800 dark:text-neutral-200"
                              title="Excused — does not affect your course grade for this item."
                            >
                              {(display ?? '').trim() || 'EX'}
                            </span>
                          ) : (display ?? '').trim() ? (
                            display
                          ) : (earned ?? '').trim() ? (
                            earned
                          ) : (
                            '—'
                          )}
                        </td>
                        <td className="px-4 py-3 text-slate-800 dark:text-neutral-200">
                          {max != null && max > 0 ? String(max) : '—'}
                        </td>
                        <td className="px-4 py-3 text-slate-800 dark:text-neutral-200">
                          {excused ? '—' : formatRowPercent(effectiveEarned, max)}
                        </td>
                        <td className="px-4 py-3 text-slate-600 dark:text-neutral-400">
                          {excused ? (
                            <span
                              className="inline-flex rounded-md border border-slate-200/90 bg-slate-50 px-1.5 py-0.5 text-xs font-medium text-slate-800 dark:border-neutral-600 dark:bg-neutral-800/60 dark:text-neutral-200"
                              title="Exempt from the course average; does not affect your final percentage."
                            >
                              Excused
                            </span>
                          ) : dropped ? (
                            <span
                              className="inline-flex rounded-md border border-amber-200/80 bg-amber-50 px-1.5 py-0.5 text-xs font-medium text-amber-900 dark:border-amber-800/60 dark:bg-amber-950/50 dark:text-amber-200"
                              title="This score is excluded from your course total by the group’s drop rules."
                            >
                              Dropped
                            </span>
                          ) : (
                            '—'
                          )}
                        </td>
                        <td className="px-4 py-3">
                          <button
                            type="button"
                            className="text-sm text-indigo-600 hover:underline dark:text-indigo-400"
                            onClick={() => openGradeHistory(col.id, col.title)}
                          >
                            View
                          </button>
                        </td>
                      </tr>
                    )
                  })}
                </tbody>
              </table>
            </div>
          )}
        </>
      )}
      {historyItem && courseCode && (
        <div
          className="fixed inset-0 z-[80] flex items-end justify-center bg-black/40 p-4 sm:items-center"
          role="presentation"
        >
          <button
            type="button"
            className="absolute inset-0 cursor-default"
            aria-label="Close"
            onClick={() => closeGradeHistory()}
          />
          <div
            role="dialog"
            aria-modal="true"
            className="relative z-[1] w-full max-w-md rounded-xl border border-slate-200 bg-white p-5 shadow-xl dark:border-neutral-700 dark:bg-neutral-900"
            onClick={(e) => e.stopPropagation()}
          >
            <button
              type="button"
              className="absolute end-3 top-3 rounded p-1 text-slate-500 hover:bg-slate-100 dark:hover:bg-neutral-800"
              onClick={() => closeGradeHistory()}
            >
              <span className="sr-only">Close</span>✕
            </button>
            <GradeHistoryPanel
              title={historyItem.title}
              events={historyEvents}
              loading={historyLoad === 'loading'}
              error={historyErr}
            />
            <div className="mt-4 flex justify-end">
              <button
                type="button"
                className="rounded-lg border border-slate-200 bg-white px-3 py-1.5 text-sm text-slate-800 hover:bg-slate-50 dark:border-neutral-600 dark:bg-neutral-800"
                onClick={() => closeGradeHistory()}
              >
                Close
              </button>
            </div>
          </div>
        </div>
      )}
    </LmsPage>
  )
}
