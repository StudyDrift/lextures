import { describe, expect, it } from 'vitest'
import {
  initialClickToMoveState,
  isValidClickToMoveTarget,
  reduceClickToMove,
} from '../click-to-move'

describe('reduceClickToMove', () => {
  it('enters picking-target on select-source and announces', () => {
    const { state, effect } = reduceClickToMove(initialClickToMoveState(), {
      type: 'select-source',
      id: 'a',
    })
    expect(state).toEqual({ mode: 'picking-target', sourceId: 'a' })
    expect(effect).toEqual({ type: 'announce', messageKey: 'enter' })
  })

  it('commits when a different target is selected', () => {
    const mid = reduceClickToMove(initialClickToMoveState(), {
      type: 'select-source',
      id: 'a',
    }).state
    const { state, effect } = reduceClickToMove(mid, { type: 'select-target', id: 'b' })
    expect(state).toEqual({ mode: 'idle' })
    expect(effect).toEqual({ type: 'commit', sourceId: 'a', targetId: 'b' })
  })

  it('cancels when source is re-selected or Escape path fires cancel', () => {
    const mid = reduceClickToMove(initialClickToMoveState(), {
      type: 'select-source',
      id: 'a',
    }).state
    const { state, effect } = reduceClickToMove(mid, { type: 'cancel' })
    expect(state).toEqual({ mode: 'idle' })
    expect(effect).toEqual({ type: 'announce', messageKey: 'cancel' })
  })

  it('marks only non-source ids as valid targets', () => {
    const mid = reduceClickToMove(initialClickToMoveState(), {
      type: 'select-source',
      id: 'a',
    }).state
    expect(isValidClickToMoveTarget(mid, 'a')).toBe(false)
    expect(isValidClickToMoveTarget(mid, 'b')).toBe(true)
    expect(isValidClickToMoveTarget(initialClickToMoveState(), 'b')).toBe(false)
  })
})
