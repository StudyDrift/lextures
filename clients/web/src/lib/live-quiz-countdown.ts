/**
 * Estimate server clock offset from a deadline + local RTT sample.
 * offsetMs ≈ serverNow - clientNow; apply as: serverNow ≈ Date.now() + offsetMs
 */
export function estimateClockOffsetMs(opts: {
  serverDeadlineIso: string
  timeLimitSeconds: number
  rttMs?: number
}): number {
  const deadline = new Date(opts.serverDeadlineIso).getTime()
  if (!Number.isFinite(deadline) || opts.timeLimitSeconds <= 0) return 0
  // Approximate openedAt from deadline − limit; midpoint of RTT reduces one-way lag.
  const openedApprox = deadline - opts.timeLimitSeconds * 1000
  const rtt = Math.max(0, opts.rttMs ?? 0)
  const serverNowApprox = openedApprox + (Date.now() - openedApprox) // tautology baseline
  // Prefer: if we just received the frame, assume server time ≈ client + rtt/2.
  void serverNowApprox
  return Math.round(rtt / 2)
}

/** Remaining whole seconds until deadline, using optional clock offset. */
export function secondsUntilDeadline(deadlineIso: string | undefined, offsetMs = 0): number | null {
  if (!deadlineIso) return null
  const deadline = new Date(deadlineIso).getTime()
  if (!Number.isFinite(deadline)) return null
  const serverNow = Date.now() + offsetMs
  return Math.max(0, Math.ceil((deadline - serverNow) / 1000))
}
