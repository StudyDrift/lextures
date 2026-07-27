import {
  checkpointRequired,
  isCheckpointDone,
  type Checkpoint,
  type CheckpointAnswer,
} from './types'

export const CHECKPOINT_TOLERANCE_SEC = 0.25

/** Find the next unanswered checkpoint at or before currentTime (within tolerance). */
export function findDueCheckpoint(
  checkpoints: Checkpoint[],
  answers: Record<string, CheckpointAnswer>,
  currentTime: number,
  alreadyPromptedIds: Set<string>,
): Checkpoint | null {
  const sorted = [...checkpoints].sort((a, b) => a.atSec - b.atSec)
  for (const cp of sorted) {
    if (alreadyPromptedIds.has(cp.id)) continue
    if (isCheckpointDone(answers, cp)) continue
    if (currentTime + CHECKPOINT_TOLERANCE_SEC >= cp.atSec && currentTime + 2 >= cp.atSec) {
      // Fire when playback reaches/passes the timestamp (within a small forward window).
      if (currentTime >= cp.atSec - CHECKPOINT_TOLERANCE_SEC) {
        return cp
      }
    }
  }
  return null
}

export function earliestUnansweredRequiredSec(
  checkpoints: Checkpoint[],
  answers: Record<string, CheckpointAnswer>,
): number | null {
  let earliest: number | null = null
  for (const cp of checkpoints) {
    if (!checkpointRequired(cp)) continue
    if (isCheckpointDone(answers, cp)) continue
    if (earliest == null || cp.atSec < earliest) earliest = cp.atSec
  }
  return earliest
}

/** Clamp seeks past unanswered required checkpoints when preventSkip is on. */
export function clampSeekTime(
  preventSkip: boolean,
  checkpoints: Checkpoint[],
  answers: Record<string, CheckpointAnswer>,
  targetSec: number,
): { time: number; clamped: boolean } {
  if (!preventSkip) return { time: targetSec, clamped: false }
  const limit = earliestUnansweredRequiredSec(checkpoints, answers)
  if (limit == null) return { time: targetSec, clamped: false }
  if (targetSec > limit + 0.05) return { time: limit, clamped: true }
  return { time: targetSec, clamped: false }
}

/** Merge a played span into coarse ≥5s segments (client-side preview of server normalize). */
export function mergeLocalSegments(
  existing: [number, number][],
  start: number,
  end: number,
  granularity = 5,
): [number, number][] {
  if (end <= start) return existing
  const floor = (v: number) => Math.floor(Math.max(0, v) / granularity) * granularity
  const ceil = (v: number) => {
    const f = floor(v)
    return v > f ? f + granularity : f
  }
  let s = floor(start)
  let e = ceil(end)
  if (e <= s) e = s + granularity
  const all = [...existing, [s, e] as [number, number]].sort((a, b) => a[0] - b[0] || a[1] - b[1])
  const merged: [number, number][] = []
  for (const seg of all) {
    const last = merged[merged.length - 1]
    if (!last || seg[0] > last[1]) {
      merged.push([seg[0], seg[1]])
    } else if (seg[1] > last[1]) {
      last[1] = seg[1]
    }
  }
  return merged
}
