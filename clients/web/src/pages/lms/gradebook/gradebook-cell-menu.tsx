import {
  type ReactNode,
  useEffect,
  useRef,
} from 'react'
import { createPortal } from 'react-dom'
import { MoreVertical } from 'lucide-react'
import type { GradebookCellMenuItem, GradebookCellMenuState } from './gradebook-cell-menu-utils'

function menuItemClass() {
  return 'block w-full px-2.5 py-1.5 text-start text-sm text-fg-default hover:bg-surface-sunken dark:text-fg-default dark:hover:bg-surface-overlay'
}

export function GradebookCellMenuPortal({
  menu,
  onClose,
  children,
}: {
  menu: Exclude<GradebookCellMenuState, null>
  onClose: () => void
  children: ReactNode
}) {
  const panelRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    function onDocMouseDown(e: MouseEvent) {
      if (!(e.target instanceof Node)) return
      if (panelRef.current?.contains(e.target)) return
      onClose()
    }
    function onKey(e: globalThis.KeyboardEvent) {
      if (e.key === 'Escape') onClose()
    }
    document.addEventListener('mousedown', onDocMouseDown)
    document.addEventListener('keydown', onKey)
    return () => {
      document.removeEventListener('mousedown', onDocMouseDown)
      document.removeEventListener('keydown', onKey)
    }
  }, [onClose])

  useEffect(() => {
    function onScroll() {
      onClose()
    }
    window.addEventListener('scroll', onScroll, true)
    window.addEventListener('resize', onScroll)
    return () => {
      window.removeEventListener('scroll', onScroll, true)
      window.removeEventListener('resize', onScroll)
    }
  }, [onClose])

  return createPortal(
    <div
      ref={panelRef}
      role="menu"
      className="fixed z-[200] min-w-[11rem] rounded-lg border border-border-default bg-surface-raised py-1 shadow-lg dark:border-border-default dark:bg-surface-raised"
      style={{ top: menu.top, left: menu.left }}
    >
      {children}
    </div>,
    document.body,
  )
}

export function GradebookCellMenuItems({
  items,
  onSelect,
}: {
  items: GradebookCellMenuItem[]
  onSelect: (item: GradebookCellMenuItem) => void
}) {
  return items.map((item) => (
    <button
      key={item.kind}
      type="button"
      role="menuitem"
      className={menuItemClass()}
      onClick={() => onSelect(item)}
    >
      {item.label}
    </button>
  ))
}

export function GradebookCellMenuTrigger({
  studentName,
  columnTitle,
  onOpen,
}: {
  studentName: string
  columnTitle: string
  onOpen: (e: React.MouseEvent<HTMLButtonElement>) => void
}) {
  return (
    <button
      type="button"
      className="absolute start-0 top-0 z-[2] inline-flex size-5 items-center justify-center rounded text-fg-subtle hover:bg-surface-sunken hover:text-fg-muted dark:hover:bg-surface-overlay dark:hover:text-fg-default"
      aria-label={`Actions for ${studentName}, ${columnTitle}`}
      aria-haspopup="menu"
      onClick={onOpen}
    >
      <MoreVertical className="size-3.5" aria-hidden />
    </button>
  )
}
