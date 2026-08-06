import {
  useEffect,
  useId,
  useLayoutEffect,
  useRef,
  useState,
  type CSSProperties,
  type ReactNode,
  type RefObject,
} from 'react'
import { createPortal } from 'react-dom'
import { cx } from './utils'

export type PopoverProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  anchorRef: RefObject<HTMLElement | null>
  children: ReactNode
  /** Accessible label when no visible title. */
  'aria-label'?: string
  className?: string
  placement?: 'bottom-start' | 'bottom-end' | 'top-start' | 'top-end'
}

export function Popover({
  open,
  onOpenChange,
  anchorRef,
  children,
  'aria-label': ariaLabel,
  className = '',
  placement = 'bottom-start',
}: PopoverProps) {
  const panelRef = useRef<HTMLDivElement>(null)
  const id = useId()
  const [pos, setPos] = useState<CSSProperties>({})

  useLayoutEffect(() => {
    if (!open || !anchorRef.current) return
    const r = anchorRef.current.getBoundingClientRect()
    const style: CSSProperties = { position: 'fixed', zIndex: 450 }
    if (placement.startsWith('bottom')) style.top = r.bottom + 6
    else style.bottom = window.innerHeight - r.top + 6
    if (placement.endsWith('start')) style.left = r.left
    else style.right = window.innerWidth - r.right
    setPos(style)
  }, [open, anchorRef, placement])

  useEffect(() => {
    if (!open) return
    function onDoc(e: MouseEvent) {
      const t = e.target as Node
      if (panelRef.current?.contains(t) || anchorRef.current?.contains(t)) return
      onOpenChange(false)
    }
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape') {
        e.preventDefault()
        onOpenChange(false)
        anchorRef.current?.focus()
      }
    }
    document.addEventListener('mousedown', onDoc)
    window.addEventListener('keydown', onKey)
    return () => {
      document.removeEventListener('mousedown', onDoc)
      window.removeEventListener('keydown', onKey)
    }
  }, [open, onOpenChange, anchorRef])

  if (!open || typeof document === 'undefined') return null

  return createPortal(
    <div
      ref={panelRef}
      id={id}
      role="dialog"
      aria-label={ariaLabel}
      style={pos}
      className={cx(
        'min-w-[12rem] rounded-xl border border-border-default bg-surface-raised p-3 shadow-lg',
        className,
      )}
    >
      {children}
    </div>,
    document.body,
  )
}
