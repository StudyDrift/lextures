import { useCallback, useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { CalendarDays, CheckSquare, Square } from 'lucide-react'
import { formatDate } from '../../lib/format'
import { NOTEBOOK_TASKS_CHANGED } from '../../lib/notebook-task-sync'
import {
  fetchNotebookTasks,
  patchNotebookTask,
  type NotebookTask,
} from '../../lib/notebook-tasks-api'
import {
  GLOBAL_STUDENT_NOTEBOOK_KEY,
  GLOBAL_STUDENT_NOTEBOOK_TITLE,
  hrefForNotebookPage,
  markNotebookTaskComplete,
} from '../../lib/student-notebook-storage'

function taskCourseLabel(courseCode: string, courseTitles: Record<string, string>): string {
  if (courseCode === GLOBAL_STUDENT_NOTEBOOK_KEY) return GLOBAL_STUDENT_NOTEBOOK_TITLE
  return courseTitles[courseCode] ?? courseCode
}

export type NotebookTasksCardProps = {
  courseTitles: Record<string, string>
}

export function NotebookTasksCard({ courseTitles }: NotebookTasksCardProps) {
  const [tasks, setTasks] = useState<NotebookTask[] | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [completingId, setCompletingId] = useState<string | null>(null)

  const load = useCallback(() => {
    void fetchNotebookTasks()
      .then((rows) => {
        setTasks(rows)
        setError(null)
      })
      .catch((e: unknown) => {
        setTasks([])
        setError(e instanceof Error ? e.message : 'Could not load notebook tasks.')
      })
  }, [])

  useEffect(() => {
    load()
  }, [load])

  useEffect(() => {
    function onTasksChanged() {
      load()
    }
    function onFocus() {
      load()
    }
    window.addEventListener(NOTEBOOK_TASKS_CHANGED, onTasksChanged)
    window.addEventListener('focus', onFocus)
    return () => {
      window.removeEventListener(NOTEBOOK_TASKS_CHANGED, onTasksChanged)
      window.removeEventListener('focus', onFocus)
    }
  }, [load])

  const onComplete = useCallback(
    async (task: NotebookTask) => {
      if (completingId) return
      setCompletingId(task.id)
      try {
        await patchNotebookTask(task.id, { completed: true })
        markNotebookTaskComplete(task.courseCode, task.notebookPageId, task.id)
        setTasks((prev) => (prev ?? []).filter((t) => t.id !== task.id))
      } catch (e: unknown) {
        setError(e instanceof Error ? e.message : 'Could not complete task.')
      } finally {
        setCompletingId(null)
      }
    },
    [completingId],
  )

  if (tasks === null) return null

  return (
    <section aria-label="Notebook tasks">
      <h2 className="text-sm font-semibold uppercase tracking-wide text-slate-500 dark:text-neutral-400">
        Notebook tasks
      </h2>
      {error ? (
        <p className="mt-3 rounded-xl border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-900 dark:border-amber-900/40 dark:bg-amber-950/30 dark:text-amber-100">
          {error}
        </p>
      ) : null}
      {tasks.length === 0 && !error ? (
        <p className="mt-3 rounded-xl border border-slate-200 bg-slate-50/80 px-4 py-3 text-sm text-slate-600 dark:border-neutral-700 dark:bg-neutral-900/50 dark:text-neutral-300">
          Add tasks in a notebook with <span className="font-mono text-xs">/task</span> or{' '}
          <span className="font-mono text-xs">/todo</span>. Open tasks show up here.
        </p>
      ) : null}
      <ul className="mt-3 space-y-2">
        {tasks.map((task) => {
          const label = task.taskText.trim() || 'Untitled task'
          const href = hrefForNotebookPage(task.courseCode, task.notebookPageId)
          const busy = completingId === task.id
          return (
            <li
              key={task.id}
              className="flex items-start gap-3 rounded-xl bg-white px-3 py-3 shadow-card dark:bg-neutral-900"
            >
              <button
                type="button"
                disabled={busy}
                onClick={() => void onComplete(task)}
                className="mt-0.5 shrink-0 rounded p-0.5 text-slate-400 transition-[background-color,color,border-color] hover:bg-slate-100 hover:text-indigo-600 disabled:opacity-50 dark:hover:bg-neutral-800 dark:hover:text-indigo-400"
                aria-label={`Complete task: ${label}`}
              >
                <Square className="h-4 w-4" aria-hidden />
              </button>
              <div className="min-w-0 flex-1">
                <Link
                  to={href}
                  className="block text-sm font-medium text-slate-900 hover:text-indigo-700 dark:text-neutral-100 dark:hover:text-indigo-300"
                >
                  {label}
                </Link>
                <p className="mt-0.5 text-xs text-slate-500 dark:text-neutral-400">
                  {taskCourseLabel(task.courseCode, courseTitles)}
                </p>
                {task.dueAt ? (
                  <p className="mt-1 flex items-center gap-1 text-xs text-slate-500 dark:text-neutral-400">
                    <CalendarDays className="h-3 w-3 shrink-0" aria-hidden />
                    Due {formatDate(task.dueAt, { dateStyle: 'medium' })}
                  </p>
                ) : null}
              </div>
              <Link
                to={href}
                className="shrink-0 self-center text-xs font-medium text-indigo-600 hover:text-indigo-500 dark:text-indigo-400"
              >
                Open
              </Link>
            </li>
          )
        })}
      </ul>
      {tasks.length > 0 ? (
        <p className="mt-2 flex items-center gap-1.5 text-xs text-slate-500 dark:text-neutral-400">
          <CheckSquare className="h-3.5 w-3.5" aria-hidden />
          Complete a task here to remove it from this list. In your notebook it stays visible, crossed out.
        </p>
      ) : null}
    </section>
  )
}
