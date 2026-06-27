import { useId, useState, type KeyboardEvent } from 'react'
import { Loader2, Sparkles, X } from 'lucide-react'
import { useTranslation } from 'react-i18next'

type AiBuildPanelProps = {
  building: boolean
  onBuild: (instruction: string) => Promise<boolean>
}

/**
 * Floating canvas control: describe the grading logic in plain English and the
 * registered AI builds/modifies the node graph in place for review.
 */
export function AiBuildPanel({ building, onBuild }: AiBuildPanelProps) {
  const { t } = useTranslation('common')
  const [open, setOpen] = useState(false)
  const [instruction, setInstruction] = useState('')
  const textareaId = useId()

  const submit = async () => {
    if (building || instruction.trim() === '') return
    const ok = await onBuild(instruction)
    if (ok) {
      setInstruction('')
      setOpen(false)
    }
  }

  const handleKeyDown = (e: KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === 'Enter' && (e.metaKey || e.ctrlKey)) {
      e.preventDefault()
      void submit()
    }
  }

  if (!open) {
    return (
      <button
        type="button"
        onClick={() => setOpen(true)}
        className="absolute bottom-4 left-4 z-10 inline-flex items-center gap-2 rounded-full bg-indigo-600 px-4 py-2 text-sm font-semibold text-white shadow-lg hover:bg-indigo-700"
      >
        <Sparkles className="h-4 w-4" aria-hidden />
        {t('gradingAgent.aiBuilder.open')}
      </button>
    )
  }

  return (
    <div className="absolute bottom-4 left-4 z-10 w-[22rem] max-w-[calc(100%-2rem)] rounded-xl border border-slate-200 bg-white p-3 shadow-xl dark:border-neutral-700 dark:bg-neutral-900">
      <div className="mb-2 flex items-center justify-between">
        <span className="inline-flex items-center gap-1.5 text-sm font-semibold text-slate-900 dark:text-neutral-50">
          <Sparkles className="h-4 w-4 text-indigo-500" aria-hidden />
          {t('gradingAgent.aiBuilder.title')}
        </span>
        <button
          type="button"
          onClick={() => setOpen(false)}
          disabled={building}
          aria-label={t('gradingAgent.aiBuilder.close')}
          className="rounded-md p-1 text-slate-500 hover:bg-slate-100 disabled:opacity-50 dark:text-neutral-400 dark:hover:bg-neutral-800"
        >
          <X className="h-4 w-4" aria-hidden />
        </button>
      </div>
      <label htmlFor={textareaId} className="sr-only">
        {t('gradingAgent.aiBuilder.title')}
      </label>
      <textarea
        id={textareaId}
        value={instruction}
        onChange={(e) => setInstruction(e.target.value)}
        onKeyDown={handleKeyDown}
        rows={4}
        disabled={building}
        placeholder={t('gradingAgent.aiBuilder.placeholder')}
        className="w-full resize-none rounded-lg border border-slate-300 px-3 py-2 text-sm text-slate-900 placeholder:text-slate-400 focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500 disabled:opacity-60 dark:border-neutral-700 dark:bg-neutral-950 dark:text-neutral-100"
      />
      <div className="mt-2 flex items-center justify-between gap-2">
        <p className="text-xs text-slate-500 dark:text-neutral-400">{t('gradingAgent.aiBuilder.hint')}</p>
        <button
          type="button"
          onClick={() => void submit()}
          disabled={building || instruction.trim() === ''}
          className="inline-flex shrink-0 items-center gap-2 rounded-lg bg-indigo-600 px-3 py-2 text-sm font-semibold text-white hover:bg-indigo-700 disabled:opacity-50"
        >
          {building ? (
            <>
              <Loader2 className="h-4 w-4 motion-safe:animate-spin" aria-hidden />
              {t('gradingAgent.aiBuilder.generating')}
            </>
          ) : (
            t('gradingAgent.aiBuilder.generate')
          )}
        </button>
      </div>
    </div>
  )
}
