import Foundation

/// Course settings helpers (M13.1) — permission gate, validation, dirty detection, schedule/theme helpers.
enum CourseSettingsLogic {
    static func courseItemCreatePermission(courseCode: String) -> String {
        "course:\(courseCode):item:create"
    }

    static func canManageCourse(courseCode: String, permissions: [String]) -> Bool {
        permissions.contains(courseItemCreatePermission(courseCode: courseCode))
    }

    static func settingsEnabled(_ features: MobilePlatformFeatures) -> Bool {
        features.ffMobileCourseSettings
    }

    static func shouldShowWorkspaceSection(
        course: CourseSummary,
        permissions: [String],
        features: MobilePlatformFeatures
    ) -> Bool {
        guard settingsEnabled(features) else { return false }
        return canManageCourse(courseCode: course.courseCode, permissions: permissions)
    }

    enum CourseSettingsSection: String, CaseIterable, Identifiable, Hashable {
        case general
        case features
        case sections
        case grading
        case outcomes
        case gradingAgents
        case plagiarism
        case accessibility
        case translations
        case importExport
        case blueprint
        case archive

        var id: String { rawValue }

        var labelKey: String {
            switch self {
            case .general: return "mobile.courseSettings.section.general"
            case .features: return "mobile.courseSettings.section.features"
            case .sections: return "mobile.courseSettings.section.sections"
            case .grading: return "mobile.courseSettings.section.grading"
            case .outcomes: return "mobile.courseSettings.section.outcomes"
            case .gradingAgents: return "mobile.courseSettings.section.gradingAgents"
            case .plagiarism: return "mobile.courseSettings.section.plagiarism"
            case .accessibility: return "mobile.courseSettings.section.accessibility"
            case .translations: return "mobile.courseSettings.section.translations"
            case .importExport: return "mobile.courseSettings.section.importExport"
            case .blueprint: return "mobile.courseSettings.section.blueprint"
            case .archive: return "mobile.courseSettings.section.archive"
            }
        }

        var label: String { L.text(String.LocalizationValue(labelKey)) }

        var systemImage: String {
            switch self {
            case .general: return "info.circle"
            case .features: return "slider.horizontal.3"
            case .sections: return "square.grid.2x2"
            case .grading: return "scalemass"
            case .outcomes: return "target"
            case .gradingAgents: return "cpu"
            case .plagiarism: return "shield"
            case .accessibility: return "eye"
            case .translations: return "globe"
            case .importExport: return "arrow.up.arrow.down"
            case .blueprint: return "doc.on.doc"
            case .archive: return "archivebox"
            }
        }
    }

    static func visibleSettingsSections(
        course: CourseSummary,
        features: MobilePlatformFeatures
    ) -> [CourseSettingsSection] {
        var sections: [CourseSettingsSection] = [.general, .features]
        if course.isSectionsEnabled {
            sections.append(.sections)
        }
        sections.append(contentsOf: [.grading, .outcomes])
        if features.graderAgentEnabled {
            sections.append(.gradingAgents)
        }
        if features.ffPlagiarismChecks {
            sections.append(.plagiarism)
        }
        if features.altTextEnforcementEnabled || features.translationMemoryEnabled {
            if features.altTextEnforcementEnabled {
                sections.append(.accessibility)
            }
            if features.translationMemoryEnabled {
                sections.append(.translations)
            }
        }
        sections.append(contentsOf: [.importExport, .blueprint, .archive])
        return sections
    }

    static let gradeLevels: [String] = [
        "", "K", "1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11", "12",
        "K-2", "3-5", "6-8", "9-12", "K-12",
    ]

    static let markdownThemePresets: [String] = [
        "classic", "reader", "serif", "contrast", "night", "accent", "custom",
    ]

    static let articleWidths: [String] = ["narrow", "comfortable", "wide", "full"]
    static let fontFamilies: [String] = ["sans", "serif"]

    enum RelativeDurationUnit: String, CaseIterable {
        case days = "D"
        case weeks = "W"
        case months = "M"
        case years = "Y"
    }

    enum CourseHomeLanding: String, CaseIterable {
        case data, calendar
        case contentPage = "content_page"
    }

    enum ScheduleMode: String, CaseIterable {
        case fixed, relative
    }

    enum SaveStatus: Equatable {
        case idle, saving, saved, error(String)
    }

    struct ValidationError: Equatable {
        var title: String?
        var courseHome: String?
    }

