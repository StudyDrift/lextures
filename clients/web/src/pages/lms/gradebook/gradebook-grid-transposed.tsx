import { useCallback, useEffect, useRef, useState } from 'react'
import { Link } from 'react-router-dom'
import { Lock } from 'lucide-react'
import { studentProgressFeatureEnabled } from '../../../lib/student-progress'
import { formatFinalPercent } from './compute-course-final-percent'
import type { GradebookColumn, GradebookStudent } from './gradebook-grid-types'
import {
  getGradebookCellMenuItems,
  openGradebookCellMenuFromButton,
  openGradebookCellMenuFromEvent,
  type GradebookCellMenuState,
} from './gradebook-cell-menu-utils'
import {
  GradebookCellMenuItems,
  GradebookCellMenuPortal,
  GradebookCellMenuTrigger,
} from './gradebook-cell-menu'

function parseGradeNumber(raw: string): number | null {
  const s = raw.trim().replace(/,/g, '')
  if (s === '') return null
  const n = Number.parseFloat(s)
  return Number.isFinite(n) ? n : null
}

function formatStat(n: number): string {
  if (!Number.isFinite(n)) return '—'
  if (Math.abs(n - Math.round(n)) < 1e-6) return String(Math.round(n))
  let s = n.toFixed(2)
  while (s.includes('.') && (s.endsWith('0') || s.endsWith('.'))) {
    s = s.slice(0, -1)
  }
  return s
}

function heatMapCellClass(t: number): string {
  if (!Number.isFinite(t)) return ''
  const u = Math.max(0, Math.min(1, t))
  if (u <= 0.17) return 'bg-sky-100/90 dark:bg-sky-950/35'
  if (u <= 0.33) return 'bg-sky-50/80 dark:bg-sky-950/20'
  if (u <= 0.5) return 'bg-slate-50 dark:bg-neutral-800/70'
  if (u <= 0.67) return 'bg-amber-50/90 dark:bg-amber-950/25'
  if (u <= 0.83) return 'bg-amber-100/85 dark:bg-amber-950/40'
  return 'bg-orange-100/80 dark:bg-orange-950/45'
}

function gradingSelectOptions(
  col: GradebookColumn,
  scheme: { type: string; scaleJson: unknown } | null | undefined,
): string[] {
  const eff = col.effectiveDisplayType ?? 'points'
  if (col.rubric) return []
  if (eff === 'pass_fail') return ['Pass', 'Fail']
  if (eff === 'complete_incomplete') return ['Complete', 'Incomplete']
  if (eff === 'letter' || eff === 'gpa') {
    const raw = scheme?.scaleJson
    if (!Array.isArray(raw)) return []
    const labels = raw
      .map((x) =>
        x && typeof x === 'object' && 'label' in x ? String((x as { label: unknown }).label).trim() : '',
      )
      .filter(Boolean)
    return [...new Set(labels)]
  }
  return []
}

type AssignmentColumnStats = { avg: number | null; med: number | null }

const MIN_ASSIGNMENT_COL_WIDTH_PX = 128
const MAX_ASSIGNMENT_COL_WIDTH_PX = 480
const TRANSPOSED_ASSIGNMENT_COL_WIDTH_KEY = 'lextures.gradebook.transposedAssignmentColWidth'

function clampAssignmentColWidth(px: number): number {
  return Math.round(Math.max(MIN_ASSIGNMENT_COL_WIDTH_PX, Math.min(MAX_ASSIGNMENT_COL_WIDTH_PX, px)))
}

function readStoredAssignmentColWidth(defaultPx: number): number {
  if (typeof window === 'undefined') return defaultPx
  try {
    const raw = window.localStorage.getItem(TRANSPOSED_ASSIGNMENT_COL_WIDTH_KEY)
    if (!raw) return defaultPx
    const n = Number.parseInt(raw, 10)
    if (!Number.isFinite(n)) return defaultPx
    return clampAssignmentColWidth(n)
  } catch {
    return defaultPx
  }
}

function assignmentColSurfaceClass(extra = ''): string {
  return [
    'sticky start-0 border-e border-slate-200 dark:border-neutral-700',
    extra,
  ].join(' ')
}

