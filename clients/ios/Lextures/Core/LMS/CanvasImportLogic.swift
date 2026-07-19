import Foundation

/// Canvas course import helpers (MOB.2) — credentials, include map, WS parsing, gates.
/// The Canvas access token is held only in memory for the request lifetime and must never
/// be written to Keychain, UserDefaults, logs, or crash reports.
enum CanvasImportLogic {
    static let courseCreatePermission = CourseCreateLogic.courseCreatePermission
    static let cancelledMessage = "Import cancelled."
    static let tokenMustNotPersistPolicy =
        "Canvas access tokens stay in memory for the active request only and are never persisted."

    enum ImportStep: Int, CaseIterable, Identifiable, Comparable {
        case credentials = 0
        case select = 1
        case importing = 2

        var id: Int { rawValue }

        static func < (lhs: ImportStep, rhs: ImportStep) -> Bool {
            lhs.rawValue < rhs.rawValue
        }

        var labelKey: String {
            switch self {
            case .credentials: return "mobile.canvasImport.step.credentials"
            case .select: return "mobile.canvasImport.step.select"
            case .importing: return "mobile.canvasImport.step.importing"
            }
        }
    }

    enum TargetMode: String, CaseIterable, Identifiable, Hashable {
        case newCourse
        case existingCourse

        var id: String { rawValue }

        var titleKey: String {
            switch self {
            case .newCourse: return "mobile.canvasImport.target.new"
            case .existingCourse: return "mobile.canvasImport.target.existing"
            }
        }
    }

    enum IncludeCategory: String, CaseIterable, Identifiable, Hashable {
        case modules
        case assignments
        case quizzes
        case enrollments
        case grades
        case settings
        case files
        case announcements

        var id: String { rawValue }

        var labelKey: String {
            "mobile.canvasImport.include.\(rawValue)"
        }
    }

    struct Include: Equatable, Hashable, Codable {
        var modules: Bool
        var assignments: Bool
        var quizzes: Bool
        var enrollments: Bool
        var grades: Bool
        var settings: Bool
        var files: Bool
        var announcements: Bool

        static let all = Include(
            modules: true,
            assignments: true,
            quizzes: true,
            enrollments: true,
            grades: true,
            settings: true,
            files: true,
            announcements: true
        )

        func value(for category: IncludeCategory) -> Bool {
            switch category {
            case .modules: return modules
            case .assignments: return assignments
            case .quizzes: return quizzes
            case .enrollments: return enrollments
            case .grades: return grades
            case .settings: return settings
            case .files: return files
            case .announcements: return announcements
            }
        }

        mutating func set(_ category: IncludeCategory, _ enabled: Bool) {
            switch category {
            case .modules: modules = enabled
            case .assignments: assignments = enabled
            case .quizzes: quizzes = enabled
            case .enrollments: enrollments = enabled
            case .grades: grades = enabled
            case .settings: settings = enabled
            case .files: files = enabled
            case .announcements: announcements = enabled
            }
        }

        /// Category counts for telemetry — never includes the token.
        var enabledCategoryCounts: [String: Int] {
            var counts: [String: Int] = [:]
            for category in IncludeCategory.allCases {
                counts[category.rawValue] = value(for: category) ? 1 : 0
            }
            return counts
        }
    }

    enum WSMessageType: String, Equatable {
        case progress
        case complete
        case coursesUpdated = "courses_updated"
        case error
        case unknown
    }

    struct WSMessage: Equatable {
        var type: WSMessageType
        var message: String?
        var courseCode: String?
    }

    static func canvasImportEnabled(_ features: MobilePlatformFeatures) -> Bool {
        features.ffMobileCanvasImport
    }

    static func shouldShowCanvasImportEntry(
        permissions: [String],
        features: MobilePlatformFeatures,
        isOnline: Bool
    ) -> Bool {
        guard canvasImportEnabled(features) else { return false }
        guard CourseCreateLogic.courseCreateV2Enabled(features) else { return false }
        guard CourseCreateLogic.canCreateCourses(permissions: permissions) else { return false }
        return isOnline
    }

