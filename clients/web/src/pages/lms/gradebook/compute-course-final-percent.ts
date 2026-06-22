const UNGROUPED = '__ungrouped__'

export type GradebookColumnForFinal = {
  id: string
  maxPoints: number | null
  assignmentGroupId?: string | null
  neverDrop?: boolean
  replaceWithFinal?: boolean
  dueAt?: string | null
}

export type AssignmentGroupWeight = {
  id: string
  weightPercent: number
  /** Plan 3.9 */
  dropLowest?: number
  dropHighest?: number
  replaceLowestWithFinal?: boolean
}

type GroupPolicy = { dropLowest: number; dropHighest: number; replaceLowestWithFinal: boolean }

type Scored = {
  id: string
  max: number
  earned: number
  pct: number
  canDrop: boolean
  isFinal: boolean
}

export type ComputeFinalOptions = {
  /** When `whatIf`, items with hypothetical overrides are included even if not yet due. */
  mode?: 'actual' | 'whatIf'
  /** Hypothetical score overrides (itemId → points). Client-only; never persisted. */
  whatIfOverrides?: Record<string, string>
  /** Held/unposted items — real scores are never merged into what-if calculations. */
  heldItemIds?: ReadonlySet<string>
  now?: Date | string | number
}

export type ScoreNeededResult =
  | { achievable: false; reason: string }
  | { achievable: true; scorePercent: number; itemIds: string[] }

function parseEarned(raw: string | undefined): number {
  const t = (raw ?? '').trim()
  if (!t) return 0
  const n = Number.parseFloat(t.replace(/,/g, ''))
  return Number.isFinite(n) ? n : 0
}

/**
 * Merge actual grades with what-if overrides. Held items never expose real scores.
 */
export function mergeGradesForWhatIf(
  actualGrades: Record<string, string>,
  overrides: Record<string, string>,
  heldItemIds: ReadonlySet<string>,
): Record<string, string> {
  const merged: Record<string, string> = { ...actualGrades }
  for (const id of heldItemIds) {
    delete merged[id]
  }
  for (const [id, val] of Object.entries(overrides)) {
    const t = val.trim()
    if (t === '') delete merged[id]
    else merged[id] = t
  }
  return merged
}

/**
 * Port of `server::services::grading::assignment_groups::compute_group_average_with_drops` (plan 3.9).
 * Returns effective earned/max for one assignment group and one student.
 */
export function groupEffectiveEarnedAndMax(
  policy: GroupPolicy,
  lines: { itemId: string; max: number; earned: number; neverDrop: boolean; isFinal: boolean }[],
): { effectiveEarned: number; effectiveMax: number; droppedIds: Set<string> } {
  const dropped = new Set<string>()
  if (lines.length === 0) {
    return { effectiveEarned: 0, effectiveMax: 0, droppedIds: dropped }
  }
  const rows: Scored[] = lines
    .map((l) => {
      const max = l.max > 0 && Number.isFinite(l.max) ? l.max : 0
      const earned = Math.max(0, l.earned)
      const pct = max > 0 ? earned / max : 0
      const isFinal = l.isFinal
      const canDrop = !l.neverDrop && !isFinal
      return {
        id: l.itemId,
        max,
        earned,
        pct: Number.isFinite(pct) ? pct : 0,
        canDrop,
        isFinal,
      }
    })
    .filter((r) => r.max > 0)

  rows.sort((a, b) => (a.pct !== b.pct ? a.pct - b.pct : a.id.localeCompare(b.id)))
  const work: Scored[] = rows.filter((r) => r.canDrop)

  const nLow = Math.max(0, policy.dropLowest)
  const nHigh = Math.max(0, policy.dropHighest)
  for (let i = 0; i < nLow; i++) {
    if (work.length === 0) break
    dropped.add(work.shift()!.id)
  }
  for (let i = 0; i < nHigh; i++) {
    if (work.length === 0) break
    dropped.add(work.pop()!.id)
  }

  let effectiveMax = 0
  let effectiveEarned = 0
  for (const r of rows) {
    if (dropped.has(r.id)) continue
    effectiveMax += r.max
    effectiveEarned += r.earned
  }

  if (policy.replaceLowestWithFinal) {
    const f = rows.find((r) => r.isFinal && !dropped.has(r.id))
    if (f && f.pct > 0) {
      const others = rows.filter((r) => !r.isFinal && !dropped.has(r.id))
      if (others.length > 0) {
        let t = others[0]!
        for (const r of others) {
          if (r.pct < t.pct) t = r
          else if (r.pct === t.pct && r.id < t.id) t = r
        }
        if (f.pct > t.pct + 1e-12) {
          effectiveEarned -= t.earned
          effectiveEarned += t.max * f.pct
        }
      }
    }
  }
  return { effectiveEarned, effectiveMax, droppedIds: dropped }
}

