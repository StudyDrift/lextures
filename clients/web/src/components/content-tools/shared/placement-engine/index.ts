/** Shared pick-up / drop placement model for CT.14 (reused by CT.15). */

export type PlacementTarget =
  | { kind: 'tray' }
  | { kind: 'bucket'; bucketId: string }
  | { kind: 'position'; index: number }

export type PlacementEngineMode = 'categorize' | 'order'

export type PlacementEngineState = {
  /** Item currently picked up (keyboard / tap-to-place), or null. */
  grabbedId: string | null
  /** Focused target while holding an item. */
  target: PlacementTarget | null
  /** Categorize: itemId → bucketId | null (tray). Order: ordered item ids (tray items omitted). */
  placement: Record<string, string | null> | string[]
}

export type PlacementEngineOptions = {
  mode: PlacementEngineMode
  itemIds: string[]
  bucketIds?: string[]
  lockedItemIds?: string[]
  announce: (message: string) => void
  labels: {
    pickedUp: (itemLabel: string) => string
    cancelled: (itemLabel: string) => string
    droppedInBucket: (itemLabel: string, bucketLabel: string, index: number, total: number) => string
    droppedAtPosition: (itemLabel: string, position: number, total: number) => string
    returnedToTray: (itemLabel: string) => string
    locked: (itemLabel: string) => string
    targetBucket: (bucketLabel: string, index: number, total: number, count: number) => string
    targetPosition: (position: number, total: number) => string
    targetTray: () => string
  }
  itemLabel: (id: string) => string
  bucketLabel: (id: string) => string
}

export function emptyPlacement(mode: PlacementEngineMode, itemIds: string[]): PlacementEngineState['placement'] {
  if (mode === 'order') return []
  const out: Record<string, string | null> = {}
  for (const id of itemIds) out[id] = null
  return out
}

export function isLocked(locked: string[] | undefined, id: string): boolean {
  return Boolean(locked?.includes(id))
}

export function createInitialEngineState(
  mode: PlacementEngineMode,
  itemIds: string[],
  existing?: PlacementEngineState['placement'],
): PlacementEngineState {
  return {
    grabbedId: null,
    target: null,
    placement: existing ?? emptyPlacement(mode, itemIds),
  }
}

export function trayItemIds(
  mode: PlacementEngineMode,
  itemIds: string[],
  placement: PlacementEngineState['placement'],
): string[] {
  if (mode === 'order') {
    const order = Array.isArray(placement) ? placement : []
    const placed = new Set(order)
    return itemIds.filter((id) => !placed.has(id))
  }
  const map = placement && !Array.isArray(placement) ? placement : {}
  return itemIds.filter((id) => map[id] == null)
}

export function itemsInBucket(
  placement: PlacementEngineState['placement'],
  bucketId: string,
): string[] {
  if (Array.isArray(placement)) return []
  return Object.entries(placement)
    .filter(([, b]) => b === bucketId)
    .map(([id]) => id)
}

export function pickUp(
  state: PlacementEngineState,
  opts: PlacementEngineOptions,
  itemId: string,
): PlacementEngineState {
  if (isLocked(opts.lockedItemIds, itemId)) {
    opts.announce(opts.labels.locked(opts.itemLabel(itemId)))
    return state
  }
  opts.announce(opts.labels.pickedUp(opts.itemLabel(itemId)))
  const target: PlacementTarget =
    opts.mode === 'order'
      ? { kind: 'position', index: Array.isArray(state.placement) ? state.placement.length : 0 }
      : { kind: 'tray' }
  return { ...state, grabbedId: itemId, target }
}

export function cancelGrab(state: PlacementEngineState, opts: PlacementEngineOptions): PlacementEngineState {
  if (!state.grabbedId) return state
  opts.announce(opts.labels.cancelled(opts.itemLabel(state.grabbedId)))
  return { ...state, grabbedId: null, target: null }
}

export function moveTarget(
  state: PlacementEngineState,
  opts: PlacementEngineOptions,
  direction: 1 | -1,
): PlacementEngineState {
  if (!state.grabbedId) return state
  if (opts.mode === 'order') {
    const total = opts.itemIds.length
    const cur = state.target?.kind === 'position' ? state.target.index : 0
    const next = Math.max(0, Math.min(total, cur + direction))
    opts.announce(opts.labels.targetPosition(next + 1, total + 1))
    return { ...state, target: { kind: 'position', index: next } }
  }
  const buckets = opts.bucketIds ?? []
  const targets: PlacementTarget[] = [{ kind: 'tray' }, ...buckets.map((id) => ({ kind: 'bucket' as const, bucketId: id }))]
  let idx = 0
  if (state.target?.kind === 'tray') idx = 0
  else if (state.target?.kind === 'bucket') {
    const bi = buckets.indexOf(state.target.bucketId)
    idx = bi >= 0 ? bi + 1 : 0
  }
  idx = (idx + direction + targets.length) % targets.length
  const next = targets[idx]!
  if (next.kind === 'tray') {
    opts.announce(opts.labels.targetTray())
  } else if (next.kind === 'bucket') {
    const bi = buckets.indexOf(next.bucketId)
    const count = itemsInBucket(state.placement, next.bucketId).length
    opts.announce(opts.labels.targetBucket(opts.bucketLabel(next.bucketId), bi + 1, buckets.length, count))
  }
  return { ...state, target: next }
}

