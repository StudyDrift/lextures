import Foundation

/// Full port of `clients/web/src/pages/lms/gradebook/compute-course-final-percent.ts`.
enum GradeCalculator {
    private static let ungrouped = "__ungrouped__"

    struct ColumnForFinal: Hashable {
        var id: String
        var maxPoints: Double?
        var assignmentGroupId: String?
        var neverDrop: Bool = false
        var replaceWithFinal: Bool = false
        var dueAt: String?
    }

    struct GroupWeight: Hashable {
        var id: String
        var weightPercent: Double
        var dropLowest: Int = 0
        var dropHighest: Int = 0
        var replaceLowestWithFinal: Bool = false
    }

    struct ComputeOptions {
        var mode: Mode = .actual
        var whatIfOverrides: [String: String] = [:]
        var heldItemIds: Set<String> = []
        var now: Date = Date()
    }

    enum Mode { case actual, whatIf }

    static func mergeGradesForWhatIf(
        actualGrades: [String: String],
        overrides: [String: String],
        heldItemIds: Set<String>
    ) -> [String: String] {
        var merged = actualGrades
        for id in heldItemIds { merged.removeValue(forKey: id) }
        for (id, val) in overrides {
            let trimmed = val.trimmingCharacters(in: .whitespacesAndNewlines)
            if trimmed.isEmpty { merged.removeValue(forKey: id) }
            else { merged[id] = trimmed }
        }
        return merged
    }

    static func groupEffectiveEarnedAndMax(
        policy: GroupWeight,
        lines: [(itemId: String, max: Double, earned: Double, neverDrop: Bool, isFinal: Bool)]
    ) -> (effectiveEarned: Double, effectiveMax: Double, droppedIds: Set<String>) {
        struct Scored {
            var id: String
            var max: Double
            var earned: Double
            var pct: Double
            var canDrop: Bool
            var isFinal: Bool
        }

        var dropped = Set<String>()
        guard !lines.isEmpty else { return (0, 0, dropped) }

        var rows: [Scored] = lines.map { line in
            let maxPoints = line.max > 0 && line.max.isFinite ? line.max : 0
            let earned = Swift.max(0, line.earned)
            let pct = maxPoints > 0 ? earned / maxPoints : 0
            let canDrop = !line.neverDrop && !line.isFinal
            return Scored(
                id: line.itemId,
                max: maxPoints,
                earned: earned,
                pct: pct.isFinite ? pct : 0,
                canDrop: canDrop,
                isFinal: line.isFinal
            )
        }.filter { $0.max > 0 }

        rows.sort { a, b in
            if a.pct != b.pct { return a.pct < b.pct }
            return a.id < b.id
        }

        var work = rows.filter(\.canDrop)
        for _ in 0 ..< max(0, policy.dropLowest) {
            guard !work.isEmpty else { break }
            dropped.insert(work.removeFirst().id)
        }
        for _ in 0 ..< max(0, policy.dropHighest) {
            guard !work.isEmpty else { break }
            dropped.insert(work.removeLast().id)
        }

        var effectiveMax = 0.0
        var effectiveEarned = 0.0
        for row in rows where !dropped.contains(row.id) {
            effectiveMax += row.max
            effectiveEarned += row.earned
        }

        if policy.replaceLowestWithFinal {
            if let finalRow = rows.first(where: { $0.isFinal && !dropped.contains($0.id) && $0.pct > 0 }) {
                let others = rows.filter { !$0.isFinal && !dropped.contains($0.id) }
                if let lowest = others.min(by: { a, b in
                    if a.pct != b.pct { return a.pct < b.pct }
                    return a.id < b.id
                }), finalRow.pct > lowest.pct + 1e-12 {
                    effectiveEarned -= lowest.earned
                    effectiveEarned += lowest.max * finalRow.pct
                }
            }
        }

        return (effectiveEarned, effectiveMax, dropped)
    }

