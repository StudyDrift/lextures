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

export type ButtonVariant = 'primary' | 'secondary' | 'ghost' | 'danger'

export type ButtonProps = ButtonHTMLAttributes<HTMLButtonElement> & {
  variant?: ButtonVariant
  /** Disables press-scale feedback when motion would distract (e.g. drag handles). */
  static?: boolean
  /** Shows spinner and disables the control while preserving width (FR-6). */
  loading?: boolean
  children: ReactNode
}

const variantClasses: Record<ButtonVariant, string> = {
  primary:
    'bg-indigo-600 text-white shadow-sm hover:bg-indigo-500 focus-visible:ring-indigo-500/20 dark:bg-indigo-600 dark:hover:bg-indigo-500',
  secondary:
    'border border-slate-200 bg-white text-slate-800 shadow-sm hover:bg-slate-50 dark:border-neutral-600 dark:bg-neutral-900 dark:text-neutral-100 dark:hover:bg-neutral-800',
  ghost:
    'text-slate-600 hover:bg-slate-100 hover:text-slate-900 dark:text-neutral-400 dark:hover:bg-neutral-850 dark:hover:text-neutral-200',
  danger:
    'bg-rose-600 text-white shadow-sm hover:bg-rose-500 focus-visible:ring-rose-500/20 dark:bg-rose-600 dark:hover:bg-rose-500',
}

export const Button = forwardRef<HTMLButtonElement, ButtonProps>(function Button(
  {
    variant = 'primary',
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
  const press = !isStatic
    ? pressClassName({ enabled: motionEnabled, reduceMotion })
    : ''

  useLayoutEffect(() => {
    if (!loading && labelRef.current) {
      setLabelWidth(labelRef.current.offsetWidth)
    }
  }, [loading, children])

  useEffect(() => {
    if (!loading) return
    // Keep last measured width while loading so the button does not jump (AC-6).
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
      className={[
        'lx-control-btn inline-flex items-center justify-center gap-2 rounded-xl px-4 py-2 text-sm font-semibold',
        'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-offset-2 dark:focus-visible:ring-offset-neutral-950',
        'disabled:cursor-not-allowed disabled:opacity-50',
        press,
        isStatic && 'lex-btn-static',
        loading && 'lx-control-loading',
        variantClasses[variant],
        className,
      ]
        .filter(Boolean)
        .join(' ')}
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
        className={[
          'lx-control-btn-label inline-flex items-center gap-2',
          loading ? 'lx-control-btn-label-exit' : 'lx-control-btn-label-enter',
          !loadState.crossfade && loading ? 'sr-only' : '',
        ]
          .filter(Boolean)
          .join(' ')}
      >
        {children}
      </span>
      {loading ? (
        <span className="lx-control-btn-spinner" aria-hidden="true">
          <span className="lx-control-spinner" />
        </span>
      ) : null}
    </button>
  )
})
