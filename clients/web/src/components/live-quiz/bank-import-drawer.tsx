import { useCallback, useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { listBankCandidates, type BankCandidate } from '../../lib/live-quiz-api'
import { toastMutationError } from '../../lib/lms-toast'

type Props = {
  open: boolean
  courseCode: string
  kitId: string
  onClose: () => void
  onImport: (questionIds: string[]) => Promise<void>
}

export function BankImportDrawer({ open, courseCode, kitId, onClose, onImport }: Props) {
  const { t } = useTranslation('common')
  const [q, setQ] = useState('')
  const [rows, setRows] = useState<BankCandidate[]>([])
  const [selected, setSelected] = useState<Set<string>>(new Set())
  const [loading, setLoading] = useState(false)
  const [importing, setImporting] = useState(false)

  const load = useCallback(async () => {
    if (!open) return
    setLoading(true)
    try {
      const list = await listBankCandidates(courseCode, kitId, { q: q.trim() || undefined })
      setRows(list)
    } catch (err) {
      toastMutationError(err instanceof Error ? err.message : String(err))
    } finally {
      setLoading(false)
    }
  }, [courseCode, kitId, open, q])

  useEffect(() => {
    void load()
  }, [load])

  if (!open) return null

  function toggle(id: string) {
    setSelected((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }

  async function handleImport() {
    if (selected.size === 0) return
    setImporting(true)
    try {
      await onImport([...selected])
      setSelected(new Set())
      onClose()
    } catch (err) {
      toastMutationError(err instanceof Error ? err.message : String(err))
    } finally {
      setImporting(false)
    }
  }

  return (
    <div className="fixed inset-0 z-40 flex justify-end bg-black/30" role="dialog" aria-modal>
      <div className="flex h-full w-full max-w-md flex-col bg-white shadow-xl dark:bg-neutral-900">
        <div className="flex items-center justify-between border-b border-slate-200 px-4 py-3 dark:border-neutral-700">
          <h2 className="text-lg font-semibold text-slate-900 dark:text-neutral-100">
            {t('liveQuiz.editor.importBank')}
          </h2>
          <button type="button" onClick={onClose} className="min-h-11 px-3 text-sm">
            {t('dialogs.cancel')}
          </button>
        </div>
        <div className="border-b border-slate-200 p-4 dark:border-neutral-700">
          <label className="block text-sm">
            <span className="sr-only">{t('liveQuiz.editor.bankSearch')}</span>
            <input
              value={q}
              onChange={(e) => setQ(e.target.value)}
              placeholder={t('liveQuiz.editor.bankSearchPlaceholder')}
              className="w-full min-h-11 rounded-md border border-slate-300 px-3 py-2 dark:border-neutral-600 dark:bg-neutral-800 dark:text-neutral-100"
            />
          </label>
        </div>
        <div className="flex-1 overflow-y-auto p-4">
          {loading ? (
            <p className="text-sm text-slate-500">{t('common.loading')}</p>
          ) : rows.length === 0 ? (
            <p className="text-sm text-slate-500">{t('liveQuiz.editor.bankEmpty')}</p>
          ) : (
            <ul className="space-y-2">
              {rows.map((row) => (
                <li key={row.id}>
                  <label className="flex min-h-11 cursor-pointer items-start gap-3 rounded-md border border-slate-200 p-3 dark:border-neutral-700">
                    <input
                      type="checkbox"
                      checked={selected.has(row.id)}
                      onChange={() => toggle(row.id)}
                      className="mt-1"
                    />
                    <span className="min-w-0 flex-1">
                      <span className="block text-sm font-medium text-slate-900 dark:text-neutral-100">
                        {row.stem.slice(0, 120) || t('liveQuiz.editor.untitledBank')}
                      </span>
                      <span className="text-xs text-slate-500">{row.questionType}</span>
                    </span>
                  </label>
                </li>
              ))}
            </ul>
          )}
        </div>
        <div className="border-t border-slate-200 p-4 dark:border-neutral-700">
          <button
            type="button"
            disabled={importing || selected.size === 0}
            onClick={() => {
              void handleImport()
            }}
            className="w-full min-h-11 rounded-md bg-indigo-600 px-3 py-2 text-sm font-medium text-white disabled:opacity-50"
          >
            {t('liveQuiz.editor.importSelected', { count: selected.size })}
          </button>
        </div>
      </div>
    </div>
  )
}
