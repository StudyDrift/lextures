/**
 * AN.5 — Shared overlay presentation: portal, scrim fade, panel enter/exit.
 *
 * Focus trap/return remain the caller's responsibility; this component only
 * animates presence and keeps the layer mounted through exit (FR-6, FR-9).
 */

import { useEffect, type CSSProperties, type ReactNode } from 'react'
import { createPortal } from 'react-dom'
import {
  overlayClassNames,
  type OverlayEdge,
  type OverlayKind,
} from '../../lib/overlay-motion'
import { useOverlayPresence } from '../../lib/use-overlay-presence'
import { usePlatformFeatures } from '../../context/platform-features-context'

export type OverlaySurfaceProps = {
  open: boolean
  onClose?: () => void
  /** Overlay motion kind (dialog scale, sheet slide, menu grow, …). */
  kind?: OverlayKind
  edge?: OverlayEdge
  /** Override kill-switch; defaults to `ffMotionOverlays`. */
  enabled?: boolean
  backdropLabel?: string
  /** Extra classes on the fixed root. */
  className?: string
  /** Extra classes on the panel wrapper around children. */
  panelClassName?: string
  /** Extra classes on the scrim. */
  scrimClassName?: string
  /** z-index utility classes; default matches confirm dialogs. */
  zClassName?: string
  /** Lock body scroll while mounted. Default true. */
  lockScroll?: boolean
  /** Called when exit animation begins (focus return). */
  onExitStart?: () => void
  children: ReactNode
  /**
   * When false, children are not wrapped in an animated panel — caller applies
   * `panelClassName` / motion classes themselves (e.g. custom drawer markup).
   */
  wrapPanel?: boolean
}

export function OverlaySurface({
  open,
  onClose,
  kind = 'dialog',
  edge,
  enabled: enabledProp,
  backdropLabel = 'Close',
  className = '',
  panelClassName = '',
  scrimClassName = '',
  zClassName = 'z-[400]',
  lockScroll = true,
  onExitStart,
  children,
  wrapPanel = true,
}: OverlaySurfaceProps) {
  const { ffMotionOverlays } = usePlatformFeatures()
  const enabled = enabledProp ?? ffMotionOverlays !== false

  const presence = useOverlayPresence({ open, kind, enabled, onExitStart })
  const { panel, scrim, durationMs } = overlayClassNames({
    kind,
    phase: presence.phase,
    enabled: presence.enabled,
    reduceMotion: presence.reducedMotion,
    edge,
  })

  useEffect(() => {
    if (!lockScroll || !presence.mounted) return
    const prev = document.body.style.overflow
    document.body.style.overflow = 'hidden'
    return () => {
      document.body.style.overflow = prev
    }
  }, [lockScroll, presence.mounted])

  if (!presence.mounted || typeof document === 'undefined') return null

  const durationStyle = {
    '--lx-overlay-duration': `${durationMs}ms`,
  } as CSSProperties

  const shell = (
    <div
      className={`fixed inset-0 ${zClassName} flex items-center justify-center p-4 ${className}`.trim()}
      role="presentation"
      style={durationStyle}
      data-overlay-phase={presence.phase}
      data-overlay-kind={kind}
    >
      {onClose ? (
        <button
          type="button"
          aria-label={backdropLabel}
          className={`lex-btn-static absolute inset-0 cursor-default border-0 bg-slate-950/55 p-0 backdrop-blur-[2px] dark:bg-black/80 ${scrim} ${scrimClassName}`.trim()}
          onClick={onClose}
          tabIndex={-1}
        />
      ) : (
        <div
          aria-hidden
          className={`absolute inset-0 bg-slate-950/55 backdrop-blur-[2px] dark:bg-black/80 ${scrim} ${scrimClassName}`.trim()}
        />
      )}
      {wrapPanel ? (
        <div className={`relative z-10 ${panel} ${panelClassName}`.trim()}>{children}</div>
      ) : (
        children
      )}
    </div>
  )

  return createPortal(shell, document.body)
}
