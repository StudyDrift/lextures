import { useId, useRef, useState, type ReactNode } from 'react'
import { Button, type ButtonProps, type ButtonVariant } from './button'
import { Menu, type MenuItem } from './menu'
import { cx } from './utils'

export type SplitButtonProps = {
  label: ReactNode
  items: MenuItem[]
  onPrimaryClick?: () => void
  variant?: ButtonVariant
  disabled?: boolean
  loading?: boolean
  className?: string
  /** Accessible name for the menu trigger. */
  menuLabel: string
  primaryProps?: Omit<ButtonProps, 'children' | 'variant' | 'disabled' | 'loading'>
}

export function SplitButton({
  label,
  items,
  onPrimaryClick,
  variant = 'primary',
  disabled,
  loading,
  className,
  menuLabel,
  primaryProps,
}: SplitButtonProps) {
  const [open, setOpen] = useState(false)
  const triggerRef = useRef<HTMLButtonElement>(null)
  const menuId = useId()

  return (
    <div className={cx('inline-flex', className)}>
      <Button
        variant={variant}
        disabled={disabled}
        loading={loading}
        onClick={onPrimaryClick}
        className="rounded-e-none"
        {...primaryProps}
      >
        {label}
      </Button>
      <Button
        ref={triggerRef}
        variant={variant}
        disabled={disabled || loading}
        aria-label={menuLabel}
        aria-haspopup="menu"
        aria-expanded={open}
        aria-controls={open ? menuId : undefined}
        className="rounded-s-none border-s border-s-white/20 px-2"
        onClick={() => setOpen((v) => !v)}
      >
        <span aria-hidden className="text-xs">
          ▾
        </span>
      </Button>
      <Menu
        id={menuId}
        open={open}
        onOpenChange={setOpen}
        items={items}
        anchorRef={triggerRef}
      />
    </div>
  )
}
