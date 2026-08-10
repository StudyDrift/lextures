/**
 * UX.5 — Reorderable primitives: "Move to…" menu + click-to-move drop zone.
 *
 * Drag surfaces keep @dnd-kit for pointer/keyboard drag; this module adds the
 * single-pointer alternative required by WCAG 2.2 SC 2.5.7 without extra bundle
 * weight on dnd-kit itself. The click-to-move hook lives in `./use-click-to-move`.
 */
import { useId, useRef, useState, type ReactNode } from 'react'
import { announce } from '../../lib/a11y'
import { Menu, type MenuItem } from './menu'
import { Button } from './button'
import { cx } from './utils'

export type ReorderableItemMeta = {
  id: string
  title: string
}

export type MoveToPositionMenuProps = {
  /** Item being moved. */
  itemId: string
  itemTitle: string
  /** Sibling items in display order (including the current item). */
  siblings: ReorderableItemMeta[]
  /** Commit absolute 0-based index. */
  onMoveToIndex: (itemId: string, toIndex: number) => void
  disabled?: boolean
  /** Optional trigger label override. */
  label?: string
  className?: string
}

/**
 * Overflow "Move to…" control: pick an absolute position (1-based labels).
 * Discoverable single-pointer alternative (UX.5 FR-5).
 */
export function MoveToPositionMenu({
  itemId,
  itemTitle,
  siblings,
  onMoveToIndex,
  disabled = false,
  label = 'Move to…',
  className = '',
}: MoveToPositionMenuProps) {
  const [open, setOpen] = useState(false)
  const triggerRef = useRef<HTMLButtonElement>(null)
  const menuId = useId()
  const currentIndex = siblings.findIndex((s) => s.id === itemId)

  const items: MenuItem[] = siblings.map((s, index) => {
    const isSelf = s.id === itemId
    return {
      id: `pos-${index}`,
      textValue: `Position ${index + 1}${isSelf ? ' (current)' : ''}`,
      disabled: isSelf,
      label: (
        <span className="flex w-full items-center justify-between gap-3">
          <span className="truncate">
            {index + 1}. {s.title}
          </span>
          {isSelf ? (
            <span className="shrink-0 text-xs text-fg-muted">Current</span>
          ) : null}
        </span>
      ),
      onSelect: () => {
        if (isSelf) return
        onMoveToIndex(itemId, index)
        announce(
          `"${itemTitle}" moved to position ${index + 1} of ${siblings.length}.`,
        )
      },
    }
  })

  if (siblings.length < 2) return null

  return (
    <div className={cx('relative inline-flex', className)}>
      <Button
        ref={triggerRef}
        type="button"
        variant="ghost"
        size="sm"
        disabled={disabled || currentIndex < 0}
        aria-haspopup="menu"
        aria-expanded={open}
        aria-controls={open ? menuId : undefined}
        aria-label={`${label} ${itemTitle}`}
        data-testid="move-to-position-trigger"
        onClick={() => setOpen((o) => !o)}
      >
        {label}
      </Button>
      <Menu
        open={open}
        onOpenChange={setOpen}
        id={menuId}
        anchorRef={triggerRef}
        placement="bottom-end"
        aria-label={`${label} ${itemTitle}`}
        items={items}
      />
    </div>
  )
}

export type ClickToMoveDropZoneProps = {
  id: string
  active: boolean
  isSource: boolean
  isValidTarget: boolean
  onSelect: () => void
  children: ReactNode
  className?: string
}

/** Visual affordance for a row that can be a click-to-move source or target. */
export function ClickToMoveDropZone({
  id,
  active,
  isSource,
  isValidTarget,
  onSelect,
  children,
  className = '',
}: ClickToMoveDropZoneProps) {
  return (
    <div
      data-reorderable-id={id}
      data-click-to-move={active ? (isSource ? 'source' : isValidTarget ? 'target' : 'idle') : undefined}
      className={cx(
        className,
        active && isSource && 'ring-2 ring-accent-solid/50 rounded-xl',
        active && isValidTarget && 'ring-2 ring-dashed ring-border-focus rounded-xl cursor-pointer',
      )}
    >
      {active && isValidTarget ? (
        <button
          type="button"
          className="w-full text-start"
          onClick={onSelect}
          aria-label="Move here"
        >
          {children}
        </button>
      ) : (
        children
      )}
    </div>
  )
}
