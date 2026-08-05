import type { KeyboardEvent, RefObject } from 'react'
import type { PassageUnit } from '../../../../lib/text-anchoring'
import { type Annotation, type Tag, underlineStyle } from './types'

export type PassagePanelProps = {
  passageRef: RefObject<HTMLDivElement | null>
  promptId: string
  units: PassageUnit[]
  tags: Tag[]
  active: Annotation[]
  focusedUnit: number
  menuUnitIndex: number | null
  unitMode: boolean
  readOnly: boolean
  showStartCue: boolean
  t: (key: string, options?: Record<string, unknown>) => string
  onFocusUnit: (index: number) => void
  onOpenUnit: (index: number) => void
  onUnitKeyDown: (e: KeyboardEvent, index: number) => void
  onPointerUp: () => void
}

export function PassagePanel({
  passageRef,
  promptId,
  units,
  tags,
  active,
  focusedUnit,
  menuUnitIndex,
  unitMode,
  readOnly,
  showStartCue,
  t,
  onFocusUnit,
  onOpenUnit,
  onUnitKeyDown,
  onPointerUp,
}: PassagePanelProps) {
  return (
    <div
      ref={passageRef}
      className={[
        'rounded-xl border p-4 text-base leading-relaxed text-slate-900 dark:text-neutral-100',
        'border-slate-200 bg-white dark:border-neutral-700 dark:bg-neutral-950',
        !readOnly && unitMode
          ? 'border-dashed border-indigo-200 dark:border-indigo-900/60'
          : '',
      ]
        .filter(Boolean)
        .join(' ')}
      data-testid="ha-passage"
      aria-labelledby={promptId}
      onMouseUp={onPointerUp}
    >
      {units.map((unit) => {
        const unitAnns = active.filter((a) => a.anchor.unitIndex === unit.index)
        const tag = unitAnns[0] ? tags.find((tg) => tg.id === unitAnns[0]!.tagId) : null
        const tagIdx = tag ? tags.findIndex((tg) => tg.id === tag.id) : 0
        const isMenuTarget = menuUnitIndex === unit.index
        const isStartCue = showStartCue && unit.index === 0 && unitAnns.length === 0
        const label = t('contentTools.tools.highlight_annotate.unitLabel', {
          n: unit.index + 1,
          total: units.length,
          tags:
            unitAnns
              .map((a) => tags.find((tg) => tg.id === a.tagId)?.label ?? a.tagId)
              .join(', ') || t('contentTools.tools.highlight_annotate.noTags'),
        })
        return (
          <span key={unit.index} className="inline">
            <button
              type="button"
              data-unit-index={unit.index}
              data-testid={`ha-unit-${unit.index}`}
              className={[
                'rounded px-1 py-0.5 text-start align-baseline transition-colors',
                'focus:outline-none focus-visible:ring-2 focus-visible:ring-indigo-400 focus-visible:ring-offset-1',
                !readOnly && unitMode
                  ? 'cursor-pointer hover:bg-indigo-50 dark:hover:bg-indigo-950/50'
                  : '',
                isMenuTarget
                  ? 'bg-indigo-100 ring-2 ring-indigo-400 dark:bg-indigo-950/70 dark:ring-indigo-500'
                  : '',
                isStartCue
                  ? 'bg-indigo-50/80 ring-1 ring-indigo-200 dark:bg-indigo-950/40 dark:ring-indigo-800'
                  : '',
                unitAnns.length === 0 && !readOnly && unitMode
                  ? 'decoration-slate-300 underline decoration-dotted decoration-1 underline-offset-4 dark:decoration-neutral-600'
                  : '',
              ]
                .filter(Boolean)
                .join(' ')}
              style={tag ? underlineStyle(tag.color, Math.max(0, tagIdx)) : undefined}
              aria-label={label}
              tabIndex={focusedUnit === unit.index ? 0 : -1}
              disabled={readOnly}
              onFocus={() => onFocusUnit(unit.index)}
              onClick={() => {
                if (unitMode) onOpenUnit(unit.index)
              }}
              onKeyDown={(e) => onUnitKeyDown(e, unit.index)}
            >
              {unit.text}
            </button>
            {unitAnns.length > 0 ? (
              <span className="ms-1 inline-flex flex-wrap gap-0.5 align-super">
                {unitAnns.map((a) => {
                  const tg = tags.find((x) => x.id === a.tagId)
                  return (
                    <span
                      key={a.id}
                      className="inline-flex items-center rounded-full px-1.5 py-px text-[10px] font-semibold leading-tight text-white shadow-sm"
                      style={{ backgroundColor: tg?.color ?? '#64748b' }}
                    >
                      {tg?.label ?? a.tagId}
                    </span>
                  )
                })}
              </span>
            ) : null}{' '}
          </span>
        )
      })}
    </div>
  )
}
