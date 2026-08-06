import { useEffect, useId, useMemo, useState, type KeyboardEvent } from 'react'
import type { ContentToolRendererProps } from '../../host/runtime-contract'
import {
  allPlaced,
  createInitialEngineState,
  drop,
  moveTarget,
  pickUp,
  placeViaPointer,
  cancelGrab,
  shuffleStable,
  tapItemOrTarget,
  trayItemIds,
  type PlacementEngineOptions,
  type PlacementEngineState,
} from '../../shared/placement-engine'
import { CategorizeBoard } from './categorize-board'
import { SortItemChip } from './item-chip'
import { OrderList } from './order-list'
import {
  prefersReducedMotion,
  type CheckResult,
  type SortBucket,
  type SortItem,
} from './types'

export default function SortSequenceRenderer({
  instanceId,
  config,
  state,
  readOnly,
  save,
  runAction,
  t,
  announce,
}: ContentToolRendererProps) {
  const liveHelpId = useId()
  const mode = config.mode === 'order' ? 'order' : 'categorize'
  const prompt = typeof config.prompt === 'string' ? config.prompt : ''
  const items = Array.isArray(config.items) ? (config.items as SortItem[]).slice(0, 30) : []
  const buckets = Array.isArray(config.buckets) ? (config.buckets as SortBucket[]).slice(0, 6) : []
  const shuffleItems = config.shuffleItems !== false
  const showPerItem = config.showPerItemCorrectness !== false
  const itemIds = useMemo(() => items.map((i) => i.id), [items])
  const bucketIds = useMemo(() => buckets.map((b) => b.id), [buckets])

  const lockedItemIds = Array.isArray(state.lockedItemIds)
    ? (state.lockedItemIds as string[])
    : []
  const attempts = Array.isArray(state.attempts) ? (state.attempts as unknown[]) : []
  const lastPerItem =
    state.lastPerItem && typeof state.lastPerItem === 'object'
      ? (state.lastPerItem as Record<string, boolean>)
      : {}

  const trayOrderStored = Array.isArray(state.trayOrder) ? (state.trayOrder as string[]) : null
  const orderedItems = useMemo(() => {
    if (!shuffleItems) return items
    if (trayOrderStored && trayOrderStored.length === items.length) {
      const byId = new Map(items.map((i) => [i.id, i]))
      const ordered = trayOrderStored.map((id) => byId.get(id)).filter(Boolean) as SortItem[]
      if (ordered.length === items.length) return ordered
    }
    return shuffleStable(items, `${instanceId}:tray`)
  }, [items, shuffleItems, trayOrderStored, instanceId])

  useEffect(() => {
    if (!shuffleItems || trayOrderStored) return
    const order = orderedItems.map((i) => i.id)
    void save({
      v: 1,
      trayOrder: order,
      placement:
        state.placement ??
        (mode === 'order' ? [] : Object.fromEntries(itemIds.map((id) => [id, null]))),
      attempts: state.attempts ?? [],
      lockedItemIds: state.lockedItemIds ?? [],
    })
    // Persist tray order once per enrollment.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [shuffleItems, trayOrderStored, orderedItems])

  const existingPlacement = useMemo((): PlacementEngineState['placement'] => {
    if (mode === 'order') {
      return Array.isArray(state.placement) ? (state.placement as string[]) : []
    }
    if (state.placement && typeof state.placement === 'object' && !Array.isArray(state.placement)) {
      return state.placement as Record<string, string | null>
    }
    return Object.fromEntries(itemIds.map((id) => [id, null]))
  }, [mode, state.placement, itemIds])

  const placementKey = JSON.stringify(existingPlacement)
  const itemIdsKey = JSON.stringify(itemIds)

  const [engine, setEngine] = useState<PlacementEngineState>(() =>
    createInitialEngineState(mode, itemIds, existingPlacement),
  )
  const [busy, setBusy] = useState(false)
  const [checkResult, setCheckResult] = useState<CheckResult | null>(null)
  const [dragItemId, setDragItemId] = useState<string | null>(null)
  const [helpOpen, setHelpOpen] = useState(false)
  const reducedMotion = prefersReducedMotion()

  useEffect(() => {
    setEngine(createInitialEngineState(mode, itemIds, existingPlacement))
  }, [mode, placementKey, itemIdsKey, itemIds, existingPlacement])

  const itemLabel = (id: string) => items.find((i) => i.id === id)?.text ?? id
  const bucketLabel = (id: string) => buckets.find((b) => b.id === id)?.label ?? id

  const labels: PlacementEngineOptions['labels'] = {
    pickedUp: (l) => t('contentTools.tools.sort_sequence.announce.pickedUp', { item: l }),
    cancelled: (l) => t('contentTools.tools.sort_sequence.announce.cancelled', { item: l }),
    droppedInBucket: (l, b, index, total) =>
      t('contentTools.tools.sort_sequence.announce.droppedInBucket', {
        item: l,
        bucket: b,
        index,
        total,
      }),
    droppedAtPosition: (l, position, total) =>
      t('contentTools.tools.sort_sequence.announce.droppedAtPosition', {
        item: l,
        position,
        total,
      }),
    returnedToTray: (l) =>
      t('contentTools.tools.sort_sequence.announce.returnedToTray', { item: l }),
    locked: (l) => t('contentTools.tools.sort_sequence.announce.locked', { item: l }),
    targetBucket: (b, index, total, count) =>
      t('contentTools.tools.sort_sequence.announce.targetBucket', {
        bucket: b,
        index,
        total,
        count,
      }),
    targetPosition: (position, total) =>
      t('contentTools.tools.sort_sequence.announce.targetPosition', { position, total }),
    targetTray: () => t('contentTools.tools.sort_sequence.announce.targetTray'),
  }

  const engineOpts = (): PlacementEngineOptions => ({
    mode,
    itemIds,
    bucketIds,
    lockedItemIds,
    announce,
    labels,
    itemLabel,
    bucketLabel,
  })

  function persistPlacement(next: PlacementEngineState['placement']) {
    void save({
      v: 1,
      placement: next,
      attempts: state.attempts ?? [],
      lockedItemIds: state.lockedItemIds ?? [],
      trayOrder: state.trayOrder ?? orderedItems.map((i) => i.id),
      ...(state.lastPerItem ? { lastPerItem: state.lastPerItem } : {}),
      ...(state.completedAt ? { completedAt: state.completedAt } : {}),
    })
  }

  function applyEngine(next: PlacementEngineState) {
    setEngine(next)
    if (next.placement !== engine.placement) {
      persistPlacement(next.placement)
    }
  }

  function onItemKeyDown(e: KeyboardEvent, itemId: string) {
    if (readOnly || lockedItemIds.includes(itemId)) return
    if (e.key === 'Enter' || e.key === ' ') {
      e.preventDefault()
      if (engine.grabbedId === itemId) {
        applyEngine(drop(engine, engineOpts()))
      } else if (!engine.grabbedId) {
        applyEngine(pickUp(engine, engineOpts(), itemId))
      } else {
        applyEngine(tapItemOrTarget(engine, engineOpts(), { type: 'item', id: itemId }))
      }
      return
    }
    if (!engine.grabbedId) return
    if (e.key === 'ArrowRight' || e.key === 'ArrowDown') {
      e.preventDefault()
      applyEngine(moveTarget(engine, engineOpts(), 1))
    } else if (e.key === 'ArrowLeft' || e.key === 'ArrowUp') {
      e.preventDefault()
      applyEngine(moveTarget(engine, engineOpts(), -1))
    } else if (e.key === 'Escape') {
      e.preventDefault()
      applyEngine(cancelGrab(engine, engineOpts()))
    }
  }

  async function onCheck() {
    if (readOnly || busy) return
    if (!allPlaced(mode, itemIds, engine.placement)) return
    setBusy(true)
    try {
      const raw = await runAction('check', { placement: engine.placement })
      const result =
        raw && typeof raw === 'object' ? (raw as CheckResult) : ({ error: 'error' } as CheckResult)
      setCheckResult(result)
      if (result.error) {
        announce(result.message || result.error)
      } else {
        announce(
          t('contentTools.tools.sort_sequence.checkAnnounce', {
            score: Math.round(result.scorePct ?? 0),
          }),
        )
      }
    } catch {
      setCheckResult({ error: 'error', message: t('contentTools.runtime.retry') })
    } finally {
      setBusy(false)
    }
  }

  async function onTryAgain() {
    if (readOnly || busy) return
    setBusy(true)
    try {
      await runAction('reset_attempt', {})
      setCheckResult(null)
      announce(t('contentTools.tools.sort_sequence.tryAgainAnnounce'))
    } catch {
      announce(t('contentTools.runtime.retry'))
    } finally {
      setBusy(false)
    }
  }

  const trayIds = trayItemIds(
    mode,
    orderedItems.map((i) => i.id),
    engine.placement,
  )
  const canCheck = allPlaced(mode, itemIds, engine.placement) && !readOnly
  const attemptsLeft =
    checkResult?.attemptsRemaining ??
    (typeof config.attempts === 'number' ? Math.max(0, config.attempts - attempts.length) : null)
  const exhausted = attemptsLeft === 0
  const dragClass = reducedMotion ? '' : 'motion-safe:transition-transform motion-safe:duration-150'

  function renderChip(item: SortItem, opts?: { inList?: boolean }) {
    const locked = lockedItemIds.includes(item.id)
    const grabbed = engine.grabbedId === item.id
    const correctness = showPerItem
      ? (checkResult?.perItem?.[item.id]?.correct ?? lastPerItem[item.id])
      : undefined
    const feedback = showPerItem ? checkResult?.perItem?.[item.id]?.feedback : undefined
    return (
      <SortItemChip
        key={item.id}
        item={item}
        mode={mode}
        locked={locked}
        grabbed={grabbed}
        readOnly={readOnly}
        correctness={correctness}
        feedback={feedback}
        dragClass={dragClass}
        inList={opts?.inList}
        t={t}
        onClick={() => {
          if (readOnly || locked) return
          applyEngine(tapItemOrTarget(engine, engineOpts(), { type: 'item', id: item.id }))
        }}
        onKeyDown={(e) => onItemKeyDown(e, item.id)}
        onDragStart={(e) => {
          if (locked || readOnly) return
          setDragItemId(item.id)
          e.dataTransfer.setData('text/plain', item.id)
          e.dataTransfer.effectAllowed = 'move'
        }}
        onDragEnd={() => setDragItemId(null)}
      />
    )
  }

  return (
    <div
      className="space-y-4"
      data-content-tool="sort_sequence"
      data-testid="sort-sequence"
      data-mode={mode}
      aria-describedby={liveHelpId}
    >
      <div className="flex items-start justify-between gap-2">
        <p className="text-sm font-semibold text-fg-default">{prompt}</p>
        <button
          type="button"
          className="shrink-0 text-xs text-sky-700 underline dark:text-sky-300"
          onClick={() => setHelpOpen((v) => !v)}
          aria-expanded={helpOpen}
        >
          {t('contentTools.tools.sort_sequence.keyboardHelp')}
        </button>
      </div>
      {helpOpen ? (
        <p
          id={liveHelpId}
          className="rounded bg-surface-base px-3 py-2 text-xs text-fg-muted dark:bg-surface-overlay dark:text-fg-default"
        >
          {t('contentTools.tools.sort_sequence.keyboardHelpBody')}
        </p>
      ) : (
        <span id={liveHelpId} className="sr-only">
          {t('contentTools.tools.sort_sequence.keyboardHelpBody')}
        </span>
      )}

      <section
        aria-label={t('contentTools.tools.sort_sequence.tray')}
        data-testid="sort-tray"
        onDragOver={(e) => {
          if (dragItemId) e.preventDefault()
        }}
        onDrop={(e) => {
          e.preventDefault()
          const id = e.dataTransfer.getData('text/plain') || dragItemId
          if (!id) return
          applyEngine(placeViaPointer(engine, engineOpts(), id, { kind: 'tray' }))
          setDragItemId(null)
        }}
        onClick={() => {
          if (engine.grabbedId) {
            applyEngine(tapItemOrTarget(engine, engineOpts(), { type: 'tray' }))
          }
        }}
        className={`min-h-12 rounded border border-dashed border-border-strong p-2 dark:border-border-default ${ engine.target?.kind === 'tray' && engine.grabbedId ? 'ring-2 ring-sky-500' : '' }`}
      >
        <p className="mb-1 text-xs font-medium text-fg-muted">
          {t('contentTools.tools.sort_sequence.tray')} ({trayIds.length})
        </p>
        <div className="flex flex-wrap gap-2">
          {trayIds.map((id) => {
            const item = orderedItems.find((i) => i.id === id)
            return item ? renderChip(item) : null
          })}
        </div>
      </section>

      {mode === 'categorize' ? (
        <CategorizeBoard
          buckets={buckets}
          orderedItems={orderedItems}
          placement={engine.placement}
          grabbedId={engine.grabbedId}
          target={engine.target}
          dragItemId={dragItemId}
          setDragItemId={setDragItemId}
          t={t}
          onBucketActivate={(bucketId) =>
            applyEngine(tapItemOrTarget(engine, engineOpts(), { type: 'bucket', id: bucketId }))
          }
          onDropItem={(id, bucketId) =>
            applyEngine(
              placeViaPointer(engine, engineOpts(), id, { kind: 'bucket', bucketId }),
            )
          }
          renderChip={renderChip}
        />
      ) : (
        <OrderList
          orderedItems={orderedItems}
          placement={Array.isArray(engine.placement) ? engine.placement : []}
          grabbedId={engine.grabbedId}
          target={engine.target}
          lockedItemIds={lockedItemIds}
          readOnly={readOnly}
          dragItemId={dragItemId}
          setDragItemId={setDragItemId}
          t={t}
          onDropAt={(id, index) =>
            applyEngine(placeViaPointer(engine, engineOpts(), id, { kind: 'position', index }))
          }
          onMove={(id, index) =>
            applyEngine(placeViaPointer(engine, engineOpts(), id, { kind: 'position', index }))
          }
          onEndActivate={() => {
            const len = Array.isArray(engine.placement) ? engine.placement.length : 0
            applyEngine(tapItemOrTarget(engine, engineOpts(), { type: 'position', index: len }))
          }}
          renderChip={renderChip}
        />
      )}

      <div className="flex flex-wrap items-center gap-2">
        <button
          type="button"
          data-testid="sort-check"
          className="rounded bg-sky-700 px-3 py-1.5 text-sm font-medium text-white disabled:opacity-50"
          disabled={!canCheck || busy || exhausted}
          onClick={() => void onCheck()}
        >
          {busy
            ? t('contentTools.tools.sort_sequence.checking')
            : t('contentTools.tools.sort_sequence.check')}
        </button>
        {attempts.length > 0 && !exhausted ? (
          <button
            type="button"
            data-testid="sort-try-again"
            className="rounded border border-border-strong px-3 py-1.5 text-sm dark:border-border-default"
            disabled={busy || readOnly}
            onClick={() => void onTryAgain()}
          >
            {t('contentTools.tools.sort_sequence.tryAgain')}
          </button>
        ) : null}
        {typeof checkResult?.scorePct === 'number' ? (
          <span className="text-sm text-fg-default" data-testid="sort-score">
            {t('contentTools.tools.sort_sequence.score', {
              score: Math.round(checkResult.scorePct),
            })}
          </span>
        ) : null}
        {attemptsLeft != null && attemptsLeft >= 0 ? (
          <span className="text-xs text-fg-muted">
            {t('contentTools.tools.sort_sequence.attemptsLeft', { count: attemptsLeft })}
          </span>
        ) : null}
        {exhausted ? (
          <span className="text-sm text-amber-800 dark:text-amber-200">
            {t('contentTools.tools.sort_sequence.exhausted')}
          </span>
        ) : null}
      </div>
      {checkResult?.error ? (
        <p className="text-sm text-rose-600" role="alert">
          {checkResult.message || checkResult.error}
        </p>
      ) : null}
    </div>
  )
}
