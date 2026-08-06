import type { KeyboardEvent } from 'react'
import type { CheckResult, DiagramLabel } from './types'

export type LabelTrayProps = {
  trayIds: string[]
  labels: DiagramLabel[]
  grabbedId: string | null
  lockedIds: string[]
  readOnly: boolean
  showPerItem: boolean
  checkResult: CheckResult | null
  lastPerItem: Record<string, boolean>
  t: (key: string, opts?: Record<string, unknown>) => string
  onActivate: (id: string) => void
  onKeyDown: (e: KeyboardEvent, id: string) => void
  onPickUp: (id: string) => void
}

export function LabelTray({
  trayIds,
  labels,
  grabbedId,
  lockedIds,
  readOnly,
  showPerItem,
  checkResult,
  lastPerItem,
  t,
  onActivate,
  onKeyDown,
  onPickUp,
}: LabelTrayProps) {
  return (
    <section
      aria-label={t('contentTools.tools.diagram_hotspot.tray')}
      data-testid="diagram-tray"
      className="min-h-12 overflow-x-auto rounded border border-dashed border-border-strong p-2 dark:border-border-default"
    >
      <p className="mb-1 text-xs font-medium text-fg-muted">
        {t('contentTools.tools.diagram_hotspot.tray')} ({trayIds.length})
      </p>
      <div className="flex flex-nowrap gap-2 sm:flex-wrap">
        {trayIds.map((id) => {
          const label = labels.find((l) => l.id === id)
          if (!label) return null
          const grabbed = grabbedId === id
          const correctness = showPerItem
            ? (checkResult?.perItem?.[id]?.correct ?? lastPerItem[id])
            : undefined
          return (
            <button
              key={id}
              type="button"
              data-testid={`diagram-label-${id}`}
              draggable={!readOnly && !lockedIds.includes(id)}
              className={`shrink-0 rounded-full border px-3 py-1 text-sm ${ grabbed ? 'border-sky-600 bg-sky-100 dark:bg-sky-950' : 'border-border-strong bg-surface-raised dark:border-border-default dark:bg-surface-raised' } ${ correctness === true ? 'ring-2 ring-teal-600' : correctness === false ? 'ring-2 ring-rose-500' : '' }`}
              aria-grabbed={grabbed}
              disabled={readOnly || lockedIds.includes(id)}
              onClick={() => {
                if (readOnly || lockedIds.includes(id)) return
                onActivate(id)
              }}
              onKeyDown={(e) => onKeyDown(e, id)}
              onDragStart={(e) => {
                e.dataTransfer.setData('text/plain', id)
                e.dataTransfer.effectAllowed = 'move'
                onPickUp(id)
              }}
            >
              {label.text}
              {correctness === true
                ? ` — ${t('contentTools.tools.diagram_hotspot.correct')}`
                : correctness === false
                  ? ` — ${t('contentTools.tools.diagram_hotspot.incorrect')}`
                  : ''}
            </button>
          )
        })}
      </div>
    </section>
  )
}
