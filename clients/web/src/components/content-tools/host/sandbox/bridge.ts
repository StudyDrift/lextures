import {
  BRIDGE_MAX_MESSAGE_BYTES,
  BridgeRateLimiter,
  isBridgeFromTool,
  measureMessageBytes,
  type BridgeFromTool,
  type BridgeToTool,
} from '@lextures/tool-sdk'

export type HostBridgeHandlers = {
  onReady: (contract: string) => void
  onSave: (state: unknown, revision: number) => void | Promise<void>
  onRunAction: (id: string, action: string, input: unknown) => void | Promise<void>
  onResize: (height: number) => void
  onAnnounce: (message: string, assertive?: boolean) => void
  onInvalid: (reason: string) => void
}

/**
 * Host-side postMessage bridge: origin-checked, schema-validated, size/rate capped (FR-11).
 */
export function createHostBridge(opts: {
  toolId: string
  /** Expected event.origin; use 'null' for opaque srcDoc sandboxes. */
  expectedOrigin: string
  source: Window | null
  handlers: HostBridgeHandlers
  onMetric?: (type: string, outcome: string) => void
}) {
  const limiter = new BridgeRateLimiter()

  function post(msg: BridgeToTool) {
    if (!opts.source) return
    const targetOrigin = opts.expectedOrigin === 'null' ? '*' : opts.expectedOrigin
    opts.source.postMessage(msg, targetOrigin)
  }

  function onMessage(event: MessageEvent) {
    if (opts.source && event.source !== opts.source) return
    // Opaque sandboxed iframes report origin "null".
    if (event.origin !== opts.expectedOrigin && !(opts.expectedOrigin === 'null' && event.origin === 'null')) {
      opts.handlers.onInvalid('origin')
      opts.onMetric?.('unknown', 'bad_origin')
      return
    }
    if (!limiter.allow()) {
      opts.handlers.onInvalid('rate')
      opts.onMetric?.('unknown', 'rate_limited')
      return
    }
    if (measureMessageBytes(event.data) > BRIDGE_MAX_MESSAGE_BYTES) {
      opts.handlers.onInvalid('size')
      opts.onMetric?.('unknown', 'oversized')
      return
    }
    if (!isBridgeFromTool(event.data)) {
      opts.handlers.onInvalid('schema')
      opts.onMetric?.('unknown', 'malformed')
      return
    }
    const msg = event.data as BridgeFromTool
    opts.onMetric?.(msg.t, 'ok')
    switch (msg.t) {
      case 'ready':
        opts.handlers.onReady(msg.contract)
        break
      case 'save':
        void opts.handlers.onSave(msg.state, msg.revision)
        break
      case 'runAction':
        void opts.handlers.onRunAction(msg.id, msg.action, msg.input)
        break
      case 'resize':
        opts.handlers.onResize(msg.height)
        break
      case 'announce':
        opts.handlers.onAnnounce(msg.message, msg.assertive)
        break
      default: {
        const _exhaustive: never = msg
        void _exhaustive
        opts.handlers.onInvalid('unknown_type')
      }
    }
  }

  window.addEventListener('message', onMessage)
  return {
    post,
    dispose: () => window.removeEventListener('message', onMessage),
  }
}

export function opaqueParticipantId(instanceId: string, enrollmentHint?: string): string {
  // Opaque per-instance participant id — not email, not user id (FR-12).
  const raw = `${instanceId}:${enrollmentHint ?? 'anon'}`
  let h = 0
  for (let i = 0; i < raw.length; i++) h = (Math.imul(31, h) + raw.charCodeAt(i)) | 0
  return `p_${(h >>> 0).toString(16)}`
}
