export const STUDENT_TODO_COLUMN_IDS = ['mon', 'tue', 'wed', 'thu', 'fri', 'sat', 'sun', 'done'] as const

export type StudentTodoColumnId = (typeof STUDENT_TODO_COLUMN_IDS)[number]

export const STUDENT_TODO_WEEKDAY_COLUMN_IDS = ['mon', 'tue', 'wed', 'thu', 'fri', 'sat', 'sun'] as const

export type StudentTodoWeekdayColumnId = (typeof STUDENT_TODO_WEEKDAY_COLUMN_IDS)[number]

export const STUDENT_TODO_COLUMN_LABELS: Record<StudentTodoColumnId, string> = {
  mon: 'Monday',
  tue: 'Tuesday',
  wed: 'Wednesday',
  thu: 'Thursday',
  fri: 'Friday',
  sat: 'Saturday',
  sun: 'Sunday',
  done: 'Done',
}

export const STUDENT_TODO_COLUMN_SHORT_LABELS: Record<StudentTodoColumnId, string> = {
  mon: 'Mon',
  tue: 'Tue',
  wed: 'Wed',
  thu: 'Thu',
  fri: 'Fri',
  sat: 'Sat',
  sun: 'Sun',
  done: 'Done',
}

export type StudentTodoItemKind = 'notebook_task' | 'due_item'

export type StudentTodoItem = {
  key: string
  kind: StudentTodoItemKind
  title: string
  courseCode: string
  courseTitle: string
  dueAt?: string | null
  href: string
  notebookPageId?: string
  notebookTaskId?: string
}

export type StudentTodoPlacement = {
  itemKey: string
  columnId: StudentTodoColumnId
  sortOrder: number
}

export function isStudentTodoColumnId(value: string): value is StudentTodoColumnId {
  return (STUDENT_TODO_COLUMN_IDS as readonly string[]).includes(value)
}

export function emptyStudentTodoBoard(): Record<StudentTodoColumnId, StudentTodoItem[]> {
  return {
    mon: [],
    tue: [],
    wed: [],
    thu: [],
    fri: [],
    sat: [],
    sun: [],
    done: [],
  }
}