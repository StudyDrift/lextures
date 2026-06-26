import type { CoursePublic, CourseStructureItem } from './courses-api'
import type { NotebookTask } from './notebook-tasks-api'
import {
  GLOBAL_STUDENT_NOTEBOOK_KEY,
  GLOBAL_STUDENT_NOTEBOOK_TITLE,
  hrefForNotebookPage,
} from './student-notebook-storage'
import {
  emptyStudentTodoBoard,
  isStudentTodoColumnId,
  STUDENT_TODO_COLUMN_IDS,
  type StudentTodoColumnId,
  type StudentTodoItem,
  type StudentTodoPlacement,
} from './student-todo-types'

/** Local hour (0–23) before which same-day due times roll back an extra day. */
export const DEFAULT_PFT_HOUR = 9

const WEEKDAY_TO_COLUMN: Record<number, StudentTodoColumnId> = {
  0: 'sun',
  1: 'mon',
  2: 'tue',
  3: 'wed',
  4: 'thu',
  5: 'fri',
  6: 'sat',
}

export function notebookTaskItemKey(taskId: string): string {
  return `notebook:${taskId}`
}

export function dueItemKey(courseCode: string, itemId: string): string {
  return `due:${courseCode}:${itemId}`
}

export function courseLabel(course: CoursePublic): string {
  return course.title.trim() || course.courseCode
}

function taskCourseTitle(courseCode: string, courseTitles: Record<string, string>): string {
  if (courseCode === GLOBAL_STUDENT_NOTEBOOK_KEY) return GLOBAL_STUDENT_NOTEBOOK_TITLE
  return courseTitles[courseCode] ?? courseCode
}

function moduleHref(courseCode: string, item: CourseStructureItem): string {
  const base = `/courses/${encodeURIComponent(courseCode)}/modules`
  if (item.kind === 'assignment') return `${base}/assignment/${encodeURIComponent(item.id)}`
  if (item.kind === 'quiz') return `${base}/quiz/${encodeURIComponent(item.id)}`
  return `${base}/content/${encodeURIComponent(item.id)}`
}

function isDueStructureItem(
  item: CourseStructureItem,
): item is CourseStructureItem & { kind: 'content_page' | 'assignment' | 'quiz'; dueAt: string } {
  return (item.kind === 'content_page' || item.kind === 'assignment' || item.kind === 'quiz') && Boolean(item.dueAt)
}

/** Weekday bucket for a due instant, applying push-forward-time (P.F.T.). */
export function weekdayColumnForDue(
  dueAt: string | null | undefined,
  now = new Date(),
  pftHour = DEFAULT_PFT_HOUR,
): StudentTodoColumnId {
  if (!dueAt) {
    return WEEKDAY_TO_COLUMN[now.getDay()] ?? 'mon'
  }
  const due = new Date(dueAt)
  if (Number.isNaN(due.getTime())) {
    return WEEKDAY_TO_COLUMN[now.getDay()] ?? 'mon'
  }
  const bucket = new Date(due.getFullYear(), due.getMonth(), due.getDate() - 1)
  if (due.getHours() < pftHour) {
    bucket.setDate(bucket.getDate() - 1)
  }
  return WEEKDAY_TO_COLUMN[bucket.getDay()] ?? 'mon'
}

export function collectStudentTodoItems(input: {
  studentCourses: CoursePublic[]
  structureByCourseCode: Record<string, CourseStructureItem[]>
  notebookTasks: NotebookTask[]
}): StudentTodoItem[] {
  const courseTitles = Object.fromEntries(
    input.studentCourses.map((c) => [c.courseCode, courseLabel(c)]),
  )
  const studentCodes = new Set(input.studentCourses.map((c) => c.courseCode))
  const items: StudentTodoItem[] = []

  for (const task of input.notebookTasks) {
    if (task.completed) continue
    if (task.courseCode !== GLOBAL_STUDENT_NOTEBOOK_KEY && !studentCodes.has(task.courseCode)) continue
    const title = task.taskText.trim() || 'Untitled task'
    items.push({
      key: notebookTaskItemKey(task.id),
      kind: 'notebook_task',
      title,
      courseCode: task.courseCode,
      courseTitle: taskCourseTitle(task.courseCode, courseTitles),
      dueAt: task.dueAt ?? null,
      href: hrefForNotebookPage(task.courseCode, task.notebookPageId),
      notebookPageId: task.notebookPageId,
      notebookTaskId: task.id,
    })
  }

  for (const course of input.studentCourses) {
    if (course.calendarEnabled === false) continue
    const structure = input.structureByCourseCode[course.courseCode] ?? []
    for (const row of structure) {
      if (!isDueStructureItem(row)) continue
      items.push({
        key: dueItemKey(course.courseCode, row.id),
        kind: 'due_item',
        title: row.title.trim() || 'Untitled',
        courseCode: course.courseCode,
        courseTitle: courseLabel(course),
        dueAt: row.dueAt,
        href: moduleHref(course.courseCode, row),
      })
    }
  }

  return items
}

export function buildStudentTodoBoard(
  items: StudentTodoItem[],
  placements: StudentTodoPlacement[],
  now = new Date(),
): Record<StudentTodoColumnId, StudentTodoItem[]> {
  const board = emptyStudentTodoBoard()
  const itemByKey = new Map(items.map((item) => [item.key, item]))
  const placed = new Set<string>()

  const sortedPlacements = [...placements].sort((a, b) => {
    if (a.columnId !== b.columnId) return a.columnId.localeCompare(b.columnId)
    return a.sortOrder - b.sortOrder
  })

  for (const placement of sortedPlacements) {
    if (!isStudentTodoColumnId(placement.columnId)) continue
    const item = itemByKey.get(placement.itemKey)
    if (!item) continue
    board[placement.columnId].push(item)
    placed.add(item.key)
  }

  for (const item of items) {
    if (placed.has(item.key)) continue
    const column = weekdayColumnForDue(item.dueAt, now)
    board[column].push(item)
  }

  return board
}

export function boardToColumnKeys(
  board: Record<StudentTodoColumnId, StudentTodoItem[]>,
): Record<StudentTodoColumnId, string[]> {
  return {
    mon: board.mon.map((item) => item.key),
    tue: board.tue.map((item) => item.key),
    wed: board.wed.map((item) => item.key),
    thu: board.thu.map((item) => item.key),
    fri: board.fri.map((item) => item.key),
    sat: board.sat.map((item) => item.key),
    sun: board.sun.map((item) => item.key),
    done: board.done.map((item) => item.key),
  }
}

export function computeNextStudentTodoBoard(
  board: Record<StudentTodoColumnId, StudentTodoItem[]>,
  itemKey: string,
  targetColumn: StudentTodoColumnId,
  targetIndex?: number,
): Record<StudentTodoColumnId, StudentTodoItem[]> {
  const next = emptyStudentTodoBoard()
  for (const col of STUDENT_TODO_COLUMN_IDS) {
    next[col] = [...board[col]]
  }

  let moving: StudentTodoItem | undefined
  for (const col of STUDENT_TODO_COLUMN_IDS) {
    const idx = next[col].findIndex((item) => item.key === itemKey)
    if (idx >= 0) {
      moving = next[col][idx]
      next[col].splice(idx, 1)
      break
    }
  }
  if (!moving) return board

  const insertAt =
    targetIndex == null || targetIndex < 0 || targetIndex > next[targetColumn].length
      ? next[targetColumn].length
      : targetIndex
  next[targetColumn].splice(insertAt, 0, moving)
  return next
}