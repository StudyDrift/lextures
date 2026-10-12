import { Delete } from 'lucide-react'
import { Button, Field, Input } from '../ui'
import { PIN_MAX } from './picture-choices'

type Props = {
  id: string
  label: string
  value: string
  onChange: (next: string) => void
  disabled?: boolean
  /** Show the large on-screen keypad (touch-friendly picker). */
  keypad?: boolean
  autoFocus?: boolean
}

const DIGITS = ['1', '2', '3', '4', '5', '6', '7', '8', '9']

/** 4–6 digit PIN entry with an optional on-screen keypad (#771). */
export function PinPad({ id, label, value, onChange, disabled = false, keypad = false, autoFocus = false }: Props) {
  const append = (d: string) => {
    if (value.length < PIN_MAX) onChange(value + d)
  }
  return (
    <div className="space-y-3">
      <Field label={label} htmlFor={id}>
        <Input
          id={id}
          type="password"
          inputMode="numeric"
          autoComplete="off"
          pattern="[0-9]*"
          maxLength={PIN_MAX}
          value={value}
          autoFocus={autoFocus}
          disabled={disabled}
          onChange={(e) => onChange(e.target.value.replace(/\D/g, '').slice(0, PIN_MAX))}
          className="text-center text-title-lg tracking-[0.5em]"
        />
      </Field>
      {keypad ? (
        <div role="group" aria-label={`${label} keypad`} className="grid grid-cols-3 gap-2">
          {DIGITS.map((d) => (
            <Button key={d} type="button" variant="secondary" disabled={disabled} onClick={() => append(d)} className="h-14 text-title">
              {d}
            </Button>
          ))}
          <span aria-hidden />
          <Button type="button" variant="secondary" disabled={disabled} onClick={() => append('0')} className="h-14 text-title">
            0
          </Button>
          <Button
            type="button"
            variant="ghost"
            disabled={disabled || value.length === 0}
            onClick={() => onChange(value.slice(0, -1))}
            aria-label="Delete last digit"
            className="h-14"
          >
            <Delete className="h-5 w-5" aria-hidden />
          </Button>
        </div>
      ) : null}
    </div>
  )
}
