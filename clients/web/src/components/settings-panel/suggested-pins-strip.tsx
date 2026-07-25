/**
 * Cold-start suggested pins strip (PS.4 FR-1–FR-7).
 * Renders only when `pins.suggestionsEligible` — zero pins, not dismissed,
 * flag on, pins loaded, and at least one resolvable curated suggestion.
 */
import { useId } from 'react'
import { pinnedSettingsCopy } from '../../lib/pinned-settings-copy'
import type { UsePinnedSettingsResult } from './use-pinned-settings'

export type SuggestedPinsStripProps = {
  pins: UsePinnedSettingsResult
}

export function SuggestedPinsStrip({ pins }: SuggestedPinsStripProps) {
  const headingId = useId()

  if (!pins.suggestionsEligible) return null

  const suggestions = pins.suggestedPins
  if (suggestions.length === 0) return null

  return (
    <section
      aria-labelledby={headingId}
      className="rounded-lg border border-dashed border-indigo-200/80 bg-indigo-50/40 px-3 py-2.5 dark:border-indigo-800/50 dark:bg-indigo-950/20"
      data-testid="suggested-pins-strip"
    >
      <div className="flex items-start justify-between gap-2">
        <div className="min-w-0 space-y-1">
          <h3
            id={headingId}
            className="text-[12px] font-medium leading-snug text-slate-700 dark:text-neutral-200"
          >
            {pinnedSettingsCopy.suggestions.heading}
          </h3>
          <p className="text-[11px] leading-relaxed text-slate-500 dark:text-neutral-400">
            {pinnedSettingsCopy.suggestions.intro}
          </p>
        </div>
        <button
          type="button"
          onClick={pins.dismissSuggestions}
          className="shrink-0 rounded-md px-2 py-1 text-[12px] font-medium text-slate-500 outline-none hover:bg-white/80 hover:text-slate-700 focus-visible:ring-2 focus-visible:ring-indigo-400 dark:text-neutral-400 dark:hover:bg-neutral-800 dark:hover:text-neutral-200"
          aria-label={pinnedSettingsCopy.suggestions.dismissAria}
        >
          {pinnedSettingsCopy.suggestions.dismiss}
        </button>
      </div>
      <ul className="mt-2 flex flex-wrap gap-1.5">
        {suggestions.map((d) => (
          <li key={d.id}>
            <button
              type="button"
              onClick={() => pins.pin(d.id, { fromSuggestion: true })}
              className="inline-flex max-w-full items-center rounded-full border border-indigo-200/90 bg-white px-2.5 py-1 text-[12px] font-medium text-indigo-700 outline-none hover:border-indigo-300 hover:bg-indigo-50 focus-visible:ring-2 focus-visible:ring-indigo-400 dark:border-indigo-800 dark:bg-neutral-900 dark:text-indigo-300 dark:hover:border-indigo-700 dark:hover:bg-indigo-950/40"
              aria-label={pinnedSettingsCopy.suggestions.pinAction(d.label)}
            >
              <span className="truncate">{d.label}</span>
            </button>
          </li>
        ))}
      </ul>
      {/* Live region for pin acceptance announcements (shares PS.3 path via pins.announce). */}
      <div role="status" aria-live="polite" className="sr-only">
        {pins.liveMessage}
      </div>
    </section>
  )
}
