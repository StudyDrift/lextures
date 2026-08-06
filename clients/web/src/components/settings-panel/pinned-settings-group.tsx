import {
  useCallback,
  useEffect,
  useId,
  useMemo,
  type KeyboardEvent as ReactKeyboardEvent,
  type ReactNode,
} from 'react'
import {
  DndContext,
  PointerSensor,
  TouchSensor,
  closestCenter,
  useSensor,
  useSensors,
  type DragEndEvent,
} from '@dnd-kit/core'
import {
  SortableContext,
  arrayMove,
  useSortable,
  verticalListSortingStrategy,
} from '@dnd-kit/sortable'
import { CSS } from '@dnd-kit/utilities'
import { GripVertical, X } from 'lucide-react'
import { defaultKeyboardSensorOptions, KeyboardSensor } from '../../lib/dnd/keyboardSensorConfig'
import { pinnedSettingsCopy } from '../../lib/pinned-settings-copy'
import type { SettingDescriptor } from '../../lib/settings-registry'
import { setPinHost } from './pin-host-registry'
import { useSettingsPanelContext } from './settings-panel-context'
import { SuggestedPinsStrip } from './suggested-pins-strip'
import type { UsePinnedSettingsResult } from './use-pinned-settings'

export type PinnedSettingsGroupProps = {
  pins: UsePinnedSettingsResult
  /**
   * Descriptors that are currently mounted (registered) and match search.
   * Order follows pin list; conditional unmounted controls are omitted (FR-10).
   */
  visiblePinned: SettingDescriptor[]
}

function SortablePinSlot({
  id,
  label,
  onMoveByOffset,
  children,
}: {
  id: string
  label: string
  onMoveByOffset: (id: string, delta: -1 | 1) => void
  children: ReactNode
}) {
  const reorderHelpId = useId()
  const {
    attributes,
    listeners,
    setNodeRef,
    setActivatorNodeRef,
    transform,
    transition,
    isDragging,
  } = useSortable({ id })

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.85 : undefined,
    zIndex: isDragging ? 10 : undefined,
  }

  return (
    <div
      ref={setNodeRef}
      style={style}
      className="group/pin-slot relative rounded-md border border-transparent px-0.5 py-1 hover:border-border-subtle dark:hover:border-border-subtle"
      data-pinned-setting-id={id}
    >
      <div className="flex items-start gap-1">
        <button
          ref={setActivatorNodeRef}
          type="button"
          className="mt-1 inline-flex h-6 w-6 shrink-0 cursor-grab items-center justify-center rounded-md text-fg-subtle outline-none hover:bg-surface-sunken hover:text-fg-muted focus-visible:ring-2 focus-visible:ring-indigo-400 active:cursor-grabbing dark:hover:bg-surface-overlay dark:hover:text-fg-muted"
          {...attributes}
          {...listeners}
          aria-label={pinnedSettingsCopy.reorderHandle(label)}
          aria-describedby={[reorderHelpId, attributes['aria-describedby']]
            .filter(Boolean)
            .join(' ')}
          onKeyDown={(e: ReactKeyboardEvent) => {
            const listenerKeyDown = listeners?.onKeyDown as
              | ((ev: ReactKeyboardEvent) => void)
              | undefined
            listenerKeyDown?.(e)
            if (!e.altKey) return
            if (e.key === 'ArrowUp') {
              e.preventDefault()
              onMoveByOffset(id, -1)
            } else if (e.key === 'ArrowDown') {
              e.preventDefault()
              onMoveByOffset(id, 1)
            }
          }}
        >
          <GripVertical className="h-3.5 w-3.5" aria-hidden />
        </button>
        <span id={reorderHelpId} className="sr-only">
          {pinnedSettingsCopy.reorderHelp}
        </span>
        <div className="min-w-0 flex-1">{children}</div>
      </div>
    </div>
  )
}

/**
 * Always-open Pinned group at the top of the settings panel (PS.3 FR-8–FR-14).
 */
