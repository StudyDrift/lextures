import Foundation

/// Parsed navigation target from a push tap, universal link, or in-app notification action URL.
enum DeepLinkDestination: Equatable {
    case home
    case inbox
    case review
    case insights
    case course(code: String, section: CourseDeepLinkSection?, itemId: String?)
}

enum CourseDeepLinkSection: String, Equatable {
    case overview
    case modules
    case grades
    case officeHours
    case feed
    case discussions
    case live
    case files
    case attendance
    case people
    case evaluations
    case library
    case groups
    case collabDocs
}

/// Maps web-style action URLs and `lextures://` links to native navigation intents.
enum DeepLinkRouter {
    static func resolve(_ raw: String?) -> DeepLinkDestination {
        guard let raw, !raw.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty else {
            return .home
        }
        let trimmed = raw.trimmingCharacters(in: .whitespacesAndNewlines)
        if let path = extractPath(from: trimmed) {
            return resolvePath(path)
        }
        return .home
    }

    private static func extractPath(from value: String) -> String? {
        if value.hasPrefix("lextures://") {
            let stripped = String(value.dropFirst("lextures://".count))
            return stripped.hasPrefix("/") ? stripped : "/\(stripped)"
        }
        if value.hasPrefix("/") {
            return value
        }
        if let url = URL(string: value), let host = url.host?.lowercased() {
            if host == "lextures.com" || host.hasSuffix(".lextures.com") || host == "localhost" {
                var path = url.path
                if !path.hasPrefix("/") { path = "/\(path)" }
                return path
            }
        }
        return nil
    }

    private static func resolvePath(_ path: String) -> DeepLinkDestination {
        let segments = path.split(separator: "/").map(String.init)
        guard let first = segments.first?.lowercased(), first == "courses", segments.count >= 2 else {
            return resolveNonCoursePath(segments)
        }
        return resolveCoursePath(segments)
    }

    private static func resolveNonCoursePath(_ segments: [String]) -> DeepLinkDestination {
        switch segments.first?.lowercased() {
        case "inbox":
            return .inbox
        case "review":
            return .review
        default:
            if segments.count >= 2,
               segments[0].lowercased() == "me",
               segments[1].lowercased() == "study-insights" {
                return .insights
            }
            return .home
        }
    }

    private static func resolveCoursePath(_ segments: [String]) -> DeepLinkDestination {
        let courseCode = segments[1]
        guard segments.count > 2 else {
            return .course(code: courseCode, section: .overview, itemId: nil)
        }
        return resolveCourseSection(courseCode: courseCode, segments: segments)
    }

    private static func resolveCourseSection(courseCode: String, segments: [String]) -> DeepLinkDestination {
        switch segments[2].lowercased() {
        case "discussions":
            let itemId = segments.count >= 5 && segments[3].lowercased() == "threads" ? segments[4] : nil
            return .course(code: courseCode, section: .discussions, itemId: itemId)
        case "collab-docs":
            let itemId = segments.count >= 4 ? segments[3] : nil
            return .course(code: courseCode, section: .collabDocs, itemId: itemId)
        case "assignments", "quizzes", "modules":
            let itemId = segments.count >= 4 ? segments[3] : nil
            return .course(code: courseCode, section: .modules, itemId: itemId)
        case "gradebook":
            return .course(code: courseCode, section: .grades, itemId: nil)
        default:
            let section = simpleCourseSection(for: segments[2].lowercased())
            return .course(code: courseCode, section: section, itemId: nil)
        }
    }

    private static func simpleCourseSection(for key: String) -> CourseDeepLinkSection {
        switch key {
        case "grades":
            return .grades
        case "office-hours":
            return .officeHours
        case "feed":
            return .feed
        case "live", "live-sessions":
            return .live
        case "files":
            return .files
        case "attendance":
            return .attendance
        case "people", "enrollments":
            return .people
        case "evaluations", "evaluation-results":
            return .evaluations
        case "library", "reading-dashboard":
            return .library
        case "groups":
            return .groups
        default:
            return .overview
        }
    }
}
