/** Versioned postMessage bridge protocol (CT.5 FR-11). */

export const BRIDGE_VERSION = 1 as const

export const BRIDGE_MAX_MESSAGE_BYTES = 64 * 1024
export const BRIDGE_MAX_MESSAGES_PER_SEC = 20

export type BridgeToTool =
  | {
      t: 'init'
      v: 1
      instanceId: string
      config: unknown
      state: unknown
      revision: number
      locale: string
      dir: 'ltr' | 'rtl'
      readOnly: boolean
      participantId: string
    }
  | { t: 'stateAccepted'; v: 1; revision: number }
  | { t: 'actionResult'; v: 1; id: string; result: unknown }
  | { t: 'error'; v: 1; id?: string; code: string; message: string }

export type BridgeFromTool =
  | { t: 'ready'; v: 1; contract: string }
  | { t: 'save'; v: 1; state: unknown; revision: number }
  | { t: 'runAction'; v: 1; id: string; action: string; input: unknown }
  | { t: 'resize'; v: 1; height: number }
  | { t: 'announce'; v: 1; message: string; assertive?: boolean }

export type BridgeMessage = BridgeToTool | BridgeFromTool

const FROM_TOOL_TYPES = new Set(['ready', 'save', 'runAction', 'resize', 'announce'])
const TO_TOOL_TYPES = new Set(['init', 'stateAccepted', 'actionResult', 'error'])

export function isBridgeFromTool(msg: unknown): msg is BridgeFromTool {
  if (!msg || typeof msg !== 'object') return false
  const m = msg as { t?: unknown; v?: unknown }
  return m.v === 1 && typeof m.t === 'string' && FROM_TOOL_TYPES.has(m.t)
}

export function isBridgeToTool(msg: unknown): msg is BridgeToTool {
  if (!msg || typeof msg !== 'object') return false
  const m = msg as { t?: unknown; v?: unknown }
  return m.v === 1 && typeof m.t === 'string' && TO_TOOL_TYPES.has(m.t)
}

export function measureMessageBytes(msg: unknown): number {
  try {
    return new TextEncoder().encode(JSON.stringify(msg)).length
  } catch {
    return Number.POSITIVE_INFINITY
  }
}

/** Sliding-window rate limiter for host-side bridge ingress. */
export class BridgeRateLimiter {
  private timestamps: number[] = []
  private maxPerSec: number

  constructor(maxPerSec = BRIDGE_MAX_MESSAGES_PER_SEC) {
    this.maxPerSec = maxPerSec
  }

  allow(now = Date.now()): boolean {
    const cut = now - 1000
    this.timestamps = this.timestamps.filter((t) => t >= cut)
    if (this.timestamps.length >= this.maxPerSec) return false
    this.timestamps.push(now)
    return true
  }
}
