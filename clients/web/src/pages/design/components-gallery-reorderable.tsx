import { useMemo, useState } from 'react'
import { ClickToMoveDropZone, MoveToPositionMenu, Stack } from '../../components/ui'
import { GalleryBlock } from './gallery-block'

/** UX.5 reorderable primitives gallery demo (file-size split). */
export function ReorderableGalleryDemo() {
  const [reorderIds, setReorderIds] = useState(['a', 'b', 'c'])
  const reorderSiblings = useMemo(
    () =>
      reorderIds.map((id) => ({
        id,
        title: id === 'a' ? 'Alpha' : id === 'b' ? 'Beta' : 'Gamma',
      })),
    [reorderIds],
  )

  return (
    <GalleryBlock
      id="reorderable"
      title="Reorderable"
      pattern="Move-to menu + click-to-move drop zone (UX.5 SC 2.5.7)"
      keyboard="Open Move to…; Escape cancels click-to-move"
    >
      <Stack gap="sm">
        {reorderSiblings.map((item) => (
          <ClickToMoveDropZone
            key={item.id}
            id={item.id}
            active={false}
            isSource={false}
            isValidTarget={false}
            onSelect={() => undefined}
            className="flex items-center justify-between gap-2 rounded-lg border border-border-subtle px-3 py-2"
          >
            <span className="text-sm text-fg-default">{item.title}</span>
            <MoveToPositionMenu
              itemId={item.id}
              itemTitle={item.title}
              siblings={reorderSiblings}
              onMoveToIndex={(id, toIndex) => {
                setReorderIds((prev) => {
                  const next = prev.filter((x) => x !== id)
                  next.splice(toIndex, 0, id)
                  return next
                })
              }}
            />
          </ClickToMoveDropZone>
        ))}
      </Stack>
    </GalleryBlock>
  )
}
