import {
  cloneElement,
  isValidElement,
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
 */
export function Tooltip({ content, children, delayMs = 400, placement = 'top' }: TooltipProps) {
  const id = useId()
  const [open, setOpen] = useState(false)
  const [pos, setPos] = useState<CSSProperties>({})
  const triggerRef = useRef<HTMLElement | null>(null)
  const timer = useRef<number | null>(null)

  function clearTimer() {
    if (timer.current != null) {
      window.clearTimeout(timer.current)
      timer.current = null
    }
  }

  function show() {
    clearTimer()
    timer.current = window.setTimeout(() => setOpen(true), delayMs)
  }

  function hide() {
    clearTimer()
    setOpen(false)
  }

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
      hide()
    },
    onFocus: (e: FocusEvent) => {
      child.props.onFocus?.(e)
      show()
    },
    onBlur: (e: FocusEvent) => {
      child.props.onBlur?.(e)
      hide()
    },
  } as Partial<TriggerProps>)

  return (
    <>
      {trigger}
      {open && typeof document !== 'undefined'
        ? createPortal(
            <div
              id={id}
              role="tooltip"
              style={pos}
              className={cx(
                'pointer-events-none max-w-xs rounded-md bg-surface-inverse px-2 py-1 text-xs font-medium text-fg-inverse shadow-md',
              )}
            >
              {content}
            </div>,
            document.body,
          )
        : null}
    </>
  )
}
