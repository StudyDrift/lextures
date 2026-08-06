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
        className="absolute bottom-4 left-4 z-10 inline-flex items-center gap-2 rounded-full bg-accent-solid px-4 py-2 text-sm font-semibold text-white shadow-lg hover:bg-accent"
      >
        <Sparkles className="h-4 w-4" aria-hidden />
        {t('gradingAgent.aiBuilder.open')}
      </button>
    )
  }

  return (
    <div className="absolute bottom-4 left-4 z-10 w-[22rem] max-w-[calc(100%-2rem)] rounded-xl border border-border-default bg-surface-raised p-3 shadow-xl dark:border-border-default dark:bg-surface-raised">
      <div className="mb-2 flex items-center justify-between">
        <span className="inline-flex items-center gap-1.5 text-sm font-semibold text-fg-default">
          <Sparkles className="h-4 w-4 text-indigo-500" aria-hidden />
          {t('gradingAgent.aiBuilder.title')}
        </span>
        <button
          type="button"
          onClick={() => setOpen(false)}
          disabled={building}
          aria-label={t('gradingAgent.aiBuilder.close')}
          className="rounded-md p-1 text-fg-muted hover:bg-surface-sunken disabled:opacity-50 dark:text-fg-muted dark:hover:bg-surface-overlay"
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
        className="w-full resize-none rounded-lg border border-border-strong px-3 py-2 text-sm text-fg-default placeholder:text-fg-subtle focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500 disabled:opacity-60 dark:border-border-default dark:bg-surface-base dark:text-fg-default"
      />
      <div className="mt-2 flex items-center justify-between gap-2">
        <p className="text-xs text-fg-muted">{t('gradingAgent.aiBuilder.hint')}</p>
        <button
          type="button"
          onClick={() => void submit()}
          disabled={building || instruction.trim() === ''}
          className="inline-flex shrink-0 items-center gap-2 rounded-lg bg-accent-solid px-3 py-2 text-sm font-semibold text-white hover:bg-accent disabled:opacity-50"
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