export function PinnedSettingsGroup({ pins, visiblePinned }: PinnedSettingsGroupProps) {
  const { matches } = useSettingsPanelContext()
  const headingId = useId()
  const capHelpId = 'pinned-settings-cap-help'
  const hostedIdsRef = useMemo(() => new Set<string>(), [])

  // Filter by search (FR-11) — parent usually already did this; double-guard.
  const items = useMemo(
    () => visiblePinned.filter((d) => matches(d.id)),
    [visiblePinned, matches],
  )

  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 6 } }),
    useSensor(TouchSensor, { activationConstraint: { delay: 200, tolerance: 8 } }),
    useSensor(KeyboardSensor, defaultKeyboardSensorOptions),
  )

  const setHostRef = useCallback(
    (id: string, el: HTMLDivElement | null) => {
      if (el) hostedIdsRef.add(id)
      else hostedIdsRef.delete(id)
      setPinHost(id, el)
    },
    [hostedIdsRef],
  )

  // Unregister hosts on unmount.
  useEffect(() => {
    return () => {
      for (const id of hostedIdsRef) {
        setPinHost(id, null)
      }
      hostedIdsRef.clear()
    }
  }, [hostedIdsRef])

  const handleDragEnd = useCallback(
    (event: DragEndEvent) => {
      const { active, over } = event
      if (!over || active.id === over.id) return
      const ids = items.map((d) => d.id)
      const oldIndex = ids.indexOf(String(active.id))
      const newIndex = ids.indexOf(String(over.id))
      if (oldIndex < 0 || newIndex < 0) return
      const reorderedVisible = arrayMove(ids, oldIndex, newIndex)
      const visibleSet = new Set(ids)
      let vi = 0
      const next = pins.keys.map((k) => {
        if (visibleSet.has(k)) {
          return reorderedVisible[vi++] ?? k
        }
        return k
      })
      while (vi < reorderedVisible.length) {
        next.push(reorderedVisible[vi++])
      }
      pins.reorder(next)
      const moved = items.find((d) => d.id === String(active.id))
      if (moved) {
        pins.announce(
          pinnedSettingsCopy.announceMoved(moved.label, newIndex + 1, reorderedVisible.length),
        )
      }
    },
    [items, pins],
  )

  // PS.4 suggestion strip replaces the PS.3 first-run hint when eligible (FR-2).
  if (pins.suggestionsEligible) {
    return <SuggestedPinsStrip pins={pins} />
  }

  // Fallback first-run hint when no curated suggestions resolve (PS.3 FR-20).
  if (pins.showFirstRunHint) {
    return (
      <div className="rounded-lg border border-dashed border-indigo-200/80 bg-indigo-50/40 px-3 py-2.5 dark:border-indigo-800/50 dark:bg-indigo-950/20">
        <div className="flex items-start justify-between gap-2">
          <p className="text-[12px] leading-relaxed text-fg-muted">
            {pinnedSettingsCopy.hint}
          </p>
          <button
            type="button"
            onClick={pins.dismissFirstRunHint}
            className="inline-flex h-6 w-6 shrink-0 items-center justify-center rounded-md text-fg-subtle hover:bg-white/80 hover:text-fg-muted dark:hover:bg-surface-overlay dark:hover:text-fg-default"
            aria-label={pinnedSettingsCopy.dismissHint}
          >
            <X className="h-3.5 w-3.5" aria-hidden />
          </button>
        </div>
      </div>
    )
  }

  if (!pins.enabled || pins.status === 'unavailable' || pins.status === 'loading') {
    return null
  }

  if (items.length === 0) {
    // Live region still mounted so pin announcements work from home sections.
    return (
      <div role="status" aria-live="polite" className="sr-only">
        {pins.liveMessage}
      </div>
    )
  }

  return (
    <section
      aria-labelledby={headingId}
      className="overflow-hidden rounded-lg border border-indigo-200/70 bg-surface-raised motion-safe:animate-in motion-safe:fade-in motion-safe:duration-200 dark:border-indigo-900/40/20"
    >
      <div className="flex items-center justify-between gap-2 border-b border-indigo-100/80 px-3 py-2 dark:border-indigo-900/30">
        <h3
          id={headingId}
          className="inline-flex items-center gap-2 text-[13px] font-medium text-fg-default"
        >
          {pinnedSettingsCopy.title}
          <span
            className="inline-flex h-5 min-w-5 items-center justify-center rounded-full bg-indigo-50 px-1.5 text-[11px] font-semibold text-accent-fg dark:bg-indigo-950/60 dark:text-indigo-300"
            aria-label={`${items.length}`}
          >
            {items.length}
          </span>
        </h3>
      </div>
      {/* Cap explanation target for disabled pin toggles (FR-5). */}
      <p id={capHelpId} className="sr-only">
        {pinnedSettingsCopy.capReached(pins.uiCap)}
      </p>
      <div className="px-2 py-1">
        <DndContext sensors={sensors} collisionDetection={closestCenter} onDragEnd={handleDragEnd}>
          <SortableContext items={items.map((d) => d.id)} strategy={verticalListSortingStrategy}>
            {items.map((d) => (
              <SortablePinSlot
                key={d.id}
                id={d.id}
                label={d.label}
                onMoveByOffset={pins.moveByOffset}
              >
                <div
                  ref={(el) => setHostRef(d.id, el)}
                  className="min-w-0"
                  data-pin-host={d.id}
                />
              </SortablePinSlot>
            ))}
          </SortableContext>
        </DndContext>
      </div>
      <div role="status" aria-live="polite" className="sr-only">
        {pins.liveMessage}
      </div>
    </section>
  )
}

/** Muted section-level hint: "N pinned to top" (FR-7). */
export function PinnedSectionHint({ count }: { count: number }) {
  if (count <= 0) return null
  return (
    <p className="pb-1 text-[11px] leading-snug text-fg-subtle">
      {pinnedSettingsCopy.sectionHint(count)}
    </p>
  )
}
