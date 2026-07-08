import Foundation

/// Course grading scale, weighted groups, and item mapping helpers (M13.4).
enum CourseGradingLogic {
    struct GradingScaleOption: Identifiable, Hashable {
        var id: String
        var labelKey: String
        var descriptionKey: String
    }

    struct SchemeDisplayType: Identifiable, Hashable {
        var id: String
        var labelKey: String
    }

    struct EditableAssignmentGroup: Equatable, Hashable {
        var clientKey: String
        var id: String?
        var name: String
        var sortOrder: Int
        var weightPercent: String
    }

    struct GradingSchemeBand: Identifiable, Equatable, Hashable {
        var clientKey: String
        var label: String
        var minPct: String
        var gpa: String

        var id: String { clientKey }
    }

    struct GradableRow: Identifiable, Hashable {
        var item: CourseStructureItem
        var moduleTitle: String
        var id: String { item.id }
    }

    struct FormBaseline: Equatable {
        var gradingScale: String
        var groups: [EditableAssignmentGroup]
        var schemeType: String
        var bands: [GradingSchemeBand]
        var passMinPct: String
        var completeMinPct: String
    }

    enum ValidationError: Equatable {
        case groupsNeedNames
        case bandsInvalid(String)
        case schemeInvalid(String)
    }

    static let gradingScaleOptions: [GradingScaleOption] = [
        .init(id: "letter_standard", labelKey: "mobile.courseSettings.grading.scale.letterStandard.label", descriptionKey: "mobile.courseSettings.grading.scale.letterStandard.description"),
        .init(id: "letter_plus_minus", labelKey: "mobile.courseSettings.grading.scale.letterPlusMinus.label", descriptionKey: "mobile.courseSettings.grading.scale.letterPlusMinus.description"),
        .init(id: "percent", labelKey: "mobile.courseSettings.grading.scale.percent.label", descriptionKey: "mobile.courseSettings.grading.scale.percent.description"),
        .init(id: "pass_fail", labelKey: "mobile.courseSettings.grading.scale.passFail.label", descriptionKey: "mobile.courseSettings.grading.scale.passFail.description"),
    ]

    static let schemeDisplayTypes: [SchemeDisplayType] = [
        .init(id: "points", labelKey: "mobile.courseSettings.grading.scheme.type.points"),
        .init(id: "percentage", labelKey: "mobile.courseSettings.grading.scheme.type.percentage"),
        .init(id: "letter", labelKey: "mobile.courseSettings.grading.scheme.type.letter"),
        .init(id: "gpa", labelKey: "mobile.courseSettings.grading.scheme.type.gpa"),
        .init(id: "pass_fail", labelKey: "mobile.courseSettings.grading.scheme.type.passFail"),
        .init(id: "complete_incomplete", labelKey: "mobile.courseSettings.grading.scheme.type.completeIncomplete"),
    ]

    static func cacheKeyGrading(courseCode: String) -> String {
        "course:\(courseCode):grading-settings"
    }

    static func settingsIdempotencyKey(courseCode: String) -> String {
        "course-grading:\(courseCode):settings"
    }

    static func schemeIdempotencyKey(courseCode: String) -> String {
        "course-grading:\(courseCode):scheme"
    }

    static func itemMappingIdempotencyKey(courseCode: String, itemId: String) -> String {
        "course-grading:\(courseCode):item-group:\(itemId)"
    }

    static func newClientKey() -> String {
        "new-\(UUID().uuidString)"
    }

    static func defaultGroups() -> [EditableAssignmentGroup] {
        [
            .init(
                clientKey: newClientKey(),
                id: nil,
                name: "Assignments",
                sortOrder: 0,
                weightPercent: "100"
            ),
        ]
    }

