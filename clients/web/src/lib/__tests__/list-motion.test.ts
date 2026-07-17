import { describe, expect, it } from 'vitest'
import {
  computeListTransitions,
  diffListKeys,
  listDragStyle,
  listDropAnimation,
  listPhaseClassName,
  LIST_DRAG_LIFT_SCALE,
  LIST_MOTION_MAX_CONCURRENT,
} from '../list-motion'
import { durations } from '../motion'

describe('AN.4 list motion pure helpers', () => {
  it('diffListKeys reports enter, exit, move, steady (FR-1 / FR-2 / FR-3)', () => {
    const diff = diffListKeys(['a', 'b', 'c'], ['a', 'd', 'b'])
    expect(diff.entered).toEqual(['d'])
    expect(diff.exited).toEqual(['c'])
    expect(diff.moved).toEqual(['b'])
    expect(diff.steady).toEqual(['a'])
  })

  it('computeListTransitions animates enter/exit/move when enabled', () => {
    const items = computeListTransitions({
      prevKeys: ['a', 'b', 'c'],
      nextKeys: ['a', 'd', 'b'],
      exitingKeys: ['c'],
      enabled: true,
      reduceMotion: false,
    })
    const byKey = Object.fromEntries(items.map((i) => [i.key, i]))
    expect(byKey.d?.phase).toBe('enter')
    expect(byKey.d?.animate).toBe(true)
    expect(byKey.c?.phase).toBe('exit')
    expect(byKey.c?.animate).toBe(true)
    expect(byKey.b?.phase).toBe('move')
    expect(byKey.a?.phase).toBe('steady')
  })

  it('reduced motion: enter/exit opacity-only, move not animated (FR-8 / AC-4)', () => {
    const items = computeListTransitions({
      prevKeys: ['a', 'b'],
      nextKeys: ['b', 'a', 'c'],
      exitingKeys: [],
      enabled: true,
      reduceMotion: true,
    })
    const byKey = Object.fromEntries(items.map((i) => [i.key, i]))
    expect(byKey.c?.phase).toBe('enter')
    expect(byKey.c?.animate).toBe(true)
    expect(byKey.a?.phase).toBe('move')
    expect(byKey.a?.animate).toBe(false)
    expect(listPhaseClassName('enter', true, true)).toContain('enter-fade')
    expect(listPhaseClassName('move', false, true)).not.toContain('move')
  })

  it('caps concurrent animations (FR-9)', () => {
    const prev = Array.from({ length: 40 }, (_, i) => `p${i}`)
    const next = Array.from({ length: 40 }, (_, i) => `n${i}`)
    const items = computeListTransitions({
      prevKeys: prev,
      nextKeys: next,
      enabled: true,
      reduceMotion: false,
      maxConcurrent: 5,
    })
    const animated = items.filter((i) => i.animate)
    expect(animated.length).toBeLessThanOrEqual(5)
    expect(LIST_MOTION_MAX_CONCURRENT).toBeGreaterThanOrEqual(8)
  })

  it('virtualization: only visible keys animate (FR-9 / AC-5)', () => {
    const items = computeListTransitions({
      prevKeys: ['a', 'b', 'c'],
      nextKeys: ['a', 'b', 'd'],
      exitingKeys: ['c'],
      enabled: true,
      reduceMotion: false,
      visibleKeys: new Set(['a', 'b']),
    })
    const byKey = Object.fromEntries(items.map((i) => [i.key, i]))
    expect(byKey.d?.animate).toBe(false)
    expect(byKey.c?.animate).toBe(false)
  })

  it('append mode only enters new keys (FR-5)', () => {
    const items = computeListTransitions({
      prevKeys: ['a', 'b'],
      nextKeys: ['a', 'b', 'c', 'd'],
      mode: 'append',
      enabled: true,
      reduceMotion: false,
    })
    const byKey = Object.fromEntries(items.map((i) => [i.key, i]))
    expect(byKey.a?.phase).toBe('steady')
    expect(byKey.c?.phase).toBe('enter')
    expect(byKey.d?.phase).toBe('enter')
  })

  it('kill-switch disables animation', () => {
    const items = computeListTransitions({
      prevKeys: ['a'],
      nextKeys: ['a', 'b'],
      enabled: false,
      reduceMotion: false,
    })
    expect(items.every((i) => !i.animate)).toBe(true)
  })

  it('listDragStyle lifts with scale when dragging; reduced skips scale (FR-4 / FR-8)', () => {
    const full = listDragStyle({
      transform: { x: 10, y: 0, scaleX: 1, scaleY: 1 },
      isDragging: true,
      reduceMotion: false,
      enabled: true,
    })
    expect(String(full.transform)).toContain(String(LIST_DRAG_LIFT_SCALE))
    expect(full.boxShadow).toBeTruthy()

    const reduced = listDragStyle({
      transform: { x: 10, y: 0, scaleX: 1, scaleY: 1 },
      isDragging: true,
      reduceMotion: true,
      enabled: true,
    })
    expect(String(reduced.transform)).not.toContain(String(LIST_DRAG_LIFT_SCALE))
    expect(reduced.boxShadow).toBeTruthy()
  })

  it('listDropAnimation uses bubble duration; reduced → instant', () => {
    const full = listDropAnimation({ reduceMotion: false, enabled: true })
    expect(full).toMatchObject({ duration: durations.deliberate })
    const reduced = listDropAnimation({ reduceMotion: true, enabled: true })
    expect(reduced).toMatchObject({ duration: durations.instant })
    const off = listDropAnimation({ enabled: false })
    expect(off).toMatchObject({ duration: 0 })
  })
})
