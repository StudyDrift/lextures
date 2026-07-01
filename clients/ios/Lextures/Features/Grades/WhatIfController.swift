import Foundation
import Observation

/// Client-only what-if projection state layered over real grades (M6.1 / plan 3.16).
@Observable
final class WhatIfController {
    var mode = false
    var overrides: [String: String] = [:]

    var hasOverrides: Bool {
        overrides.contains { !$0.value.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty }
    }

    func toggleMode() {
        mode.toggle()
    }

    func reset() {
        overrides = [:]
    }

    func setOverride(itemId: String, value: String) {
        let trimmed = value.trimmingCharacters(in: .whitespacesAndNewlines)
        if trimmed.isEmpty {
            overrides.removeValue(forKey: itemId)
        } else {
            overrides[itemId] = trimmed
        }
    }

    func projectedPercent(for grades: MyGradesResponse) -> Double? {
        guard mode else { return nil }
        return GradeCalculator.computeWhatIfFinalPercent(
            columns: GradeCalculator.calcColumns(from: grades),
            actualGrades: grades.grades,
            assignmentGroups: GradeCalculator.groups(from: grades),
            excusedByItemId: GradeCalculator.excusedByItemId(from: grades),
            whatIfOverrides: overrides,
            heldItemIds: GradeCalculator.heldSet(from: grades)
        )
    }

    func actualPercent(for grades: MyGradesResponse) -> Double? {
        GradeCalculator.overallPercent(grades)
    }

    func activeDropped(for grades: MyGradesResponse) -> [String: Bool] {
        GradeCalculator.activeDroppedGrades(
            response: grades,
            whatIfMode: mode,
            whatIfOverrides: overrides
        )
    }
}

enum GradesDisplayLogic {
    struct Section: Identifiable {
        var id: String
        var title: String
        var weightPercent: Double?
        var columns: [GradeColumn]
    }

    static func buildSections(from response: MyGradesResponse) -> [Section] {
        let groupIds = Set(response.assignmentGroups.map(\.id))
        var grouped: [String: [GradeColumn]] = [:]
        var ungrouped: [GradeColumn] = []

        for column in response.columns {
            if let gid = column.assignmentGroupId?.trimmingCharacters(in: .whitespacesAndNewlines),
               !gid.isEmpty,
               groupIds.contains(gid) {
                grouped[gid, default: []].append(column)
            } else {
                ungrouped.append(column)
            }
        }

        var sections: [Section] = response.assignmentGroups.compactMap { group in
            guard let cols = grouped[group.id], !cols.isEmpty else { return nil }
            return Section(
                id: group.id,
                title: group.name.isEmpty ? "Assignments" : group.name,
                weightPercent: group.weightPercent > 0 ? group.weightPercent : nil,
                columns: cols
            )
        }

        if !ungrouped.isEmpty {
            sections.append(Section(id: "__ungrouped__", title: "Other", weightPercent: nil, columns: ungrouped))
        }
        return sections
    }

    static func groupLabel(for column: GradeColumn, in response: MyGradesResponse) -> String? {
        guard let gid = column.assignmentGroupId,
              let group = response.assignmentGroups.first(where: { $0.id == gid }) else { return nil }
        if group.weightPercent > 0 {
            return "\(group.name) · \(Int(group.weightPercent))%"
        }
        return group.name.nilIfEmpty
    }

    static func statusBadges(
        column: GradeColumn,
        response: MyGradesResponse,
        dropped: [String: Bool]
    ) -> [String] {
        var badges: [String] = []
        if response.gradeStatuses[column.id] == "excused" { badges.append("Excused") }
        else if response.heldGradeItemIds.contains(column.id) { badges.append("Pending") }
        else if dropped[column.id] == true { badges.append("Dropped") }
        else if response.gradeStatuses[column.id] == "late" { badges.append("Late") }
        return badges
    }
}

private extension String {
    var nilIfEmpty: String? {
        isEmpty ? nil : self
    }
}
