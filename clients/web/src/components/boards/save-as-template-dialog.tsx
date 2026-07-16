import { useEffect, useId, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { saveBoardAsTemplate, type Board } from '../../lib/boards-api'
import { toastMutationError } from '../../lib/lms-toast'

type Props = {
  open: boolean
  onClose: () => void
  courseCode: string
  board: Board
  onSaved?: () => void
}

export function SaveAsTemplateDialog({ open, onClose, courseCode, board, onSaved }: Props) {
  const { t } = useTranslation('common')
  const titleId = useId()
  const [scope, setScope] = useState<'course' | 'org'>('course')
  const [title, setTitle] = useState(board.title)
  const [description, setDescription] = useState(board.description)
  const [includePosts, setIncludePosts] = useState(false)
  const [confirmPosts, setConfirmPosts] = useState(false)
  const [submitting, setSubmitting] = useState(false)

  useEffect(() => {
    if (!open) return
    setScope('course')
    setTitle(board.title)
    setDescription(board.description)
    setIncludePosts(false)
    setConfirmPosts(false)
  }, [open, board])

  if (!open) return null

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (includePosts && !confirmPosts) return
    setSubmitting(true)
    try {
      await saveBoardAsTemplate(courseCode, board.id, {
        scope,
        title: title.trim() || board.title,
        description: description.trim(),
        includePosts,
      })
      onSaved?.()
      onClose()
    } catch (err) {
      toastMutationError(err instanceof Error ? err.message : String(err))
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div
      className="fixed inset-0 z-50 flex items-end justify-center bg-black/40 p-0 sm:items-center sm:p-4"
      role="presentation"
      onClick={onClose}
    >
      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        className="w-full max-w-md rounded-none bg-white p-4 shadow-xl dark:bg-neutral-900 sm:rounded-lg"
        onClick={(e) => e.stopPropagation()}
      >
        <h2 id={titleId} className="text-lg font-semibold text-slate-900 dark:text-neutral-100">
          {t('boards.template.saveTitle')}
        </h2>
        <p className="mt-1 text-sm text-slate-600 dark:text-neutral-300">
          {t('boards.template.saveSubtitle')}
        </p>
        <form
          onSubmit={(e) => {
            void handleSubmit(e)
          }}
          className="mt-4 space-y-3"
        >
          <label className="block text-sm">
            <span className="font-medium text-slate-700 dark:text-neutral-200">
              {t('boards.template.scopeLabel')}
            </span>
            <select
              value={scope}
              onChange={(e) => setScope(e.target.value as 'course' | 'org')}
              className="mt-1 w-full rounded-md border border-slate-300 px-3 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-800"
            >
              <option value="course">{t('boards.template.scopeCourse')}</option>
              <option value="org">{t('boards.template.scopeOrg')}</option>
            </select>
          </label>
          <label className="block text-sm">
            <span className="font-medium text-slate-700 dark:text-neutral-200">
              {t('boards.create.titleLabel')}
            </span>
            <input
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              maxLength={200}
              required
              className="mt-1 w-full rounded-md border border-slate-300 px-3 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-800"
            />
          </label>
          <label className="block text-sm">
            <span className="font-medium text-slate-700 dark:text-neutral-200">
              {t('boards.create.descriptionLabel')}
            </span>
            <textarea
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              rows={2}
              className="mt-1 w-full rounded-md border border-slate-300 px-3 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-800"
            />
          </label>
          <label className="flex items-start gap-2 text-sm">
            <input
              type="checkbox"
              checked={includePosts}
              onChange={(e) => {
                setIncludePosts(e.target.checked)
                if (!e.target.checked) setConfirmPosts(false)
              }}
              className="mt-1"
            />
            <span>{t('boards.template.includePosts')}</span>
          </label>
          {includePosts ? (
            <div className="rounded-md border border-amber-300 bg-amber-50 p-3 text-sm text-amber-900 dark:border-amber-700 dark:bg-amber-950/40 dark:text-amber-100">
              <p>{t('boards.template.ferpaWarning')}</p>
              <label className="mt-2 flex items-start gap-2">
                <input
                  type="checkbox"
                  checked={confirmPosts}
                  onChange={(e) => setConfirmPosts(e.target.checked)}
                  className="mt-1"
                />
                <span>{t('boards.template.ferpaConfirm')}</span>
              </label>
            </div>
          ) : (
            <p className="text-xs text-slate-500 dark:text-neutral-400">
              {t('boards.template.structureOnlyHint')}
            </p>
          )}
          <div className="flex justify-end gap-2 pt-2">
            <button
              type="button"
              onClick={onClose}
              className="rounded-md px-3 py-2 text-sm font-medium text-slate-700 hover:bg-slate-100 dark:text-neutral-200 dark:hover:bg-neutral-800"
            >
              {t('dialogs.cancel')}
            </button>
            <button
              type="submit"
              disabled={submitting || (includePosts && !confirmPosts) || !title.trim()}
              className="rounded-md bg-indigo-600 px-3 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50"
            >
              {submitting ? t('common.loading') : t('boards.template.saveSubmit')}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}
