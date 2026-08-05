/**
 * Exponential backoff for long-lived app WebSockets (notifications, mailbox).
 * Caps delay so a dead origin (e.g. Cloudflare 504) does not reconnect every 2s forever.
 */
const BASE_MS = 1000
const MAX_MS = 60_000

/** Delay before reconnect attempt `attempt` (0-based). */
export function wsReconnectDelayMs(attempt: number): number {
  const exp = Math.min(Math.max(0, attempt), 6)
  const base = Math.min(MAX_MS, BASE_MS * 2 ** exp)
  // 50%–100% jitter to avoid thundering herds across tabs.
  return Math.floor(base * (0.5 + Math.random() * 0.5))
}
