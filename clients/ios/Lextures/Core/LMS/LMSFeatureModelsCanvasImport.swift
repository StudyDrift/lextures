import Foundation

/// Canvas import API models (MOB.2).
struct CanvasCourseListItem: Codable, Identifiable, Hashable, Equatable {
    var id: Int
    var name: String
    var courseCode: String?
    var workflowState: String?
    var termName: String?
}

struct CanvasCoursesResponse: Codable, Equatable {
    var courses: [CanvasCourseListItem]?
}

struct CanvasListCoursesRequest: Encodable {
    var canvasBaseUrl: String
    var accessToken: String
}

struct CanvasImportIncludeBody: Encodable, Equatable {
    var modules: Bool
    var assignments: Bool
    var quizzes: Bool
    var enrollments: Bool
    var grades: Bool
    var settings: Bool
    var files: Bool
    var announcements: Bool

    init(_ include: CanvasImportLogic.Include) {
        modules = include.modules
        assignments = include.assignments
        quizzes = include.quizzes
        enrollments = include.enrollments
        grades = include.grades
        settings = include.settings
        files = include.files
        announcements = include.announcements
    }
}

struct PostCourseImportCanvasRequest: Encodable {
    var mode: String
    var canvasBaseUrl: String
    var canvasCourseId: String
    var accessToken: String
    var include: CanvasImportIncludeBody
    var canvasGradeSyncEnabled: Bool?

    init(
        mode: CourseImportExportLogic.ImportMode,
        canvasBaseUrl: String,
        canvasCourseId: String,
        accessToken: String,
        include: CanvasImportLogic.Include,
        canvasGradeSyncEnabled: Bool
    ) {
        self.mode = mode.rawValue
        self.canvasBaseUrl = canvasBaseUrl
        self.canvasCourseId = canvasCourseId
        self.accessToken = accessToken
        self.include = CanvasImportIncludeBody(include)
        self.canvasGradeSyncEnabled = canvasGradeSyncEnabled ? true : nil
    }
}

struct CanvasImportQueuedResponse: Codable, Equatable {
    var jobId: String?
    var message: String?
}
