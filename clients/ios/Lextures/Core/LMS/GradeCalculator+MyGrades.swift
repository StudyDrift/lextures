import Foundation

extension GradeCalculator {
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
}