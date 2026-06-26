import { authorizedFetch } from './api'
import { readApiErrorMessage } from './errors'
import { isStudentTodoColumnId, type StudentTodoColumnId, type StudentTodoPlacement } from './student-todo-types'

async function parseError(res: Response, fallback: string): Promise<never> {
  const raw: unknown = await res.json().catch(() => null)
  throw new Error(readApiErrorMessage(raw) || fallback)
}

export async function fetchStudentTodoBoardPlacements(): Promise<StudentTodoPlacement[]> {
  const res = await authorizedFetch('/api/v1/me/student-todo-board')
  if (!res.ok) await parseError(res, 'Could not load todo board.')
  const data = (await res.json()) as {
    placements?: Array<{ itemKey?: string; columnId?: string; sortOrder?: number }>
  }
  const out: StudentTodoPlacement[] = []
  for (const row of data.placements ?? []) {
    const itemKey = row.itemKey?.trim()
    const columnId = row.columnId?.trim()
    if (!itemKey || !columnId || !isStudentTodoColumnId(columnId)) continue
    out.push({
      itemKey,
      columnId,
      sortOrder: typeof row.sortOrder === 'number' ? row.sortOrder : 0,
    })
  }
  return out
}

export async function saveStudentTodoBoard(
  columns: Record<StudentTodoColumnId, string[]>,
): Promise<void> {
  const res = await authorizedFetch('/api/v1/me/student-todo-board', {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ columns }),
  })
  if (!res.ok) await parseError(res, 'Could not save todo board.')
}