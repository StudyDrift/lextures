import { Pin } from 'lucide-react'
import { PINNED_SETTINGS_UI_CAP, pinnedSettingsCopy } from '../../lib/pinned-settings-copy'

export type PinToggleProps = {
  label: string
  pinned: boolean
  disabledAtCap?: boolean
  onToggle: () => void
  /** When true, always fully visible (pinned row or touch). */
  alwaysVisible?: boolean
  className?: string
}

/**
 * Per-row pin/unpin button (PS.3 FR-1–FR-5).
 * Opacity-based reveal so it stays keyboard-reachable when unfocused.
 */
export function PinToggle({
  label,
  pinned,
  disabledAtCap = false,
  onToggle,
  alwaysVisible = false,
  className = '',
}: PinToggleProps) {
  const disabled = !pinned && disabledAtCap
  const capId = disabled ? 'pinned-settings-cap-help' : undefined
  const name = pinned
    ? pinnedSettingsCopy.unpinAction(label)
    : pinnedSettingsCopy.pinAction(label)

  return (
    <button
      type="button"
      aria-pressed={pinned}
      aria-label={name}
      aria-describedby={capId}
      disabled={disabled}
      data-pin-toggle=""
      title={disabled ? pinnedSettingsCopy.capReached(PINNED_SETTINGS_UI_CAP) : name}
      onClick={(e) => {
        e.preventDefault()
        e.stopPropagation()
        if (!disabled) onToggle()
      }}
      className={[
        'inline-flex h-6 w-6 shrink-0 items-center justify-center rounded-md',
        'text-slate-400 outline-none motion-safe:transition-opacity',
        'hover:bg-slate-100 hover:text-indigo-600',
        'focus-visible:opacity-100 focus-visible:ring-2 focus-visible:ring-indigo-400',
        'dark:text-neutral-500 dark:hover:bg-neutral-800 dark:hover:text-indigo-400',
        'disabled:cursor-not-allowed disabled:opacity-40',
        pinned || alwaysVisible
          ? 'opacity-100'
          : 'opacity-0 group-hover/setting-row:opacity-100 group-focus-within/setting-row:opacity-100',
        pinned ? 'text-indigo-600 dark:text-indigo-400' : '',
        className,
      ]
        .filter(Boolean)
        .join(' ')}
    >
      <Pin
        className="h-3.5 w-3.5"
        aria-hidden
        fill={pinned ? 'currentColor' : 'none'}
        strokeWidth={pinned ? 1.5 : 2}
      />
    </button>
  )
}