function shouldIncludeColumn(
  col: GradebookColumnForFinal,
  gradeStr: string | undefined,
  hasOverride: boolean,
  mode: 'actual' | 'whatIf',
  nowMs: number,
): boolean {
  if (mode === 'whatIf' && hasOverride) return true
  const hasGrade = typeof gradeStr === 'string' && gradeStr.trim() !== ''
  let isPastDue = false
  if (col.dueAt) {
    const d = new Date(col.dueAt)
    if (!Number.isNaN(d.getTime())) {
      isPastDue = d.getTime() < nowMs
    }
  }
  return hasGrade || isPastDue
}

function resolveOptions(
  nowOrOptions: Date | string | number | ComputeFinalOptions,
): Required<Pick<ComputeFinalOptions, 'mode' | 'whatIfOverrides' | 'heldItemIds' | 'now'>> {
  if (
    typeof nowOrOptions === 'object' &&
    !(nowOrOptions instanceof Date) &&
    ('mode' in nowOrOptions ||
      'whatIfOverrides' in nowOrOptions ||
      'heldItemIds' in nowOrOptions ||
      'now' in nowOrOptions)
  ) {
    const o = nowOrOptions as ComputeFinalOptions
    return {
      mode: o.mode ?? 'actual',
      whatIfOverrides: o.whatIfOverrides ?? {},
      heldItemIds: o.heldItemIds ?? new Set<string>(),
      now: o.now ?? new Date(),
    }
  }
  return {
    mode: 'actual',
    whatIfOverrides: {},
    heldItemIds: new Set<string>(),
    now: nowOrOptions as Date | string | number,
  }
}

/**
 * Course final as a percentage (0–100) with assignment-group drop / replace policy (3.9).
 * Ungrouped columns are summed without drops.
 *
 * Only includes assignments that (a) have a grade entered for the student, or (b) are past their
 * due date (missing work counts as 0 toward the average). Future/not-due assignments with no
 * grade are excluded from the denominator and numerator unless `mode` is `whatIf` and the item
 * has a hypothetical override.
 */
