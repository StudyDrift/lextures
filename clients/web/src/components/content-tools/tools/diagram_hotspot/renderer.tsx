import { useEffect, useId, useMemo, useRef, useState, type KeyboardEvent } from 'react'
import type { ContentToolRendererProps } from '../../host/runtime-contract'
import {
  allPlaced,
  cancelGrab,
  createInitialEngineState,
  drop,
  moveTarget,
  pickUp,
  placeViaPointer,
  tapItemOrTarget,
  trayItemIds,
  type PlacementEngineOptions,
  type PlacementEngineState,
} from '../../shared/placement-engine'
import { hitTestRegions, pointerToNormalized } from '../../shared/region-geometry'
import { DiagramBoard } from './diagram-board'
import { ListModeView } from './list-mode'
import { LabelTray } from './label-tray'
import { ZoomControls } from './zoom-controls'
import { DiagramActionBar } from './action-bar'
import { HotspotPromptBar } from './hotspot-prompt-bar'
import { diagramPlacementLabels } from './placement-labels'
import {
  prefersReducedMotion,
  regionPositionLabel,
  type CheckResult,
  type DiagramImage,
  type DiagramLabel,
  type DiagramPrompt,
  type DiagramRegion,
} from './types'

export default function DiagramHotspotRenderer({
  config,
  state,
  readOnly,
  save,
  runAction,
  t,
  announce,
}: ContentToolRendererProps) {
  const liveHelpId = useId()
  const mode = config.mode === 'hotspot' ? 'hotspot' : 'label'
  const prompt = typeof config.prompt === 'string' ? config.prompt : ''
  const image = (config.image ?? {}) as Partial<DiagramImage>
  const labels = useMemo(
    () => (Array.isArray(config.labels) ? (config.labels as DiagramLabel[]).slice(0, 40) : []),
    [config.labels],
  )
  const prompts = useMemo(
    () => (Array.isArray(config.prompts) ? (config.prompts as DiagramPrompt[]).slice(0, 20) : []),
    [config.prompts],
  )
  const regions = useMemo(
    () => (Array.isArray(config.regions) ? (config.regions as DiagramRegion[]).slice(0, 40) : []),
    [config.regions],
  )
  const showPerItem = config.showPerItemCorrectness !== false
  const outlinesPref =
    config.showRegionOutlines === 'always' || config.showRegionOutlines === 'after_check'
      ? config.showRegionOutlines
      : 'on_focus'

  const itemIds = useMemo(
    () => (mode === 'hotspot' ? prompts.map((p) => p.id) : labels.map((l) => l.id)),
    [mode, prompts, labels],
  )
  const regionIds = useMemo(() => regions.map((r) => r.id), [regions])
  const lockedIds = Array.isArray(state.lockedIds) ? (state.lockedIds as string[]) : []
  const attempts = Array.isArray(state.attempts) ? (state.attempts as unknown[]) : []
  const lastPerItem =
    state.lastPerItem && typeof state.lastPerItem === 'object'
      ? (state.lastPerItem as Record<string, boolean>)
      : {}

  const existingPlacement = useMemo((): Record<string, string | null> => {
    if (state.assignments && typeof state.assignments === 'object' && !Array.isArray(state.assignments)) {
      return state.assignments as Record<string, string | null>
    }
    return Object.fromEntries(itemIds.map((id) => [id, null]))
  }, [state.assignments, itemIds])

  const placementKey = JSON.stringify(existingPlacement)
  const itemIdsKey = JSON.stringify(itemIds)

  const [engine, setEngine] = useState<PlacementEngineState>(() =>
    createInitialEngineState('categorize', itemIds, existingPlacement),
  )
  const [busy, setBusy] = useState(false)
  const [checkResult, setCheckResult] = useState<CheckResult | null>(null)
  const [listMode, setListMode] = useState(Boolean(state.usedListMode))
  const [usedListMode, setUsedListMode] = useState(Boolean(state.usedListMode))
  const [imageFailed, setImageFailed] = useState(false)
  const [zoom, setZoom] = useState(1)
  const [pan, setPan] = useState({ x: 0, y: 0 })
  const [helpOpen, setHelpOpen] = useState(false)
  const [activePromptIndex, setActivePromptIndex] = useState(0)
  const surfaceRef = useRef<HTMLDivElement | null>(null)
  const reducedMotion = prefersReducedMotion()

  useEffect(() => {
    setEngine(createInitialEngineState('categorize', itemIds, existingPlacement))
  }, [placementKey, itemIdsKey, itemIds, existingPlacement])

  useEffect(() => {
    if (imageFailed && !listMode) {
      setListMode(true)
      setUsedListMode(true)
      announce(t('contentTools.tools.diagram_hotspot.imageFailedAnnounce'))
    }
  }, [imageFailed, listMode, announce, t])

  const itemLabel = (id: string) =>
    mode === 'hotspot'
      ? (prompts.find((p) => p.id === id)?.text ?? id)
      : (labels.find((l) => l.id === id)?.text ?? id)
  const bucketLabel = (id: string) => regions.find((r) => r.id === id)?.label ?? id

  const labelsI18n = diagramPlacementLabels(t)

  const engineOpts = (): PlacementEngineOptions => ({
    mode: 'categorize',
    itemIds,
    bucketIds: regionIds,
    lockedItemIds: lockedIds,
    announce,
    labels: labelsI18n,
    itemLabel,
    bucketLabel,
  })

  function persistAssignments(
    next: Record<string, string | null>,
    extras?: { usedListMode?: boolean },
  ) {
    void save({
      v: 1,
      assignments: next,
      attempts: state.attempts ?? [],
      lockedIds: state.lockedIds ?? [],
      usedListMode: extras?.usedListMode ?? usedListMode ?? state.usedListMode ?? false,
      ...(state.lastPerItem ? { lastPerItem: state.lastPerItem } : {}),
      ...(state.completedAt ? { completedAt: state.completedAt } : {}),
    })
  }

  function applyEngine(next: PlacementEngineState) {
    setEngine(next)
    if (next.placement !== engine.placement && !Array.isArray(next.placement)) {
      persistAssignments(next.placement)
    }
  }

  function setAssignment(itemId: string, regionId: string | null, fromList = false) {
    if (readOnly || lockedIds.includes(itemId)) return
    const map = !Array.isArray(engine.placement)
      ? { ...engine.placement }
      : Object.fromEntries(itemIds.map((id) => [id, null as string | null]))
    map[itemId] = regionId
    const next: PlacementEngineState = { grabbedId: null, target: null, placement: map }
    setEngine(next)
    if (fromList) {
      setUsedListMode(true)
      persistAssignments(map, { usedListMode: true })
    } else {
      persistAssignments(map)
    }
  }

  const placementMap = !Array.isArray(engine.placement)
    ? engine.placement
    : Object.fromEntries(itemIds.map((id) => [id, null as string | null]))

  const itemByRegion: Record<string, string> = {}
  for (const [itemId, regionId] of Object.entries(placementMap)) {
    if (regionId) itemByRegion[regionId] = itemLabel(itemId)
  }

  const focusedRegionId =
    engine.target?.kind === 'bucket' ? engine.target.bucketId : null
  const activePromptId = mode === 'hotspot' ? prompts[activePromptIndex]?.id : null
  const selectedRegionId =
    mode === 'hotspot' && activePromptId ? placementMap[activePromptId] : null

  const showOutlines =
    outlinesPref === 'always' ||
    (outlinesPref === 'after_check' && attempts.length > 0) ||
    Boolean(engine.grabbedId)

  function onItemKeyDown(e: KeyboardEvent, itemId: string) {
    if (readOnly || lockedIds.includes(itemId) || listMode) return
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
      const next = moveTarget(engine, engineOpts(), 1)
      applyEngine(next)
      if (next.target?.kind === 'bucket') {
        const idx = regionIds.indexOf(next.target.bucketId)
        const region = regions[idx]
        if (region) {
          announce(
            t('contentTools.tools.diagram_hotspot.announce.cycleRegion', {
              position: regionPositionLabel(region, idx, regions.length),
              label: region.label,
            }),
          )
        }
      }
    } else if (e.key === 'ArrowLeft' || e.key === 'ArrowUp') {
      e.preventDefault()
      applyEngine(moveTarget(engine, engineOpts(), -1))
    } else if (e.key === 'Escape') {
      e.preventDefault()
      applyEngine(cancelGrab(engine, engineOpts()))
    }
  }

  function assignFromPointer(clientX: number, clientY: number) {
    if (readOnly || listMode || imageFailed) return
    const el = surfaceRef.current
    if (!el || !image.naturalWidth || !image.naturalHeight) return
    const norm = pointerToNormalized(
      clientX,
      clientY,
      el,
      image.naturalWidth,
      image.naturalHeight,
      zoom,
      pan.x,
      pan.y,
    )
    if (!norm) return
    const hit = hitTestRegions(regions, norm[0], norm[1])
    if (!hit) return
    if (mode === 'hotspot') {
      if (!activePromptId || lockedIds.includes(activePromptId)) return
      setAssignment(activePromptId, hit.id)
      announce(
        t('contentTools.tools.diagram_hotspot.announce.selectedRegion', {
          region: hit.label,
        }),
      )
      return
    }
    if (engine.grabbedId) {
      applyEngine(
        placeViaPointer(engine, engineOpts(), engine.grabbedId, {
          kind: 'bucket',
          bucketId: hit.id,
        }),
      )
    }
  }

  async function onCheck() {
    if (readOnly || busy) return
    if (!allPlaced('categorize', itemIds, placementMap)) return
    setBusy(true)
    try {
      const raw = await runAction('check', {
        assignments: placementMap,
        usedListMode,
      })
      const result =
        raw && typeof raw === 'object' ? (raw as CheckResult) : ({ error: 'error' } as CheckResult)
      setCheckResult(result)
      if (result.error) {
        announce(result.message || result.error)
      } else {
        announce(
          t('contentTools.tools.diagram_hotspot.checkAnnounce', {
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
      announce(t('contentTools.tools.diagram_hotspot.tryAgainAnnounce'))
    } catch {
      announce(t('contentTools.runtime.retry'))
    } finally {
      setBusy(false)
    }
  }

  const trayIds = trayItemIds('categorize', itemIds, placementMap)
  const canCheck = allPlaced('categorize', itemIds, placementMap) && !readOnly
  const attemptsLeft =
    checkResult?.attemptsRemaining ??
    (typeof config.attempts === 'number' ? Math.max(0, config.attempts - attempts.length) : null)
  const exhausted = attemptsLeft === 0

  return (
    <div
      className="space-y-4"
      data-content-tool="diagram_hotspot"
      data-testid="diagram-hotspot"
      data-mode={mode}
      data-view={listMode ? 'list' : 'spatial'}
      aria-describedby={liveHelpId}
    >
      <div className="flex flex-wrap items-start justify-between gap-2">
        <p className="text-sm font-semibold text-slate-800 dark:text-neutral-100">{prompt}</p>
        <div className="flex flex-wrap gap-2">
          <button
            type="button"
            data-testid="diagram-toggle-list"
            className="rounded border border-slate-300 px-2 py-1 text-xs dark:border-neutral-600"
            aria-pressed={listMode}
            onClick={() => {
              const next = !listMode
              setListMode(next)
              if (next) {
                setUsedListMode(true)
                persistAssignments(placementMap, { usedListMode: true })
              }
            }}
          >
            {listMode
              ? t('contentTools.tools.diagram_hotspot.spatialView')
              : t('contentTools.tools.diagram_hotspot.listView')}
          </button>
          <button
            type="button"
            className="text-xs text-sky-700 underline dark:text-sky-300"
            onClick={() => setHelpOpen((v) => !v)}
            aria-expanded={helpOpen}
          >
            {t('contentTools.tools.diagram_hotspot.keyboardHelp')}
          </button>
        </div>
      </div>

      {helpOpen ? (
        <p
          id={liveHelpId}
          className="rounded bg-slate-50 px-3 py-2 text-xs text-slate-700 dark:bg-neutral-800 dark:text-neutral-200"
        >
          {t('contentTools.tools.diagram_hotspot.keyboardHelpBody')}
        </p>
      ) : (
        <span id={liveHelpId} className="sr-only">
          {t('contentTools.tools.diagram_hotspot.keyboardHelpBody')}
        </span>
      )}

      {mode === 'hotspot' && !listMode ? (
        <HotspotPromptBar
          prompts={prompts}
          activePromptIndex={activePromptIndex}
          selectedRegionId={selectedRegionId}
          selectedRegionLabel={selectedRegionId ? bucketLabel(selectedRegionId) : ''}
          t={t}
          onPrev={() => setActivePromptIndex((i) => Math.max(0, i - 1))}
          onNext={() => setActivePromptIndex((i) => Math.min(prompts.length - 1, i + 1))}
        />
      ) : null}

      {listMode || imageFailed ? (
        <ListModeView
          mode={mode}
          labels={labels}
          prompts={prompts}
          regions={regions}
          assignments={placementMap}
          lockedIds={lockedIds}
          readOnly={readOnly}
          t={t}
          onAssign={(itemId, regionId) => setAssignment(itemId, regionId, true)}
        />
      ) : (
        <>
          <ZoomControls
            zoom={zoom}
            t={t}
            onZoomOut={() => setZoom((z) => Math.max(1, Math.round((z - 0.5) * 10) / 10))}
            onZoomIn={() => setZoom((z) => Math.min(4, Math.round((z + 0.5) * 10) / 10))}
            onReset={() => {
              setZoom(1)
              setPan({ x: 0, y: 0 })
            }}
            onPan={(dx, dy) => setPan((p) => ({ x: p.x + dx, y: p.y + dy }))}
          />

          <DiagramBoard
            imageUrl={image.url ?? ''}
            imageAlt={image.alt ?? ''}
            naturalWidth={image.naturalWidth ?? 800}
            naturalHeight={image.naturalHeight ?? 600}
            regions={regions}
            showOutlines={showOutlines}
            focusedRegionId={focusedRegionId}
            selectedRegionId={selectedRegionId}
            assignments={placementMap}
            itemByRegion={itemByRegion}
            zoom={zoom}
            pan={pan}
            reducedMotion={reducedMotion}
            imageFailed={imageFailed}
            surfaceRef={surfaceRef}
            onImageError={() => setImageFailed(true)}
            onPointerSelect={assignFromPointer}
            onRegionActivate={(regionId) => {
              if (mode === 'hotspot') {
                if (!activePromptId) return
                setAssignment(activePromptId, regionId)
                announce(
                  t('contentTools.tools.diagram_hotspot.announce.selectedRegion', {
                    region: bucketLabel(regionId),
                  }),
                )
                return
              }
              if (engine.grabbedId) {
                applyEngine(
                  tapItemOrTarget(engine, engineOpts(), { type: 'bucket', id: regionId }),
                )
              }
            }}
            onRegionKeyDown={(e, regionId) => {
              if (e.key === 'Enter' || e.key === ' ') {
                e.preventDefault()
                if (mode === 'hotspot' && activePromptId) {
                  setAssignment(activePromptId, regionId)
                } else if (engine.grabbedId) {
                  applyEngine(
                    tapItemOrTarget(engine, engineOpts(), { type: 'bucket', id: regionId }),
                  )
                }
              }
            }}
          />

          {mode === 'label' ? (
            <LabelTray
              trayIds={trayIds}
              labels={labels}
              grabbedId={engine.grabbedId}
              lockedIds={lockedIds}
              readOnly={readOnly}
              showPerItem={showPerItem}
              checkResult={checkResult}
              lastPerItem={lastPerItem}
              t={t}
              onActivate={(id) =>
                applyEngine(tapItemOrTarget(engine, engineOpts(), { type: 'item', id }))
              }
              onKeyDown={onItemKeyDown}
              onPickUp={(id) => {
                if (!engine.grabbedId) {
                  applyEngine(pickUp(engine, engineOpts(), id))
                }
              }}
            />
          ) : null}
        </>
      )}

      <DiagramActionBar
        canCheck={canCheck}
        busy={busy}
        exhausted={exhausted}
        readOnly={readOnly}
        attemptCount={attempts.length}
        attemptsLeft={attemptsLeft}
        checkResult={checkResult}
        t={t}
        onCheck={() => void onCheck()}
        onTryAgain={() => void onTryAgain()}
      />
    </div>
  )
}
