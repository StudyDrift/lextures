import type { ReactNode } from 'react'
import { itemsInBucket, type PlacementEngineState } from '../../shared/placement-engine'
import type { SortBucket, SortItem } from './types'

export type CategorizeBoardProps = {
  buckets: SortBucket[]
  orderedItems: SortItem[]
  placement: PlacementEngineState['placement']
  grabbedId: string | null
  target: PlacementEngineState['target']
  dragItemId: string | null
  setDragItemId: (id: string | null) => void
  t: (key: string, options?: Record<string, unknown>) => string
  onBucketActivate: (bucketId: string) => void
  onDropItem: (itemId: string, bucketId: string) => void
  renderChip: (item: SortItem, opts?: { inList?: boolean }) => ReactNode
}

export function CategorizeBoard({
  buckets,
  orderedItems,
  placement,
  grabbedId,
  target,
  dragItemId,
  setDragItemId,
  t,
  onBucketActivate,
  onDropItem,
  renderChip,
}: CategorizeBoardProps) {
  return (
    <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3" data-testid="sort-buckets">
      {buckets.map((bucket, bi) => {
        const inBucket = itemsInBucket(placement, bucket.id)
        const focused =
          target?.kind === 'bucket' && target.bucketId === bucket.id && Boolean(grabbedId)
        return (
          <section
            key={bucket.id}
            data-testid={`sort-bucket-${bucket.id}`}
            aria-label={`${bucket.label}, ${inBucket.length} ${t('contentTools.tools.sort_sequence.items')}`}
            onDragOver={(e) => {
              if (dragItemId) e.preventDefault()
            }}
            onDrop={(e) => {
              e.preventDefault()
              const id = e.dataTransfer.getData('text/plain') || dragItemId
              if (!id) return
              onDropItem(id, bucket.id)
              setDragItemId(null)
            }}
            onClick={() => {
              if (grabbedId) onBucketActivate(bucket.id)
            }}
            className={`sticky top-0 min-h-24 rounded border border-border-default bg-slate-50/80 p-2 dark:border-border-default/60 ${ focused ? 'ring-2 ring-sky-500' : '' }`}
          >
            <header className="mb-2">
              <h3 className="text-sm font-semibold text-fg-default">
                {bucket.label}
              </h3>
              {bucket.description ? (
                <p className="text-xs text-fg-muted">{bucket.description}</p>
              ) : null}
              <p className="text-xs text-fg-muted">
                {t('contentTools.tools.sort_sequence.bucketCount', {
                  count: inBucket.length,
                  index: bi + 1,
                  total: buckets.length,
                })}
              </p>
            </header>
            <div className="flex flex-col gap-2">
              {inBucket.map((id) => {
                const item = orderedItems.find((i) => i.id === id)
                return item ? renderChip(item, { inList: true }) : null
              })}
            </div>
          </section>
        )
      })}
    </div>
  )
}
