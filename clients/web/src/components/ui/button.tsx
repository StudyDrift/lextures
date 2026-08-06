import {
  forwardRef,
  useEffect,
  useLayoutEffect,
  useRef,
  useState,
  type ButtonHTMLAttributes,
  type ReactNode,
} from 'react'
import { usePlatformFeatures } from '../../context/platform-features-context'
import { loadingButtonState, pressClassName, useHaptics } from '../../lib/control-motion'
import { usePrefersReducedMotion } from '../../lib/motion'
import { controlBaseClass, cx, focusRingClass, sizeClasses, type ControlSize } from './utils'
import { Spinner } from './spinner'

export type ButtonVariant = 'primary' | 'secondary' | 'ghost' | 'danger'

export type ButtonProps = ButtonHTMLAttributes<HTMLButtonElement> & {
  variant?: ButtonVariant
  size?: ControlSize
  /** Disables press-scale feedback when motion would distract (e.g. drag handles). */
  static?: boolean
  /** Shows spinner and disables the control while preserving width (FR-6). */
  loading?: boolean
  children: ReactNode
}

const variantClasses: Record<ButtonVariant, string> = {
  primary:
    'bg-accent-solid text-fg-on-accent shadow-sm hover:opacity-90 focus-visible:ring-accent-solid/30',
  secondary:
    'border border-border-default bg-surface-raised text-fg-default shadow-sm hover:bg-surface-sunken',
  ghost: 'text-fg-muted hover:bg-surface-sunken hover:text-fg-default',
  danger:
    'bg-danger-fg text-fg-on-accent shadow-sm hover:opacity-90 focus-visible:ring-danger-fg/30',
}

export const Button = forwardRef<HTMLButtonElement, ButtonProps>(function Button(
  {
    variant = 'primary',
    size = 'md',
    static: isStatic,
    loading = false,
    className = '',
    disabled,
    children,
    type = 'button',
    onClick,
    ...props
  },
  ref,
) {
  const { ffMotionControls } = usePlatformFeatures()
  const reduceMotion = usePrefersReducedMotion()
  const { trigger } = useHaptics()
  const labelRef = useRef<HTMLSpanElement>(null)
  const [labelWidth, setLabelWidth] = useState<number | undefined>(undefined)

  const motionEnabled = ffMotionControls !== false
  const press = !isStatic ? pressClassName({ enabled: motionEnabled, reduceMotion }) : ''

  useLayoutEffect(() => {
    if (!loading && labelRef.current) {
      setLabelWidth(labelRef.current.offsetWidth)
    }
  }, [loading, children])

  useEffect(() => {
    if (!loading) return
  }, [loading])

  const loadState = loadingButtonState({
    loading,
    labelWidthPx: labelWidth,
    enabled: motionEnabled,
    reduceMotion,
  })

  return (
    <button
      ref={ref}
      type={type}
      disabled={disabled || loading}
      aria-busy={loadState.ariaBusy}
      data-loading={loading ? 'true' : undefined}
      data-motion-controls={motionEnabled ? 'on' : 'off'}
      className={cx(
        'lx-control-btn',
        controlBaseClass,
        focusRingClass,
        sizeClasses[size],
        press,
        isStatic && 'lex-btn-static',
        loading && 'lx-control-loading',
        variantClasses[variant],
        className,
      )}
      style={loadState.minWidth ? { minWidth: loadState.minWidth } : undefined}
      onClick={(e) => {
        // FR-9: haptic/motion never gates the handler.
        if (variant === 'primary' || variant === 'danger') {
          trigger('tap')
        }
        onClick?.(e)
      }}
      {...props}
    >
      <span
        ref={labelRef}
        className={cx(
          'lx-control-btn-label inline-flex items-center gap-2',
          loading ? 'lx-control-btn-label-exit' : 'lx-control-btn-label-enter',
          !loadState.crossfade && loading ? 'sr-only' : '',
        )}
      >
        {children}
      </span>
      {loading ? (
        <span className="lx-control-btn-spinner" aria-hidden="true">
          <Spinner size="sm" className="lx-control-spinner border-current" />
        </span>
      ) : null}
    </button>
  )
})