export function computeCourseFinalPercent(
  columns: GradebookColumnForFinal[],
  gradesByItemId: Record<string, string>,
  assignmentGroups: AssignmentGroupWeight[],
  excusedByItemId: Record<string, boolean> = {},
  nowOrOptions: Date | string | number | ComputeFinalOptions = new Date(),
): number | null {
  const { mode, whatIfOverrides, heldItemIds, now } = resolveOptions(nowOrOptions)
  const mergedGrades =
    mode === 'whatIf'
      ? mergeGradesForWhatIf(gradesByItemId, whatIfOverrides, heldItemIds)
      : gradesByItemId

  const settingsIds = new Set(assignmentGroups.map((g) => g.id))
  const polByG = new Map<string, GroupPolicy>()
  for (const g of assignmentGroups) {
    polByG.set(g.id, {
      dropLowest: g.dropLowest != null && g.dropLowest > 0 ? g.dropLowest : 0,
      dropHighest: g.dropHighest != null && g.dropHighest > 0 ? g.dropHighest : 0,
      replaceLowestWithFinal: g.replaceLowestWithFinal === true,
    })
  }

  const maxByBucket = new Map<string, number>()
  const earnedByBucket = new Map<string, number>()

  const byGroup: Map<
    string,
    { itemId: string; max: number; earned: number; neverDrop: boolean; isFinal: boolean }[]
  > = new Map()

  const nowDate = now instanceof Date ? now : new Date(now)
  const nowMs = nowDate.getTime()

  for (const col of columns) {
    const max = col.maxPoints
    if (max == null || max <= 0) continue
    if (excusedByItemId[col.id] === true) continue

    const hasOverride =
      mode === 'whatIf' && (whatIfOverrides[col.id] ?? '').trim() !== ''
    const gradeStr = mergedGrades[col.id]
    if (!shouldIncludeColumn(col, gradeStr, hasOverride, mode, nowMs)) continue

    const earned = parseEarned(gradeStr)
    const gid = col.assignmentGroupId?.trim()
    const bucket = gid && settingsIds.has(gid) ? gid : UNGROUPED
    const isFinal = col.replaceWithFinal === true
    const neverDrop = col.neverDrop === true

    if (bucket === UNGROUPED) {
      maxByBucket.set(bucket, (maxByBucket.get(bucket) ?? 0) + max)
      earnedByBucket.set(bucket, (earnedByBucket.get(bucket) ?? 0) + earned)
    } else {
      if (!byGroup.has(bucket)) byGroup.set(bucket, [])
      byGroup.get(bucket)!.push({
        itemId: col.id,
        max,
        earned,
        neverDrop,
        isFinal,
      })
    }
  }

  for (const [gid, lines] of byGroup) {
    const p = polByG.get(gid) ?? { dropLowest: 0, dropHighest: 0, replaceLowestWithFinal: false }
    const { effectiveEarned, effectiveMax } = groupEffectiveEarnedAndMax(p, lines)
    maxByBucket.set(gid, (maxByBucket.get(gid) ?? 0) + effectiveMax)
    earnedByBucket.set(gid, (earnedByBucket.get(gid) ?? 0) + effectiveEarned)
  }

  const totalMaxPoints = [...maxByBucket.values()].reduce((a, b) => a + b, 0)
  if (totalMaxPoints <= 0) return null

  const bucketsWithColumns = new Set(
    [...maxByBucket.entries()].filter(([, mx]) => mx > 0).map(([b]) => b),
  )
  if (bucketsWithColumns.size === 0) return null

  const configuredSum = assignmentGroups.reduce((acc, g) => {
    const w = Number.isFinite(g.weightPercent) && g.weightPercent > 0 ? g.weightPercent : 0
    return acc + w
  }, 0)
  const remainder = Math.max(0, 100 - configuredSum)

  let lostConfiguredWeight = 0
  for (const g of assignmentGroups) {
    const w = Number.isFinite(g.weightPercent) && g.weightPercent > 0 ? g.weightPercent : 0
    if (w <= 0) continue
    if (!bucketsWithColumns.has(g.id)) lostConfiguredWeight += w
  }

  const maxUngrouped = maxByBucket.get(UNGROUPED) ?? 0

  const rawWeight = new Map<string, number>()
  for (const g of assignmentGroups) {
    if (!bucketsWithColumns.has(g.id)) continue
    const w = Number.isFinite(g.weightPercent) && g.weightPercent > 0 ? g.weightPercent : 0
    if (w > 0) rawWeight.set(g.id, w)
  }

  if (bucketsWithColumns.has(UNGROUPED)) {
    let wU = remainder + lostConfiguredWeight
    if (wU <= 0 && maxUngrouped > 0 && totalMaxPoints > 0) {
      wU = (maxUngrouped / totalMaxPoints) * 100
    }
    rawWeight.set(UNGROUPED, (rawWeight.get(UNGROUPED) ?? 0) + wU)
  }

  const weightSum = [...rawWeight.values()].reduce((a, b) => a + b, 0)
  if (weightSum <= 0) {
    const earnedTotal = [...earnedByBucket.values()].reduce((a, b) => a + b, 0)
    return (earnedTotal / totalMaxPoints) * 100
  }

  let acc = 0
  for (const [bucket, rw] of rawWeight) {
    if (rw <= 0) continue
    const maxB = maxByBucket.get(bucket) ?? 0
    const earnedB = earnedByBucket.get(bucket) ?? 0
    const ratio = maxB > 0 ? earnedB / maxB : 0
    acc += ratio * (rw / weightSum)
  }

  return acc * 100
}

