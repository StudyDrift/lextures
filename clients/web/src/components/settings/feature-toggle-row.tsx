import type { ReactNode } from 'react'
import { usePlatformFeatures } from '../../context/platform-features-context'
import { pressClassName, useHaptics } from '../../lib/control-motion'
import { usePrefersReducedMotion } from '../../lib/motion'

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
  /** Short human hint for DERIVE (credential-gated) flags, e.g. "SAML SP certificate + private key". */
  deriveFrom?: string
}

export function FeatureToggleRow({
  label,
  description,
  enabled,
  disabled = false,
  disabledReason,
  onToggle,
  meta,
  deriveFrom,
}: Props) {
  const descriptionId = disabledReason ? `${slug(label)}-disabled-reason` : undefined
  const { ffMotionControls } = usePlatformFeatures()
  const reduceMotion = usePrefersReducedMotion()
  const { trigger } = useHaptics()
  const motionEnabled = ffMotionControls !== false
  const press = pressClassName({ enabled: motionEnabled && !disabled, reduceMotion })

  return (
    <div className="flex flex-wrap items-start justify-between gap-4 py-4">
      <div className="min-w-0 flex-1">
        <p className="text-sm font-semibold text-slate-900 dark:text-neutral-100">
          {label}
          {meta ? <span className="font-normal">{meta}</span> : null}
          {deriveFrom ? (
            <span
              className="ms-2 rounded-md bg-amber-100 px-1.5 py-0.5 text-[10px] font-semibold uppercase tracking-wide text-amber-800 dark:bg-amber-950/60 dark:text-amber-200"
              data-testid="feature-toggle-config-gated-badge"
            >
              Config-gated
            </span>
          ) : null}
        </p>
        <p className="mt-1 text-sm text-slate-500 dark:text-neutral-400">{description}</p>
        {deriveFrom ? (
          <p
            className="mt-1 text-xs text-slate-400 dark:text-neutral-500"
            data-testid="feature-toggle-derive-note"
          >
            Requires: {deriveFrom}
            {!enabled ? ' — won\u2019t take effect until these credentials are configured.' : ''}
          </p>
        ) : null}
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
        onClick={() => {
          trigger('selection')
          onToggle()
        }}
        disabled={disabled}
        title={disabled && disabledReason ? disabledReason : undefined}
        data-motion-controls={motionEnabled ? 'on' : 'off'}
        className={[
          'lx-control-switch relative mt-0.5 inline-flex h-7 w-12 shrink-0 rounded-full border-2 border-transparent',
          'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-indigo-500 focus-visible:ring-offset-2',
          'disabled:cursor-not-allowed disabled:opacity-50',
          press,
          enabled ? 'bg-indigo-600' : 'bg-slate-200 dark:bg-neutral-700',
        ]
          .filter(Boolean)
          .join(' ')}
      >
        <span
          className={[
            'lx-control-switch-thumb pointer-events-none inline-block h-6 w-6 transform rounded-full bg-white shadow ring-0',
            enabled ? 'translate-x-5' : 'translate-x-0.5',
          ]
            .filter(Boolean)
            .join(' ')}
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