    static func validateGeneralForm(
        title: String,
        courseHomeLanding: CourseHomeLanding,
        courseHomeContentItemId: String
    ) -> ValidationError? {
        var error = ValidationError()
        if title.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
            error.title = L.text("mobile.courseSettings.validation.titleRequired")
        }
        if courseHomeLanding == .contentPage,
           courseHomeContentItemId.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
            error.courseHome = L.text("mobile.courseSettings.validation.contentPageRequired")
        }
        if error.title == nil && error.courseHome == nil { return nil }
        return error
    }

    static func normalizeCourseHomeLanding(_ value: String?) -> CourseHomeLanding {
        if value == CourseHomeLanding.calendar.rawValue { return .calendar }
        if value == CourseHomeLanding.contentPage.rawValue { return .contentPage }
        return .data
    }

    static func isoDurationToParts(
        iso: String?
    ) -> (amount: String, unit: RelativeDurationUnit) {
        guard let iso, !iso.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty else {
            return ("", .months)
        }
        let pattern = /^P(\d+)([DWMY])$/
        guard let match = iso.trimmingCharacters(in: .whitespacesAndNewlines).wholeMatch(of: pattern) else {
            return ("", .months)
        }
        let amount = String(match.output.1)
        let unitChar = String(match.output.2).uppercased()
        let unit = RelativeDurationUnit(rawValue: unitChar) ?? .months
        return (amount, unit)
    }

    static func partsToIsoDuration(amount: String, unit: RelativeDurationUnit) -> String? {
        guard let parsedAmount = Int(amount.trimmingCharacters(in: .whitespacesAndNewlines)), parsedAmount >= 1 else {
            return nil
        }
        return "P\(parsedAmount)\(unit.rawValue)"
    }

    static func isoToLocalDateString(_ iso: String?) -> String {
        guard let iso, !iso.isEmpty, let date = ISO8601DateFormatter().date(from: iso) else {
            return ""
        }
        let formatter = DateFormatter()
        formatter.dateFormat = "yyyy-MM-dd'T'HH:mm"
        formatter.timeZone = .current
        return formatter.string(from: date)
    }

    static func localDateStringToIso(_ value: String) -> String? {
        let trimmed = value.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else { return nil }
        let formatter = DateFormatter()
        formatter.dateFormat = "yyyy-MM-dd'T'HH:mm"
        formatter.timeZone = .current
        guard let date = formatter.date(from: trimmed) else { return nil }
        return ISO8601DateFormatter().string(from: date)
    }

    static func normalizeIso(_ iso: String?) -> String? {
        localDateStringToIso(isoToLocalDateString(iso))
    }

    static func parseHeroObjectPosition(_ pos: String?) -> (x: Double, y: Double) {
        guard let pos, !pos.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty else {
            return (50, 50)
        }
        let pattern = /^(\d+(?:\.\d+)?)%\s+(\d+(?:\.\d+)?)%$/
        guard let match = pos.trimmingCharacters(in: .whitespacesAndNewlines).wholeMatch(of: pattern) else {
            return (50, 50)
        }
        let posX = min(100, max(0, Double(match.output.1) ?? 50))
        let posY = min(100, max(0, Double(match.output.2) ?? 50))
        return (posX, posY)
    }

    static func formatHeroObjectPosition(x: Double, y: Double) -> String? {
        let rx = round(x)
        let ry = round(y)
        if rx == 50 && ry == 50 { return nil }
        return "\(Int(rx))% \(Int(ry))%"
    }

    static func defaultImagePrompt(title: String, description: String) -> String {
        """
        Generate an image for a course banner with the following title and description:
        Title: \(title)
        Description: \(description)
        """
    }

    static func gradeLevelLabel(_ value: String) -> String {
        let trimmed = value.trimmingCharacters(in: .whitespacesAndNewlines)
        if trimmed.isEmpty { return L.text("mobile.courseSettings.gradeLevel.none") }
        return trimmed
    }

    static func timezoneOptions() -> [String] {
        TimeZone.knownTimeZoneIdentifiers.sorted()
    }

    static func defaultTimezone() -> String {
        TimeZone.current.identifier
    }

    static func isGeneralFormDirty(form: CourseGeneralFormState, course: CourseSummary) -> Bool {
        isBasicInfoDirty(form: form, course: course)
            || isCourseHomeDirty(form: form, course: course)
            || isScheduleDirty(form: form, course: course)
            || isMarkdownThemeDirty(form: form, course: course)
    }

    private static func isBasicInfoDirty(form: CourseGeneralFormState, course: CourseSummary) -> Bool {
        if form.title.trimmingCharacters(in: .whitespacesAndNewlines) != course.title { return true }
        if form.description.trimmingCharacters(in: .whitespacesAndNewlines) != course.description { return true }
        if form.published != (course.published ?? false) { return true }
        if (form.gradeLevel.trimmingCharacters(in: .whitespacesAndNewlines).nilIfEmpty)
            != (course.gradeLevel?.trimmingCharacters(in: .whitespacesAndNewlines).nilIfEmpty) {
            return true
        }
        if (form.courseTimezone.trimmingCharacters(in: .whitespacesAndNewlines).nilIfEmpty)
            != (course.courseTimezone?.trimmingCharacters(in: .whitespacesAndNewlines).nilIfEmpty
                ?? defaultTimezone().nilIfEmpty) {
            return true
        }
        return false
    }

    private static func isCourseHomeDirty(form: CourseGeneralFormState, course: CourseSummary) -> Bool {
        let normHome = normalizeCourseHomeLanding(course.courseHomeLanding)
        if form.courseHomeLanding != normHome { return true }
        let origHomeId = (course.courseHomeContentItemId ?? "").trimmingCharacters(in: .whitespacesAndNewlines)
        return form.courseHomeLanding == .contentPage
            && form.courseHomeContentItemId.trimmingCharacters(in: .whitespacesAndNewlines) != origHomeId
    }

    private static func isScheduleDirty(form: CourseGeneralFormState, course: CourseSummary) -> Bool {
        let mode = course.scheduleMode == ScheduleMode.relative.rawValue ? ScheduleMode.relative : .fixed
        if form.scheduleMode != mode { return true }
        if form.scheduleMode == .fixed {
            if localDateStringToIso(form.startsAt) != normalizeIso(course.startsAt) { return true }
            if localDateStringToIso(form.endsAt) != normalizeIso(course.endsAt) { return true }
            if localDateStringToIso(form.visibleFrom) != normalizeIso(course.visibleFrom) { return true }
            if localDateStringToIso(form.hiddenAt) != normalizeIso(course.hiddenAt) { return true }
            return false
        }
        let currentEnd = partsToIsoDuration(amount: form.relEndAmount, unit: form.relEndUnit)
        if currentEnd != (course.relativeEndAfter?.nilIfEmpty) { return true }
        let currentHidden = partsToIsoDuration(amount: form.relHiddenAmount, unit: form.relHiddenUnit)
        return currentHidden != (course.relativeHiddenAfter?.nilIfEmpty)
    }

    private static func isMarkdownThemeDirty(form: CourseGeneralFormState, course: CourseSummary) -> Bool {
        if (course.markdownThemePreset ?? "default") != form.markdownThemePreset { return true }
        guard form.markdownThemePreset == "custom" else { return false }
        let seed = MarkdownThemeCustom.seed
        let orig = course.markdownThemeCustom ?? seed
        let draft = form.customDraft
        if (draft.headingColor ?? seed.headingColor) != (orig.headingColor ?? seed.headingColor) { return true }
        if (draft.bodyColor ?? seed.bodyColor) != (orig.bodyColor ?? seed.bodyColor) { return true }
        if (draft.linkColor ?? seed.linkColor) != (orig.linkColor ?? seed.linkColor) { return true }
        if (draft.codeBackground ?? seed.codeBackground) != (orig.codeBackground ?? seed.codeBackground) { return true }
        if (draft.blockquoteBorder ?? seed.blockquoteBorder) != (orig.blockquoteBorder ?? seed.blockquoteBorder) { return true }
        if (draft.articleWidth ?? seed.articleWidth) != (orig.articleWidth ?? seed.articleWidth) { return true }
        return (draft.fontFamily ?? seed.fontFamily) != (orig.fontFamily ?? seed.fontFamily)
    }

    static func applyCourseToForm(_ course: CourseSummary) -> CourseGeneralFormState {
        let endParts = isoDurationToParts(iso: course.relativeEndAfter)
        let hiddenParts = isoDurationToParts(iso: course.relativeHiddenAfter)
        return CourseGeneralFormState(
            title: course.title,
            description: course.description,
            published: course.published ?? false,
            gradeLevel: course.gradeLevel ?? "",
            courseTimezone: course.courseTimezone?.trimmingCharacters(in: .whitespacesAndNewlines).nilIfEmpty
                ?? defaultTimezone(),
            courseHomeLanding: normalizeCourseHomeLanding(course.courseHomeLanding),
            courseHomeContentItemId: (course.courseHomeContentItemId ?? "").trimmingCharacters(in: .whitespacesAndNewlines),
            scheduleMode: course.scheduleMode == ScheduleMode.relative.rawValue ? .relative : .fixed,
            startsAt: isoToLocalDateString(course.startsAt),
            endsAt: isoToLocalDateString(course.endsAt),
            visibleFrom: isoToLocalDateString(course.visibleFrom),
            hiddenAt: isoToLocalDateString(course.hiddenAt),
            relEndAmount: endParts.amount,
            relEndUnit: endParts.unit,
            relHiddenAmount: hiddenParts.amount,
            relHiddenUnit: hiddenParts.unit,
            markdownThemePreset: course.markdownThemePreset ?? "default",
            customDraft: {
                var draft = MarkdownThemeCustom.seed
                if let custom = course.markdownThemeCustom {
                    draft.merge(custom)
                }
                return draft
            }()
        )
    }

    static func buildCourseUpdateRequest(form: CourseGeneralFormState) -> CourseUpdateRequest {
        let mode = form.scheduleMode
        return CourseUpdateRequest(
            title: form.title.trimmingCharacters(in: .whitespacesAndNewlines),
            description: form.description.trimmingCharacters(in: .whitespacesAndNewlines),
            published: form.published,
            startsAt: mode == .relative ? nil : localDateStringToIso(form.startsAt),
            endsAt: mode == .relative ? nil : localDateStringToIso(form.endsAt),
            visibleFrom: mode == .relative ? nil : localDateStringToIso(form.visibleFrom),
            hiddenAt: mode == .relative ? nil : localDateStringToIso(form.hiddenAt),
            scheduleMode: mode.rawValue,
            relativeEndAfter: mode == .relative ? partsToIsoDuration(amount: form.relEndAmount, unit: form.relEndUnit) : nil,
            relativeHiddenAfter: mode == .relative ? partsToIsoDuration(amount: form.relHiddenAmount, unit: form.relHiddenUnit) : nil,
            courseHomeLanding: form.courseHomeLanding.rawValue,
            courseHomeContentItemId: form.courseHomeLanding == .contentPage
                ? form.courseHomeContentItemId.trimmingCharacters(in: .whitespacesAndNewlines).nilIfEmpty
                : nil,
            courseTimezone: form.courseTimezone.trimmingCharacters(in: .whitespacesAndNewlines).nilIfEmpty,
            gradeLevel: form.gradeLevel.trimmingCharacters(in: .whitespacesAndNewlines).nilIfEmpty
        )
    }

    static func buildMarkdownThemePatch(form: CourseGeneralFormState) -> CourseMarkdownThemePatch {
        CourseMarkdownThemePatch(
            preset: form.markdownThemePreset,
            custom: form.markdownThemePreset == "custom" ? form.customDraft : nil
        )
    }

    static func courseNeedsUpdate(form: CourseGeneralFormState, course: CourseSummary) -> Bool {
        let request = buildCourseUpdateRequest(form: form)
        let baseline = buildCourseUpdateRequest(form: applyCourseToForm(course))
        return request != baseline
    }

    static func themeNeedsUpdate(form: CourseGeneralFormState, course: CourseSummary) -> Bool {
        let request = buildMarkdownThemePatch(form: form)
        let baseline = buildMarkdownThemePatch(form: applyCourseToForm(course))
        return request != baseline
    }

    static func contentPages(from items: [CourseStructureItem]) -> [CourseStructureItem] {
        items.filter { $0.kind == "content_page" }
    }

    static func cacheKeySettings(courseCode: String) -> String {
        "course:\(courseCode):settings"
    }
}

private extension String {
    var nilIfEmpty: String? {
        let trimmed = trimmingCharacters(in: .whitespacesAndNewlines)
        return trimmed.isEmpty ? nil : trimmed
    }
}

struct CourseGeneralFormState: Equatable {
    var title: String = ""
    var description: String = ""
    var published: Bool = false
    var gradeLevel: String = ""
    var courseTimezone: String = CourseSettingsLogic.defaultTimezone()
    var courseHomeLanding: CourseSettingsLogic.CourseHomeLanding = .data
    var courseHomeContentItemId: String = ""
    var scheduleMode: CourseSettingsLogic.ScheduleMode = .fixed
    var startsAt: String = ""
    var endsAt: String = ""
    var visibleFrom: String = ""
    var hiddenAt: String = ""
    var relEndAmount: String = ""
    var relEndUnit: CourseSettingsLogic.RelativeDurationUnit = .months
    var relHiddenAmount: String = ""
    var relHiddenUnit: CourseSettingsLogic.RelativeDurationUnit = .months
    var markdownThemePreset: String = "default"
    var customDraft: MarkdownThemeCustom = .seed
}
