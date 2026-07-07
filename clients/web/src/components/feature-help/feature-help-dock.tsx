import { useEffect } from 'react'
import { createPortal } from 'react-dom'
import { X } from 'lucide-react'
import { useFeatureHelp } from '../../context/feature-help-context'
import { FEATURE_HELP_BODY, FEATURE_HELP_MEDIA, FEATURE_HELP_TITLES } from '../../lib/feature-help-content'
import { FeatureHelpMediaRegion } from './feature-help-media'

export function FeatureHelpDock() {
  const {
    state: { open, topic },
    closeHelp,
  } = useFeatureHelp()

  useEffect(() => {
    if (!open) return
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') closeHelp()
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [open, closeHelp])

  if (!open || !topic) return null

  const title = FEATURE_HELP_TITLES[topic]
  const body = FEATURE_HELP_BODY[topic]
  const media = FEATURE_HELP_MEDIA[topic]

  return createPortal(
    <div className="fixed inset-0 z-[400] flex justify-end" role="dialog" aria-modal="true" aria-label={`${title} help`}>
      <button
        type="button"
        className="absolute inset-0 cursor-default bg-slate-950/40 backdrop-blur-[2px] dark:bg-black/50"
        aria-label="Close help"
        onClick={() => closeHelp()}
      />
      <aside className="relative z-10 flex h-full w-full max-w-md flex-col border-s border-slate-200 bg-white shadow-2xl dark:border-neutral-700 dark:bg-neutral-900">
        <div className="flex items-start justify-between gap-3 border-b border-slate-100 px-4 py-3 dark:border-neutral-800">
          <div>
            <p className="text-xs font-semibold uppercase tracking-wide text-indigo-600 dark:text-indigo-400">Help</p>
            <h2 className="mt-0.5 text-lg font-semibold tracking-tight text-slate-900 dark:text-neutral-100">{title}</h2>
          </div>
          <button
            type="button"
            onClick={() => closeHelp()}
            className="rounded-lg p-2 text-slate-500 transition-[background-color,color,border-color] hover:bg-slate-100 hover:text-slate-800 dark:text-neutral-400 dark:hover:bg-neutral-800 dark:hover:text-neutral-100"
            aria-label="Close"
          >
            <X className="h-5 w-5" aria-hidden />
          </button>
        </div>
        <div className="min-h-0 flex-1 overflow-y-auto px-4 py-4">
          {media ? <FeatureHelpMediaRegion media={media} active={open} /> : null}
          <p className="text-sm leading-relaxed text-slate-700 dark:text-neutral-300">{body}</p>
        </div>
      </aside>
    </div>,
    document.body,
  )
}