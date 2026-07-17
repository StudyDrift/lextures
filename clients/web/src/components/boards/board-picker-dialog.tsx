import { useEffect, useId, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { listBoards, type Board } from '../../lib/boards-api'
import { toastMutationError } from '../../lib/lms-toast'

type Props = {
  open: boolean
  courseCode: string
  onClose: () => void
  onPick: (board: Board) => void
}

export function BoardPickerDialog({ open, courseCode, onClose, onPick }: Props) {
  const { t } = useTranslation('common')
  const titleId = useId()
  const [boards, setBoards] = useState<Board[]>([])
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    if (!open || !courseCode) return
    setLoading(true)
    void listBoards(courseCode)
      .then(setBoards)
      .catch((err) => toastMutationError(err instanceof Error ? err.message : String(err)))
      .finally(() => setLoading(false))
  }, [open, courseCode])

  if (!open) return null

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4" role="presentation">
      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        className="w-full max-w-md rounded-lg bg-white p-5 shadow-xl dark:bg-neutral-900"
      >
        <h2 id={titleId} className="text-lg font-semibold text-slate-900 dark:text-neutral-100">
          {t('boards.embed.pickerTitle')}
        </h2>
        <p className="mt-1 text-sm text-slate-600 dark:text-neutral-300">{t('boards.embed.pickerSubtitle')}</p>
        <div className="mt-4 max-h-72 overflow-auto">
          {loading ? (
            <p className="text-sm text-slate-500">{t('common.loading')}</p>
          ) : boards.length === 0 ? (
            <p className="text-sm text-slate-500">{t('boards.embed.pickerEmpty')}</p>
          ) : (
            <ul className="space-y-1">
              {boards.map((b) => (
                <li key={b.id}>
                  <button
                    type="button"
                    className="w-full rounded-md px-3 py-2 text-start text-sm hover:bg-slate-50 dark:hover:bg-neutral-800"
                    onClick={() => {
                      onPick(b)
                      onClose()
                    }}
                  >
                    <span className="font-medium text-slate-900 dark:text-neutral-100">{b.title}</span>
                    {b.description ? (
                      <span className="mt-0.5 block truncate text-xs text-slate-500">{b.description}</span>
                    ) : null}
                  </button>
                </li>
              ))}
            </ul>
          )}
        </div>
        <div className="mt-4 flex justify-end">
          <button
            type="button"
            className="rounded-md px-3 py-1.5 text-sm text-slate-600 dark:text-neutral-300"
            onClick={onClose}
          >
            {t('dialogs.cancel')}
          </button>
        </div>
      </div>
    </div>
  )
}
