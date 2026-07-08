import Foundation

/// Course outcomes CRUD, mapping, and class-progress helpers (M13.5).
enum CourseOutcomesLogic {
    struct GradableOption: Identifiable, Hashable {
        var id: String
        var label: String
        var kind: String
    }

    struct OutcomeDraft: Equatable, Hashable {
        var title: String
        var description: String
    }

    struct QuizQuestionOption: Identifiable, Hashable {
        var id: String
        var prompt: String
    }

    enum MeasurementLevelId: String, CaseIterable, Identifiable {
        case diagnostic
        case formative
        case summative
        case performance

        var id: String { rawValue }

        var labelKey: String {
            "mobile.courseSettings.outcomes.measurement.\(rawValue)"
        }
    }

    enum IntensityLevelId: String, CaseIterable, Identifiable {
        case low
        case medium
        case high

        var id: String { rawValue }

        var labelKey: String {
            "mobile.courseSettings.outcomes.intensity.\(rawValue)"
        }
    }

    enum ValidationError: Equatable {
        case titleRequired
        case selectItem
        case selectQuestion
    }

    static let defaultMeasurement: MeasurementLevelId = .formative
    static let defaultIntensity: IntensityLevelId = .medium

    static func cacheKeyOutcomes(courseCode: String) -> String {
        "course:\(courseCode):outcomes-settings"
    }

    static func saveIdempotencyKey(courseCode: String) -> String {
        "course-outcomes:\(courseCode):save"
    }

    static func createOutcomeIdempotencyKey(courseCode: String) -> String {
        "course-outcomes:\(courseCode):create"
    }

    static func deleteOutcomeIdempotencyKey(courseCode: String, outcomeId: String) -> String {
        "course-outcomes:\(courseCode):delete:\(outcomeId)"
    }

    static func addLinkIdempotencyKey(courseCode: String, outcomeId: String, itemId: String) -> String {
        "course-outcomes:\(courseCode):link:\(outcomeId):\(itemId)"
    }

    static func deleteLinkIdempotencyKey(courseCode: String, outcomeId: String, linkId: String) -> String {
        "course-outcomes:\(courseCode):unlink:\(outcomeId):\(linkId)"
    }

    static func gradableOptions(from structure: [CourseStructureItem]) -> [GradableOption] {
        let byId = Dictionary(uniqueKeysWithValues: structure.map { ($0.id, $0) })
        let rows = structure.filter { item in
            (item.kind == "assignment" || item.kind == "quiz") && item.archived != true
        }
        let withLabels: [GradableOption] = rows.map { item in
            var moduleTitle = ""
            var parent = item.parentId.flatMap { byId[$0] }
            var guardIds = Set<String>()
            while let currentParent = parent, !guardIds.contains(currentParent.id) {
                guardIds.insert(currentParent.id)
                if currentParent.kind == "module" {
                    moduleTitle = currentParent.title
                    break
                }
                parent = currentParent.parentId.flatMap { byId[$0] }
            }
            let label = moduleTitle.isEmpty ? item.title : "\(moduleTitle) — \(item.title)"
            return GradableOption(id: item.id, label: label, kind: item.kind)
        }
        return withLabels.sorted { $0.label.localizedCaseInsensitiveCompare($1.label) == .orderedAscending }
    }

    static func drafts(from outcomes: [CourseOutcome]) -> [String: OutcomeDraft] {
        Dictionary(uniqueKeysWithValues: outcomes.map { outcome in
            (outcome.id, OutcomeDraft(title: outcome.title, description: outcome.description))
        })
    }

    static func dirtyOutcomeIds(
        drafts: [String: OutcomeDraft],
        outcomes: [CourseOutcome]
    ) -> [String] {
        drafts.compactMap { id, draft in
            guard let original = outcomes.first(where: { $0.id == id }) else { return nil }
            let titleChanged = draft.title.trimmingCharacters(in: .whitespacesAndNewlines)
                != original.title.trimmingCharacters(in: .whitespacesAndNewlines)
            let descriptionChanged = draft.description.trimmingCharacters(in: .whitespacesAndNewlines)
                != original.description.trimmingCharacters(in: .whitespacesAndNewlines)
            return titleChanged || descriptionChanged ? id : nil
        }
    }

    static func isDirty(drafts: [String: OutcomeDraft], outcomes: [CourseOutcome]) -> Bool {
        !dirtyOutcomeIds(drafts: drafts, outcomes: outcomes).isEmpty
    }

