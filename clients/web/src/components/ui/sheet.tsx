import { useEffect, useId, useRef, type CSSProperties, type ReactNode } from 'react'
import { createPortal } from 'react-dom'
import { createFocusTrap } from '../../lib/a11y/focus-trap'
import { overlayClassNames, type OverlayEdge } from '../../lib/overlay-motion'
import { useOverlayPresence } from '../../lib/use-overlay-presence'
import { usePlatformFeatures } from '../../context/platform-features-context'
import { useInertBackground } from './use-inert-background'
import { cx } from './utils'
import { IconButton } from './icon-button'

export type SheetProps = {
  open: boolean
  onClose: () => void
  title: ReactNode
  children: ReactNode
  edge?: OverlayEdge
  closeLabel?: string
  className?: string
  panelClassName?: string
}

/**
 * Edge-anchored modal sheet (mobile-friendly dialog). Focus trap + inert + Escape.
 */
export function Sheet({
  open,
  onClose,
  title,
  children,
  edge = 'end',
  closeLabel = 'Close',
  className = '',
  panelClassName = '',
}: SheetProps) {
  const titleId = useId()
  const panelRef = useRef<HTMLDivElement>(null)
  const { ffMotionOverlays } = usePlatformFeatures()
  const presence = useOverlayPresence({
    open,
    kind: 'sheet',
    enabled: ffMotionOverlays !== false,
  })

  useInertBackground(presence.mounted && presence.phase !== 'closing')

  useEffect(() => {
    if (!presence.entered || !panelRef.current) return
    const trap = createFocusTrap(panelRef.current)
    trap.activate()
    return () => trap.deactivate()
  }, [presence.entered])

  useEffect(() => {
    if (!presence.mounted) return
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape') {
        e.preventDefault()
        onClose()
      }
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [presence.mounted, onClose])

  if (!presence.mounted || typeof document === 'undefined') return null

  const classes = overlayClassNames({
    kind: 'sheet',
    phase: presence.phase,
    enabled: presence.enabled,
    reduceMotion: presence.reducedMotion,
    edge,
  })
  const durationStyle = {
    '--lx-overlay-duration': `${classes.durationMs}ms`,
  } as CSSProperties
  const exiting = presence.phase === 'closing'

  const edgePos =
    edge === 'start'
      ? 'inset-y-0 start-0'
      : edge === 'end'
        ? 'inset-y-0 end-0'
        : edge === 'top'
          ? 'inset-x-0 top-0'
          : 'inset-x-0 bottom-0'

  const edgeSize =
    edge === 'start' || edge === 'end' ? 'h-full w-full max-w-md' : 'w-full max-h-[85vh]'

  return createPortal(
    <div
      className={cx('fixed inset-0 z-[400]', exiting && 'pointer-events-none', className)}
      role="presentation"
      style={durationStyle}
    >
      <button
        type="button"
        aria-label={closeLabel}
        tabIndex={-1}
        className={cx('lex-btn-static absolute inset-0 border-0 bg-black/45 p-0', classes.scrim)}
        onClick={onClose}
      />
      <div
        ref={panelRef}
        role="dialog"
        aria-modal
        aria-labelledby={titleId}
        className={cx(
          'absolute flex flex-col border border-border-default bg-surface-raised shadow-xl',
          edgePos,
          edgeSize,
          classes.panel,
          panelClassName,
        )}
      >
        <div className="flex items-center justify-between gap-3 border-b border-border-default px-4 py-3">
          <h2 id={titleId} className="text-base font-semibold text-fg-default">
            {title}
          </h2>
          <IconButton variant="ghost" size="sm" aria-label={closeLabel} onClick={onClose}>
            <span aria-hidden>×</span>
          </IconButton>
        </div>
        <div className="min-h-0 flex-1 overflow-auto p-4">{children}</div>
      </div>
    </div>,
    document.body,
  )
}

/** Alias used in some product copy — same component. */
export const Drawer = Sheet
