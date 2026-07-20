import { useEffect, type CSSProperties, type ReactNode } from 'react'
import { createPortal } from 'react-dom'
import { usePlatformFeatures } from '../../context/platform-features-context'
import { overlayClassNames } from '../../lib/overlay-motion'
import { useOverlayPresence } from '../../lib/use-overlay-presence'

export type FullScreenModalShellProps = {
  open: boolean
  onClose?: () => void
  backdropLabel: string
  children: ReactNode
}

/**
 * Full-viewport modal shell portaled to document.body so the backdrop is not clipped
 * by app-shell overflow (e.g. SpeedGrader over the course sidebar).
 * AN.5: scale+fade dialog enter/exit with synced scrim.
 */
export function FullScreenModalShell({
  open,
  onClose,
  backdropLabel,
  children,
}: FullScreenModalShellProps) {
  const { ffMotionOverlays } = usePlatformFeatures()
  const presence = useOverlayPresence({
    open,
    kind: 'dialog',
    enabled: ffMotionOverlays !== false,
  })
  const classes = overlayClassNames({
    kind: 'dialog',
    phase: presence.phase,
    enabled: presence.enabled,
    reduceMotion: presence.reducedMotion,
  })

  useEffect(() => {
    if (!presence.mounted) return
    const prevOverflow = document.body.style.overflow
    document.body.style.overflow = 'hidden'
    return () => {
      document.body.style.overflow = prevOverflow
    }
  }, [presence.mounted])

  if (!presence.mounted) return null

  const durationStyle = {
    '--lx-overlay-duration': `${classes.durationMs}ms`,
  } as CSSProperties

  const shell = (
    <div
      className="fixed inset-0 z-[500] flex items-center justify-center p-3 sm:p-6"
      role="presentation"
      style={durationStyle}
      data-overlay-phase={presence.phase}
    >
      {onClose ? (
        <button
          type="button"
          aria-label={backdropLabel}
          className={`absolute inset-0 cursor-default border-0 bg-slate-950/55 p-0 backdrop-blur-[2px] dark:bg-black/80 ${classes.scrim}`}
          onClick={onClose}
          tabIndex={-1}
        />
      ) : (
        <div
          aria-hidden
          className={`absolute inset-0 bg-slate-950/55 backdrop-blur-[2px] dark:bg-black/80 ${classes.scrim}`}
        />
      )}
      <div className={`relative z-10 ${classes.panel}`}>{children}</div>
    </div>
  )

  return createPortal(shell, document.body)
}
