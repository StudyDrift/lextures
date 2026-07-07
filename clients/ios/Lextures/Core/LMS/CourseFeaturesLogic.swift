import Foundation

/// Course features / tools helpers (M13.2).
enum CourseFeaturesLogic {
    enum Tool: String, CaseIterable, Identifiable, Hashable {
        case adaptivePaths
        case aiTutor
        case attendance
        case calendar
        case collabDocs
        case sections
        case discussions
        case feed
        case files
        case liveSessions
        case misconceptionDetection
        case multilingualMessaging
        case notebook
        case officeHours
        case diagnosticAssessments
        case questionBank
        case reportCards
        case hintScaffolding
        case lockdownMode
        case srs
        case standardsAlignment
        case whiteboard

        var id: String { rawValue }

        var labelKey: String { "mobile.courseSettings.features.tool.\(rawValue).label" }
        var descriptionKey: String { "mobile.courseSettings.features.tool.\(rawValue).description" }
    }

    struct ToolRow: Identifiable, Hashable {
        var tool: Tool
        var id: String { tool.id }
    }

    static let allToolRows: [ToolRow] = [
        .init(tool: .adaptivePaths),
        .init(tool: .aiTutor),
        .init(tool: .attendance),
        .init(tool: .calendar),
        .init(tool: .collabDocs),
        .init(tool: .sections),
        .init(tool: .discussions),
        .init(tool: .feed),
        .init(tool: .files),
        .init(tool: .liveSessions),
        .init(tool: .misconceptionDetection),
        .init(tool: .multilingualMessaging),
        .init(tool: .notebook),
        .init(tool: .officeHours),
        .init(tool: .diagnosticAssessments),
        .init(tool: .questionBank),
        .init(tool: .reportCards),
        .init(tool: .hintScaffolding),
        .init(tool: .lockdownMode),
        .init(tool: .srs),
        .init(tool: .standardsAlignment),
        .init(tool: .whiteboard),
    ]

    static func filterTools(_ tools: [ToolRow], query: String) -> [ToolRow] {
        let trimmed = query.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else { return tools }
        let q = trimmed.lowercased()
        return tools.filter { row in
            L.text(String.LocalizationValue(row.tool.labelKey)).lowercased().contains(q)
                || L.text(String.LocalizationValue(row.tool.descriptionKey)).lowercased().contains(q)
        }
    }

    static func isEnabled(_ tool: Tool, course: CourseSummary) -> Bool {
        switch tool {
        case .adaptivePaths: return course.adaptivePathsEnabled == true
        case .aiTutor: return course.aiTutorEnabled == true
        case .attendance: return course.attendanceEnabled == true
        case .calendar: return course.calendarEnabled != false
        case .collabDocs: return course.collabDocsEnabled == true
        case .sections: return course.sectionsEnabled == true
        case .discussions: return course.discussionsEnabled == true
        case .feed: return course.feedEnabled != false
        case .files: return course.filesEnabled != false
        case .liveSessions: return course.liveSessionsEnabled == true
        case .misconceptionDetection: return course.misconceptionDetectionEnabled == true
        case .multilingualMessaging: return course.multilingualMessagingEnabled == true
        case .notebook: return course.notebookEnabled != false
        case .officeHours: return course.officeHoursEnabled == true
        case .diagnosticAssessments: return course.diagnosticAssessmentsEnabled == true
        case .questionBank: return course.questionBankEnabled == true
        case .reportCards: return course.reportCardsEnabled == true
        case .hintScaffolding: return course.hintScaffoldingEnabled == true
        case .lockdownMode: return course.lockdownModeEnabled == true
        case .srs: return course.srsEnabled == true
        case .standardsAlignment: return course.standardsAlignmentEnabled == true
        case .whiteboard: return course.whiteboardEnabled == true
        }
    }

    static func applyToggle(course: CourseSummary, tool: Tool, enabled: Bool) -> CourseSummary {
        var updated = course
        switch tool {
        case .adaptivePaths: updated.adaptivePathsEnabled = enabled
        case .aiTutor: updated.aiTutorEnabled = enabled
        case .attendance: updated.attendanceEnabled = enabled
        case .calendar: updated.calendarEnabled = enabled
        case .collabDocs: updated.collabDocsEnabled = enabled
        case .sections: updated.sectionsEnabled = enabled
        case .discussions: updated.discussionsEnabled = enabled
        case .feed: updated.feedEnabled = enabled
        case .files: updated.filesEnabled = enabled
        case .liveSessions: updated.liveSessionsEnabled = enabled
        case .misconceptionDetection: updated.misconceptionDetectionEnabled = enabled
        case .multilingualMessaging: updated.multilingualMessagingEnabled = enabled
        case .notebook: updated.notebookEnabled = enabled
        case .officeHours: updated.officeHoursEnabled = enabled
        case .diagnosticAssessments: updated.diagnosticAssessmentsEnabled = enabled
        case .questionBank: updated.questionBankEnabled = enabled
        case .reportCards: updated.reportCardsEnabled = enabled
        case .hintScaffolding: updated.hintScaffoldingEnabled = enabled
        case .lockdownMode: updated.lockdownModeEnabled = enabled
        case .srs: updated.srsEnabled = enabled
        case .standardsAlignment: updated.standardsAlignmentEnabled = enabled
        case .whiteboard: updated.whiteboardEnabled = enabled
        }
        return updated
    }

