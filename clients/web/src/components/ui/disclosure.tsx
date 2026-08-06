import { useId, useState, type ReactNode } from 'react'
import { cx, focusRingClass } from './utils'

export type DisclosureProps = {
  title: ReactNode
  children: ReactNode
  defaultOpen?: boolean
  open?: boolean
  onOpenChange?: (open: boolean) => void
  className?: string
}

export function Disclosure({
  title,
  children,
  defaultOpen = false,
  open: openProp,
  onOpenChange,
  className = '',
}: DisclosureProps) {
  const panelId = useId()
  const [uncontrolled, setUncontrolled] = useState(defaultOpen)
  const open = openProp ?? uncontrolled
  const setOpen = (v: boolean) => {
    if (openProp === undefined) setUncontrolled(v)
    onOpenChange?.(v)
  }

  return (
    <div className={cx('rounded-xl border border-border-default bg-surface-raised', className)}>
      <h3 className="m-0">
        <button
          type="button"
          aria-expanded={open}
          aria-controls={panelId}
          className={cx(
            'flex w-full min-h-11 items-center justify-between gap-2 px-4 py-3 text-start text-sm font-semibold text-fg-default',
            focusRingClass,
          )}
          onClick={() => setOpen(!open)}
        >
          <span>{title}</span>
          <span aria-hidden className={cx('text-fg-muted transition-transform', open && 'rotate-180')}>
            ▾
          </span>
        </button>
      </h3>
      {open ? (
        <div id={panelId} className="border-t border-border-default px-4 py-3 text-sm text-fg-muted">
          {children}
        </div>
      ) : (
        <div id={panelId} hidden />
      )}
    </div>
  )
}