    static func validateDrafts(
        drafts: [String: OutcomeDraft],
        outcomes: [CourseOutcome]
    ) -> ValidationError? {
        for id in dirtyOutcomeIds(drafts: drafts, outcomes: outcomes) {
            guard let draft = drafts[id] else { continue }
            if draft.title.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
                return .titleRequired
            }
        }
        return nil
    }

    static func validateCreateTitle(_ title: String) -> ValidationError? {
        title.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty ? .titleRequired : nil
    }

    static func buildCreateBody(title: String, description: String) -> CreateCourseOutcomeBody {
        CreateCourseOutcomeBody(
            title: title.trimmingCharacters(in: .whitespacesAndNewlines),
            description: description.trimmingCharacters(in: .whitespacesAndNewlines)
        )
    }

    static func buildPatchBody(draft: OutcomeDraft) -> PatchCourseOutcomeBody {
        PatchCourseOutcomeBody(
            title: draft.title.trimmingCharacters(in: .whitespacesAndNewlines),
            description: draft.description.trimmingCharacters(in: .whitespacesAndNewlines)
        )
    }

    static func buildAddLinkBody(
        structureItemId: String,
        targetKind: String,
        quizQuestionId: String?,
        measurementLevel: String,
        intensityLevel: String
    ) -> AddCourseOutcomeLinkBody {
        AddCourseOutcomeLinkBody(
            structureItemId: structureItemId,
            targetKind: targetKind,
            quizQuestionId: quizQuestionId,
            measurementLevel: measurementLevel,
            intensityLevel: intensityLevel,
            subOutcomeId: nil
        )
    }

    static func targetKind(
        gradableKind: String,
        quizScopeWhole: Bool
    ) -> String {
        if gradableKind == "assignment" { return "assignment" }
        return quizScopeWhole ? "quiz" : "quiz_question"
    }

    static func questionOptions(from questions: [QuizQuestion]?) -> [QuizQuestionOption] {
        (questions ?? []).map { question in
            QuizQuestionOption(id: question.id, prompt: truncatedPrompt(question.prompt))
        }
    }

    static func truncatedPrompt(_ prompt: String) -> String {
        let collapsed = prompt.replacingOccurrences(of: "\\s+", with: " ", options: .regularExpression)
            .trimmingCharacters(in: .whitespacesAndNewlines)
        guard collapsed.count > 120 else { return collapsed }
        return String(collapsed.prefix(120)) + "…"
    }

    static func measurementLabelKey(_ level: String) -> String {
        MeasurementLevelId(rawValue: level)?.labelKey ?? level
    }

    static func intensityLabelKey(_ level: String) -> String {
        IntensityLevelId(rawValue: level)?.labelKey ?? level
    }

    static func formatLevels(measurementLevel: String, intensityLevel: String) -> String {
        let measurement = L.text(String.LocalizationValue(measurementLabelKey(measurementLevel)))
        let intensity = L.text(String.LocalizationValue(intensityLabelKey(intensityLevel)))
        return L.format("mobile.courseSettings.outcomes.levelsSummary", measurement, intensity)
    }

    static func progressPercentLabel(_ value: Double?) -> String {
        guard let value, value.isFinite else { return "—" }
        return "\(Int(value.rounded()))%"
    }

    static func progressLabel(for link: CourseOutcomeLink) -> String {
        L.format(
            "mobile.courseSettings.outcomes.progressLabel",
            progressPercentLabel(link.progress.avgScorePercent),
            String(link.progress.gradedLearners),
            String(link.progress.enrolledLearners)
        )
    }

    static func linkItemTitle(for link: CourseOutcomeLink) -> String {
        switch link.targetKind {
        case "quiz":
            return L.format("mobile.courseSettings.outcomes.linkWholeQuiz", link.itemTitle)
        case "quiz_question":
            return L.format("mobile.courseSettings.outcomes.linkQuestion", link.itemTitle)
        default:
            return link.itemTitle
        }
    }

    static func linkSummary(for link: CourseOutcomeLink) -> String {
        "\(linkItemTitle(for: link)) · \(formatLevels(measurementLevel: link.measurementLevel, intensityLevel: link.intensityLevel))"
    }

    static func rollupPercentLabel(_ value: Double?) -> String? {
        guard let value, value.isFinite else { return nil }
        return "\(Int(value.rounded()))"
    }
}