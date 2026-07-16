import { useEffect, useState } from 'react'
import type { Awareness } from 'y-protocols/awareness'
import type { BoardPresenceUser } from '../../lib/boards-realtime'

type Props = {
  awareness: Awareness
  /** When false, cursors are not rendered (non-spatial layouts). */
  enabled: boolean
}

type PresenceState = {
  user?: BoardPresenceUser
  cursor?: { x: number; y: number } | null
}

export function BoardLiveCursors({ awareness, enabled }: Props) {
  const [states, setStates] = useState<Map<number, PresenceState>>(new Map())
  const localClient = awareness.clientID

  useEffect(() => {
    if (!enabled) return
    const onChange = () => {
      setStates(new Map(awareness.getStates() as Map<number, PresenceState>))
    }
    awareness.on('change', onChange)
    onChange()
    return () => {
      awareness.off('change', onChange)
    }
  }, [awareness, enabled])

  if (!enabled) return null

  const cursors: { clientId: number; user: BoardPresenceUser; x: number; y: number }[] = []
  states.forEach((state, clientId) => {
    if (clientId === localClient) return
    if (!state.user || !state.cursor) return
    cursors.push({
      clientId,
      user: state.user,
      x: state.cursor.x,
      y: state.cursor.y,
    })
  })

  return (
    <div className="pointer-events-none absolute inset-0 z-20 overflow-hidden" aria-hidden="true">
      {cursors.map(({ clientId, user, x, y }) => (
        <div
          key={clientId}
          className="absolute"
          style={{ left: x, top: y, transform: 'translate(-2px, -2px)' }}
        >
          <svg width="16" height="20" viewBox="0 0 16 20" fill="none">
            <path
              d="M1 1L1 15L5.5 11.5L8.5 18L11 16.5L8 10L14 10L1 1Z"
              fill={user.color}
              stroke="white"
              strokeWidth="1"
            />
          </svg>
          <span
            className="mt-0.5 ms-3 inline-block max-w-[8rem] truncate rounded px-1.5 py-0.5 text-[10px] font-medium text-white"
            style={{ backgroundColor: user.color }}
          >
            {user.displayName}
          </span>
        </div>
      ))}
    </div>
  )
}
