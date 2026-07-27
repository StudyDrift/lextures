import { describe, expect, it, vi } from 'vitest'
import {
  allPlaced,
  cancelGrab,
  createInitialEngineState,
  drop,
  moveTarget,
  pickUp,
  placeViaPointer,
  shuffleStable,
  tapItemOrTarget,
  trayItemIds,
  type PlacementEngineOptions,
} from '../index'

function opts(overrides: Partial<PlacementEngineOptions> = {}): PlacementEngineOptions {
  const announce = vi.fn()
  return {
    mode: 'categorize',
    itemIds: ['a', 'b', 'c'],
    bucketIds: ['x', 'y'],
    lockedItemIds: [],
    announce,
    labels: {
      pickedUp: (l) => `picked:${l}`,
      cancelled: (l) => `cancel:${l}`,
      droppedInBucket: (l, b) => `drop:${l}:${b}`,
      droppedAtPosition: (l, p) => `pos:${l}:${p}`,
      returnedToTray: (l) => `tray:${l}`,
      locked: (l) => `locked:${l}`,
      targetBucket: (b) => `tgt:${b}`,
      targetPosition: (p) => `tpos:${p}`,
      targetTray: () => 'tgt:tray',
    },
    itemLabel: (id) => id,
    bucketLabel: (id) => id,
    ...overrides,
  }
}

describe('placement-engine', () => {
  it('pick up, move, drop categorize via keyboard path', () => {
    const o = opts()
    let s = createInitialEngineState('categorize', o.itemIds)
    s = pickUp(s, o, 'a')
    expect(s.grabbedId).toBe('a')
    s = moveTarget(s, o, 1)
    expect(s.target).toEqual({ kind: 'bucket', bucketId: 'x' })
    s = drop(s, o)
    expect(s.grabbedId).toBeNull()
    expect((s.placement as Record<string, string | null>).a).toBe('x')
    expect(trayItemIds('categorize', o.itemIds, s.placement)).toEqual(['b', 'c'])
  })

  it('cancel grab with Esc semantics', () => {
    const o = opts()
    let s = createInitialEngineState('categorize', o.itemIds)
    s = pickUp(s, o, 'b')
    s = cancelGrab(s, o)
    expect(s.grabbedId).toBeNull()
  })

  it('tap-to-place path', () => {
    const o = opts()
    let s = createInitialEngineState('categorize', o.itemIds)
    s = tapItemOrTarget(s, o, { type: 'item', id: 'a' })
    s = tapItemOrTarget(s, o, { type: 'bucket', id: 'y' })
    expect((s.placement as Record<string, string | null>).a).toBe('y')
  })

  it('respects locked items', () => {
    const o = opts({ lockedItemIds: ['a'] })
    let s = createInitialEngineState('categorize', o.itemIds, { a: 'x', b: null, c: null })
    s = pickUp(s, o, 'a')
    expect(s.grabbedId).toBeNull()
    s = placeViaPointer(s, o, 'a', { kind: 'bucket', bucketId: 'y' })
    expect((s.placement as Record<string, string | null>).a).toBe('x')
  })

  it('order mode drop at position', () => {
    const o = opts({ mode: 'order', bucketIds: undefined })
    let s = createInitialEngineState('order', o.itemIds)
    s = placeViaPointer(s, o, 'b', { kind: 'position', index: 0 })
    s = placeViaPointer(s, o, 'a', { kind: 'position', index: 0 })
    s = placeViaPointer(s, o, 'c', { kind: 'position', index: 2 })
    expect(s.placement).toEqual(['a', 'b', 'c'])
    expect(allPlaced('order', o.itemIds, s.placement)).toBe(true)
  })

  it('shuffleStable is deterministic', () => {
    const items = [
      { id: '1' },
      { id: '2' },
      { id: '3' },
      { id: '4' },
      { id: '5' },
      { id: '6' },
      { id: '7' },
      { id: '8' },
    ]
    expect(shuffleStable(items, 'seed-a').map((i) => i.id)).toEqual(
      shuffleStable(items, 'seed-a').map((i) => i.id),
    )
    expect(shuffleStable(items, 'enrollment-1').map((i) => i.id)).not.toEqual(
      shuffleStable(items, 'enrollment-2').map((i) => i.id),
    )
  })
})
