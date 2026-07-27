import type { Parameter } from './types'

type Props = {
  parameters: Parameter[]
  values: Record<string, number | boolean | string>
  onChange: (id: string, value: number | boolean | string) => void
  readOnly?: boolean
  t: (key: string, options?: Record<string, unknown>) => string
}

export function ParameterControls({ parameters, values, onChange, readOnly, t }: Props) {
  return (
    <div className="space-y-3" data-testid="parameter-explorer-controls">
      {parameters.map((p) => {
        if (p.kind === 'boolean') {
          const checked = Boolean(values[p.id] ?? p.default)
          return (
            <label key={p.id} className="flex items-center gap-2 text-sm">
              <input
                type="checkbox"
                checked={checked}
                disabled={readOnly}
                onChange={(e) => onChange(p.id, e.target.checked)}
                aria-describedby={p.description ? `${p.id}-desc` : undefined}
              />
              <span className="font-medium">{p.label}</span>
              {p.description ? (
                <span id={`${p.id}-desc`} className="text-xs text-slate-500">
                  {p.description}
                </span>
              ) : null}
            </label>
          )
        }
        if (p.kind === 'choice') {
          const val = String(values[p.id] ?? p.default)
          return (
            <label key={p.id} className="block space-y-1 text-sm">
              <span className="font-medium">{p.label}</span>
              <select
                className="w-full rounded border border-slate-300 bg-white px-2 py-1.5 dark:border-neutral-600 dark:bg-neutral-950"
                value={val}
                disabled={readOnly}
                onChange={(e) => onChange(p.id, e.target.value)}
              >
                {p.options.map((o) => (
                  <option key={o.value} value={o.value}>
                    {o.label}
                  </option>
                ))}
              </select>
            </label>
          )
        }
        // number
        const num = typeof values[p.id] === 'number' ? (values[p.id] as number) : p.default
        const valueText = p.unit ? `${num} ${p.unit}` : String(num)
        return (
          <div key={p.id} className="space-y-1 text-sm">
            <div className="flex items-baseline justify-between gap-2">
              <label htmlFor={`pe-range-${p.id}`} className="font-medium">
                {p.label}
                {p.unit ? (
                  <span className="ml-1 font-normal text-slate-500">({p.unit})</span>
                ) : null}
              </label>
              <span className="tabular-nums text-xs text-slate-600 dark:text-neutral-400">
                {valueText}
              </span>
            </div>
            {p.description ? (
              <p className="text-xs text-slate-500" id={`pe-desc-${p.id}`}>
                {p.description}
              </p>
            ) : null}
            <div className="flex items-center gap-2">
              <input
                id={`pe-range-${p.id}`}
                type="range"
                min={p.min}
                max={p.max}
                step={p.step}
                value={num}
                disabled={readOnly}
                aria-valuemin={p.min}
                aria-valuemax={p.max}
                aria-valuenow={num}
                aria-valuetext={valueText}
                aria-describedby={p.description ? `pe-desc-${p.id}` : undefined}
                className="min-w-0 flex-1"
                data-testid={`parameter-explorer-slider-${p.id}`}
                onChange={(e) => onChange(p.id, Number(e.target.value))}
                onKeyDown={(e) => {
                  if (readOnly) return
                  if (e.key === 'Home') {
                    e.preventDefault()
                    onChange(p.id, p.min)
                  } else if (e.key === 'End') {
                    e.preventDefault()
                    onChange(p.id, p.max)
                  }
                }}
              />
              <input
                type="number"
                inputMode="decimal"
                min={p.min}
                max={p.max}
                step={p.step}
                value={num}
                disabled={readOnly}
                aria-label={t('contentTools.tools.parameter_explorer.numericInput', {
                  label: p.label,
                })}
                className="w-20 rounded border border-slate-300 bg-white px-1.5 py-1 text-sm tabular-nums dark:border-neutral-600 dark:bg-neutral-950"
                data-testid={`parameter-explorer-number-${p.id}`}
                onChange={(e) => {
                  const v = Number(e.target.value)
                  if (!Number.isFinite(v)) return
                  const clamped = Math.min(p.max, Math.max(p.min, v))
                  onChange(p.id, clamped)
                }}
              />
            </div>
          </div>
        )
      })}
    </div>
  )
}