export function drop(
  state: PlacementEngineState,
  opts: PlacementEngineOptions,
): PlacementEngineState {
  const itemId = state.grabbedId
  if (!itemId || !state.target) return state
  if (isLocked(opts.lockedItemIds, itemId)) {
    opts.announce(opts.labels.locked(opts.itemLabel(itemId)))
    return { ...state, grabbedId: null, target: null }
  }

  if (opts.mode === 'order') {
    const order = Array.isArray(state.placement) ? [...state.placement] : []
    const without = order.filter((id) => id !== itemId)
    const index = state.target.kind === 'position' ? state.target.index : without.length
    const clamped = Math.max(0, Math.min(without.length, index))
    without.splice(clamped, 0, itemId)
    opts.announce(opts.labels.droppedAtPosition(opts.itemLabel(itemId), clamped + 1, without.length))
    return { grabbedId: null, target: null, placement: without }
  }

  const map: Record<string, string | null> = {
    ...(typeof state.placement === 'object' && !Array.isArray(state.placement) ? state.placement : {}),
  }
  if (state.target.kind === 'tray') {
    map[itemId] = null
    opts.announce(opts.labels.returnedToTray(opts.itemLabel(itemId)))
  } else if (state.target.kind === 'bucket') {
    map[itemId] = state.target.bucketId
    const buckets = opts.bucketIds ?? []
    const bi = buckets.indexOf(state.target.bucketId)
    opts.announce(
      opts.labels.droppedInBucket(
        opts.itemLabel(itemId),
        opts.bucketLabel(state.target.bucketId),
        bi + 1,
        buckets.length,
      ),
    )
  }
  return { grabbedId: null, target: null, placement: map }
}

/** Tap-to-place: if nothing grabbed, pick up; if same item, cancel; else drop onto target. */
export function tapItemOrTarget(
  state: PlacementEngineState,
  opts: PlacementEngineOptions,
  hit: { type: 'item'; id: string } | { type: 'bucket'; id: string } | { type: 'tray' } | { type: 'position'; index: number },
): PlacementEngineState {
  if (!state.grabbedId) {
    if (hit.type === 'item') return pickUp(state, opts, hit.id)
    return state
  }
  if (hit.type === 'item' && hit.id === state.grabbedId) {
    return cancelGrab(state, opts)
  }
  let target: PlacementTarget
  switch (hit.type) {
    case 'tray':
      target = { kind: 'tray' }
      break
    case 'bucket':
      target = { kind: 'bucket', bucketId: hit.id }
      break
    case 'position':
      target = { kind: 'position', index: hit.index }
      break
    case 'item': {
      // Drop onto the bucket/position of the tapped item.
      if (opts.mode === 'order' && Array.isArray(state.placement)) {
        const idx = state.placement.indexOf(hit.id)
        target = { kind: 'position', index: idx >= 0 ? idx : state.placement.length }
      } else if (!Array.isArray(state.placement)) {
        const bucket = state.placement[hit.id]
        target = bucket ? { kind: 'bucket', bucketId: bucket } : { kind: 'tray' }
      } else {
        target = { kind: 'tray' }
      }
      break
    }
    default: {
      const _exhaustive: never = hit
      return _exhaustive
    }
  }
  return drop({ ...state, target }, opts)
}

export function placeViaPointer(
  state: PlacementEngineState,
  opts: PlacementEngineOptions,
  itemId: string,
  target: PlacementTarget,
): PlacementEngineState {
  if (isLocked(opts.lockedItemIds, itemId)) {
    opts.announce(opts.labels.locked(opts.itemLabel(itemId)))
    return state
  }
  return drop({ ...state, grabbedId: itemId, target }, opts)
}

export function allPlaced(
  mode: PlacementEngineMode,
  itemIds: string[],
  placement: PlacementEngineState['placement'],
): boolean {
  if (itemIds.length === 0) return false
  if (mode === 'order') {
    const order = Array.isArray(placement) ? placement : []
    return itemIds.every((id) => order.includes(id))
  }
  const map = !Array.isArray(placement) ? placement : {}
  return itemIds.every((id) => typeof map[id] === 'string' && map[id])
}

export function shuffleStable<T extends { id: string }>(items: T[], seed: string): T[] {
  if (items.length <= 1) return items
  const out = [...items]
  let h = 0
  for (let i = 0; i < seed.length; i++) h = (h * 31 + seed.charCodeAt(i)) >>> 0
  for (let i = out.length - 1; i > 0; i--) {
    h = (h * 1103515245 + 12345) >>> 0
    const j = h % (i + 1)
    ;[out[i], out[j]] = [out[j]!, out[i]!]
  }
  return out
}
