import { forwardRef, type ComponentPropsWithoutRef, type ReactNode } from 'react'
import { Link } from 'react-router-dom'
import { type ButtonVariant } from './button'
import { controlBaseClass, cx, focusRingClass, sizeClasses, type ControlSize } from './utils'
import { pressClassName } from '../../lib/control-motion'
import { usePrefersReducedMotion } from '../../lib/motion'
import { usePlatformFeatures } from '../../context/platform-features-context'

export type LinkButtonProps = ComponentPropsWithoutRef<typeof Link> & {
  variant?: ButtonVariant
  size?: ControlSize
  children: ReactNode
}

const variantClasses: Record<ButtonVariant, string> = {
  primary:
    'bg-accent-solid text-fg-on-accent shadow-sm hover:opacity-90 focus-visible:ring-accent-solid/30',
  secondary:
    'border border-border-default bg-surface-raised text-fg-default shadow-sm hover:bg-surface-sunken',
  ghost: 'text-fg-muted hover:bg-surface-sunken hover:text-fg-default',
  danger: 'bg-danger-fg text-fg-on-accent shadow-sm hover:opacity-90 focus-visible:ring-danger-fg/30',
}

export const LinkButton = forwardRef<HTMLAnchorElement, LinkButtonProps>(function LinkButton(
  { variant = 'primary', size = 'md', className = '', children, ...props },
  ref,
) {
  const { ffMotionControls } = usePlatformFeatures()
  const reduceMotion = usePrefersReducedMotion()
  const motionEnabled = ffMotionControls !== false
  const press = pressClassName({ enabled: motionEnabled, reduceMotion })

  return (
    <Link
      ref={ref}
      className={cx(
        controlBaseClass,
        focusRingClass,
        sizeClasses[size],
        press,
        variantClasses[variant],
        className,
      )}
      data-motion-controls={motionEnabled ? 'on' : 'off'}
      {...props}
    >
      {children}
    </Link>
  )
})