/** What-if projection using merged actual + hypothetical scores (plan 3.16). */
export function computeWhatIfFinalPercent(
  columns: GradebookColumnForFinal[],
  actualGrades: Record<string, string>,
  assignmentGroups: AssignmentGroupWeight[],
  excusedByItemId: Record<string, boolean>,
  whatIfOverrides: Record<string, string>,
  heldItemIds: ReadonlySet<string>,
  now: Date | string | number = new Date(),
): number | null {
  return computeCourseFinalPercent(columns, actualGrades, assignmentGroups, excusedByItemId, {
    mode: 'whatIf',
    whatIfOverrides,
    heldItemIds,
    now,
  })
}

/** Recompute which items are dropped under current grades/overrides (plan 3.9 + 3.16). */
export function computeDroppedGrades(
  columns: GradebookColumnForFinal[],
  gradesByItemId: Record<string, string>,
  assignmentGroups: AssignmentGroupWeight[],
  excusedByItemId: Record<string, boolean> = {},
  options: ComputeFinalOptions = {},
): Record<string, boolean> {
  const { mode, whatIfOverrides, heldItemIds, now } = resolveOptions(options)
  const mergedGrades =
    mode === 'whatIf'
      ? mergeGradesForWhatIf(gradesByItemId, whatIfOverrides, heldItemIds)
      : gradesByItemId

  const settingsIds = new Set(assignmentGroups.map((g) => g.id))
  const polByG = new Map<string, GroupPolicy>()
  for (const g of assignmentGroups) {
    polByG.set(g.id, {
      dropLowest: g.dropLowest != null && g.dropLowest > 0 ? g.dropLowest : 0,
      dropHighest: g.dropHighest != null && g.dropHighest > 0 ? g.dropHighest : 0,
      replaceLowestWithFinal: g.replaceLowestWithFinal === true,
    })
  }

  const byGroup: Map<
    string,
    { itemId: string; max: number; earned: number; neverDrop: boolean; isFinal: boolean }[]
  > = new Map()

  const nowDate = now instanceof Date ? now : new Date(now)
  const nowMs = nowDate.getTime()
  const dropped: Record<string, boolean> = {}

  for (const col of columns) {
    const max = col.maxPoints
    if (max == null || max <= 0) continue
    if (excusedByItemId[col.id] === true) continue

    const hasOverride =
      mode === 'whatIf' && (whatIfOverrides[col.id] ?? '').trim() !== ''
    const gradeStr = mergedGrades[col.id]
    if (!shouldIncludeColumn(col, gradeStr, hasOverride, mode, nowMs)) continue

    const earned = parseEarned(gradeStr)
    const gid = col.assignmentGroupId?.trim()
    const bucket = gid && settingsIds.has(gid) ? gid : UNGROUPED
    if (bucket === UNGROUPED) continue

    if (!byGroup.has(bucket)) byGroup.set(bucket, [])
    byGroup.get(bucket)!.push({
      itemId: col.id,
      max,
      earned,
      neverDrop: col.neverDrop === true,
      isFinal: col.replaceWithFinal === true,
    })
  }

  for (const [gid, lines] of byGroup) {
    const p = polByG.get(gid) ?? { dropLowest: 0, dropHighest: 0, replaceLowestWithFinal: false }
    const { droppedIds } = groupEffectiveEarnedAndMax(p, lines)
    for (const id of droppedIds) dropped[id] = true
  }

  return dropped
}