export type GradebookTransposedTableProps = {
  columns: GradebookColumn[]
  students: GradebookStudent[]
  grades: Record<string, Record<string, string>>
  onGradeChange: (studentId: string, columnId: string, value: string) => void
  readOnly: boolean
  gradingScheme?: { type: string; scaleJson: unknown } | null
  gradeExcused?: Record<string, Record<string, boolean>>
  gradeHeld?: Record<string, Record<string, boolean>>
  droppedGrades?: Record<string, Record<string, boolean>>
  finalPercentByStudentId: Record<string, number | null>
  assignmentStats: AssignmentColumnStats[]
  colorScaleEnabled: boolean
  courseCode?: string
  highlightStudentId?: string | null
  onRubricClick?: (studentId: string, columnId: string) => void
  onGradeSubmission?: (studentId: string, columnId: string) => void
  onOpenGradeHistory?: (studentId: string, columnId: string) => void
  onToggleExcused?: (studentId: string, columnId: string, excused: boolean) => void | Promise<void>
  onPostAssignmentGrades?: (itemId: string) => void
  postGradesPending?: string | null
  pad: string
  defaultAssignmentColWidthPx: number
  studentColMin: string
}

type EditingCell = { studentId: string; columnId: string }

function displayCellValue(
  grades: Record<string, Record<string, string>>,
  gradeExcused: Record<string, Record<string, boolean>> | undefined,
  studentId: string,
  columnId: string,
): string {
  if (gradeExcused?.[studentId]?.[columnId] === true) return 'EX'
  return grades[studentId]?.[columnId] ?? ''
}

