import { forwardRef, type ButtonHTMLAttributes, type ReactNode } from 'react'
import { Button, type ButtonVariant } from './button'
import { cx, iconSizeClasses, type ControlSize } from './utils'

export type IconButtonProps = Omit<ButtonHTMLAttributes<HTMLButtonElement>, 'children'> & {
  /** Required accessible name — never rely on the icon alone. */
  'aria-label': string
  variant?: ButtonVariant
  size?: ControlSize
  loading?: boolean
  static?: boolean
  children: ReactNode
}

export const IconButton = forwardRef<HTMLButtonElement, IconButtonProps>(function IconButton(
  { className = '', size = 'md', children, ...props },
  ref,
) {
  return (
    <Button
      ref={ref}
      {...props}
      className={cx(iconSizeClasses[size], 'rounded-xl p-0', className)}
    >
      {children}
    </Button>
  )
})
