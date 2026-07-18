import type { ReactNode } from 'react'

type Props = {
  label: string
  description: string
  enabled: boolean
  disabled?: boolean
  /** Explains why the switch is disabled (e.g. environment-owned). Announced to AT. */
  disabledReason?: string
  onToggle: () => void
  /** Optional badge or hint shown beside the label (e.g. env/database source). */
  meta?: ReactNode
}

export function FeatureToggleRow({
  label,
  description,
  enabled,
  disabled = false,
  disabledReason,
  onToggle,
  meta,
}: Props) {
  const descriptionId = disabledReason ? `${slug(label)}-disabled-reason` : undefined

  return (
    <div className="flex flex-wrap items-start justify-between gap-4 py-4">
      <div className="min-w-0 flex-1">
        <p className="text-sm font-semibold text-slate-900 dark:text-neutral-100">
          {label}
          {meta ? <span className="font-normal">{meta}</span> : null}
        </p>
        <p className="mt-1 text-sm text-slate-500 dark:text-neutral-400">{description}</p>
        {disabled && disabledReason ? (
          <p
            id={descriptionId}
            className="mt-1 text-xs text-amber-800 dark:text-amber-200"
            data-testid="feature-toggle-disabled-reason"
          >
            {disabledReason}
          </p>
        ) : null}
      </div>
      <button
        type="button"
        role="switch"
        aria-checked={enabled}
        aria-describedby={descriptionId}
        onClick={onToggle}
        disabled={disabled}
        title={disabled && disabledReason ? disabledReason : undefined}
        className={`relative mt-0.5 inline-flex h-7 w-12 shrink-0 rounded-full border-2 border-transparent transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-indigo-500 focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50 ${
          enabled ? 'bg-indigo-600' : 'bg-slate-200 dark:bg-neutral-700'
        }`}
      >
        <span
          className={`pointer-events-none inline-block h-6 w-6 transform rounded-full bg-white shadow ring-0 transition-transform ${
            enabled ? 'translate-x-5' : 'translate-x-0.5'
          }`}
        />
      </button>
    </div>
  )
}

function slug(value: string): string {
  return value
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-|-$/g, '')
}
