import type { DiagramPrompt } from './types'

export type HotspotPromptBarProps = {
  prompts: DiagramPrompt[]
  activePromptIndex: number
  selectedRegionId: string | null
  selectedRegionLabel: string
  t: (key: string, opts?: Record<string, unknown>) => string
  onPrev: () => void
  onNext: () => void
}

export function HotspotPromptBar({
  prompts,
  activePromptIndex,
  selectedRegionId,
  selectedRegionLabel,
  t,
  onPrev,
  onNext,
}: HotspotPromptBarProps) {
  return (
    <div className="space-y-1" data-testid="diagram-hotspot-prompt">
      <p className="text-sm text-slate-700 dark:text-neutral-200">
        {prompts[activePromptIndex]?.text ?? ''}
      </p>
      {prompts.length > 1 ? (
        <div className="flex gap-2">
          <button
            type="button"
            className="text-xs underline"
            disabled={activePromptIndex <= 0}
            onClick={onPrev}
          >
            {t('contentTools.tools.diagram_hotspot.prevPrompt')}
          </button>
          <button
            type="button"
            className="text-xs underline"
            disabled={activePromptIndex >= prompts.length - 1}
            onClick={onNext}
          >
            {t('contentTools.tools.diagram_hotspot.nextPrompt')}
          </button>
        </div>
      ) : null}
      {selectedRegionId ? (
        <p className="text-xs text-slate-600 dark:text-neutral-300" data-testid="diagram-selected">
          {t('contentTools.tools.diagram_hotspot.selected', { region: selectedRegionLabel })}
        </p>
      ) : null}
    </div>
  )
}
