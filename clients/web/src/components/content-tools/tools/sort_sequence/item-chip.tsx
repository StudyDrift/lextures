import type { DragEvent, KeyboardEvent } from 'react'
import type { SortItem } from './types'

export type SortItemChipProps = {
  item: SortItem
  mode: 'categorize' | 'order'
  locked: boolean
  grabbed: boolean
  readOnly: boolean
  correctness?: boolean
  feedback?: string
  dragClass: string
  inList?: boolean
  t: (key: string, options?: Record<string, unknown>) => string
  onClick: () => void
  onKeyDown: (e: KeyboardEvent) => void
  onDragStart: (e: DragEvent) => void
  onDragEnd: () => void
}

export function SortItemChip({
  item,
  mode,
  locked,
  grabbed,
  readOnly,
  correctness,
  feedback,
  dragClass,
  inList,
  t,
  onClick,
  onKeyDown,
  onDragStart,
  onDragEnd,
}: SortItemChipProps) {
  return (
    <button
      type="button"
      data-testid={`sort-item-${item.id}`}
      data-locked={locked ? 'true' : undefined}
      data-grabbed={grabbed ? 'true' : undefined}
      aria-roledescription={t('contentTools.tools.sort_sequence.roleDescription')}
      aria-grabbed={grabbed}
      aria-disabled={locked || readOnly}
      disabled={locked || readOnly}
      draggable={!locked && !readOnly}
      onDragStart={onDragStart}
      onDragEnd={onDragEnd}
      onClick={onClick}
      onKeyDown={onKeyDown}
      className={`inline-flex max-w-full items-center gap-2 rounded border px-2.5 py-1.5 text-start text-sm ${ grabbed ? 'border-sky-600 bg-sky-50 ring-2 ring-sky-500 dark:bg-sky-950/40' : 'border-border-strong bg-surface-raised dark:border-border-default dark:bg-surface-raised' } ${locked ? 'opacity-70' : ''} ${dragClass} ${inList ? 'w-full' : ''}`}
    >
      {mode === 'order' ? (
        <span className="text-fg-subtle" aria-hidden="true">
          ⋮⋮
        </span>
      ) : null}
      <span className="min-w-0 flex-1">
        <span className="block truncate font-medium text-fg-default">
          {item.text}
        </span>
        {item.imageUrl ? (
          <img src={item.imageUrl} alt={item.imageAlt || ''} className="mt-1 max-h-12 rounded" />
        ) : null}
        {feedback ? (
          <span className="mt-0.5 block text-xs text-fg-muted">{feedback}</span>
        ) : null}
      </span>
      {correctness === true ? (
        <span className="text-emerald-700" aria-label={t('contentTools.tools.sort_sequence.correct')}>
          ✓
        </span>
      ) : null}
      {correctness === false ? (
        <span className="text-rose-700" aria-label={t('contentTools.tools.sort_sequence.incorrect')}>
          ✗
        </span>
      ) : null}
      {locked ? (
        <span className="text-xs text-fg-muted">{t('contentTools.tools.sort_sequence.locked')}</span>
      ) : null}
    </button>
  )
}
