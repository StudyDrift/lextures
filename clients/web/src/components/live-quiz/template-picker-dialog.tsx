import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { createKitFromTemplate, listQuizTemplates, type QuizKit } from '../../lib/live-quiz-api'
import { toastMutationError } from '../../lib/lms-toast'

type Props = {
  courseCode: string
  open: boolean
  onClose: () => void
  onCreated: (kit: QuizKit) => void
}

export function TemplatePickerDialog({ courseCode, open, onClose, onCreated }: Props) {
  const { t } = useTranslation('common')
  const [templates, setTemplates] = useState<QuizKit[]>([])
  const [loading, setLoading] = useState(false)
  const [creatingId, setCreatingId] = useState<string | null>(null)

  useEffect(() => {
    if (!open) return
    setLoading(true)
    void listQuizTemplates({ courseCode })
      .then(setTemplates)
      .catch((err) => toastMutationError(err instanceof Error ? err.message : String(err)))
      .finally(() => setLoading(false))
  }, [open, courseCode])

  if (!open) return null

  async function handleCreate(templateId: string) {
    setCreatingId(templateId)
    try {
      const kit = await createKitFromTemplate(templateId, courseCode)
      onCreated(kit)
      onClose()
    } catch (err) {
      toastMutationError(err instanceof Error ? err.message : String(err))
    } finally {
      setCreatingId(null)
    }
  }

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4"
      role="dialog"
      aria-modal="true"
      aria-labelledby="template-picker-title"
    >
      <div className="flex max-h-[80vh] w-full max-w-2xl flex-col rounded-lg border border-slate-200 bg-white shadow-xl dark:border-neutral-700 dark:bg-neutral-900">
        <div className="flex items-start justify-between gap-3 border-b border-slate-200 p-4 dark:border-neutral-700">
          <div>
            <h2
              id="template-picker-title"
              className="text-lg font-semibold text-slate-900 dark:text-neutral-100"
            >
              {t('liveQuiz.template.pickerTitle')}
            </h2>
            <p className="mt-1 text-sm text-slate-600 dark:text-neutral-300">
              {t('liveQuiz.template.pickerSubtitle')}
            </p>
          </div>
          <button
            type="button"
            onClick={onClose}
            className="min-h-11 rounded-md px-3 text-sm text-slate-600 hover:bg-slate-100 dark:text-neutral-300 dark:hover:bg-neutral-800"
          >
            {t('dialogs.cancel')}
          </button>
        </div>
        <div className="overflow-y-auto p-4">
          {loading ? (
            <p className="text-sm text-slate-500">{t('common.loading')}</p>
          ) : templates.length === 0 ? (
            <p className="text-sm text-slate-500">{t('liveQuiz.template.empty')}</p>
          ) : (
            <ul className="space-y-2">
              {templates.map((tmpl) => (
                <li
                  key={tmpl.id}
                  className="flex flex-wrap items-center justify-between gap-3 rounded-md border border-slate-200 p-3 dark:border-neutral-700"
                >
                  <div>
                    <p className="font-medium text-slate-900 dark:text-neutral-100">{tmpl.title}</p>
                    {tmpl.description ? (
                      <p className="mt-0.5 text-sm text-slate-600 dark:text-neutral-300">
                        {tmpl.description}
                      </p>
                    ) : null}
                    <p className="mt-1 text-xs text-slate-500">
                      {tmpl.templateScope ?? 'system'} ·{' '}
                      {t('liveQuiz.gallery.questionCount', { count: tmpl.questionCount })}
                    </p>
                  </div>
                  <button
                    type="button"
                    disabled={creatingId === tmpl.id}
                    onClick={() => void handleCreate(tmpl.id)}
                    className="min-h-11 rounded-md bg-indigo-600 px-3 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50"
                  >
                    {creatingId === tmpl.id ? t('common.loading') : t('liveQuiz.template.use')}
                  </button>
                </li>
              ))}
            </ul>
          )}
        </div>
      </div>
    </div>
  )
}