    static func defaultBands() -> [GradingSchemeBand] {
        [
            .init(clientKey: newClientKey(), label: "A", minPct: "90", gpa: "4"),
            .init(clientKey: newClientKey(), label: "B", minPct: "80", gpa: "3"),
            .init(clientKey: newClientKey(), label: "C", minPct: "70", gpa: "2"),
            .init(clientKey: newClientKey(), label: "D", minPct: "60", gpa: "1"),
            .init(clientKey: newClientKey(), label: "F", minPct: "0", gpa: "0"),
        ]
    }

    static func groupsFromSettings(_ settings: CourseGradingSettings) -> [EditableAssignmentGroup] {
        if settings.assignmentGroups.isEmpty { return defaultGroups() }
        return settings.assignmentGroups.map { group in
            EditableAssignmentGroup(
                clientKey: group.id,
                id: group.id,
                name: group.name,
                sortOrder: group.sortOrder,
                weightPercent: String(group.weightPercent)
            )
        }
    }

    static func baseline(
        settings: CourseGradingSettings,
        scheme: CourseGradingSchemeRecord?
    ) -> FormBaseline {
        let schemeType = scheme?.type.trimmingCharacters(in: .whitespacesAndNewlines).nilIfEmpty ?? "points"
        let parsed = parseBands(from: scheme?.scaleJson)
        let passMin = parsePassMinPct(from: scheme?.scaleJson) ?? "60"
        let completeMin = parseCompleteMinPct(from: scheme?.scaleJson) ?? "50"
        return FormBaseline(
            gradingScale: settings.gradingScale.nilIfEmpty ?? "letter_standard",
            groups: groupsFromSettings(settings),
            schemeType: schemeType,
            bands: parsed.isEmpty ? defaultBands() : parsed,
            passMinPct: passMin,
            completeMinPct: completeMin
        )
    }

    static func weightTotal(_ groups: [EditableAssignmentGroup]) -> Double {
        var total = 0.0
        for group in groups {
            let value = Double(group.weightPercent.trimmingCharacters(in: .whitespacesAndNewlines)) ?? 0
            if value.isFinite { total += value }
        }
        return (total * 1000).rounded() / 1000
    }

    static func hasWeightWarning(_ total: Double) -> Bool {
        abs(total - 100) >= 0.01
    }

    static func weightTotalLabel(total: Double) -> String {
        L.format("mobile.courseSettings.grading.weightTotal", String(format: "%.2f", total))
    }

    static func gradableRows(from structure: [CourseStructureItem]) -> [GradableRow] {
        var rows: [GradableRow] = []
        var moduleTitle = ""
        for item in structure {
            if item.kind == "module" {
                moduleTitle = item.title
            } else if item.isGradable {
                rows.append(.init(item: item, moduleTitle: moduleTitle))
            }
        }
        return rows
    }

    static func namedGroupsWithIds(_ groups: [EditableAssignmentGroup]) -> [EditableAssignmentGroup] {
        groups.filter { !$0.name.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty && $0.id != nil }
    }

    static func isSettingsDirty(current: FormBaseline, baseline: FormBaseline) -> Bool {
        if current.gradingScale != baseline.gradingScale { return true }
        return normalizedGroups(current.groups) != normalizedGroups(baseline.groups)
    }

    static func isSchemeDirty(current: FormBaseline, baseline: FormBaseline) -> Bool {
        if current.schemeType != baseline.schemeType { return true }
        if current.schemeType == "letter" || current.schemeType == "gpa" {
            return normalizedBands(current.bands) != normalizedBands(baseline.bands)
        }
        if current.schemeType == "pass_fail" {
            return current.passMinPct.trimmingCharacters(in: .whitespacesAndNewlines)
                != baseline.passMinPct.trimmingCharacters(in: .whitespacesAndNewlines)
        }
        if current.schemeType == "complete_incomplete" {
            return current.completeMinPct.trimmingCharacters(in: .whitespacesAndNewlines)
                != baseline.completeMinPct.trimmingCharacters(in: .whitespacesAndNewlines)
        }
        return false
    }

    static func validateGroups(_ groups: [EditableAssignmentGroup]) -> ValidationError? {
        let named = groups.filter { !$0.name.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty }
        if named.isEmpty || named.count != groups.count {
            return .groupsNeedNames
        }
        return nil
    }

