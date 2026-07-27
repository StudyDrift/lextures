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
  unitMode: boolean
  readOnly: boolean
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
  unitMode,
  readOnly,
  t,
  onFocusUnit,
  onOpenUnit,
  onUnitKeyDown,
  onPointerUp,
}: PassagePanelProps) {
  return (
    <div
      ref={passageRef}
      className="rounded border border-slate-200 p-3 text-sm leading-relaxed text-slate-900 dark:border-neutral-700 dark:text-neutral-100"
      data-testid="ha-passage"
      aria-labelledby={promptId}
      onMouseUp={onPointerUp}
    >
      {units.map((unit) => {
        const unitAnns = active.filter((a) => a.anchor.unitIndex === unit.index)
        const tag = unitAnns[0] ? tags.find((tg) => tg.id === unitAnns[0]!.tagId) : null
        const tagIdx = tag ? tags.findIndex((tg) => tg.id === tag.id) : 0
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
              className="rounded px-0.5 text-start align-baseline focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-400"
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
              <span className="ms-1 inline-flex gap-0.5 align-super text-[10px] font-medium text-slate-600 dark:text-neutral-300">
                {unitAnns.map((a) => {
                  const tg = tags.find((x) => x.id === a.tagId)
                  return (
                    <span key={a.id} style={{ color: tg?.color }}>
                      [{tg?.label ?? a.tagId}]
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
