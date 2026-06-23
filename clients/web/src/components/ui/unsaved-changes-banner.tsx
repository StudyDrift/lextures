import { Loader2, Save } from 'lucide-react'
import { useExitTransition } from '../../hooks/use-exit-transition'
import { Button } from './button'

export type UnsavedChangesBannerProps = {
  visible: boolean
  description: string
  saveStatus: 'idle' | 'saving' | 'error' | 'saved'
  saveMessage?: string | null
  onDiscard: () => void
  onSave: () => void
}

export function UnsavedChangesBanner({
  visible,
  description,
  saveStatus,
  saveMessage,
  onDiscard,
  onSave,
}: UnsavedChangesBannerProps) {
  const { rendered, exiting, onTransitionEnd } = useExitTransition(visible)

  if (!rendered) return null

  return (
    <div
      onTransitionEnd={onTransitionEnd}
      className={[
        'fixed bottom-6 start-1/2 z-50 w-full max-w-2xl -translate-x-1/2 px-4',
        exiting
          ? 'lex-banner-exit'
          : 'motion-safe:animate-in motion-safe:fade-in motion-safe:slide-in-from-bottom-4 duration-300',
      ].join(' ')}
    >
      <div className="flex items-center justify-between rounded-2xl border border-slate-200 bg-white/90 px-6 py-4 shadow-xl backdrop-blur-md dark:border-neutral-800 dark:bg-neutral-900/90">
        <div className="flex flex-col">
          <span className="text-sm font-semibold text-slate-900 dark:text-neutral-50">Unsaved changes</span>
          <span className="text-xs text-slate-500 dark:text-neutral-400">
            {saveStatus === 'error' && saveMessage ? (
              <span className="font-medium text-rose-600 dark:text-rose-400">{saveMessage}</span>
            ) : (
              description
            )}
          </span>
        </div>
        <div className="flex items-center gap-3">
          <Button variant="ghost" onClick={onDiscard} disabled={saveStatus === 'saving'} className="py-2.5">
            Discard
          </Button>
          <Button
            onClick={onSave}
            disabled={saveStatus === 'saving'}
            className="gap-2 px-5 py-2.5 shadow-md"
          >
            {saveStatus === 'saving' ? (
              <>
                <Loader2 className="h-4 w-4 motion-safe:animate-spin" aria-hidden />
                Saving...
              </>
            ) : (
              <>
                <Save className="h-4 w-4" aria-hidden />
                Save changes
              </>
            )}
          </Button>
        </div>
      </div>
    </div>
  )
}