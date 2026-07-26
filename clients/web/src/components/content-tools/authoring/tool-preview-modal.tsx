import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  getContentToolState,
  putContentToolState,
  type ContentToolInstance,
} from '../../../lib/courses-api'

export type ToolPreviewModalProps = {
  open: boolean
  courseCode: string
  instance: ContentToolInstance | null
  onClose: () => void
}

export function ToolPreviewModal({ open, courseCode, instance, onClose }: ToolPreviewModalProps) {
  const { t } = useTranslation('contentTools')
  const [answer, setAnswer] = useState('')
  const [revision, setRevision] = useState(0)
  const [localOnly, setLocalOnly] = useState(false)
  const [status, setStatus] = useState<string | null>(null)

  useEffect(() => {
    if (!open || !instance) return
    setAnswer('')
    setStatus(null)
    setLocalOnly(false)
    setRevision(0)
    let cancelled = false
    void getContentToolState(courseCode, instance.id, { scope: 'preview' })
      .then((st) => {
        if (cancelled) return
        setRevision(st.revision)
        const response = st.stateJson.response
        if (typeof response === 'string') setAnswer(response)
      })
      .catch(() => {
        if (cancelled) return
        setLocalOnly(true)
      })
    return () => {
      cancelled = true
    }
  }, [open, courseCode, instance])

  if (!open || !instance) return null

  const prompt =
    typeof instance.config.prompt === 'string'
      ? instance.config.prompt
      : t('contentTools.authoring.previewNoPrompt')

  async function submitPreview() {
    if (!instance) return
    if (localOnly) {
      setStatus(t('contentTools.authoring.previewLocalSaved'))
      return
    }
    try {
      const next = await putContentToolState(courseCode, instance.id, {
        stateJson: { response: answer, attempts: 1 },
        revision,
        scope: 'preview',
      })
      setRevision(next.revision)
      setStatus(t('contentTools.authoring.previewSaved'))
    } catch {
      setLocalOnly(true)
      setStatus(t('contentTools.authoring.previewLocalSaved'))
    }
  }

  function resetPreview() {
    setAnswer('')
    setStatus(null)
    setRevision(0)
  }

  return (
    <div
      className="fixed inset-0 z-[80] flex items-center justify-center bg-slate-900/40 p-4"
      role="dialog"
      aria-modal="true"
      aria-labelledby="content-tool-preview-title"
    >
      <div className="max-h-[90vh] w-full max-w-lg overflow-y-auto rounded-lg border border-slate-200 bg-white shadow-xl dark:border-neutral-700 dark:bg-neutral-900">
        <div className="border-b border-amber-200 bg-amber-50 px-4 py-2 text-sm text-amber-900 dark:border-amber-900/50 dark:bg-amber-950/40 dark:text-amber-200">
          {t('contentTools.authoring.previewBanner')}
        </div>
        <div className="space-y-3 p-4">
          <div className="flex items-start justify-between gap-2">
            <h2
              id="content-tool-preview-title"
              className="text-sm font-semibold text-slate-900 dark:text-neutral-100"
            >
              {t('contentTools.authoring.previewTitle')}
            </h2>
            <button
              type="button"
              onClick={onClose}
              className="rounded px-2 py-1 text-xs font-medium text-slate-600 hover:bg-slate-100 dark:text-neutral-300 dark:hover:bg-neutral-800"
            >
              {t('contentTools.authoring.close')}
            </button>
          </div>
          {localOnly ? (
            <p className="text-xs text-slate-500 dark:text-neutral-400">
              {t('contentTools.authoring.previewLocalOnly')}
            </p>
          ) : null}
          <p className="text-sm text-slate-800 dark:text-neutral-200">{prompt}</p>
          {instance.toolId === 'noop_probe' ? (
            <label className="block space-y-1">
              <span className="text-xs font-medium text-slate-600 dark:text-neutral-300">
                {t('contentTools.authoring.previewAnswer')}
              </span>
              <input
                type="text"
                value={answer}
                onChange={(e) => setAnswer(e.target.value)}
                className="w-full rounded-md border border-slate-200 bg-white px-2.5 py-1.5 text-sm dark:border-neutral-600 dark:bg-neutral-950 dark:text-neutral-100"
              />
            </label>
          ) : (
            <p className="text-xs text-slate-500 dark:text-neutral-400">
              {t('contentTools.authoring.previewGeneric')}
            </p>
          )}
          {status ? (
            <p className="text-xs text-slate-600 dark:text-neutral-300" role="status">
              {status}
            </p>
          ) : null}
          <div className="flex flex-wrap gap-2">
            <button
              type="button"
              onClick={() => void submitPreview()}
              className="rounded-md bg-slate-800 px-3 py-1.5 text-xs font-medium text-white hover:bg-slate-700 dark:bg-neutral-200 dark:text-neutral-900"
            >
              {t('contentTools.authoring.previewSubmit')}
            </button>
            <button
              type="button"
              onClick={resetPreview}
              className="rounded-md px-3 py-1.5 text-xs font-medium text-slate-700 hover:bg-slate-100 dark:text-neutral-200 dark:hover:bg-neutral-800"
            >
              {t('contentTools.authoring.resetPreview')}
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}
