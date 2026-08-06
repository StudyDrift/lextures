import { useEffect, useId, useRef, type ReactNode } from 'react'
import { createFocusTrap } from '../../lib/a11y/focus-trap'
import { useInertBackground } from './use-inert-background'
import { OverlaySurface } from './overlay-surface'
import { cx } from './utils'
import { IconButton } from './icon-button'

export type DialogProps = {
  open: boolean
  onClose: () => void
  title: ReactNode
  description?: ReactNode
  children?: ReactNode
  footer?: ReactNode
  /** Hide the header close control. */
  hideClose?: boolean
  /** Label for the close button (required when visible — pass i18n string). */
  closeLabel?: string
  /**
   * Accessible name for the scrim control. Defaults to `closeLabel`.
   * Pass a distinct string when the footer already has a control with the same name
   * (e.g. AlertDialog cancel).
   */
  backdropLabel?: string
  /** Backdrop click dismiss. Default true. */
  closeOnBackdrop?: boolean
  /** Escape dismiss. Default true. */
  closeOnEscape?: boolean
  className?: string
  panelClassName?: string
  size?: 'sm' | 'md' | 'lg' | 'xl'
  /** Called when exit animation finishes. */
  onExited?: () => void
  /** Initial focus selector within the panel; default first focusable. */
  initialFocusRef?: React.RefObject<HTMLElement | null>
  /** Optional test id for the overlay root (e.g. confirm-dialog-root). */
  rootTestId?: string
}

const sizeClass: Record<NonNullable<DialogProps['size']>, string> = {
  sm: 'max-w-sm',
  md: 'max-w-md',
  lg: 'max-w-lg',
  xl: 'max-w-2xl',
}

export function Dialog({
  open,
  onClose,
  title,
  description,
  children,
  footer,
  hideClose = false,
  closeLabel = 'Close',
  backdropLabel,
  closeOnBackdrop = true,
  closeOnEscape = true,
  className = '',
  panelClassName = '',
  size = 'md',
  onExited,
  initialFocusRef,
  rootTestId,
}: DialogProps) {
  const titleId = useId()
  const descId = useId()
  const panelRef = useRef<HTMLDivElement>(null)

  useInertBackground(open)

  useEffect(() => {
    if (!open || !panelRef.current) return
    const trap = createFocusTrap(panelRef.current)
    trap.activate()
    if (initialFocusRef?.current) {
      initialFocusRef.current.focus()
    }
    return () => trap.deactivate()
  }, [open, initialFocusRef])

  useEffect(() => {
    if (!open || !closeOnEscape) return
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape') {
        e.preventDefault()
        e.stopPropagation()
        onClose()
      }
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [open, closeOnEscape, onClose])

  return (
    <OverlaySurface
      open={open}
      onClose={closeOnBackdrop ? onClose : undefined}
      kind="dialog"
      backdropLabel={backdropLabel ?? closeLabel}
      className={className}
      wrapPanel={false}
      onExited={onExited}
      rootTestId={rootTestId}
    >
      <div
        ref={panelRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        aria-describedby={description ? descId : undefined}
        className={cx(
          'relative z-10 w-full rounded-2xl border border-border-default bg-surface-raised p-5 shadow-xl',
          sizeClass[size],
          panelClassName,
        )}
        data-testid="ui-dialog-panel"
      >
        <div className="flex items-start justify-between gap-3">
          <h2 id={titleId} className="text-lg font-semibold text-fg-default">
            {title}
          </h2>
          {!hideClose ? (
            <IconButton
              variant="ghost"
              size="sm"
              aria-label={closeLabel}
              onClick={onClose}
              className="shrink-0"
            >
              <span aria-hidden className="text-base leading-none">
                ×
              </span>
            </IconButton>
          ) : null}
        </div>
        {description ? (
          <div id={descId} className="mt-2 text-sm text-fg-muted">
            {description}
          </div>
        ) : null}
        {children ? <div className="mt-4">{children}</div> : null}
        {footer ? <div className="mt-6 flex flex-wrap justify-end gap-2">{footer}</div> : null}
      </div>
    </OverlaySurface>
  )
}