    static func normalizeBaseURL(_ raw: String) -> String {
        var value = raw.trimmingCharacters(in: .whitespacesAndNewlines)
        while value.hasSuffix("/") {
            value.removeLast()
        }
        return value
    }

    /// Returns a localization key when credentials are incomplete/invalid.
    static func validateCredentials(baseURL: String, accessToken: String) -> String? {
        let url = normalizeBaseURL(baseURL)
        let token = accessToken.trimmingCharacters(in: .whitespacesAndNewlines)
        if url.isEmpty {
            return "mobile.canvasImport.error.urlRequired"
        }
        let lower = url.lowercased()
        if !(lower.hasPrefix("https://") || lower.hasPrefix("http://")) {
            return "mobile.canvasImport.error.urlInvalid"
        }
        if token.isEmpty {
            return "mobile.canvasImport.error.tokenRequired"
        }
        return nil
    }

    static func filterCourses(_ courses: [CanvasCourseListItem], query: String) -> [CanvasCourseListItem] {
        let normalizedQuery = query.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
        guard !normalizedQuery.isEmpty else { return courses }
        return courses.filter { course in
            let haystack = [
                course.name,
                course.courseCode ?? "",
                course.termName ?? "",
                String(course.id),
            ]
            .joined(separator: " ")
            .lowercased()
            return haystack.contains(normalizedQuery)
        }
    }

    static func isUnpublished(_ workflowState: String?) -> Bool {
        workflowState?.trimmingCharacters(in: .whitespacesAndNewlines).lowercased() == "unpublished"
    }

    static func parseWSMessage(from data: Data) -> WSMessage? {
        guard
            let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
            let typeRaw = json["type"] as? String
        else { return nil }
        let type: WSMessageType
        switch typeRaw {
        case "progress": type = .progress
        case "complete": type = .complete
        case "courses_updated": type = .coursesUpdated
        case "error": type = .error
        default: type = .unknown
        }
        return WSMessage(
            type: type,
            message: json["message"] as? String,
            courseCode: json["courseCode"] as? String
        )
    }

    static func isTerminal(_ type: WSMessageType) -> Bool {
        switch type {
        case .complete, .coursesUpdated, .error:
            return true
        case .progress, .unknown:
            return false
        }
    }

    static func isCancelledError(_ error: Error) -> Bool {
        if let api = error as? CanvasImportError {
            return api == .cancelled
        }
        return (error as NSError).domain == NSURLErrorDomain && (error as NSError).code == NSURLErrorCancelled
    }

    /// Security helper for tests: forbidden persistence keys must never contain a token.
    static func storageContainsToken(haystacks: [String], token: String) -> Bool {
        let trimmed = token.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else { return false }
        return haystacks.contains { $0.contains(trimmed) }
    }

    static func jobWebSocketPath(jobId: String) -> String {
        "/api/v1/ws/canvas-import/\(LMSAPI.encodePath(jobId))"
    }

    static func defaultImportMode(for target: TargetMode) -> CourseImportExportLogic.ImportMode {
        switch target {
        case .newCourse: return .erase
        case .existingCourse: return .mergeAdd
        }
    }

    enum CanvasImportError: LocalizedError, Equatable {
        case cancelled
        case missingJobId
        case connectionClosed
        case connectionError
        case server(String)

        var errorDescription: String? {
            switch self {
            case .cancelled:
                return cancelledMessage
            case .missingJobId:
                return L.text("mobile.canvasImport.error.missingJobId")
            case .connectionClosed:
                return L.text("mobile.canvasImport.error.connectionClosed")
            case .connectionError:
                return L.text("mobile.canvasImport.error.connectionError")
            case .server(let message):
                return message
            }
        }
    }
}
