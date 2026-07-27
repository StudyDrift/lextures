import type { ReactNode } from 'react'
import type { PlacementEngineState } from '../../shared/placement-engine'
import type { SortItem } from './types'

export type OrderListProps = {
  orderedItems: SortItem[]
  placement: string[]
  grabbedId: string | null
  target: PlacementEngineState['target']
  lockedItemIds: string[]
  readOnly: boolean
  dragItemId: string | null
  setDragItemId: (id: string | null) => void
  t: (key: string, options?: Record<string, unknown>) => string
  onDropAt: (itemId: string, index: number) => void
  onMove: (itemId: string, index: number) => void
  onEndActivate: () => void
  renderChip: (item: SortItem, opts?: { inList?: boolean }) => ReactNode
}

export function OrderList({
  orderedItems,
  placement,
  grabbedId,
  target,
  lockedItemIds,
  readOnly,
  dragItemId,
  setDragItemId,
  t,
  onDropAt,
  onMove,
  onEndActivate,
  renderChip,
}: OrderListProps) {
  return (
    <ol
      className="space-y-2"
      data-testid="sort-order-list"
      aria-label={t('contentTools.tools.sort_sequence.sequence')}
    >
      {placement.map((id, index) => {
        const item = orderedItems.find((i) => i.id === id)
        if (!item) return null
        const focused = target?.kind === 'position' && target.index === index && Boolean(grabbedId)
        return (
          <li
            key={id}
            className={`flex items-center gap-2 ${focused ? 'ring-2 ring-sky-500 rounded' : ''}`}
            onDragOver={(e) => {
              if (dragItemId) e.preventDefault()
            }}
            onDrop={(e) => {
              e.preventDefault()
              const dropId = e.dataTransfer.getData('text/plain') || dragItemId
              if (!dropId) return
              onDropAt(dropId, index)
              setDragItemId(null)
            }}
          >
            <span className="w-6 text-xs tabular-nums text-slate-500">{index + 1}</span>
            <div className="min-w-0 flex-1">{renderChip(item, { inList: true })}</div>
            <div className="flex flex-col gap-0.5">
              <button
                type="button"
                className="rounded border border-slate-300 px-1.5 text-xs dark:border-neutral-600"
                aria-label={t('contentTools.tools.sort_sequence.moveUp')}
                disabled={readOnly || lockedItemIds.includes(id) || index === 0}
                onClick={() => {
                  if (index === 0) return
                  onMove(id, index - 1)
                }}
              >
                ↑
              </button>
              <button
                type="button"
                className="rounded border border-slate-300 px-1.5 text-xs dark:border-neutral-600"
                aria-label={t('contentTools.tools.sort_sequence.moveDown')}
                disabled={readOnly || lockedItemIds.includes(id) || index >= placement.length - 1}
                onClick={() => onMove(id, index + 1)}
              >
                ↓
              </button>
            </div>
          </li>
        )
      })}
      <li
        className={`min-h-10 rounded border border-dashed border-slate-300 p-2 text-xs text-slate-500 dark:border-neutral-600 ${
          target?.kind === 'position' && target.index === placement.length ? 'ring-2 ring-sky-500' : ''
        }`}
        onDragOver={(e) => {
          if (dragItemId) e.preventDefault()
        }}
        onDrop={(e) => {
          e.preventDefault()
          const dropId = e.dataTransfer.getData('text/plain') || dragItemId
          if (!dropId) return
          onDropAt(dropId, placement.length)
          setDragItemId(null)
        }}
        onClick={() => {
          if (grabbedId) onEndActivate()
        }}
      >
        {t('contentTools.tools.sort_sequence.dropHere')}
      </li>
    </ol>
  )
}