/** Items eligible for equal-score target calculation (ungraded, non-held, non-excused). */
export function remainingItemsForTarget(
  columns: GradebookColumnForFinal[],
  actualGrades: Record<string, string>,
  excusedByItemId: Record<string, boolean>,
  heldItemIds: ReadonlySet<string>,
  whatIfOverrides: Record<string, string>,
): GradebookColumnForFinal[] {
  return columns.filter((col) => {
    if (col.maxPoints == null || col.maxPoints <= 0) return false
    if (excusedByItemId[col.id] === true) return false
    if (heldItemIds.has(col.id)) return false
    const hasActual = (actualGrades[col.id] ?? '').trim() !== ''
    const hasOverride = (whatIfOverrides[col.id] ?? '').trim() !== ''
    return !hasActual && !hasOverride
  })
}

/**
 * Given a target course percentage, estimate the equal score (0–100%) needed on each remaining
 * ungraded item. Uses binary search over the shared grade-calculation engine.
 */
export function computeScoreNeededForTarget(
  targetPercent: number,
  columns: GradebookColumnForFinal[],
  actualGrades: Record<string, string>,
  assignmentGroups: AssignmentGroupWeight[],
  excusedByItemId: Record<string, boolean>,
  heldItemIds: ReadonlySet<string>,
  whatIfOverrides: Record<string, string>,
  now: Date | string | number = new Date(),
): ScoreNeededResult {
  if (!Number.isFinite(targetPercent)) {
    return { achievable: false, reason: 'Enter a valid target grade.' }
  }

  const remaining = remainingItemsForTarget(
    columns,
    actualGrades,
    excusedByItemId,
    heldItemIds,
    whatIfOverrides,
  )
  if (remaining.length === 0) {
    return { achievable: false, reason: 'No remaining ungraded items to model.' }
  }

  const baseOpts: ComputeFinalOptions = {
    mode: 'whatIf',
    whatIfOverrides,
    heldItemIds,
    now,
  }

  const current = computeCourseFinalPercent(
    columns,
    actualGrades,
    assignmentGroups,
    excusedByItemId,
    baseOpts,
  )
  if (current != null && current + 1e-9 >= targetPercent) {
    return { achievable: false, reason: 'You already meet or exceed this target with current grades.' }
  }

  const itemIds = remaining.map((c) => c.id)

  function finalWithEqualScore(scorePct: number): number | null {
    const trialOverrides = { ...whatIfOverrides }
    for (const col of remaining) {
      const max = col.maxPoints ?? 0
      trialOverrides[col.id] = String(Math.round((scorePct / 100) * max * 1000) / 1000)
    }
    return computeCourseFinalPercent(columns, actualGrades, assignmentGroups, excusedByItemId, {
      ...baseOpts,
      whatIfOverrides: trialOverrides,
    })
  }

  const at100 = finalWithEqualScore(100)
  if (at100 == null || at100 + 1e-9 < targetPercent) {
    return { achievable: false, reason: 'This target is not achievable even with 100% on remaining items.' }
  }

  let lo = 0
  let hi = 100
  for (let i = 0; i < 48; i++) {
    const mid = (lo + hi) / 2
    const pct = finalWithEqualScore(mid)
    if (pct != null && pct + 1e-9 >= targetPercent) hi = mid
    else lo = mid
  }

  return { achievable: true, scorePercent: Math.ceil(hi * 10) / 10, itemIds }
}

export function formatFinalPercent(pct: number | null): string {
  if (pct == null || !Number.isFinite(pct)) return '—'
  const rounded = Math.round(pct * 10) / 10
  return `${rounded}%`
}

/** Lightweight client-only usage metric (plan 3.16 observability). */
export function recordWhatIfSession(): void {
  try {
    sessionStorage.setItem('whatif_sessions', '1')
  } catch {
    // ignore storage errors
  }
}

export function hasWhatIfSession(): boolean {
  try {
    return sessionStorage.getItem('whatif_sessions') === '1'
  } catch {
    return false
  }
}
