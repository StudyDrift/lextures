import { useEffect, type ReactNode } from 'react'
import { createPortal } from 'react-dom'

export type FullScreenModalShellProps = {
  open: boolean
  onClose?: () => void
  backdropLabel: string
  children: ReactNode
}

/**
 * Full-viewport modal shell portaled to document.body so the backdrop is not clipped
 * by app-shell overflow (e.g. SpeedGrader over the course sidebar).
 */
export function FullScreenModalShell({
  open,
  onClose,
  backdropLabel,
  children,
}: FullScreenModalShellProps) {
  useEffect(() => {
    if (!open) return
    const prevOverflow = document.body.style.overflow
    document.body.style.overflow = 'hidden'
    return () => {
      document.body.style.overflow = prevOverflow
    }
  }, [open])

  if (!open) return null

  const shell = (
    <div
      className="fixed inset-0 z-[500] flex items-center justify-center p-3 sm:p-6"
      role="presentation"
    >
      {onClose ? (
        <button
          type="button"
          aria-label={backdropLabel}
          className="absolute inset-0 cursor-default border-0 bg-slate-950/55 p-0 backdrop-blur-[2px] dark:bg-black/80"
          onClick={onClose}
          tabIndex={-1}
        />
      ) : (
        <div
          aria-hidden
          className="absolute inset-0 bg-slate-950/55 backdrop-blur-[2px] dark:bg-black/80"
        />
      )}
      {children}
    </div>
  )

  return createPortal(shell, document.body)
}