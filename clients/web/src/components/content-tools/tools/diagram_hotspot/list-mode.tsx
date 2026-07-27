import type { DiagramLabel, DiagramPrompt, DiagramRegion } from './types'

export type ListModeProps = {
  mode: 'label' | 'hotspot'
  labels: DiagramLabel[]
  prompts: DiagramPrompt[]
  regions: DiagramRegion[]
  assignments: Record<string, string | null>
  lockedIds: string[]
  readOnly: boolean
  t: (key: string, opts?: Record<string, unknown>) => string
  onAssign: (itemId: string, regionId: string | null) => void
}

export function ListModeView({
  mode,
  labels,
  prompts,
  regions,
  assignments,
  lockedIds,
  readOnly,
  t,
  onAssign,
}: ListModeProps) {
  const items =
    mode === 'hotspot'
      ? prompts.map((p) => ({ id: p.id, text: p.text }))
      : labels.map((l) => ({ id: l.id, text: l.text }))

  return (
    <div className="space-y-3" data-testid="diagram-list-mode">
      <p className="text-xs text-slate-600 dark:text-neutral-300">
        {t('contentTools.tools.diagram_hotspot.listModeHelp')}
      </p>
      <ul className="space-y-3">
        {items.map((item) => {
          const locked = lockedIds.includes(item.id)
          return (
            <li
              key={item.id}
              className="grid gap-2 rounded border border-slate-200 p-3 sm:grid-cols-2 dark:border-neutral-700"
            >
              <div>
                <p className="text-sm font-medium text-slate-800 dark:text-neutral-100">{item.text}</p>
                {locked ? (
                  <p className="text-xs text-slate-500">{t('contentTools.tools.diagram_hotspot.locked')}</p>
                ) : null}
              </div>
              <label className="block space-y-1 text-xs">
                <span className="font-medium text-slate-600 dark:text-neutral-300">
                  {t('contentTools.tools.diagram_hotspot.assignRegion')}
                </span>
                <select
                  data-testid={`diagram-list-select-${item.id}`}
                  className="w-full rounded border border-slate-300 bg-white px-2 py-1.5 text-sm dark:border-neutral-600 dark:bg-neutral-950"
                  disabled={readOnly || locked}
                  value={assignments[item.id] ?? ''}
                  onChange={(e) => onAssign(item.id, e.target.value || null)}
                >
                  <option value="">{t('contentTools.tools.diagram_hotspot.unassigned')}</option>
                  {regions.map((r) => (
                    <option key={r.id} value={r.id}>
                      {r.label} — {r.description}
                    </option>
                  ))}
                </select>
              </label>
            </li>
          )
        })}
      </ul>
    </div>
  )
}
