import { useLayoutEffect, useRef, useState, type ReactNode } from 'react'
import { createPortal } from 'react-dom'

type Placement = 'top' | 'bottom'

type ActionErrorTooltipProps = {
  message: string | null | undefined
  children: ReactNode
  placement?: Placement
}

/** Hover/focus tooltip for disabled action buttons (wraps the control so disabled buttons still receive pointer events). */
export function ActionErrorTooltip({
  message,
  children,
  placement = 'bottom',
}: ActionErrorTooltipProps) {
  const [open, setOpen] = useState(false)
  const [pos, setPos] = useState<{ top: number; left: number } | null>(null)
  const ref = useRef<HTMLSpanElement>(null)

  useLayoutEffect(() => {
    if (!open || !message || !ref.current) return
    const measure = () => {
      if (!ref.current) return
      const rect = ref.current.getBoundingClientRect()
      setPos({
        top: placement === 'top' ? rect.top - 8 : rect.bottom + 8,
        left: rect.left + rect.width / 2,
      })
    }
    measure()
    window.addEventListener('scroll', measure, true)
    window.addEventListener('resize', measure)
    return () => {
      window.removeEventListener('scroll', measure, true)
      window.removeEventListener('resize', measure)
    }
  }, [message, open, placement])

  if (!message) return <>{children}</>

  const show = () => setOpen(true)
  const hide = () => {
    setOpen(false)
    setPos(null)
  }

  return (
    <span
      ref={ref}
      className="inline-flex"
      onMouseEnter={show}
      onMouseLeave={hide}
      onFocusCapture={show}
      onBlurCapture={hide}
    >
      {children}
      {open && pos
        ? createPortal(
            <div
              role="tooltip"
              style={{
                top: pos.top,
                left: pos.left,
                transform:
                  placement === 'top' ? 'translate(-50%, -100%)' : 'translate(-50%, 0)',
              }}
              className="pointer-events-none fixed z-[560] max-w-xs rounded-lg border border-rose-200 bg-rose-50 px-3 py-2 text-sm text-rose-800 shadow-lg dark:border-rose-900/60 dark:bg-rose-950/90 dark:text-rose-100"
            >
              {message}
            </div>,
            document.body,
          )
        : null}
    </span>
  )
}