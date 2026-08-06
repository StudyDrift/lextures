import { useCallback, useRef, useState, type ReactNode } from 'react'
import { Menu, type MenuItem } from './menu'

export type ContextMenuProps = {
  items: MenuItem[]
  children: ReactNode
  className?: string
}

/**
 * Right-click / context-menu surface. Wraps children and opens {@link Menu} at the pointer.
 */
export function ContextMenu({ items, children, className = '' }: ContextMenuProps) {
  const [open, setOpen] = useState(false)
  const anchorRef = useRef<HTMLDivElement>(null)
  const [anchorBox, setAnchorBox] = useState({ x: 0, y: 0 })

  const onContextMenu = useCallback((e: React.MouseEvent) => {
    e.preventDefault()
    setAnchorBox({ x: e.clientX, y: e.clientY })
    setOpen(true)
  }, [])

  // Position via a zero-size anchor at the pointer.
  return (
    <div className={className} onContextMenu={onContextMenu}>
      {children}
      <div
        ref={anchorRef}
        aria-hidden
        className="pointer-events-none fixed h-0 w-0"
        style={{ left: anchorBox.x, top: anchorBox.y }}
      />
      <Menu open={open} onOpenChange={setOpen} items={items} anchorRef={anchorRef} />
    </div>
  )
}
