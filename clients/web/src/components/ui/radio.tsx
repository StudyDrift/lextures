import {
  createContext,
  forwardRef,
  useContext,
  useId,
  type InputHTMLAttributes,
  type ReactNode,
} from 'react'
import { cx, focusRingClass } from './utils'

type RadioGroupContextValue = {
  name: string
  value?: string
  onChange?: (value: string) => void
  disabled?: boolean
}

const RadioGroupContext = createContext<RadioGroupContextValue | null>(null)

export type RadioGroupProps = {
  name?: string
  value?: string
  defaultValue?: string
  onChange?: (value: string) => void
  disabled?: boolean
  legend?: ReactNode
  children: ReactNode
  className?: string
  orientation?: 'horizontal' | 'vertical'
}

export function RadioGroup({
  name: nameProp,
  value,
  onChange,
  disabled,
  legend,
  children,
  className = '',
  orientation = 'vertical',
}: RadioGroupProps) {
  const autoName = useId()
  const name = nameProp ?? autoName

  return (
    <fieldset className={cx('min-w-0 border-0 p-0', className)} disabled={disabled}>
      {legend ? <legend className="mb-2 text-sm font-medium text-fg-default">{legend}</legend> : null}
      <div
        role="radiogroup"
        aria-orientation={orientation}
        className={cx('flex gap-3', orientation === 'vertical' ? 'flex-col' : 'flex-row flex-wrap')}
      >
        <RadioGroupContext.Provider value={{ name, value, onChange, disabled }}>
          {children}
        </RadioGroupContext.Provider>
      </div>
    </fieldset>
  )
}

export type RadioProps = Omit<InputHTMLAttributes<HTMLInputElement>, 'type' | 'size'> & {
  label?: ReactNode
  value: string
}

export const Radio = forwardRef<HTMLInputElement, RadioProps>(function Radio(
  { className = '', label, value, id, disabled, onChange, checked, name, ...props },
  ref,
) {
  const ctx = useContext(RadioGroupContext)
  const autoId = useId()
  const controlId = id ?? autoId
  const isDisabled = disabled || ctx?.disabled
  const isChecked = checked ?? (ctx?.value != null ? ctx.value === value : undefined)

  const control = (
    <input
      ref={ref}
      id={controlId}
      type="radio"
      name={name ?? ctx?.name}
      value={value}
      checked={isChecked}
      disabled={isDisabled}
      className={cx(
        'h-6 w-6 shrink-0 border-border-default text-accent-solid accent-accent-solid disabled:opacity-50',
        focusRingClass,
        className,
      )}
      onChange={(e) => {
        onChange?.(e)
        if (e.target.checked) ctx?.onChange?.(value)
      }}
      {...props}
    />
  )

  if (!label) return control

  return (
    <label
      htmlFor={controlId}
      className={cx(
        'inline-flex min-h-6 cursor-pointer items-center gap-2.5 text-sm font-medium text-fg-default',
        isDisabled && 'cursor-not-allowed opacity-50',
      )}
    >
      {control}
      {label}
    </label>
  )
})
