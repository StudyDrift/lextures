import { Maximize2 } from 'lucide-react'
import { useEffect, useId, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'

type InspectorTextareaExpandModalProps = {
  title: string
  onDone: () => void
  onCancel: () => void
  children: ReactNode
}

/** Room for the in-field expand control without overlapping wrapped text. */
export const EXPANDABLE_TEXTAREA_FIELD_CLASSES = 'pe-9 pb-7 [scrollbar-gutter:stable]'

export function InspectorTextareaExpandButton({
  label,
  onClick,
}: {
  label: string
  onClick: () => void
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className="absolute bottom-1.5 end-7 z-10 inline-flex items-center justify-center rounded-md bg-white/90 p-1 text-slate-500 shadow-sm ring-1 ring-slate-200/80 hover:bg-slate-100 hover:text-slate-700 dark:bg-neutral-950/90 dark:text-neutral-400 dark:ring-neutral-700 dark:hover:bg-neutral-800 dark:hover:text-neutral-200"
      aria-label={label}
    >
      <Maximize2 className="h-3.5 w-3.5" aria-hidden="true" />
    </button>
  )
}

export function InspectorTextareaExpandOverlay({
  label,
  onExpand,
  children,
}: {
  label: string
  onExpand: () => void
  children: ReactNode
}) {
  return (
    <div className="relative">
      {children}
      <InspectorTextareaExpandButton label={label} onClick={onExpand} />
    </div>
  )
}

export function InspectorTextareaExpandModal({
  title,
  onDone,
  onCancel,
  children,
}: InspectorTextareaExpandModalProps) {
  const { t } = useTranslation('common')
  const titleId = useId()

  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') onCancel()
    }
    window.addEventListener('keydown', onKeyDown)
    return () => window.removeEventListener('keydown', onKeyDown)
  }, [onCancel])

  return (
    <div
      className="fixed inset-0 z-[540] flex items-center justify-center bg-black/40 p-4"
      role="presentation"
      onClick={onCancel}
    >
      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        className="flex h-[min(80vh,640px)] w-full max-w-2xl flex-col rounded-2xl bg-white shadow-xl dark:bg-neutral-900"
        onClick={(event) => event.stopPropagation()}
      >
        <div className="shrink-0 border-b border-slate-200 px-5 py-4 dark:border-neutral-700">
          <h3 id={titleId} className="text-sm font-semibold text-slate-900 dark:text-neutral-100">
            {title}
          </h3>
        </div>
        <div className="flex min-h-0 flex-1 flex-col overflow-hidden px-5 py-4">{children}</div>
        <div className="flex shrink-0 justify-end gap-2 border-t border-slate-200 px-5 py-3 dark:border-neutral-700">
          <button
            type="button"
            onClick={onCancel}
            className="rounded-lg px-3 py-1.5 text-sm text-slate-600 hover:bg-slate-100 dark:text-neutral-400 dark:hover:bg-neutral-800"
          >
            {t('gradingAgent.canvas.inspector.expandTextareaCancel')}
          </button>
          <button
            type="button"
            onClick={onDone}
            className="rounded-lg bg-indigo-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-indigo-700"
          >
            {t('gradingAgent.canvas.inspector.expandTextareaDone')}
          </button>
        </div>
      </div>
    </div>
  )
}