    static func validateBands(_ bands: [GradingSchemeBand]) -> ValidationError? {
        guard !bands.isEmpty else {
            return .bandsInvalid(L.text("mobile.courseSettings.grading.validation.bandsRequired"))
        }
        var parsed: [(label: String, minPct: Double)] = []
        for (index, band) in bands.enumerated() {
            let label = band.label.trimmingCharacters(in: .whitespacesAndNewlines)
            if label.isEmpty {
                return .bandsInvalid(L.format("mobile.courseSettings.grading.validation.bandLabelRequired", index + 1))
            }
            guard let minPct = Double(band.minPct.trimmingCharacters(in: .whitespacesAndNewlines)),
                  minPct.isFinite, minPct >= 0, minPct <= 100 else {
                return .bandsInvalid(L.format("mobile.courseSettings.grading.validation.bandMinOutOfRange", index + 1))
            }
            parsed.append((label, minPct))
        }
        let ascending = parsed.sorted { $0.minPct < $1.minPct }
        if abs(ascending[0].minPct) > 0.001 {
            return .bandsInvalid(L.text("mobile.courseSettings.grading.validation.lowestBandMustBeZero"))
        }
        for index in 1 ..< ascending.count where ascending[index].minPct <= ascending[index - 1].minPct + 0.001 {
            return .bandsInvalid(L.text("mobile.courseSettings.grading.validation.bandsMustIncrease"))
        }
        return nil
    }

    static func validateScheme(form: FormBaseline) -> ValidationError? {
        switch form.schemeType {
        case "letter", "gpa":
            return validateBands(form.bands)
        case "pass_fail":
            guard let value = Double(form.passMinPct.trimmingCharacters(in: .whitespacesAndNewlines)),
                  value.isFinite, value >= 0, value <= 100 else {
                return .schemeInvalid(L.text("mobile.courseSettings.grading.validation.passMinOutOfRange"))
            }
            return nil
        case "complete_incomplete":
            guard let value = Double(form.completeMinPct.trimmingCharacters(in: .whitespacesAndNewlines)),
                  value.isFinite, value >= 0, value <= 100 else {
                return .schemeInvalid(L.text("mobile.courseSettings.grading.validation.completeMinOutOfRange"))
            }
            return nil
        default:
            return nil
        }
    }

    static func buildPutSettingsBody(form: FormBaseline) -> PutCourseGradingSettingsBody {
        PutCourseGradingSettingsBody(
            gradingScale: form.gradingScale,
            assignmentGroups: form.groups.enumerated().map { index, group in
                let weight = Double(group.weightPercent.trimmingCharacters(in: .whitespacesAndNewlines)) ?? 0
                return CourseAssignmentGroupInput(
                    id: group.id,
                    name: group.name.trimmingCharacters(in: .whitespacesAndNewlines),
                    sortOrder: index,
                    weightPercent: weight.isFinite ? weight : 0,
                    dropLowest: 0,
                    dropHighest: 0,
                    replaceLowestWithFinal: false
                )
            }
        )
    }

    static func buildPutSchemeBody(form: FormBaseline) -> PutCourseGradingSchemeBody {
        let scaleJson: JSONValue?
        switch form.schemeType {
        case "letter", "gpa":
            scaleJson = encodeBands(form.bands)
        case "pass_fail":
            let value = Double(form.passMinPct.trimmingCharacters(in: .whitespacesAndNewlines)) ?? 60
            scaleJson = .object(["pass_min_pct": .number(value)])
        case "complete_incomplete":
            let value = Double(form.completeMinPct.trimmingCharacters(in: .whitespacesAndNewlines)) ?? 50
            scaleJson = .object(["complete_min_pct": .number(value)])
        default:
            scaleJson = .object([:])
        }
        return PutCourseGradingSchemeBody(type: form.schemeType, scaleJson: scaleJson)
    }

