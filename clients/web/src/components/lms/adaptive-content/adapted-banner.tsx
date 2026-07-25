import { useEffect, useId, useRef, useState } from 'react'
import { Sparkles, ChevronDown, ChevronUp } from 'lucide-react'

export type AdaptedBannerProps = {
  adaptationReason?: string
  showingOriginal: boolean
  canViewOriginal: boolean
  optoutAllowed?: boolean
  onToggleOriginal: () => void
  onPreferStandard?: () => void
  /** AC.8 — report that this adaptation seems wrong (contest path). */
  onReportAdaptation?: () => void
  reportBusy?: boolean
  /** When true, expand the banner details on first render (mobile starts collapsed). */
  defaultExpanded?: boolean
}

/**
 * AC.6 — visible, dismissible "Adapted for you" indicator with View original toggle.
 * Announces adaptation to assistive tech via a live region on first render.
 */
export function AdaptedBanner({
  adaptationReason,
  showingOriginal,
  canViewOriginal,
  optoutAllowed,
  onToggleOriginal,
  onPreferStandard,
  onReportAdaptation,
  reportBusy,
  defaultExpanded = true,
}: AdaptedBannerProps) {
  const labelId = useId()
  const [expanded, setExpanded] = useState(defaultExpanded)
  const announcedRef = useRef(false)
  const [liveMessage, setLiveMessage] = useState('')

  useEffect(() => {
    if (announcedRef.current) return
    announcedRef.current = true
    setLiveMessage(
      showingOriginal
        ? 'Showing the original version of this section.'
        : 'This section has been adapted to your progress.',
    )
  }, [showingOriginal])

  const reason =
    adaptationReason?.trim() ||
    (showingOriginal ? undefined : 'matched to your progress')

  return (
    <div
      className="mb-4 rounded-lg border border-violet-200 bg-violet-50 px-4 py-3 text-sm text-violet-950 dark:border-violet-800 dark:bg-violet-950/40 dark:text-violet-100"
      role="region"
      aria-labelledby={labelId}
    >
      {/* Live region for first-render assistive announcement (FR-9). */}
      <div className="sr-only" role="status" aria-live="polite">
        {liveMessage}
      </div>

      <div className="flex flex-wrap items-start justify-between gap-2">
        <div className="min-w-0 flex-1">
          <p id={labelId} className="inline-flex items-center gap-1.5 font-medium">
            <Sparkles className="h-4 w-4 shrink-0 text-violet-600 dark:text-violet-300" aria-hidden />
            {showingOriginal ? 'Showing the original' : 'Adapted for you'}
            {!showingOriginal && reason ? (
              <span className="hidden font-normal text-violet-800 dark:text-violet-200 sm:inline">
                — {reason}
              </span>
            ) : null}
          </p>
          {!showingOriginal && reason ? (
            <p className="mt-0.5 text-violet-800 dark:text-violet-200 sm:hidden">{reason}</p>
          ) : null}
        </div>

        <button
          type="button"
          className="inline-flex items-center gap-1 rounded-md px-2 py-1 text-xs font-medium text-violet-800 hover:bg-violet-100 dark:text-violet-200 dark:hover:bg-violet-900/50 sm:hidden"
          aria-expanded={expanded}
          onClick={() => setExpanded((v) => !v)}
        >
          {expanded ? (
            <>
              Less <ChevronUp className="h-3.5 w-3.5" aria-hidden />
            </>
          ) : (
            <>
              More <ChevronDown className="h-3.5 w-3.5" aria-hidden />
            </>
          )}
        </button>
      </div>

      <div className={`mt-2 flex flex-wrap items-center gap-x-4 gap-y-2 ${expanded ? '' : 'hidden sm:flex'}`}>
        {canViewOriginal ? (
          <button
            type="button"
            className="text-sm font-medium underline underline-offset-2"
            aria-pressed={showingOriginal}
            onClick={onToggleOriginal}
          >
            {showingOriginal ? 'View adapted' : 'View original'}
          </button>
        ) : null}
        {optoutAllowed && onPreferStandard ? (
          <button
            type="button"
            className="text-sm text-violet-800 underline underline-offset-2 dark:text-violet-200"
            onClick={onPreferStandard}
          >
            Prefer standard content?
          </button>
        ) : null}
        {onReportAdaptation && !showingOriginal ? (
          <button
            type="button"
            className="text-sm text-violet-800 underline underline-offset-2 dark:text-violet-200"
            disabled={reportBusy}
            onClick={onReportAdaptation}
          >
            Report this adaptation
          </button>
        ) : null}
      </div>
    </div>
  )
}
