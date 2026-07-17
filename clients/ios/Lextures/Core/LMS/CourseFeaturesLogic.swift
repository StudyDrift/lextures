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
        case visualBoards
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
        .init(tool: .visualBoards),
        .init(tool: .whiteboard),
    ]

    static func filterTools(_ tools: [ToolRow], query: String) -> [ToolRow] {
        let trimmed = query.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else { return tools }
        let queryLower = trimmed.lowercased()
        return tools.filter { row in
            L.text(String.LocalizationValue(row.tool.labelKey)).lowercased().contains(queryLower)
                || L.text(String.LocalizationValue(row.tool.descriptionKey)).lowercased().contains(queryLower)
        }
    }

    private static let enabledReaders: [Tool: (CourseSummary) -> Bool] = [
        .adaptivePaths: { $0.adaptivePathsEnabled == true },
        .aiTutor: { $0.aiTutorEnabled == true },
        .attendance: { $0.attendanceEnabled == true },
        .calendar: { $0.calendarEnabled != false },
        .collabDocs: { $0.collabDocsEnabled == true },
        .sections: { $0.sectionsEnabled == true },
        .discussions: { $0.discussionsEnabled == true },
        .feed: { $0.feedEnabled != false },
        .files: { $0.filesEnabled != false },
        .liveSessions: { $0.liveSessionsEnabled == true },
        .misconceptionDetection: { $0.misconceptionDetectionEnabled == true },
        .multilingualMessaging: { $0.multilingualMessagingEnabled == true },
        .notebook: { $0.notebookEnabled != false },
        .officeHours: { $0.officeHoursEnabled == true },
        .diagnosticAssessments: { $0.diagnosticAssessmentsEnabled == true },
        .questionBank: { $0.questionBankEnabled == true },
        .reportCards: { $0.reportCardsEnabled == true },
        .hintScaffolding: { $0.hintScaffoldingEnabled == true },
        .lockdownMode: { $0.lockdownModeEnabled == true },
        .srs: { $0.srsEnabled == true },
        .standardsAlignment: { $0.standardsAlignmentEnabled == true },
        .visualBoards: { $0.visualBoardsEnabled == true },
        .whiteboard: { $0.whiteboardEnabled == true },
    ]

    private static let togglers: [Tool: (inout CourseSummary, Bool) -> Void] = [
        .adaptivePaths: { $0.adaptivePathsEnabled = $1 },
        .aiTutor: { $0.aiTutorEnabled = $1 },
        .attendance: { $0.attendanceEnabled = $1 },
        .calendar: { $0.calendarEnabled = $1 },
        .collabDocs: { $0.collabDocsEnabled = $1 },
        .sections: { $0.sectionsEnabled = $1 },
        .discussions: { $0.discussionsEnabled = $1 },
        .feed: { $0.feedEnabled = $1 },
        .files: { $0.filesEnabled = $1 },
        .liveSessions: { $0.liveSessionsEnabled = $1 },
        .misconceptionDetection: { $0.misconceptionDetectionEnabled = $1 },
        .multilingualMessaging: { $0.multilingualMessagingEnabled = $1 },
        .notebook: { $0.notebookEnabled = $1 },
        .officeHours: { $0.officeHoursEnabled = $1 },
        .diagnosticAssessments: { $0.diagnosticAssessmentsEnabled = $1 },
        .questionBank: { $0.questionBankEnabled = $1 },
        .reportCards: { $0.reportCardsEnabled = $1 },
        .hintScaffolding: { $0.hintScaffoldingEnabled = $1 },
        .lockdownMode: { $0.lockdownModeEnabled = $1 },
        .srs: { $0.srsEnabled = $1 },
        .standardsAlignment: { $0.standardsAlignmentEnabled = $1 },
        .visualBoards: { $0.visualBoardsEnabled = $1 },
        .whiteboard: { $0.whiteboardEnabled = $1 },
    ]

    static func isEnabled(_ tool: Tool, course: CourseSummary) -> Bool {
        enabledReaders[tool]?(course) ?? false
    }

    static func applyToggle(course: CourseSummary, tool: Tool, enabled: Bool) -> CourseSummary {
        var updated = course
        togglers[tool]?(&updated, enabled)
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
            reportCardsEnabled: course.reportCardsEnabled == true,
            visualBoardsEnabled: course.visualBoardsEnabled == true
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
    var visualBoardsEnabled: Bool
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
