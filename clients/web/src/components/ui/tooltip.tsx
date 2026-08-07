import {
  cloneElement,
  isValidElement,
  useEffect,
  useId,
  useLayoutEffect,
  useRef,
  useState,
  type CSSProperties,
  type FocusEvent,
  type MouseEvent,
  type ReactElement,
  type ReactNode,
  type Ref,
} from 'react'
import { createPortal } from 'react-dom'
import { cx } from './utils'

export type TooltipProps = {
  content: ReactNode
  children: ReactElement
  /** Delay before show (ms). */
  delayMs?: number
  placement?: 'top' | 'bottom'
}

type TriggerProps = {
  'aria-describedby'?: string
  onMouseEnter?: (e: MouseEvent) => void
  onMouseLeave?: (e: MouseEvent) => void
  onFocus?: (e: FocusEvent) => void
  onBlur?: (e: FocusEvent) => void
  ref?: Ref<HTMLElement>
}

/**
 * Accessible tooltip using aria-describedby (not the native `title` attribute).
 * Keyboard-reachable via focus; dismissible with Escape; hoverable for SC 1.4.13
 * (UX.4 FR-7).
 */
export function Tooltip({ content, children, delayMs = 400, placement = 'top' }: TooltipProps) {
  const id = useId()
  const [open, setOpen] = useState(false)
  const [pos, setPos] = useState<CSSProperties>({})
  const triggerRef = useRef<HTMLElement | null>(null)
  const tipRef = useRef<HTMLDivElement | null>(null)
  const timer = useRef<number | null>(null)
  const hideTimer = useRef<number | null>(null)

  function clearShowTimer() {
    if (timer.current != null) {
      window.clearTimeout(timer.current)
      timer.current = null
    }
  }

  function clearHideTimer() {
    if (hideTimer.current != null) {
      window.clearTimeout(hideTimer.current)
      hideTimer.current = null
    }
  }

  function show() {
    clearHideTimer()
    clearShowTimer()
    timer.current = window.setTimeout(() => setOpen(true), delayMs)
  }

  function hideSoon() {
    clearShowTimer()
    clearHideTimer()
    // Brief grace so pointer can move onto the tooltip (SC 1.4.13).
    hideTimer.current = window.setTimeout(() => setOpen(false), 100)
  }

  function hideNow() {
    clearShowTimer()
    clearHideTimer()
    setOpen(false)
  }

  useEffect(() => {
    if (!open) return
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape') {
        e.preventDefault()
        e.stopPropagation()
        hideNow()
      }
    }
    window.addEventListener('keydown', onKey, true)
    return () => window.removeEventListener('keydown', onKey, true)
  }, [open])

  useLayoutEffect(() => {
    if (!open || !triggerRef.current) return
    const r = triggerRef.current.getBoundingClientRect()
    setPos({
      position: 'fixed',
      zIndex: 500,
      left: r.left + r.width / 2,
      top: placement === 'top' ? r.top - 8 : r.bottom + 8,
      transform: placement === 'top' ? 'translate(-50%, -100%)' : 'translate(-50%, 0)',
    })
  }, [open, placement, content])

  if (!isValidElement(children)) return children

  const child = children as ReactElement<TriggerProps>
  const prevRef = (child as ReactElement & { ref?: Ref<HTMLElement> }).ref

  const trigger = cloneElement(child, {
    'aria-describedby': open ? id : child.props['aria-describedby'],
    ref: (node: HTMLElement | null) => {
      triggerRef.current = node
      if (typeof prevRef === 'function') prevRef(node)
      else if (prevRef && typeof prevRef === 'object') {
        ;(prevRef as { current: HTMLElement | null }).current = node
      }
    },
    onMouseEnter: (e: MouseEvent) => {
      child.props.onMouseEnter?.(e)
      show()
    },
    onMouseLeave: (e: MouseEvent) => {
      child.props.onMouseLeave?.(e)
      hideSoon()
    },
    onFocus: (e: FocusEvent) => {
      child.props.onFocus?.(e)
      show()
    },
    onBlur: (e: FocusEvent) => {
      child.props.onBlur?.(e)
      // Don't hide if focus moved into the tooltip (rare; tip is not focusable by default).
      const next = e.relatedTarget as Node | null
      if (tipRef.current?.contains(next)) return
      hideNow()
    },
  } as Partial<TriggerProps>)

  return (
    <>
      {trigger}
      {open && typeof document !== 'undefined'
        ? createPortal(
            <div
              ref={tipRef}
              id={id}
              role="tooltip"
              style={pos}
              className={cx(
                // pointer-events-auto so users can hover the tip (SC 1.4.13).
                'pointer-events-auto max-w-xs rounded-md bg-surface-inverse px-2 py-1 text-xs font-medium text-fg-inverse shadow-md',
              )}
              onMouseEnter={() => {
                clearHideTimer()
              }}
              onMouseLeave={() => hideSoon()}
            >
              {content}
            </div>,
            document.body,
          )
        : null}
    </>
  )
}