    static func buildFeaturesPatch(from course: CourseSummary) -> CourseFeaturesPatch {
        CourseFeaturesPatch(
            notebookEnabled: course.notebookEnabled != false,
            feedEnabled: course.feedEnabled != false,
            calendarEnabled: course.calendarEnabled != false,
            questionBankEnabled: course.questionBankEnabled == true,
            lockdownModeEnabled: course.lockdownModeEnabled == true,
            standardsAlignmentEnabled: course.standardsAlignmentEnabled == true,
            adaptivePathsEnabled: course.adaptivePathsEnabled == true,
            srsEnabled: course.srsEnabled == true,
            diagnosticAssessmentsEnabled: course.diagnosticAssessmentsEnabled == true,
            hintScaffoldingEnabled: course.hintScaffoldingEnabled == true,
            misconceptionDetectionEnabled: course.misconceptionDetectionEnabled == true,
            sectionsEnabled: course.sectionsEnabled == true,
            discussionsEnabled: course.discussionsEnabled == true,
            collabDocsEnabled: course.collabDocsEnabled == true,
            liveSessionsEnabled: course.liveSessionsEnabled == true,
            officeHoursEnabled: course.officeHoursEnabled == true,
            aiTutorEnabled: course.aiTutorEnabled == true,
            multilingualMessagingEnabled: course.multilingualMessagingEnabled == true,
            filesEnabled: course.filesEnabled != false,
            attendanceEnabled: course.attendanceEnabled == true,
            whiteboardEnabled: course.whiteboardEnabled == true,
            reportCardsEnabled: course.reportCardsEnabled == true
        )
    }

    static func shouldConfirmDisable(_ tool: Tool, currentlyEnabled: Bool) -> Bool {
        currentlyEnabled
    }

    static func videoCaptionsSectionEnabled(_ features: MobilePlatformFeatures) -> Bool {
        features.videoCaptionsEnabled
    }

    static func consortiumSectionEnabled(_ features: MobilePlatformFeatures) -> Bool {
        features.ffConsortiumSharing
    }

    static func cacheKeyFeatures(courseCode: String) -> String {
        "course:\(courseCode):features"
    }

    static func cacheKeyConsortium(courseCode: String) -> String {
        "course:\(courseCode):consortium"
    }

    static func toggleIdempotencyKey(courseCode: String, tool: Tool) -> String {
        "course-features:\(courseCode):\(tool.rawValue)"
    }

    static func captionPolicyIdempotencyKey(courseCode: String) -> String {
        "course-caption-policy:\(courseCode)"
    }

    static func consortiumIdempotencyKey(courseCode: String) -> String {
        "course-consortium:\(courseCode)"
    }

    static func userFacingError(_ error: Error) -> String {
        if let apiError = error as? APIError {
            return apiError.errorDescription ?? L.text("mobile.courseSettings.features.genericError")
        }
        return error.localizedDescription
    }
}

struct CourseFeaturesPatch: Codable, Equatable {
    var notebookEnabled: Bool
    var feedEnabled: Bool
    var calendarEnabled: Bool
    var questionBankEnabled: Bool
    var lockdownModeEnabled: Bool
    var standardsAlignmentEnabled: Bool
    var adaptivePathsEnabled: Bool
    var srsEnabled: Bool
    var diagnosticAssessmentsEnabled: Bool
    var hintScaffoldingEnabled: Bool
    var misconceptionDetectionEnabled: Bool
    var sectionsEnabled: Bool
    var discussionsEnabled: Bool
    var collabDocsEnabled: Bool
    var liveSessionsEnabled: Bool
    var officeHoursEnabled: Bool
    var aiTutorEnabled: Bool
    var multilingualMessagingEnabled: Bool
    var filesEnabled: Bool
    var attendanceEnabled: Bool
    var whiteboardEnabled: Bool
    var reportCardsEnabled: Bool
}

struct CourseCaptionPolicyPatch: Codable, Equatable {
    var requireCaptions: Bool
}

struct CourseConsortiumSettings: Codable, Equatable {
    var consortiumShareable: Bool
}

struct CourseConsortiumSettingsPatch: Codable, Equatable {
    var consortiumShareable: Bool
}
