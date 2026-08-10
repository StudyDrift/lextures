/**
 * Click-to-move state machine for single-pointer reorder (WCAG 2.2 SC 2.5.7).
 *
 * idle → selecting source → picking target → commit / cancel
 */

export type ClickToMoveState =
  | { mode: 'idle' }
  | { mode: 'source-selected'; sourceId: string }
  | { mode: 'picking-target'; sourceId: string }

export type ClickToMoveEvent =
  | { type: 'select-source'; id: string }
  | { type: 'select-target'; id: string }
  | { type: 'cancel' }
  | { type: 'reset' }

export type ClickToMoveEffect =
  | { type: 'none' }
  | { type: 'commit'; sourceId: string; targetId: string }
  | { type: 'announce'; messageKey: 'enter' | 'cancel' }

export function initialClickToMoveState(): ClickToMoveState {
  return { mode: 'idle' }
}

export function reduceClickToMove(
  state: ClickToMoveState,
  event: ClickToMoveEvent,
): { state: ClickToMoveState; effect: ClickToMoveEffect } {
  switch (event.type) {
    case 'reset':
    case 'cancel':
      if (state.mode === 'idle') return { state, effect: { type: 'none' } }
      return { state: { mode: 'idle' }, effect: { type: 'announce', messageKey: 'cancel' } }

    case 'select-source':
      if (state.mode !== 'idle' && state.sourceId === event.id) {
        return { state: { mode: 'idle' }, effect: { type: 'announce', messageKey: 'cancel' } }
      }
      return {
        state: { mode: 'picking-target', sourceId: event.id },
        effect: { type: 'announce', messageKey: 'enter' },
      }

    case 'select-target': {
      if (state.mode === 'idle') {
        return {
          state: { mode: 'picking-target', sourceId: event.id },
          effect: { type: 'announce', messageKey: 'enter' },
        }
      }
      if (state.sourceId === event.id) {
        return { state: { mode: 'idle' }, effect: { type: 'announce', messageKey: 'cancel' } }
      }
      return {
        state: { mode: 'idle' },
        effect: { type: 'commit', sourceId: state.sourceId, targetId: event.id },
      }
    }

    default:
      return { state, effect: { type: 'none' } }
  }
}

export function isClickToMoveActive(state: ClickToMoveState): boolean {
  return state.mode !== 'idle'
}

export function isValidClickToMoveTarget(state: ClickToMoveState, id: string): boolean {
  if (state.mode === 'idle') return false
  return state.sourceId !== id
}
