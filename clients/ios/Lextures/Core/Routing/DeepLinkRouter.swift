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
            if segments.first?.lowercased() == "inbox" {
                return .inbox
            }
            if segments.first?.lowercased() == "review" {
                return .review
            }
            if segments.count >= 2,
               segments[0].lowercased() == "me",
               segments[1].lowercased() == "study-insights" {
                return .insights
            }
            return .home
        }

        let courseCode = segments[1]
        if segments.count == 2 {
            return .course(code: courseCode, section: .overview, itemId: nil)
        }

        switch segments[2].lowercased() {
        case "grades":
            return .course(code: courseCode, section: .grades, itemId: nil)
        case "office-hours":
            return .course(code: courseCode, section: .officeHours, itemId: nil)
        case "feed":
            return .course(code: courseCode, section: .feed, itemId: nil)
        case "discussions":
            if segments.count >= 5, segments[3].lowercased() == "threads" {
                return .course(code: courseCode, section: .discussions, itemId: segments[4])
            }
            return .course(code: courseCode, section: .discussions, itemId: nil)
        case "live", "live-sessions":
            return .course(code: courseCode, section: .live, itemId: nil)
        case "files":
            return .course(code: courseCode, section: .files, itemId: nil)
        case "attendance":
            return .course(code: courseCode, section: .attendance, itemId: nil)
        case "people", "enrollments":
            return .course(code: courseCode, section: .people, itemId: nil)
        case "evaluations", "evaluation-results":
            return .course(code: courseCode, section: .evaluations, itemId: nil)
        case "library", "reading-dashboard":
            return .course(code: courseCode, section: .library, itemId: nil)
        case "groups":
            return .course(code: courseCode, section: .groups, itemId: nil)
        case "collab-docs":
            if segments.count >= 4 {
                return .course(code: courseCode, section: .collabDocs, itemId: segments[3])
            }
            return .course(code: courseCode, section: .collabDocs, itemId: nil)
        case "gradebook":
            return .course(code: courseCode, section: .grades, itemId: nil)
        case "assignments", "quizzes", "modules":
            if segments.count >= 4 {
                return .course(code: courseCode, section: .modules, itemId: segments[3])
            }
            return .course(code: courseCode, section: .modules, itemId: nil)
        default:
            return .course(code: courseCode, section: .overview, itemId: nil)
        }
    }
}
