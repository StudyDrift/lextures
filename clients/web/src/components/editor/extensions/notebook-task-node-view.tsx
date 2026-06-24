import { NodeViewContent, NodeViewWrapper, type NodeViewProps } from '@tiptap/react'
import { CalendarDays, ChevronDown, X } from 'lucide-react'
import { useCallback, useEffect, useId, useRef, useState } from 'react'
import { formatDate } from '../../../lib/format'
import { getNotebookTaskContext } from '../../../lib/notebook-task-context'
import { emitNotebookTasksChanged } from '../../../lib/notebook-task-sync'
import { patchNotebookTask, upsertNotebookTask } from '../../../lib/notebook-tasks-api'

function dueAtToDateInputValue(dueAt: string | null | undefined): string {
  if (!dueAt) return ''
  const d = new Date(dueAt)
  if (Number.isNaN(d.getTime())) return ''
  const y = d.getFullYear()
  const m = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  return `${y}-${m}-${day}`
}

function dateInputToDueAt(value: string): string | null {
  if (!value) return null
  const d = new Date(`${value}T23:59:59`)
  if (Number.isNaN(d.getTime())) return null
  return d.toISOString()
}

export function NotebookTaskNodeView(props: NodeViewProps) {
  const menuId = useId()
  const menuRef = useRef<HTMLDivElement>(null)
  const [menuOpen, setMenuOpen] = useState(false)
  const [dateDraft, setDateDraft] = useState('')
  const syncTimer = useRef<ReturnType<typeof setTimeout> | null>(null)
  const createdRef = useRef(false)

  const taskId = String(props.node.attrs.taskId ?? '')
  const checked = props.node.attrs.checked === true
  const dueAt = (props.node.attrs.dueAt as string | null) ?? null
  const editable = props.editor.isEditable
  const text = props.node.textContent

  const upsertCurrent = useCallback(() => {
    const ctx = getNotebookTaskContext()
    if (!ctx || !taskId) return
    void upsertNotebookTask({
      id: taskId,
      courseCode: ctx.courseCode,
      notebookPageId: ctx.pageId,
      taskText: text,
      completed: checked,
      dueAt,
    })
      .then(() => emitNotebookTasksChanged())
      .catch(() => {})
  }, [taskId, text, checked, dueAt])

  const syncTask = useCallback(
    (patch: { taskText?: string; completed?: boolean; dueAt?: string | null; clearDue?: boolean }) => {
      if (!taskId) return
      const ctx = getNotebookTaskContext()
      if (!ctx) return
      if (patch.taskText !== undefined) {
        upsertCurrent()
        return
      }
      void patchNotebookTask(taskId, patch)
        .then(() => emitNotebookTasksChanged())
        .catch(() => upsertCurrent())
    },
    [taskId, upsertCurrent],
  )

  useEffect(() => {
    if (!taskId || createdRef.current) return
    const ctx = getNotebookTaskContext()
    if (!ctx) return
    createdRef.current = true
    upsertCurrent()
  }, [taskId, upsertCurrent])

  useEffect(() => {
    if (!taskId) return
    if (syncTimer.current) clearTimeout(syncTimer.current)
    syncTimer.current = setTimeout(() => {
      upsertCurrent()
      syncTimer.current = null
    }, 600)
    return () => {
      if (syncTimer.current) clearTimeout(syncTimer.current)
    }
  }, [taskId, text, checked, dueAt, upsertCurrent])

  useEffect(() => {
    if (!menuOpen) return
    function onClickOutside(e: MouseEvent) {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        setMenuOpen(false)
      }
    }
    document.addEventListener('mousedown', onClickOutside)
    return () => document.removeEventListener('mousedown', onClickOutside)
  }, [menuOpen])

  const onToggleChecked = useCallback(() => {
    const next = !checked
    props.updateAttributes({ checked: next })
    syncTask({ completed: next })
  }, [checked, props.updateAttributes, syncTask])

  const onOpenDueMenu = useCallback(() => {
    setDateDraft(dueAtToDateInputValue(dueAt))
    setMenuOpen((v) => !v)
  }, [dueAt])

  const onSaveDueDate = useCallback(() => {
    const iso = dateInputToDueAt(dateDraft)
    props.updateAttributes({ dueAt: iso })
    syncTask({ dueAt: iso })
    setMenuOpen(false)
  }, [dateDraft, props.updateAttributes, syncTask])

  const onClearDueDate = useCallback(() => {
    props.updateAttributes({ dueAt: null })
    syncTask({ clearDue: true })
    setDateDraft('')
    setMenuOpen(false)
  }, [props.updateAttributes, syncTask])

  const textClass = [
    'min-w-0 flex-1 outline-none [&_p]:my-0',
    checked
      ? 'text-slate-500 line-through decoration-slate-400 dark:text-neutral-500 dark:decoration-neutral-500'
      : 'text-slate-800 dark:text-neutral-200',
  ].join(' ')

  return (
    <NodeViewWrapper
      as="div"
      className="lex-notebook-task group my-2 flex items-start gap-2 rounded-lg border border-transparent px-1 py-1 transition-[background-color,color,border-color] hover:border-slate-200 dark:hover:border-neutral-700"
      data-type="notebook-task"
    >
      <div className="min-w-0 flex-1">
        <div className="flex items-center gap-2">
          <input
            type="checkbox"
            checked={checked}
            disabled={!editable}
            onChange={onToggleChecked}
            className="h-4 w-4 shrink-0 rounded border-slate-300 text-indigo-600 focus:ring-indigo-500 dark:border-neutral-600"
            aria-label={checked ? 'Mark task incomplete' : 'Mark task complete'}
          />
          <NodeViewContent as="div" className={textClass} />
        </div>
        {dueAt ? (
          <p className="mt-0.5 flex items-center gap-1 ps-6 text-xs text-slate-500 dark:text-neutral-400">
            <CalendarDays className="h-3 w-3 shrink-0" aria-hidden />
            Due {formatDate(dueAt, { dateStyle: 'medium' })}
          </p>
        ) : null}
      </div>
      {editable ? (
        <div className="relative shrink-0 opacity-0 transition-opacity group-hover:opacity-100 group-focus-within:opacity-100" ref={menuRef}>
          <button
            type="button"
            onClick={onOpenDueMenu}
            className="inline-flex items-center rounded-md p-1 text-slate-400 transition-[background-color,color,border-color] hover:bg-slate-100 hover:text-slate-600 dark:hover:bg-neutral-800 dark:hover:text-neutral-300"
            aria-expanded={menuOpen}
            aria-haspopup="menu"
            aria-controls={menuId}
            aria-label="Task options"
          >
            <ChevronDown className="h-4 w-4" aria-hidden />
          </button>
          {menuOpen ? (
            <div
              id={menuId}
              role="menu"
              className="absolute right-0 top-full z-30 mt-1 w-52 rounded-xl border border-slate-200 bg-white p-3 shadow-lg dark:border-neutral-700 dark:bg-neutral-900"
            >
              <label className="block text-xs font-medium text-slate-600 dark:text-neutral-400">
                Due date
                <input
                  type="date"
                  value={dateDraft}
                  onChange={(e) => setDateDraft(e.target.value)}
                  className="mt-1 w-full rounded-lg border border-slate-200 px-2 py-1.5 text-sm text-slate-800 dark:border-neutral-600 dark:bg-neutral-950 dark:text-neutral-100"
                />
              </label>
              <div className="mt-2 flex gap-2">
                <button
                  type="button"
                  role="menuitem"
                  onClick={onSaveDueDate}
                  className="flex-1 rounded-lg bg-indigo-600 px-2 py-1.5 text-xs font-semibold text-white hover:bg-indigo-500"
                >
                  Save
                </button>
                {dueAt ? (
                  <button
                    type="button"
                    role="menuitem"
                    onClick={onClearDueDate}
                    className="inline-flex items-center gap-1 rounded-lg border border-slate-200 px-2 py-1.5 text-xs text-slate-600 hover:bg-slate-50 dark:border-neutral-600 dark:text-neutral-300 dark:hover:bg-neutral-800"
                    aria-label="Clear due date"
                  >
                    <X className="h-3 w-3" aria-hidden />
                  </button>
                ) : null}
              </div>
            </div>
          ) : null}
        </div>
      ) : null}
    </NodeViewWrapper>
  )
}
