import { useLayoutEffect, useRef, useState, type ReactNode } from 'react'
import { createPortal } from 'react-dom'

type Placement = 'top' | 'bottom'

type IconActionTooltipProps = {
  label: string
  children: ReactNode
  placement?: Placement
}

export function IconActionTooltip({
  label,
  children,
  placement = 'top',
}: IconActionTooltipProps) {
  const [open, setOpen] = useState(false)
  const [pos, setPos] = useState<{ top: number; left: number } | null>(null)
  const ref = useRef<HTMLSpanElement>(null)

  useLayoutEffect(() => {
    if (!open || !ref.current) {
      return
    }
    const measure = () => {
      if (!ref.current) return
      const rect = ref.current.getBoundingClientRect()
      setPos({
        top: placement === 'top' ? rect.top - 6 : rect.bottom + 6,
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
  }, [open, placement])

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
              className="lx-overlay-tooltip-in pointer-events-none fixed z-[200] whitespace-nowrap rounded-md bg-slate-950 px-2 py-1 text-xs font-medium text-white shadow-lg ring-1 ring-white/10 dark:bg-neutral-800"
            >
              {label}
            </div>,
            document.body,
          )
        : null}
    </span>
  )
}
