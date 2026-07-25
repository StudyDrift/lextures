import type { AdaptiveContentEffectiveness } from '../../../lib/courses-api'

export type EffectivenessChipProps = {
  effectiveness: AdaptiveContentEffectiveness | null | undefined
  compact?: boolean
}

function formatPts(v: number | null | undefined): string {
  if (v == null || Number.isNaN(v)) return '—'
  const rounded = Math.round(v)
  return `${rounded >= 0 ? '+' : ''}${rounded} pts`
}

/** Compact AC.7 verdict chip with icon + text (not color-only). */
export function EffectivenessChip({ effectiveness, compact = false }: EffectivenessChipProps) {
  if (!effectiveness) {
    return (
      <span
        className="inline-flex items-center gap-1 rounded-md bg-slate-100 px-2 py-0.5 text-xs text-slate-600 dark:bg-neutral-800 dark:text-neutral-300"
        data-testid="ace-effectiveness-chip"
        title="No effectiveness data yet"
      >
        <span aria-hidden="true">○</span>
        Needs data
      </span>
    )
  }

  const nT = effectiveness.nTreatment
  const nH = effectiveness.nHoldout
  const diff = effectiveness.treatmentMinusHoldout
  const verdict = effectiveness.verdict

  let label: string
  let icon: string
  let className: string
  switch (verdict) {
    case 'helping':
      icon = '▲'
      label = compact
        ? `Helping ${formatPts(diff)} vs control (n=${nT}/${nH})`
        : `▲ ${formatPts(diff)} vs control (n=${nT}/${nH})`
      className =
        'bg-emerald-100 text-emerald-900 dark:bg-emerald-950/40 dark:text-emerald-200'
      break
    case 'regressing':
      icon = '▼'
      label = compact
        ? `Regressing — review (n=${nT}/${nH})`
        : `▼ regressing — review this unit (n=${nT}/${nH})`
      className = 'bg-rose-100 text-rose-900 dark:bg-rose-950/40 dark:text-rose-200'
      break
    case 'no_effect':
      icon = '●'
      label = `No measurable effect (n=${nT}/${nH})`
      className =
        'bg-slate-100 text-slate-700 dark:bg-neutral-800 dark:text-neutral-200'
      break
    case 'insufficient_data':
    default:
      icon = '○'
      label = 'Needs more data'
      className =
        'bg-slate-100 text-slate-600 dark:bg-neutral-800 dark:text-neutral-300'
      break
  }

  return (
    <span
      className={`inline-flex items-center gap-1 rounded-md px-2 py-0.5 text-xs font-medium ${className}`}
      data-testid="ace-effectiveness-chip"
      data-verdict={verdict}
      role="status"
      aria-label={`Effectiveness: ${label}`}
    >
      <span aria-hidden="true">{icon}</span>
      {label}
    </span>
  )
}

/** Accessible table summary for effectiveness figures (AC.7 a11y). */
export function EffectivenessSummaryTable({
  effectiveness,
}: {
  effectiveness: AdaptiveContentEffectiveness
}) {
  return (
    <table className="mt-2 w-full text-left text-xs" data-testid="ace-effectiveness-table">
      <caption className="sr-only">Adaptive content effectiveness breakdown</caption>
      <thead>
        <tr className="text-slate-500">
          <th scope="col" className="py-1 pr-2 font-medium">
            Group
          </th>
          <th scope="col" className="py-1 pr-2 font-medium">
            n
          </th>
          <th scope="col" className="py-1 font-medium">
            Mean lift
          </th>
        </tr>
      </thead>
      <tbody>
        <tr>
          <td className="py-0.5 pr-2">Treatment</td>
          <td className="py-0.5 pr-2">{effectiveness.nTreatment}</td>
          <td className="py-0.5">{formatPts(effectiveness.meanLiftTreatment)}</td>
        </tr>
        <tr>
          <td className="py-0.5 pr-2">Holdout</td>
          <td className="py-0.5 pr-2">{effectiveness.nHoldout}</td>
          <td className="py-0.5">{formatPts(effectiveness.meanLiftHoldout)}</td>
        </tr>
        <tr>
          <td className="py-0.5 pr-2">Difference</td>
          <td className="py-0.5 pr-2">—</td>
          <td className="py-0.5">
            {formatPts(effectiveness.treatmentMinusHoldout)}
            {effectiveness.diffStdError != null
              ? ` (±${Math.round(effectiveness.diffStdError)} SE)`
              : ''}
          </td>
        </tr>
      </tbody>
    </table>
  )
}