    static func computeCourseFinalPercent(
        columns: [ColumnForFinal],
        gradesByItemId: [String: String],
        assignmentGroups: [GroupWeight],
        excusedByItemId: [String: Bool] = [:],
        options: ComputeOptions = ComputeOptions()
    ) -> Double? {
        let mergedGrades: [String: String]
        if options.mode == .whatIf {
            mergedGrades = mergeGradesForWhatIf(
                actualGrades: gradesByItemId,
                overrides: options.whatIfOverrides,
                heldItemIds: options.heldItemIds
            )
        } else {
            mergedGrades = gradesByItemId
        }

        let settingsIds = Set(assignmentGroups.map(\.id))
        var polByG: [String: GroupWeight] = [:]
        for group in assignmentGroups { polByG[group.id] = group }

        var maxByBucket: [String: Double] = [:]
        var earnedByBucket: [String: Double] = [:]
        var byGroup: [String: [(itemId: String, max: Double, earned: Double, neverDrop: Bool, isFinal: Bool)]] = [:]
        let nowMs = options.now.timeIntervalSince1970 * 1000

        for col in columns {
            guard let max = col.maxPoints, max > 0 else { continue }
            if excusedByItemId[col.id] == true { continue }

            let hasOverride = options.mode == .whatIf
                && !(options.whatIfOverrides[col.id] ?? "").trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
            let gradeStr = mergedGrades[col.id]
            guard shouldIncludeColumn(col, gradeStr: gradeStr, hasOverride: hasOverride, mode: options.mode, nowMs: nowMs) else {
                continue
            }

            let earned = parseEarned(gradeStr)
            let gid = col.assignmentGroupId?.trimmingCharacters(in: .whitespacesAndNewlines)
            let bucket = (gid.flatMap { settingsIds.contains($0) ? $0 : nil }) ?? ungrouped

            if bucket == ungrouped {
                maxByBucket[bucket, default: 0] += max
                earnedByBucket[bucket, default: 0] += earned
            } else {
                byGroup[bucket, default: []].append((
                    itemId: col.id,
                    max: max,
                    earned: earned,
                    neverDrop: col.neverDrop,
                    isFinal: col.replaceWithFinal
                ))
            }
        }

        for (gid, lines) in byGroup {
            let policy = polByG[gid] ?? GroupWeight(
                id: gid,
                weightPercent: 0,
                dropLowest: 0,
                dropHighest: 0,
                replaceLowestWithFinal: false
            )
            let result = groupEffectiveEarnedAndMax(policy: policy, lines: lines)
            maxByBucket[gid, default: 0] += result.effectiveMax
            earnedByBucket[gid, default: 0] += result.effectiveEarned
        }

        let totalMaxPoints = maxByBucket.values.reduce(0, +)
        guard totalMaxPoints > 0 else { return nil }

        let bucketsWithColumns = Set(maxByBucket.filter { $0.value > 0 }.map(\.key))
        guard !bucketsWithColumns.isEmpty else { return nil }

        let configuredSum = assignmentGroups.reduce(0.0) { acc, g in
            let w = g.weightPercent.isFinite && g.weightPercent > 0 ? g.weightPercent : 0
            return acc + w
        }
        let remainder = max(0, 100 - configuredSum)

        var lostConfiguredWeight = 0.0
        for group in assignmentGroups {
            let w = group.weightPercent.isFinite && group.weightPercent > 0 ? group.weightPercent : 0
            if w <= 0 { continue }
            if !bucketsWithColumns.contains(group.id) { lostConfiguredWeight += w }
        }

        let maxUngrouped = maxByBucket[ungrouped] ?? 0
        var rawWeight: [String: Double] = [:]
        for group in assignmentGroups where bucketsWithColumns.contains(group.id) {
            let w = group.weightPercent.isFinite && group.weightPercent > 0 ? group.weightPercent : 0
            if w > 0 { rawWeight[group.id] = w }
        }

        if bucketsWithColumns.contains(ungrouped) {
            var wU = remainder + lostConfiguredWeight
            if wU <= 0 && maxUngrouped > 0 && totalMaxPoints > 0 {
                wU = (maxUngrouped / totalMaxPoints) * 100
            }
            rawWeight[ungrouped, default: 0] += wU
        }

        let weightSum = rawWeight.values.reduce(0, +)
        if weightSum <= 0 {
            let earnedTotal = earnedByBucket.values.reduce(0, +)
            return (earnedTotal / totalMaxPoints) * 100
        }

        var acc = 0.0
        for (bucket, rw) in rawWeight where rw > 0 {
            let maxB = maxByBucket[bucket] ?? 0
            let earnedB = earnedByBucket[bucket] ?? 0
            let ratio = maxB > 0 ? earnedB / maxB : 0
            acc += ratio * (rw / weightSum)
        }
        return acc * 100
    }

    static func computeWhatIfFinalPercent(
        columns: [ColumnForFinal],
        actualGrades: [String: String],
        assignmentGroups: [GroupWeight],
        excusedByItemId: [String: Bool],
        whatIfOverrides: [String: String],
        heldItemIds: Set<String>,
        now: Date = Date()
    ) -> Double? {
        var opts = ComputeOptions()
        opts.mode = .whatIf
        opts.whatIfOverrides = whatIfOverrides
        opts.heldItemIds = heldItemIds
        opts.now = now
        return computeCourseFinalPercent(
            columns: columns,
            gradesByItemId: actualGrades,
            assignmentGroups: assignmentGroups,
            excusedByItemId: excusedByItemId,
            options: opts
        )
    }

