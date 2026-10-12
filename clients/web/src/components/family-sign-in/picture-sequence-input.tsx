import { Button } from '../ui'
import { PICTURE_CHOICES, PICTURE_MAX, pictureFor } from './picture-choices'

type Props = {
  value: string[]
  onChange: (next: string[]) => void
  choices?: readonly string[]
  max?: number
  disabled?: boolean
  /** Accessible name for the picture grid. */
  label: string
  /** Hide the chosen pictures (sign-in) and show dots instead. */
  masked?: boolean
}

/** Tap pictures in order to build a picture password (#771). */
export function PictureSequenceInput({
  value,
  onChange,
  choices = PICTURE_CHOICES,
  max = PICTURE_MAX,
  disabled = false,
  label,
  masked = false,
}: Props) {
  const full = value.length >= max
  return (
    <div className="space-y-3">
      <div
        className="flex min-h-12 flex-wrap items-center gap-2 rounded-xl border border-border-default bg-surface-sunken px-3 py-2"
        aria-live="polite"
        aria-label={`${value.length} of up to ${max} pictures chosen`}
      >
        {value.length === 0 ? (
          <span className="text-body-sm text-fg-muted">Tap pictures in order</span>
        ) : (
          value.map((id, i) =>
            masked ? (
              <span key={`${id}-${i}`} aria-hidden className="h-3 w-3 rounded-full bg-fg-default" />
            ) : (
              <span key={`${id}-${i}`} aria-hidden className="text-title-lg">
                {pictureFor(id).emoji}
              </span>
            ),
          )
        )}
      </div>
      <div role="group" aria-label={label} className="grid grid-cols-3 gap-2">
        {choices.map((id) => {
          const p = pictureFor(id)
          return (
            <Button
              key={id}
              type="button"
              variant="secondary"
              disabled={disabled || full}
              onClick={() => onChange([...value, id])}
              aria-label={p.label}
              className="h-16 flex-col gap-0 text-display"
            >
              <span aria-hidden>{p.emoji}</span>
            </Button>
          )
        })}
      </div>
      <div className="flex gap-2">
        <Button
          type="button"
          variant="ghost"
          size="sm"
          disabled={disabled || value.length === 0}
          onClick={() => onChange(value.slice(0, -1))}
        >
          Undo
        </Button>
        <Button type="button" variant="ghost" size="sm" disabled={disabled || value.length === 0} onClick={() => onChange([])}>
          Clear
        </Button>
      </div>
    </div>
  )
}
