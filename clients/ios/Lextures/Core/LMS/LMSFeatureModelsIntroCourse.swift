import Foundation

// MARK: - Intro course (IC07)

/// Canonical intro course URL code (matches server introcourse.CourseCode).
enum IntroCourseConstants {
    static let courseCode = "C-WLCOME"
}

typealias IntroCourseModuleStatus = String

struct IntroCourseNextItem: Codable, Equatable {
    var slug: String
    var title: String
    var route: String
}

struct IntroCourseModuleProgress: Codable, Equatable, Identifiable {
    var slug: String
    var title: String
    var status: IntroCourseModuleStatus

    var id: String { slug }
}

struct IntroCourseProgress: Codable, Equatable {
    var enrolled: Bool
    var courseCode: String?
    var modulesComplete: Int
    var modulesTotal: Int
    var percent: Int
    var runningGrade: Double?
    var completedAt: String?
    var credentialId: String?
    var nextItem: IntroCourseNextItem?
    var modules: [IntroCourseModuleProgress]?
    var welcomeBannerDismissed: Bool?
    var celebrationSeen: Bool?
}

enum IntroCourseCardState: Equatable {
    case hidden
    case loading
    case error
    case notStarted
    case inProgress
    case completed
}