    static func sortBandsDescending(_ bands: [GradingSchemeBand]) -> [GradingSchemeBand] {
        bands.sorted {
            let left = Double($0.minPct.trimmingCharacters(in: .whitespacesAndNewlines)) ?? 0
            let right = Double($1.minPct.trimmingCharacters(in: .whitespacesAndNewlines)) ?? 0
            return left > right
        }
    }

    static func kindLabelKey(for kind: String) -> String {
        switch kind {
        case "quiz": return "mobile.courseSettings.grading.mapping.type.quiz"
        case "content_page": return "mobile.courseSettings.grading.mapping.type.content"
        default: return "mobile.courseSettings.grading.mapping.type.assignment"
        }
    }

    private static func normalizedGroups(_ groups: [EditableAssignmentGroup]) -> [[String: String]] {
        groups.map { group in
            [
                "name": group.name.trimmingCharacters(in: .whitespacesAndNewlines),
                "weight": String(Double(group.weightPercent.trimmingCharacters(in: .whitespacesAndNewlines)) ?? 0),
            ]
        }
    }

    private static func normalizedBands(_ bands: [GradingSchemeBand]) -> [[String: String]] {
        sortBandsDescending(bands).map { band in
            [
                "label": band.label.trimmingCharacters(in: .whitespacesAndNewlines),
                "min_pct": band.minPct.trimmingCharacters(in: .whitespacesAndNewlines),
                "gpa": band.gpa.trimmingCharacters(in: .whitespacesAndNewlines),
            ]
        }
    }

    private static func parseBands(from scaleJson: JSONValue?) -> [GradingSchemeBand] {
        guard case let .array(items) = scaleJson else { return [] }
        return items.compactMap { value -> GradingSchemeBand? in
            guard case let .object(obj) = value else { return nil }
            let label: String
            if case let .string(text) = obj["label"] ?? .null { label = text } else { return nil }
            let minPct: String
            if case let .number(number) = obj["min_pct"] ?? .null {
                minPct = String(number)
            } else if case let .string(text) = obj["min_pct"] ?? .null {
                minPct = text
            } else {
                return nil
            }
            let gpa: String
            if case let .number(number) = obj["gpa"] ?? .null {
                gpa = String(number)
            } else if case let .string(text) = obj["gpa"] ?? .null {
                gpa = text
            } else {
                gpa = ""
            }
            return GradingSchemeBand(clientKey: newClientKey(), label: label, minPct: minPct, gpa: gpa)
        }
    }

    private static func parsePassMinPct(from scaleJson: JSONValue?) -> String? {
        guard case let .object(obj) = scaleJson else { return nil }
        if case let .number(number) = obj["pass_min_pct"] ?? .null { return String(number) }
        if case let .string(text) = obj["pass_min_pct"] ?? .null { return text }
        return nil
    }

    private static func parseCompleteMinPct(from scaleJson: JSONValue?) -> String? {
        guard case let .object(obj) = scaleJson else { return nil }
        if case let .number(number) = obj["complete_min_pct"] ?? .null { return String(number) }
        if case let .string(text) = obj["complete_min_pct"] ?? .null { return text }
        return nil
    }

    private static func encodeBands(_ bands: [GradingSchemeBand]) -> JSONValue {
        .array(
            sortBandsDescending(bands).map { band in
                var object: [String: JSONValue] = [
                    "label": .string(band.label.trimmingCharacters(in: .whitespacesAndNewlines)),
                    "min_pct": .number(Double(band.minPct.trimmingCharacters(in: .whitespacesAndNewlines)) ?? 0),
                ]
                if let gpa = Double(band.gpa.trimmingCharacters(in: .whitespacesAndNewlines)) {
                    object["gpa"] = .number(gpa)
                }
                return .object(object)
            }
        )
    }
}

private extension String {
    var nilIfEmpty: String? {
        let trimmed = trimmingCharacters(in: .whitespacesAndNewlines)
        return trimmed.isEmpty ? nil : trimmed
    }
}