export function GradebookTransposedTable({
  columns,
  students,
  grades,
  onGradeChange,
  readOnly,
  gradingScheme = null,
  gradeExcused,
  gradeHeld,
  droppedGrades,
  finalPercentByStudentId,
  assignmentStats,
  colorScaleEnabled,
  courseCode,
  highlightStudentId,
  onRubricClick,
  onGradeSubmission,
  onOpenGradeHistory,
  onToggleExcused,
  onPostAssignmentGrades,
  postGradesPending,
  pad,
  defaultAssignmentColWidthPx,
  studentColMin,
}: GradebookTransposedTableProps) {
  const [editing, setEditing] = useState<EditingCell | null>(null)
  const [draft, setDraft] = useState('')
  const [cellMenu, setCellMenu] = useState<GradebookCellMenuState>(null)
  const [assignmentColWidthPx, setAssignmentColWidthPx] = useState(() =>
    readStoredAssignmentColWidth(defaultAssignmentColWidthPx),
  )
  const [resizing, setResizing] = useState(false)
  const resizeRef = useRef<{ startX: number; startWidth: number } | null>(null)

  useEffect(() => {
    setAssignmentColWidthPx(readStoredAssignmentColWidth(defaultAssignmentColWidthPx))
  }, [defaultAssignmentColWidthPx])

  useEffect(() => {
    try {
      window.localStorage.setItem(TRANSPOSED_ASSIGNMENT_COL_WIDTH_KEY, String(assignmentColWidthPx))
    } catch {
      /* ignore */
    }
  }, [assignmentColWidthPx])

  const assignmentColStyle = {
    width: assignmentColWidthPx,
    minWidth: assignmentColWidthPx,
    maxWidth: assignmentColWidthPx,
  } as const

  const onResizePointerDown = useCallback(
    (e: React.PointerEvent<HTMLButtonElement>) => {
      e.preventDefault()
      e.stopPropagation()
      resizeRef.current = { startX: e.clientX, startWidth: assignmentColWidthPx }
      setResizing(true)
      e.currentTarget.setPointerCapture(e.pointerId)
    },
    [assignmentColWidthPx],
  )

  const onResizePointerMove = useCallback((e: React.PointerEvent<HTMLButtonElement>) => {
    const drag = resizeRef.current
    if (!drag) return
    const next = clampAssignmentColWidth(drag.startWidth + (e.clientX - drag.startX))
    setAssignmentColWidthPx(next)
  }, [])

  const finishResize = useCallback((e: React.PointerEvent<HTMLButtonElement>) => {
    if (!resizeRef.current) return
    resizeRef.current = null
    setResizing(false)
    if (e.currentTarget.hasPointerCapture(e.pointerId)) {
      e.currentTarget.releasePointerCapture(e.pointerId)
    }
  }, [])

  const heatPercentForCell = useCallback(
    (col: GradebookColumn, valStr: string): number | null => {
      const n = parseGradeNumber(valStr)
      if (n == null) return null
      if (col.maxPoints != null && col.maxPoints > 0) {
        return Math.max(0, Math.min(1, n / col.maxPoints))
      }
      const nums: number[] = []
      for (const s of students) {
        if (gradeExcused?.[s.id]?.[col.id] === true) continue
        const v = parseGradeNumber(grades[s.id]?.[col.id] ?? '')
        if (v != null) nums.push(v)
      }
      if (nums.length === 0) return 0.5
      const min = Math.min(...nums)
      const max = Math.max(...nums)
      if (max <= min) return 0.5
      return Math.max(0, Math.min(1, (n - min) / (max - min)))
    },
    [students, grades, gradeExcused],
  )

  const beginEdit = useCallback(
    (studentId: string, columnId: string) => {
      if (readOnly) return
      setEditing({ studentId, columnId })
      setDraft(displayCellValue(grades, gradeExcused, studentId, columnId))
    },
    [readOnly, grades, gradeExcused],
  )

  const commitEdit = useCallback(() => {
    if (!editing) return
    const raw = draft.trim()
    const value = raw === 'EX' ? '' : raw
    onGradeChange(editing.studentId, editing.columnId, value)
    setEditing(null)
    setDraft('')
  }, [editing, draft, onGradeChange])

  const cancelEdit = useCallback(() => {
    setEditing(null)
    setDraft('')
  }, [])

  const closeCellMenu = useCallback(() => setCellMenu(null), [])

  const handleCellMenuSelect = useCallback(
    (item: ReturnType<typeof getGradebookCellMenuItems>[number]) => {
      if (!cellMenu) return
      const { studentId, columnId } = cellMenu
      setCellMenu(null)
      switch (item.kind) {
        case 'gradeSubmission':
          onGradeSubmission?.(studentId, columnId)
          break
        case 'rubric':
          onRubricClick?.(studentId, columnId)
          break
        case 'history':
          onOpenGradeHistory?.(studentId, columnId)
          break
        case 'excuse':
          void onToggleExcused?.(studentId, columnId, item.label === 'Excuse')
          break
        default: {
          const _exhaustive: never = item
          return _exhaustive
        }
      }
    },
    [cellMenu, onGradeSubmission, onOpenGradeHistory, onRubricClick, onToggleExcused],
  )

  return (
    <>
    <div
      className={`overflow-auto rounded-xl border border-slate-200 bg-white shadow-sm dark:border-neutral-700 dark:bg-neutral-900 ${resizing ? 'cursor-col-resize select-none' : ''}`}
    >
      <table
        role="grid"
        aria-label="Grades by assignment and student"
        aria-rowcount={columns.length + 2}
        aria-colcount={1 + students.length}
        className="w-full min-w-max table-fixed border-collapse text-start"
      >
        <colgroup>
          <col style={{ width: assignmentColWidthPx }} />
          {students.map((student) => (
            <col key={student.id} />
          ))}
        </colgroup>
        <thead>
          <tr
            aria-rowindex={1}
            className="border-b border-slate-200 bg-slate-50 dark:border-neutral-700 dark:bg-neutral-800"
          >
            <th
              scope="col"
              style={assignmentColStyle}
              className={`relative ${assignmentColSurfaceClass('top-0 z-30 border-b bg-slate-50 align-bottom dark:bg-neutral-800')} ${pad}`}
            >
              <span className="text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-neutral-400">
                Assignment
              </span>
              <button
                type="button"
                aria-label="Resize assignment column"
                aria-orientation="vertical"
                title="Drag to resize assignment column"
                className="absolute inset-y-0 end-0 z-40 w-2 translate-x-1/2 cursor-col-resize touch-none border-0 bg-transparent p-0 after:absolute after:inset-y-0 after:start-1/2 after:w-px after:-translate-x-1/2 after:bg-slate-300 after:transition-colors hover:after:bg-indigo-400 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-indigo-500 dark:after:bg-neutral-600 dark:hover:after:bg-indigo-400"
                onPointerDown={onResizePointerDown}
                onPointerMove={onResizePointerMove}
                onPointerUp={finishResize}
                onPointerCancel={finishResize}
              />
            </th>
            {students.map((student) => (
              <th
                key={student.id}
                scope="col"
                title={student.name}
                className={`sticky top-0 z-20 ${pad} ${studentColMin} border-b border-slate-200 bg-slate-50 align-bottom dark:border-neutral-700 dark:bg-neutral-800 ${
                  highlightStudentId === student.id
                    ? 'bg-amber-50/90 ring-2 ring-inset ring-amber-300/90 dark:bg-amber-950/25 dark:ring-amber-500/50'
                    : ''
                }`}
              >
                <span className="block max-w-[9rem] truncate text-xs font-semibold text-slate-800 dark:text-neutral-200">
                  {studentProgressFeatureEnabled() && courseCode && student.enrollmentId ? (
                    <Link
                      to={`/courses/${encodeURIComponent(courseCode)}/students/${encodeURIComponent(student.enrollmentId)}/progress`}
                      className="text-indigo-700 hover:underline dark:text-indigo-300"
                    >
                      {student.name}
                    </Link>
                  ) : (
                    student.name
                  )}
                </span>
              </th>
            ))}
          </tr>
          <tr
            aria-rowindex={2}
            className="border-b border-slate-200 bg-slate-100 text-slate-800 dark:border-neutral-700 dark:bg-neutral-800 dark:text-neutral-200"
          >
            <th
              scope="row"
              style={assignmentColStyle}
              className={`${assignmentColSurfaceClass('z-[28] border-b bg-slate-100 text-start font-medium dark:bg-neutral-800')} ${pad}`}
            >
              <span className="text-[10px] font-semibold uppercase tracking-wide text-slate-500 dark:text-neutral-400">
                Final
              </span>
            </th>
            {students.map((student) => (
              <th
                key={`final-${student.id}`}
                scope="col"
                className={`${pad} ${studentColMin} border-b border-slate-200 bg-slate-100 text-end font-normal tabular-nums dark:border-neutral-700 dark:bg-neutral-800`}
              >
                {formatFinalPercent(finalPercentByStudentId[student.id] ?? null)}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {columns.map((col, rowIndex) => {
            const stats = assignmentStats[rowIndex]
            return (
              <tr
                key={col.id}
                aria-rowindex={rowIndex + 3}
                className="border-b border-slate-100 last:border-b-0 dark:border-neutral-700/80"
              >
                <th
                  scope="row"
                  title={col.title}
                  style={assignmentColStyle}
                  className={`${assignmentColSurfaceClass('z-10 bg-slate-100 text-start font-medium text-slate-950 dark:bg-neutral-800 dark:text-neutral-100')} ${pad}`}
                >
                  <span className="block truncate text-sm">{col.title}</span>
                  <span className="mt-0.5 block text-[0.65rem] font-normal text-slate-500 dark:text-neutral-400">
                    {col.maxPoints != null ? `Out of ${col.maxPoints}` : 'Max points not set'}
                  </span>
                  {stats ? (
                    <span className="mt-1 block text-[11px] tabular-nums leading-snug text-slate-600 dark:text-neutral-300">
                      Avg {stats.avg != null ? formatStat(stats.avg) : '—'} · Med{' '}
                      {stats.med != null ? formatStat(stats.med) : '—'}
                    </span>
                  ) : null}
                  {col.kind === 'assignment' &&
                  col.postingPolicy === 'manual' &&
                  onPostAssignmentGrades &&
                  !readOnly ? (
                    <button
                      type="button"
                      className="mt-1 inline-flex max-w-full items-center justify-center rounded-md border border-slate-200 bg-white px-1.5 py-0.5 text-[0.65rem] font-medium text-slate-700 shadow-sm hover:bg-slate-50 disabled:opacity-50 dark:border-neutral-600 dark:bg-neutral-900 dark:text-neutral-200 dark:hover:bg-neutral-800"
                      aria-label={`Post grades for ${col.title}`}
                      disabled={postGradesPending === col.id}
                      onClick={() => onPostAssignmentGrades(col.id)}
                    >
                      {postGradesPending === col.id ? 'Posting…' : 'Post grades'}
                    </button>
                  ) : null}
                </th>
                {students.map((student) => {
                  const val = displayCellValue(grades, gradeExcused, student.id, col.id)
                  const isEditing =
                    editing?.studentId === student.id && editing.columnId === col.id
                  const isExcused = Boolean(gradeExcused?.[student.id]?.[col.id])
                  const cellHeld = Boolean(gradeHeld?.[student.id]?.[col.id])
                  const cellDropped = Boolean(droppedGrades?.[student.id]?.[col.id])
                  const selectOpts = isEditing ? gradingSelectOptions(col, gradingScheme) : []
                  const cellMenuItems = getGradebookCellMenuItems({
                    col,
                    readOnly,
                    isExcused,
                    onGradeSubmission,
                    onRubricClick,
                    onOpenGradeHistory,
                    onToggleExcused,
                  })
                  const hasCellMenu = cellMenuItems.length > 0
                  const heatT =
                    colorScaleEnabled && !isExcused
                      ? heatPercentForCell(col, val === 'EX' ? '' : val)
                      : null
                  const heatSurface =
                    heatT != null && !isEditing ? heatMapCellClass(heatT) : 'bg-white dark:bg-neutral-900/80'

                  return (
                    <td
                      key={`${col.id}-${student.id}`}
                      role="gridcell"
                      className={`relative ${pad} ${studentColMin} border-s border-slate-100 text-end tabular-nums dark:border-neutral-700/80 ${heatSurface} ${
                        cellDropped ? 'opacity-65 dark:opacity-70' : ''
                      }`}
                      onDoubleClick={() => beginEdit(student.id, col.id)}
                      onContextMenu={
                        hasCellMenu
                          ? (e) => openGradebookCellMenuFromEvent(e, student.id, col.id, setCellMenu)
                          : undefined
                      }
                    >
                      {isEditing ? (
                        selectOpts.length > 0 ? (
                          <select
                            autoFocus
                            aria-label={`Grade for ${student.name}, ${col.title}`}
                            className="m-0 w-full min-w-0 border-0 bg-transparent p-0 text-end text-sm text-slate-950 shadow-none outline-none ring-0 focus:ring-0 dark:text-neutral-100"
                            value={draft}
                            onChange={(e) => setDraft(e.target.value)}
                            onBlur={commitEdit}
                            onKeyDown={(e) => {
                              if (e.key === 'Enter') commitEdit()
                              if (e.key === 'Escape') cancelEdit()
                            }}
                          >
                            <option value="">—</option>
                            {selectOpts.map((opt) => (
                              <option key={opt} value={opt}>
                                {opt}
                              </option>
                            ))}
                          </select>
                        ) : (
                          <input
                            autoFocus
                            type="text"
                            inputMode="decimal"
                            autoComplete="off"
                            aria-label={`Grade for ${student.name}, ${col.title}`}
                            className="m-0 w-full min-w-0 border-0 bg-transparent p-0 text-end text-sm tabular-nums text-slate-950 shadow-none outline-none ring-0 focus:ring-0 dark:text-neutral-100"
                            value={draft}
                            onChange={(e) => setDraft(e.target.value)}
                            onBlur={commitEdit}
                            onKeyDown={(e) => {
                              if (e.key === 'Enter') commitEdit()
                              if (e.key === 'Escape') cancelEdit()
                            }}
                          />
                        )
                      ) : (
                        <div className="relative flex min-h-[1.25rem] flex-col items-end gap-0.5">
                          {hasCellMenu ? (
                            <GradebookCellMenuTrigger
                              studentName={student.name}
                              columnTitle={col.title}
                              onOpen={(e) =>
                                openGradebookCellMenuFromButton(e, student.id, col.id, setCellMenu)
                              }
                            />
                          ) : null}
                          <span className="inline-flex max-w-full items-center justify-end gap-1">
                            {cellHeld ? (
                              <Lock
                                className="size-3.5 shrink-0 text-amber-600 dark:text-amber-400"
                                aria-hidden
                              />
                            ) : null}
                            <span
                              className={
                                val
                                  ? isExcused
                                    ? 'font-semibold text-slate-700 dark:text-neutral-200'
                                    : 'text-slate-950 dark:text-neutral-100'
                                  : 'text-neutral-400 dark:text-neutral-500'
                              }
                            >
                              {val || '—'}
                            </span>
                          </span>
                        </div>
                      )}
                    </td>
                  )
                })}
              </tr>
            )
          })}
        </tbody>
      </table>
    </div>
    {cellMenu ? (() => {
      const menuCol = columns.find((c) => c.id === cellMenu.columnId)
      if (!menuCol) return null
      return (
        <GradebookCellMenuPortal menu={cellMenu} onClose={closeCellMenu}>
          <GradebookCellMenuItems
            items={getGradebookCellMenuItems({
              col: menuCol,
              readOnly,
              isExcused: Boolean(gradeExcused?.[cellMenu.studentId]?.[cellMenu.columnId]),
              onGradeSubmission,
              onRubricClick,
              onOpenGradeHistory,
              onToggleExcused,
            })}
            onSelect={handleCellMenuSelect}
          />
        </GradebookCellMenuPortal>
      )
    })() : null}
    </>
  )
}
