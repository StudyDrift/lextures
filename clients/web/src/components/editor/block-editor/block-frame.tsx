import { ChevronDown, ChevronUp, Trash2 } from 'lucide-react'
import type { ReactNode } from 'react'
import { useBlockEditor } from './block-editor-provider'

export type BlockFrameProps = {
  blockId: string
  /** e.g. “Section 1 of 4” — always visible so the writer knows where they are. */
  positionLabel: string
  onMoveUp?: () => void
  onMoveDown?: () => void
  moveUpDisabled?: boolean
  moveDownDisabled?: boolean
  onRemove?: () => void
  /** Explains why remove is unavailable (e.g. last remaining block). */
  removeDisabledReason?: string
  /**
   * Formatting controls pinned into the header while this block is being
   * written in. Present means the action labels collapse to icons to make room.
   */
  toolbar?: ReactNode
  children: ReactNode
  className?: string
}

const ACTION_BUTTON =
  'flex h-7 items-center gap-1.5 rounded-md px-2 text-xs font-medium disabled:cursor-not-allowed disabled:opacity-40 motion-safe:transition-colors'

/**
 * Wraps a single block as a card with a persistent header.
 *
 * The header keeps its height at all times and the actions inside it only fade
 * in, so moving the pointer across the document never reflows the page.
 */
export function BlockFrame({
  blockId,
  positionLabel,
  onMoveUp,
  onMoveDown,
  moveUpDisabled,
  moveDownDisabled,
  onRemove,
  removeDisabledReason,
  toolbar,
  children,
  className,
}: BlockFrameProps) {
  const { selectedId, setSelectedId, disabled } = useBlockEditor()
  const selected = selectedId === blockId

  // Actions are revealed by opacity only — never by height or margin.
  const reveal = selected
    ? 'opacity-100'
    : 'opacity-0 pointer-events-none group-hover:opacity-100 group-hover:pointer-events-auto group-focus-within:opacity-100 group-focus-within:pointer-events-auto'

  // With the toolbar present there is no room for words next to the icons.
  const actionLabel = toolbar ? 'sr-only' : 'hidden sm:inline'

  return (
    <div
      className={[
        '@container group relative mb-3 rounded-xl border bg-white motion-safe:transition-[border-color,box-shadow] dark:bg-neutral-900',
        selected
          ? 'border-indigo-400 shadow-sm shadow-indigo-900/10 ring-1 ring-indigo-400/25 dark:border-indigo-500 dark:ring-indigo-500/25'
          : 'border-slate-200 hover:border-slate-300 dark:border-neutral-700 dark:hover:border-neutral-600',
        disabled ? 'opacity-60' : '',
        className,
      ]
        .filter(Boolean)
        .join(' ')}
      onClick={(e) => {
        e.stopPropagation()
        setSelectedId(blockId)
      }}
      onKeyDown={(e) => {
        if (e.key === 'Enter' || e.key === ' ') {
          e.stopPropagation()
          setSelectedId(blockId)
        }
      }}
      role="group"
      aria-label={positionLabel}
    >
      <div
        className={[
          'sticky top-0 z-20 flex h-10 items-center gap-2 rounded-t-xl border-b px-3',
          'bg-white/95 backdrop-blur-sm dark:bg-neutral-900/95',
          selected
            ? 'border-slate-200 dark:border-neutral-700'
            : 'border-transparent group-hover:border-slate-100 group-focus-within:border-slate-100 dark:group-hover:border-neutral-800 dark:group-focus-within:border-neutral-800',
        ].join(' ')}
      >
        <span
          className={[
            'min-w-0 truncate text-xs font-medium text-slate-500 dark:text-neutral-400',
            // With the toolbar in the row, drop the label rather than ellipsing it.
            toolbar ? 'hidden @[34rem]:block' : 'block',
          ].join(' ')}
        >
          {positionLabel}
        </span>
        <div className="ms-auto flex shrink-0 items-center gap-1">
          {toolbar}
          {toolbar && (
            <span className="mx-0.5 h-5 w-px bg-slate-200 dark:bg-neutral-600" aria-hidden />
          )}
          <div className={`flex items-center gap-0.5 duration-150 motion-safe:transition-opacity ${reveal}`}>
            {onMoveUp && (
              <button
                type="button"
                disabled={disabled || moveUpDisabled}
                onClick={onMoveUp}
                className={`${ACTION_BUTTON} text-slate-600 hover:bg-slate-100 hover:text-slate-900 dark:text-neutral-300 dark:hover:bg-neutral-800 dark:hover:text-neutral-50`}
                title="Move this section up"
              >
                <ChevronUp className="h-4 w-4" aria-hidden />
                <span className={actionLabel}>Move up</span>
              </button>
            )}
            {onMoveDown && (
              <button
                type="button"
                disabled={disabled || moveDownDisabled}
                onClick={onMoveDown}
                className={`${ACTION_BUTTON} text-slate-600 hover:bg-slate-100 hover:text-slate-900 dark:text-neutral-300 dark:hover:bg-neutral-800 dark:hover:text-neutral-50`}
                title="Move this section down"
              >
                <ChevronDown className="h-4 w-4" aria-hidden />
                <span className={actionLabel}>Move down</span>
              </button>
            )}
            {onRemove && (
              <button
                type="button"
                disabled={disabled || Boolean(removeDisabledReason)}
                onClick={onRemove}
                className={`${ACTION_BUTTON} text-slate-600 hover:bg-rose-50 hover:text-rose-700 dark:text-neutral-300 dark:hover:bg-rose-950/50 dark:hover:text-rose-400`}
                title={removeDisabledReason ?? 'Delete this section'}
              >
                <Trash2 className="h-4 w-4" aria-hidden />
                <span className={actionLabel}>Delete</span>
              </button>
            )}
          </div>
        </div>
      </div>
      <div className="px-5 pb-6 pt-4">{children}</div>
    </div>
  )
}
