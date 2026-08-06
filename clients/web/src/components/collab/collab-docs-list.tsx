import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { formatDate } from '../../lib/format'
import { Link } from 'react-router-dom'
import { FileText, LayoutTemplate, Plus, Trash2 } from 'lucide-react'
import type { CollabDoc, DocType } from '../../lib/collab-docs-api'
import { createCollabDoc, deleteCollabDoc } from '../../lib/collab-docs-api'
import { toastMutationError } from '../../lib/lms-toast'
import { useConfirm } from '../use-confirm'

type Props = {
  courseCode: string
  docs: CollabDoc[]
  canManage: boolean
  onDocsChanged: () => void
}

export function CollabDocsList({ courseCode, docs, canManage, onDocsChanged }: Props) {
  const { t } = useTranslation('common')
  const { confirm, ConfirmDialogHost } = useConfirm()
  const [creating, setCreating] = useState(false)
  const [newTitle, setNewTitle] = useState('')
  const [newType, setNewType] = useState<DocType>('rich_text')
  const [submitting, setSubmitting] = useState(false)

  const base = `/courses/${encodeURIComponent(courseCode)}/collab-docs`

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault()
    if (!newTitle.trim()) return
    setSubmitting(true)
    try {
      await createCollabDoc(courseCode, newTitle.trim(), newType)
      setNewTitle('')
      setCreating(false)
      onDocsChanged()
    } catch (err) {
      toastMutationError(err instanceof Error ? err.message : String(err))
    } finally {
      setSubmitting(false)
    }
  }

  async function handleDelete(docId: string) {
    if (
      !(await confirm({
        title: t('collab.deleteDocument.title'),
        confirmLabel: t('dialogs.delete'),
        variant: 'danger',
      }))
    ) {
      return
    }
    try {
      await deleteCollabDoc(courseCode, docId)
      onDocsChanged()
    } catch (err) {
      toastMutationError(err instanceof Error ? err.message : String(err))
    }
  }

  return (
    <>
    {ConfirmDialogHost}
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-xl font-semibold text-fg-default">
          Collaborative Documents
        </h2>
        {canManage && !creating && (
          <button
            type="button"
            onClick={() => setCreating(true)}
            className="flex items-center gap-1.5 rounded-md bg-accent-solid px-3 py-2 text-sm font-medium text-white hover:bg-accent focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-indigo-500"
          >
            <Plus className="h-4 w-4" aria-hidden="true" />
            New document
          </button>
        )}
      </div>

      {creating && (
        <form
          onSubmit={(e) => { void handleCreate(e) }}
          className="rounded-lg border border-indigo-200 bg-indigo-50 p-4 dark:border-indigo-800 dark:bg-indigo-950/30"
        >
          <div className="space-y-3">
            <div>
              <label className="block text-sm font-medium text-fg-muted" htmlFor="doc-title">
                Document title
              </label>
              <input
                id="doc-title"
                type="text"
                value={newTitle}
                onChange={(e) => setNewTitle(e.target.value)}
                placeholder="My collaborative document"
                className="mt-1 w-full rounded-md border border-border-strong px-3 py-2 text-sm shadow-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500 dark:border-border-default dark:bg-surface-overlay dark:text-fg-default"
                autoFocus
              />
            </div>
            <div>
              <span className="block text-sm font-medium text-fg-muted">
                Document type
              </span>
              <div className="mt-1 flex gap-4">
                <label className="flex items-center gap-2 text-sm text-fg-muted">
                  <input
                    type="radio"
                    name="doc-type"
                    value="rich_text"
                    checked={newType === 'rich_text'}
                    onChange={() => setNewType('rich_text')}
                  />
                  Rich text
                </label>
                <label className="flex items-center gap-2 text-sm text-fg-muted">
                  <input
                    type="radio"
                    name="doc-type"
                    value="whiteboard"
                    checked={newType === 'whiteboard'}
                    onChange={() => setNewType('whiteboard')}
                  />
                  Whiteboard
                </label>
              </div>
            </div>
          </div>
          <div className="mt-4 flex gap-2">
            <button
              type="submit"
              disabled={submitting || !newTitle.trim()}
              className="rounded-md bg-accent-solid px-4 py-2 text-sm font-medium text-white hover:bg-accent disabled:opacity-50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-indigo-500"
            >
              {submitting ? 'Creating…' : 'Create'}
            </button>
            <button
              type="button"
              onClick={() => { setCreating(false); setNewTitle('') }}
              className="rounded-md border border-border-strong px-4 py-2 text-sm font-medium text-fg-muted hover:bg-surface-base dark:border-border-default dark:text-fg-muted dark:hover:bg-surface-overlay"
            >
              Cancel
            </button>
          </div>
        </form>
      )}

      {docs.length === 0 && !creating && (
        <div className="rounded-lg border border-dashed border-border-strong p-8 text-center dark:border-border-default">
          <FileText className="mx-auto h-8 w-8 text-fg-subtle" />
          <p className="mt-2 text-sm text-fg-muted">
            No collaborative documents yet.
          </p>
          {canManage && (
            <p className="text-sm text-fg-subtle">
              Click <strong>New document</strong> to create one.
            </p>
          )}
        </div>
      )}

      <ul className="space-y-2" role="list">
        {docs.map((doc) => (
          <li
            key={doc.id}
            className="flex items-center justify-between rounded-lg border border-border-default bg-surface-raised px-4 py-3 shadow-sm hover:bg-surface-base dark:border-border-default/50 dark:hover:bg-neutral-700/50"
          >
            <Link
              to={`${base}/${doc.id}`}
              className="flex min-w-0 flex-1 items-center gap-3"
            >
              {doc.docType === 'whiteboard' ? (
                <LayoutTemplate className="h-5 w-5 shrink-0 text-indigo-500" aria-hidden="true" />
              ) : (
                <FileText className="h-5 w-5 shrink-0 text-indigo-500" aria-hidden="true" />
              )}
              <div className="min-w-0">
                <p className="truncate text-sm font-medium text-fg-default">
                  {doc.title}
                </p>
                <p className="text-xs text-fg-subtle">
                  {doc.docType === 'whiteboard' ? 'Whiteboard' : 'Rich text'} ·{' '}
                  {formatDate(doc.updatedAt, { dateStyle: 'medium' })}
                </p>
              </div>
            </Link>
            {canManage && (
              <button
                type="button"
                onClick={() => { void handleDelete(doc.id) }}
                aria-label={`Delete "${doc.title}"`}
                className="ms-4 rounded p-1 text-fg-subtle hover:text-red-500 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-red-500 dark:hover:text-red-400"
              >
                <Trash2 className="h-4 w-4" aria-hidden="true" />
              </button>
            )}
          </li>
        ))}
      </ul>
    </div>
    </>
  )
}
