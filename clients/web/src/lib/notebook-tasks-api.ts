import { authorizedFetch } from './api'
import { readApiErrorMessage } from './errors'

export type NotebookTask = {
  id: string
  courseCode: string
  notebookPageId: string
  taskText: string
  completed: boolean
  dueAt?: string | null
  createdAt: string
  updatedAt: string
}

async function parseError(res: Response, fallback: string): Promise<never> {
  const raw: unknown = await res.json().catch(() => null)
  throw new Error(readApiErrorMessage(raw) || fallback)
}

export async function fetchNotebookTasks(): Promise<NotebookTask[]> {
  const res = await authorizedFetch('/api/v1/me/notebook-tasks')
  if (!res.ok) await parseError(res, 'Could not load notebook tasks.')
  const data = (await res.json()) as { tasks?: NotebookTask[] }
  return data.tasks ?? []
}

export async function upsertNotebookTask(body: {
  id: string
  courseCode: string
  notebookPageId: string
  taskText: string
  completed?: boolean
  dueAt?: string | null
}): Promise<NotebookTask> {
  const res = await authorizedFetch('/api/v1/me/notebook-tasks', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  if (!res.ok) await parseError(res, 'Could not save notebook task.')
  return res.json() as Promise<NotebookTask>
}

export async function patchNotebookTask(
  id: string,
  body: {
    taskText?: string
    completed?: boolean
    dueAt?: string | null
    clearDue?: boolean
  },
): Promise<NotebookTask> {
  const res = await authorizedFetch(`/api/v1/me/notebook-tasks/${encodeURIComponent(id)}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  if (!res.ok) await parseError(res, 'Could not update notebook task.')
  return res.json() as Promise<NotebookTask>
}
