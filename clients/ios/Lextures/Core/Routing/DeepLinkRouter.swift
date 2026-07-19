import Foundation

/// Parsed navigation target from a push tap, universal link, or in-app notification action URL.
enum SettingsDeepLinkSection: Equatable {
    case account
    case notifications
    case learnerProfile
    /// MOB.3 Settings/Admin hub.
    case adminHub
    /// MOB.3 Audit log page inside the admin hub.
    case auditLog
}

enum DeepLinkDestination: Equatable {
    case home
    case inbox
    case review
    case insights
    case billing
    case credentials
    case checkoutSuccess(courseId: String?)
    case checkoutCancel
    case coursesList
    case settings(SettingsDeepLinkSection)
    case course(code: String, section: CourseDeepLinkSection?, itemId: String?)
    case parent(studentId: String?, section: ParentDeepLinkSection)
    /// Public board share link (`/board-links/{token}`).
    case boardLink(token: String)
    /// Live quiz join / play (`/play` or `/play/{code}`).
    case liveQuizPlay(code: String?)
}

enum ParentDeepLinkSection: Equatable {
    case dashboard
    case grades
    case attendance
    case conferences
    case notificationPrefs
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
    case boards
    case liveQuizzes
    case behavior
    case hallPass
    case insights
}

/// Maps web-style action URLs and `lextures://` links to native navigation intents.
enum DeepLinkRouter {
    static func resolve(_ raw: String?) -> DeepLinkDestination {
        guard let raw, !raw.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty else {
            return .home
        }
        let trimmed = raw.trimmingCharacters(in: .whitespacesAndNewlines)
        if let checkout = resolveCheckout(from: trimmed) {
            return checkout
        }
        if let parent = resolveParent(from: trimmed) {
            return parent
        }
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

    private static func resolveCheckout(from raw: String) -> DeepLinkDestination? {
        let urlString = raw.hasPrefix("/") ? "https://lextures.com\(raw)" : raw
        guard let url = URL(string: urlString),
              let components = URLComponents(url: url, resolvingAgainstBaseURL: false) else {
            return nil
        }
        switch components.path {
        case "/checkout/success":
            let courseId = components.queryItems?.first(where: { $0.name == "course_id" })?.value
            return .checkoutSuccess(courseId: courseId)
        case "/checkout/cancel":
            return .checkoutCancel
        case "/me/billing":
            return .billing
        case "/me/credentials":
            return .credentials
        default:
            return nil
        }
    }

    private static func resolveParent(from raw: String) -> DeepLinkDestination? {
        let urlString = raw.hasPrefix("/") ? "https://lextures.com\(raw)" : raw
        guard let url = URL(string: urlString),
              let components = URLComponents(url: url, resolvingAgainstBaseURL: false) else {
            return nil
        }
        let path = components.path
        guard path == "/parent" || path.hasPrefix("/parent/") else { return nil }
        let studentId = components.queryItems?.first(where: { $0.name == "student" })?.value
        let section: ParentDeepLinkSection
        if path.contains("conferences") {
            section = .conferences
        } else if path.contains("notification") {
            section = .notificationPrefs
        } else if path.contains("grades") {
            section = .grades
        } else if path.contains("attendance") {
            section = .attendance
        } else {
            section = .dashboard
        }
        return .parent(studentId: studentId, section: section)
    }

    private static func resolveNonCoursePath(_ segments: [String]) -> DeepLinkDestination {
        switch segments.first?.lowercased() {
        case "inbox":
            return .inbox
        case "review":
            return .review
        case "courses":
            return .coursesList
        case "settings":
            return resolveSettingsDeepLink(segments)
        case "parent":
            return resolveParentDeepLink(segments)
        case "board-links":
            if segments.count >= 2 {
                let token = segments[1].trimmingCharacters(in: .whitespacesAndNewlines)
                if !token.isEmpty {
                    return .boardLink(token: token)
                }
            }
            return .home
        case "play":
            let code = segments.count >= 2
                ? segments[1].trimmingCharacters(in: .whitespacesAndNewlines)
                : nil
            return .liveQuizPlay(code: (code?.isEmpty == false) ? code : nil)
        default:
            return resolveMeDeepLink(segments)
        }
    }

    private static func resolveSettingsDeepLink(_ segments: [String]) -> DeepLinkDestination {
        guard segments.count >= 2 else { return .settings(.account) }
        switch segments[1].lowercased() {
        case "account":
            return .settings(.account)
        case "notifications":
            return .settings(.notifications)
        case "learner-profile":
            return .settings(.learnerProfile)
        case "admin", "admin-console":
            if segments.count >= 3, segments[2].lowercased() == "audit-log" {
                return .settings(.auditLog)
            }
            return .settings(.adminHub)
        case "audit-log":
            return .settings(.auditLog)
        default:
            return .settings(.account)
        }
    }

    private static func resolveParentDeepLink(_ segments: [String]) -> DeepLinkDestination {
        guard segments.count >= 2 else { return .parent(studentId: nil, section: .dashboard) }
        switch segments[1].lowercased() {
        case "conferences":
            return .parent(studentId: nil, section: .conferences)
        case "notification-prefs":
            return .parent(studentId: nil, section: .notificationPrefs)
        case "grades":
            return .parent(studentId: nil, section: .grades)
        case "attendance":
            return .parent(studentId: nil, section: .attendance)
        default:
            return .parent(studentId: nil, section: .dashboard)
        }
    }

    private static func resolveMeDeepLink(_ segments: [String]) -> DeepLinkDestination {
        guard segments.count >= 2, segments[0].lowercased() == "me" else { return .home }
        switch segments[1].lowercased() {
        case "study-insights":
            return .insights
        case "credentials":
            return .credentials
        default:
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
        case "boards":
            let itemId = segments.count >= 4 ? segments[3] : nil
            return .course(code: courseCode, section: .boards, itemId: itemId)
        case "live-quizzes":
            return .course(code: courseCode, section: .liveQuizzes, itemId: nil)
        case "assignments", "quizzes", "modules":
            if segments.count >= 5,
               ["content", "quiz", "assignment"].contains(segments[3].lowercased()) {
                return .course(code: courseCode, section: .modules, itemId: segments[4])
            }
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
        case "behavior":
            return .behavior
        case "hall-pass":
            return .hallPass
        case "insights", "at-risk", "whats-working":
            return .insights
        default:
            return .overview
        }
    }
}
