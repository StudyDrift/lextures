import { CircleHelp } from 'lucide-react'
import type { FeatureHelpTopic } from '../../context/feature-help-context'
import { useFeatureHelp } from '../../context/feature-help-context'

export function FeatureHelpTrigger({
  topic,
  label = 'Open help for this area',
}: {
  topic: FeatureHelpTopic
  label?: string
}) {
  const { openHelp } = useFeatureHelp()
  return (
    <button
      type="button"
      onClick={() => openHelp(topic)}
      className="inline-flex h-8 w-8 shrink-0 items-center justify-center rounded-full border border-border-default bg-surface-raised text-fg-muted shadow-sm transition-[background-color,color,border-color] hover:border-indigo-200 hover:text-accent-fg dark:border-border-default dark:bg-surface-overlay dark:text-fg-muted dark:hover:border-indigo-500/40 dark:hover:text-indigo-200"
      aria-label={label}
    >
      <CircleHelp className="h-4 w-4" aria-hidden />
    </button>
  )
}
