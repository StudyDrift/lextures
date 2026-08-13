import {
  useEffect,
  useId,
  useLayoutEffect,
  useRef,
  useState,
  type CSSProperties,
  type KeyboardEvent,
  type ReactNode,
  type RefObject,
} from 'react'
import { createPortal } from 'react-dom'
import { menuPositionStyle, type MenuPlacement } from './menu-position'
import { cx, focusRingClass } from './utils'

export type { MenuPlacement }

export type MenuItem = {
  id: string
  label: ReactNode
  onSelect?: () => void
  disabled?: boolean
  danger?: boolean
  /** Typeahead text; defaults to string label. */
  textValue?: string
}

export type MenuProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  items: MenuItem[]
  anchorRef: RefObject<HTMLElement | null>
  id?: string
  className?: string
  /** Called after an item is chosen (menu already closing). */
  onAction?: (id: string) => void
  /** Placement relative to anchor. Default bottom-start. Flips/clamps to stay on screen. */
  placement?: MenuPlacement
  /** Accessible name when no visible label. */
  'aria-label'?: string
  'aria-labelledby'?: string
}

/**
 * Menu per WAI-ARIA APG: focus first item on open, arrows, Home/End, typeahead,
 * Escape closes + restores focus, Tab closes (FR-2).
 */
export function Menu({
  open,
  onOpenChange,
  items,
  anchorRef,
  id: idProp,
  className = '',
  onAction,
  placement = 'bottom-start',
  'aria-label': ariaLabel,
  'aria-labelledby': ariaLabelledby,
}: MenuProps) {
  const autoId = useId()
  const menuId = idProp ?? autoId
  const listRef = useRef<HTMLDivElement>(null)
  const [activeIndex, setActiveIndex] = useState(0)
  const [pos, setPos] = useState<CSSProperties>({})
  const typeaheadRef = useRef({ buffer: '', at: 0 })

  const enabledIndexes = items
    .map((item, i) => ({ item, i }))
    .filter(({ item }) => !item.disabled)
    .map(({ i }) => i)

  useLayoutEffect(() => {
    if (!open || !anchorRef.current) return
    const apply = () => {
      if (!anchorRef.current) return
      const r = anchorRef.current.getBoundingClientRect()
      const menuEl = listRef.current
      setPos(
        menuPositionStyle(
          r,
          { width: menuEl?.offsetWidth ?? 0, height: menuEl?.offsetHeight ?? 0 },
          placement,
        ),
      )
    }
    apply()
    // First paint can report height 0; remasure so top placement sits above the trigger.
    const raf = requestAnimationFrame(apply)
    setActiveIndex(enabledIndexes[0] ?? 0)
    return () => cancelAnimationFrame(raf)
    // enabledIndexes identity changes each render; open + items drive re-position.
    // eslint-disable-next-line react-hooks/exhaustive-deps -- intentional
  }, [open, anchorRef, items, placement])

  useEffect(() => {
    if (!open) return
    function reposition() {
      if (!anchorRef.current) return
      const r = anchorRef.current.getBoundingClientRect()
      const menuEl = listRef.current
      setPos(
        menuPositionStyle(
          r,
          { width: menuEl?.offsetWidth ?? 0, height: menuEl?.offsetHeight ?? 0 },
          placement,
        ),
      )
    }
    window.addEventListener('resize', reposition)
    window.addEventListener('scroll', reposition, true)
    return () => {
      window.removeEventListener('resize', reposition)
      window.removeEventListener('scroll', reposition, true)
    }
  }, [open, anchorRef, placement])

  useEffect(() => {
    if (!open || !listRef.current) return
    const el = listRef.current.querySelector<HTMLElement>(`[data-menu-index="${activeIndex}"]`)
    el?.focus()
  }, [open, activeIndex])

  useEffect(() => {
    if (!open) return
    function onDoc(e: MouseEvent) {
      const t = e.target as Node
      if (listRef.current?.contains(t) || anchorRef.current?.contains(t)) return
      onOpenChange(false)
    }
    document.addEventListener('mousedown', onDoc)
    return () => document.removeEventListener('mousedown', onDoc)
  }, [open, onOpenChange, anchorRef])

  function close() {
    onOpenChange(false)
    anchorRef.current?.focus()
  }

  function activate(index: number) {
    const item = items[index]
    if (!item || item.disabled) return
    item.onSelect?.()
    onAction?.(item.id)
    close()
  }

  function move(delta: number) {
    if (enabledIndexes.length === 0) return
    const curPos = enabledIndexes.indexOf(activeIndex)
    const next =
      enabledIndexes[(curPos + delta + enabledIndexes.length) % enabledIndexes.length] ??
      enabledIndexes[0]!
    setActiveIndex(next)
  }

  function onKeyDown(e: KeyboardEvent) {
    if (e.key === 'ArrowDown') {
      e.preventDefault()
      move(1)
    } else if (e.key === 'ArrowUp') {
      e.preventDefault()
      move(-1)
    } else if (e.key === 'Home') {
      e.preventDefault()
      if (enabledIndexes[0] != null) setActiveIndex(enabledIndexes[0])
    } else if (e.key === 'End') {
      e.preventDefault()
      const last = enabledIndexes[enabledIndexes.length - 1]
      if (last != null) setActiveIndex(last)
    } else if (e.key === 'Enter' || e.key === ' ') {
      e.preventDefault()
      activate(activeIndex)
    } else if (e.key === 'Escape') {
      e.preventDefault()
      close()
    } else if (e.key === 'Tab') {
      // APG: Tab closes the menu and moves focus per normal tab order (after restore).
      close()
    } else if (e.key.length === 1 && !e.ctrlKey && !e.metaKey && !e.altKey) {
      const now = Date.now()
      const state = typeaheadRef.current
      if (now - state.at > 500) state.buffer = ''
      state.at = now
      state.buffer += e.key.toLowerCase()
      const match = items.findIndex((item) => {
        if (item.disabled) return false
        const text = item.textValue ?? (typeof item.label === 'string' ? item.label : '')
        return text.toLowerCase().startsWith(state.buffer)
      })
      if (match >= 0) setActiveIndex(match)
    }
  }

  if (!open || typeof document === 'undefined') return null

  return createPortal(
    <div
      ref={listRef}
      id={menuId}
      role="menu"
      tabIndex={-1}
      aria-label={ariaLabel}
      aria-labelledby={ariaLabelledby}
      style={pos}
      className={cx(
        'w-max rounded-xl border border-border-default bg-surface-raised py-1 shadow-lg outline-none',
        className,
      )}
      onKeyDown={onKeyDown}
    >
      {items.map((item, i) => (
        <button
          key={item.id}
          type="button"
          role="menuitem"
          data-menu-index={i}
          tabIndex={i === activeIndex ? 0 : -1}
          disabled={item.disabled}
          className={cx(
            'flex w-full min-h-9 items-center whitespace-nowrap px-3 py-2 text-start text-sm font-medium outline-none',
            focusRingClass,
            i === activeIndex && 'bg-accent-surface text-accent-fg',
            item.danger ? 'text-danger-fg' : 'text-fg-default',
            item.disabled && 'opacity-50',
          )}
          onMouseEnter={() => !item.disabled && setActiveIndex(i)}
          onClick={() => activate(i)}
        >
          {item.label}
        </button>
      ))}
    </div>,
    document.body,
  )
}
