import { parseNotebookTasksFromMarkdown } from './notebook-task-markdown'
import { upsertNotebookTask } from './notebook-tasks-api'

export const NOTEBOOK_TASKS_CHANGED = 'lextures-notebook-tasks-changed'

export function emitNotebookTasksChanged(): void {
  if (typeof window === 'undefined') return
  window.dispatchEvent(new Event(NOTEBOOK_TASKS_CHANGED))
}

/** Upsert every task block in page markdown to the server. */
export async function syncNotebookTasksFromMarkdown(
  courseCode: string,
  pageId: string,
  markdown: string,
): Promise<void> {
  const tasks = parseNotebookTasksFromMarkdown(markdown)
  if (tasks.length === 0) return
  await Promise.all(
    tasks.map((task) =>
      upsertNotebookTask({
        id: task.id,
        courseCode,
        notebookPageId: pageId,
        taskText: task.text,
        completed: task.checked,
        dueAt: task.dueAt,
      }),
    ),
  )
  emitNotebookTasksChanged()
}
