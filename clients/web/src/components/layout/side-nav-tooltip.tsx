import { type ReactNode, useState, useRef, useLayoutEffect } from 'react'
import { createPortal } from 'react-dom'
import { useShellNav } from './use-shell-nav'

interface SideNavTooltipProps {
  children: ReactNode
  content: string
  /** When true, shows the tooltip immediately (e.g. after pinning a course). */
  instant?: boolean
  /** When true, hover shows the tooltip even when the sidebar is expanded. */
  hoverWhenExpanded?: boolean
}

export function SideNavTooltip({
  children,
  content,
  instant = false,
  hoverWhenExpanded = false,
}: SideNavTooltipProps) {
  const { sideNavCollapsed } = useShellNav()
  const [hoverOpen, setHoverOpen] = useState(false)
  const [pos, setPos] = useState<{ top: number; left: number } | null>(null)
  const ref = useRef<HTMLDivElement>(null)
  const hoverEnabled = sideNavCollapsed || hoverWhenExpanded
  const showTooltip = instant || (hoverEnabled && hoverOpen)
  const needsWrapper = instant || hoverEnabled

  useLayoutEffect(() => {
    if (!showTooltip) {
      setPos(null)
      return
    }
    const measure = () => {
      if (ref.current) {
        const r = ref.current.getBoundingClientRect()
        if (r.width > 0 || r.height > 0) {
          setPos({
            top: r.top + r.height / 2,
            left: r.right + 10,
          })
        }
      }
    }
    measure()
    // Extra frames after an instant flash so the newly pinned tile has laid out.
    let raf2 = 0
    const raf1 = requestAnimationFrame(() => {
      measure()
      if (instant) {
        raf2 = requestAnimationFrame(measure)
      }
    })
    window.addEventListener('scroll', measure, true)
    window.addEventListener('resize', measure)
    return () => {
      cancelAnimationFrame(raf1)
      cancelAnimationFrame(raf2)
      window.removeEventListener('scroll', measure, true)
      window.removeEventListener('resize', measure)
    }
  }, [showTooltip, sideNavCollapsed, hoverWhenExpanded, instant, content])

  if (!needsWrapper) return <>{children}</>

  return (
    <div
      ref={ref}
      onMouseEnter={() => setHoverOpen(true)}
      onMouseLeave={() => {
        setHoverOpen(false)
        if (!instant) setPos(null)
      }}
      className={
        sideNavCollapsed ? 'flex w-full min-w-0 justify-center' : 'inline-flex shrink-0'
      }
    >
      {children}
      {showTooltip && pos &&
        createPortal(
          <div
            style={{ top: pos.top, left: pos.left }}
            className="tooltip-in lx-overlay-tooltip-in fixed z-[100] -translate-y-1/2 whitespace-nowrap rounded-lg bg-slate-950 px-2.5 py-1.5 text-xs font-semibold text-white shadow-xl ring-1 ring-white/10 dark:bg-neutral-800"
          >
            {content}
            <div className="absolute -start-1 top-1/2 h-2 w-2 -translate-y-1/2 rotate-45 bg-slate-950 dark:bg-neutral-800" />
          </div>,
          document.body,
        )}
    </div>
  )
}
