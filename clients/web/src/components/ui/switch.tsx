import {
  forwardRef,
  useId,
  useState,
  type ButtonHTMLAttributes,
  type ReactNode,
} from 'react'
import { cx, focusRingClass } from './utils'

export type SwitchProps = Omit<ButtonHTMLAttributes<HTMLButtonElement>, 'children'> & {
  checked?: boolean
  defaultChecked?: boolean
  onCheckedChange?: (checked: boolean) => void
  label?: ReactNode
  description?: ReactNode
}

export const Switch = forwardRef<HTMLButtonElement, SwitchProps>(function Switch(
  {
    checked,
    defaultChecked = false,
    onCheckedChange,
    label,
    description,
    className = '',
    disabled,
    id,
    onClick,
    ...props
  },
  ref,
) {
  const autoId = useId()
  const controlId = id ?? autoId
  const isControlled = checked !== undefined
  const [uncontrolled, setUncontrolled] = useState(defaultChecked)
  const isOn = isControlled ? Boolean(checked) : uncontrolled

  const toggle = (
    <button
      ref={ref}
      id={controlId}
      type="button"
      role="switch"
      aria-checked={isOn}
      disabled={disabled}
      className={cx(
        'relative inline-flex h-6 w-11 shrink-0 items-center rounded-full border border-transparent transition-colors',
        focusRingClass,
        isOn ? 'bg-accent-solid' : 'bg-surface-sunken',
        disabled && 'cursor-not-allowed opacity-50',
        className,
      )}
      onClick={(e) => {
        onClick?.(e)
        if (e.defaultPrevented || disabled) return
        const next = !isOn
        if (!isControlled) setUncontrolled(next)
        onCheckedChange?.(next)
      }}
      {...props}
    >
      <span
        aria-hidden
        className={cx(
          'pointer-events-none inline-block h-5 w-5 rounded-full bg-surface-raised shadow-sm transition-transform',
          isOn ? 'translate-x-5 rtl:-translate-x-5' : 'translate-x-0.5 rtl:-translate-x-0.5',
        )}
      />
    </button>
  )

  if (!label) return toggle

  return (
    <div className="inline-flex items-start gap-3">
      {toggle}
      <label htmlFor={controlId} className="flex cursor-pointer flex-col gap-0.5 text-sm">
        <span className="font-medium text-fg-default">{label}</span>
        {description ? <span className="text-xs text-fg-muted">{description}</span> : null}
      </label>
    </div>
  )
})
