import { useId, useState } from 'react'
import { Loader2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'

type SaveTemplateDialogProps = {
  defaultName: string
  saving: boolean
  onClose: () => void
  onSave: (name: string) => void | Promise<void>
}

export function SaveTemplateDialog({
  defaultName,
  saving,
  onClose,
  onSave,
}: SaveTemplateDialogProps) {
  const { t } = useTranslation('common')
  const titleId = useId()
  const inputId = useId()
  const [name, setName] = useState(defaultName)
  const [error, setError] = useState<string | null>(null)

  const submit = async () => {
    const trimmed = name.trim()
    if (!trimmed) {
      setError(t('gradingAgent.save.templateNameRequired'))
      return
    }
    setError(null)
    try {
      await onSave(trimmed)
    } catch (e) {
      setError(e instanceof Error ? e.message : t('gradingAgent.error.saveTemplate'))
    }
  }

  return (
    <div
      className="fixed inset-0 z-[530] flex items-center justify-center bg-black/40 p-4"
      role="presentation"
      onClick={onClose}
    >
      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        className="w-full max-w-sm rounded-2xl bg-white p-6 shadow-xl dark:bg-neutral-900"
        onClick={(e) => e.stopPropagation()}
      >
        <h3 id={titleId} className="text-sm font-semibold text-slate-900 dark:text-neutral-100">
          {t('gradingAgent.save.templateDialogTitle')}
        </h3>
        <label htmlFor={inputId} className="mt-4 mb-1 block text-xs font-medium text-slate-600 dark:text-neutral-400">
          {t('gradingAgent.save.templateNameLabel')}
        </label>
        <input
          id={inputId}
          autoFocus
          type="text"
          value={name}
          disabled={saving}
          onChange={(e) => setName(e.target.value)}
          placeholder={t('gradingAgent.save.templateNamePlaceholder')}
          className="w-full rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 disabled:opacity-60 dark:border-neutral-700 dark:bg-neutral-800 dark:text-neutral-100"
          onKeyDown={(e) => {
            if (e.key === 'Enter') void submit()
            if (e.key === 'Escape') onClose()
          }}
        />
        {error ? (
          <p className="mt-2 text-sm text-rose-600 dark:text-rose-400" role="alert">
            {error}
          </p>
        ) : null}
        <div className="mt-4 flex justify-end gap-2">
          <button
            type="button"
            disabled={saving}
            onClick={onClose}
            className="rounded-lg px-3 py-1.5 text-sm text-slate-600 hover:bg-slate-100 disabled:opacity-60 dark:text-neutral-400 dark:hover:bg-neutral-800"
          >
            {t('gradingAgent.save.templateCancel')}
          </button>
          <button
            type="button"
            disabled={saving || !name.trim()}
            onClick={() => void submit()}
            className="inline-flex items-center gap-2 rounded-lg bg-indigo-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50"
          >
            {saving ? (
              <>
                <Loader2 className="h-4 w-4 motion-safe:animate-spin" aria-hidden />
                {t('gradingAgent.save.saving')}
              </>
            ) : (
              t('gradingAgent.save.asTemplate')
            )}
          </button>
        </div>
      </div>
    </div>
  )
}