    static func computeDroppedGrades(
        columns: [ColumnForFinal],
        gradesByItemId: [String: String],
        assignmentGroups: [GroupWeight],
        excusedByItemId: [String: Bool] = [:],
        options: ComputeOptions = ComputeOptions()
    ) -> [String: Bool] {
        let mergedGrades: [String: String]
        if options.mode == .whatIf {
            mergedGrades = mergeGradesForWhatIf(
                actualGrades: gradesByItemId,
                overrides: options.whatIfOverrides,
                heldItemIds: options.heldItemIds
            )
        } else {
            mergedGrades = gradesByItemId
        }

        let settingsIds = Set(assignmentGroups.map(\.id))
        var polByG: [String: GroupWeight] = [:]
        for group in assignmentGroups { polByG[group.id] = group }

        var byGroup: [String: [(itemId: String, max: Double, earned: Double, neverDrop: Bool, isFinal: Bool)]] = [:]
        let nowMs = options.now.timeIntervalSince1970 * 1000
        var dropped: [String: Bool] = [:]

        for col in columns {
            guard let max = col.maxPoints, max > 0 else { continue }
            if excusedByItemId[col.id] == true { continue }

            let hasOverride = options.mode == .whatIf
                && !(options.whatIfOverrides[col.id] ?? "").trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
            let gradeStr = mergedGrades[col.id]
            guard shouldIncludeColumn(col, gradeStr: gradeStr, hasOverride: hasOverride, mode: options.mode, nowMs: nowMs) else {
                continue
            }

            let earned = parseEarned(gradeStr)
            let gid = col.assignmentGroupId?.trimmingCharacters(in: .whitespacesAndNewlines)
            let bucket = (gid.flatMap { settingsIds.contains($0) ? $0 : nil }) ?? ungrouped
            guard bucket != ungrouped else { continue }

            byGroup[bucket, default: []].append((
                itemId: col.id,
                max: max,
                earned: earned,
                neverDrop: col.neverDrop,
                isFinal: col.replaceWithFinal
            ))
        }

        for (gid, lines) in byGroup {
            let policy = polByG[gid] ?? GroupWeight(
                id: gid,
                weightPercent: 0,
                dropLowest: 0,
                dropHighest: 0,
                replaceLowestWithFinal: false
            )
            let result = groupEffectiveEarnedAndMax(policy: policy, lines: lines)
            for id in result.droppedIds { dropped[id] = true }
        }
        return dropped
    }

    static func formatFinalPercent(_ pct: Double?) -> String {
        guard let pct, pct.isFinite else { return "—" }
        let rounded = (pct * 10).rounded() / 10
        return String(format: "%.1f%%", rounded)
    }

    // MARK: - MyGradesResponse helpers

    static func columns(from response: MyGradesResponse) -> [ColumnForFinal] {
        response.columns.map { col in
            ColumnForFinal(
                id: col.id,
                maxPoints: col.maxPoints,
                assignmentGroupId: col.assignmentGroupId,
                neverDrop: col.neverDrop,
                replaceWithFinal: col.replaceWithFinal,
                dueAt: col.dueAt
            )
        }
    }

    static func groups(from response: MyGradesResponse) -> [GroupWeight] {
        response.assignmentGroups.map { group in
            GroupWeight(
                id: group.id,
                weightPercent: group.weightPercent,
                dropLowest: group.dropLowest,
                dropHighest: group.dropHighest,
                replaceLowestWithFinal: group.replaceLowestWithFinal
            )
        }
    }

    static func excusedByItemId(from response: MyGradesResponse) -> [String: Bool] {
        var out: [String: Bool] = [:]
        for (id, status) in response.gradeStatuses where status == "excused" {
            out[id] = true
        }
        return out
    }

    static func heldSet(from response: MyGradesResponse) -> Set<String> {
        Set(response.heldGradeItemIds)
    }

    static func calcColumns(from response: MyGradesResponse) -> [ColumnForFinal] {
        let held = heldSet(from: response)
        return columns(from: response).filter { !held.contains($0.id) }
    }

    static func overallPercent(_ response: MyGradesResponse, options: ComputeOptions = ComputeOptions()) -> Double? {
        computeCourseFinalPercent(
            columns: calcColumns(from: response),
            gradesByItemId: response.grades,
            assignmentGroups: groups(from: response),
            excusedByItemId: excusedByItemId(from: response),
            options: options
        )
    }

    static func activeDroppedGrades(
        response: MyGradesResponse,
        whatIfMode: Bool,
        whatIfOverrides: [String: String]
    ) -> [String: Bool] {
        if whatIfMode && !whatIfOverrides.isEmpty {
            var opts = ComputeOptions()
            opts.mode = .whatIf
            opts.whatIfOverrides = whatIfOverrides
            opts.heldItemIds = heldSet(from: response)
            return computeDroppedGrades(
                columns: calcColumns(from: response),
                gradesByItemId: response.grades,
                assignmentGroups: groups(from: response),
                excusedByItemId: excusedByItemId(from: response),
                options: opts
            )
        }
        return response.droppedGrades
    }

    // MARK: - Private

    private static func parseEarned(_ raw: String?) -> Double {
        let trimmed = (raw ?? "").trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else { return 0 }
        let normalized = trimmed.replacingOccurrences(of: ",", with: "")
        return Double(normalized) ?? 0
    }

    private static func shouldIncludeColumn(
        _ col: ColumnForFinal,
        gradeStr: String?,
        hasOverride: Bool,
        mode: Mode,
        nowMs: Double
    ) -> Bool {
        if mode == .whatIf && hasOverride { return true }
        let hasGrade = !(gradeStr ?? "").trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
        var isPastDue = false
        if let dueAt = col.dueAt, let date = LMSDates.parse(dueAt) {
            isPastDue = date.timeIntervalSince1970 * 1000 < nowMs
        }
        return hasGrade || isPastDue
    }
}
