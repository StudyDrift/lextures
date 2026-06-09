import type { MouseEvent as ReactMouseEvent } from 'react'
import type { GradebookColumn } from './gradebook-grid-types'

export type GradebookCellMenuState = {
  studentId: string
  columnId: string
  top: number
  left: number
} | null

export type GradebookCellMenuItem =
  | { kind: 'gradeSubmission'; label: 'Grade submission' }
  | { kind: 'rubric'; label: 'Rubric' }
  | { kind: 'history'; label: 'History' }
  | { kind: 'excuse'; label: 'Excuse' | 'Unexcuse' }

type GradebookCellMenuContext = {
  col: GradebookColumn
  readOnly: boolean
  isExcused: boolean
  onGradeSubmission?: (studentId: string, columnId: string) => void
  onRubricClick?: (studentId: string, columnId: string) => void
  onOpenGradeHistory?: (studentId: string, columnId: string) => void
  onToggleExcused?: (studentId: string, columnId: string, excused: boolean) => void | Promise<void>
}

function isGradableColumnKind(kind: string | undefined): boolean {
  return kind === 'assignment' || kind === 'quiz' || kind === 'quiz_comprehensive'
}

export function getGradebookCellMenuItems(ctx: GradebookCellMenuContext): GradebookCellMenuItem[] {
  const { col, readOnly, isExcused } = ctx
  const items: GradebookCellMenuItem[] = []

  if (!readOnly && col.kind === 'assignment' && ctx.onGradeSubmission) {
    items.push({ kind: 'gradeSubmission', label: 'Grade submission' })
  }
  if (!readOnly && col.rubric && ctx.onRubricClick) {
    items.push({ kind: 'rubric', label: 'Rubric' })
  }
  if (ctx.onOpenGradeHistory && isGradableColumnKind(col.kind)) {
    items.push({ kind: 'history', label: 'History' })
  }
  if (!readOnly && ctx.onToggleExcused && isGradableColumnKind(col.kind)) {
    items.push({ kind: 'excuse', label: isExcused ? 'Unexcuse' : 'Excuse' })
  }

  return items
}

export function openGradebookCellMenuFromEvent(
  e: ReactMouseEvent,
  studentId: string,
  columnId: string,
  openMenu: (menu: Exclude<GradebookCellMenuState, null>) => void,
) {
  e.preventDefault()
  e.stopPropagation()
  openMenu({ studentId, columnId, top: e.clientY, left: e.clientX })
}

export function openGradebookCellMenuFromButton(
  e: ReactMouseEvent<HTMLButtonElement>,
  studentId: string,
  columnId: string,
  openMenu: (menu: Exclude<GradebookCellMenuState, null>) => void,
) {
  e.stopPropagation()
  const rect = e.currentTarget.getBoundingClientRect()
  openMenu({ studentId, columnId, top: rect.bottom + 4, left: rect.left })
}
