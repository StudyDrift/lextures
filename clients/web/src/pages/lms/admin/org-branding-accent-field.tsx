import { Palette } from 'lucide-react'
import {
  deriveAccentRamp,
  formatOklch,
  hexToOklch,
  oklchToHex,
  parseOklch,
} from '../../../lib/tokens/oklch'
import { contrastRatio, AA_NORMAL } from '../../../lib/tokens/contrast'
import { useMemo } from 'react'

type OrgBrandingAccentFieldProps = {
  formId: string
  accentInput: string
  accentHex: string
  onAccentInputChange: (value: string) => void
  onAccentHexChange: (value: string) => void
  onClear: () => void
}

/** Brand accent (OKLCH) field with live ramp preview and AA contrast check. */
export function OrgBrandingAccentField({
  formId,
  accentInput,
  accentHex,
  onAccentInputChange,
  onAccentHexChange,
  onClear,
}: OrgBrandingAccentFieldProps) {
  const accentPreview = useMemo(() => {
    const seed = accentInput.trim() ? parseOklch(accentInput.trim()) : hexToOklch(accentHex)
    if (!seed) return null
    const ramp = deriveAccentRamp(seed)
    const solid = oklchToHex(parseOklch(ramp['600'])!)
    const ratio = contrastRatio('#ffffff', solid)
    return { ramp, solid, ratio, pass: ratio >= AA_NORMAL, oklch: formatOklch(seed) }
  }, [accentInput, accentHex])

  return (
    <div className="rounded-xl border border-border-default bg-surface-sunken p-4">
      <label
        className="mb-2 flex items-center gap-2 text-sm font-medium text-fg-default"
        htmlFor={`${formId}-accent`}
      >
        <Palette className="h-4 w-4" aria-hidden />
        Brand accent (OKLCH)
      </label>
      <p className="mb-3 text-xs text-fg-muted">
        UX.1: one brand hue derives the full accent ramp. Leave empty for the product default.
        Invalid or low-contrast hues are rejected on save.
      </p>
      <div className="flex flex-wrap gap-2">
        <input
          type="color"
          aria-label="Accent colour picker"
          value={accentHex}
          onChange={(e) => {
            onAccentHexChange(e.target.value)
            const o = hexToOklch(e.target.value)
            if (o) onAccentInputChange(formatOklch(o))
          }}
          className="h-10 w-14 cursor-pointer rounded border border-border-default bg-surface-raised"
        />
        <input
          id={`${formId}-accent`}
          type="text"
          value={accentInput}
          onChange={(e) => onAccentInputChange(e.target.value)}
          placeholder="oklch(0.55 0.18 264) or leave empty"
          className="min-w-[12rem] flex-1 rounded-lg border border-border-default bg-surface-raised px-3 py-2 font-mono text-sm text-fg-default"
          autoComplete="off"
          spellCheck={false}
        />
        <button
          type="button"
          className="rounded-lg border border-border-default bg-surface-raised px-3 py-2 text-sm text-fg-muted"
          onClick={onClear}
        >
          Clear
        </button>
      </div>
      {accentPreview ? (
        <div className="mt-3 flex flex-wrap items-center gap-3 text-sm">
          <span
            className="inline-flex items-center rounded-md px-3 py-1.5 text-sm font-semibold text-fg-on-accent"
            style={{ backgroundColor: accentPreview.solid }}
          >
            Primary button
          </span>
          <span className="font-mono text-xs text-fg-muted">{accentPreview.oklch}</span>
          <span
            className={
              accentPreview.pass
                ? 'inline-flex items-center gap-1 text-success-fg'
                : 'inline-flex items-center gap-1 text-danger-fg'
            }
            role="status"
          >
            <span aria-hidden>{accentPreview.pass ? '✓' : '✗'}</span>
            onAccent {accentPreview.ratio.toFixed(2)}:1
            {accentPreview.pass ? ' (AA)' : ' (fails AA)'}
          </span>
        </div>
      ) : accentInput.trim() ? (
        <p className="mt-2 text-sm text-danger-fg" role="status">
          Invalid OKLCH — use oklch(L C H) only.
        </p>
      ) : (
        <p className="mt-2 text-xs text-fg-subtle">Using product accent until you set one.</p>
      )}
    </div>
  )
}
