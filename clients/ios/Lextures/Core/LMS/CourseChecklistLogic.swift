import Foundation

/// Pure course-checklist helpers (CC.9 FR-23). No networking / disk I/O.
enum CourseChecklistLogic {
    static let summaryMemoSeconds: TimeInterval = 60
    static let highlightDurationSeconds: TimeInterval = 4
    static let maxDismissNoteLength = 500
    static let badgeCap = 99

    /// Offline persistence must never store checklist payloads (FR-11 / AC-13).
    static let offlineCacheKeyPrefix = "checklist:"

    // MARK: - Visibility

    /// Staff + Teaching role context only (FR-3, FR-4).
    static func shouldShowWorkspaceSection(
        viewerIsStaff: Bool,
        roleContext: MobileRoleContext
    ) -> Bool {
        viewerIsStaff && roleContext == .teaching
    }

    // MARK: - Status

    static func normalizeStatus(_ raw: String) -> ChecklistStatus {
        ChecklistStatus(rawValue: raw) ?? .unknown
    }

    static func isOutstanding(_ status: String) -> Bool {
        switch normalizeStatus(status) {
        case .todo, .inProgress, .unknown: return true
        case .done, .notApplicable: return false
        }
    }

    static func accessibilityStatusValue(_ status: String) -> String {
        switch normalizeStatus(status) {
        case .done: return "Completed"
        case .todo: return "To do"
        case .inProgress: return "In progress"
        case .notApplicable: return "Not applicable"
        case .unknown: return "Unknown"
        }
    }

    static func isDone(_ status: String) -> Bool {
        normalizeStatus(status) == .done
    }

    // MARK: - Badge

    struct BadgePresentation: Equatable {
        var visible: Bool
        var text: String
        var count: Int
        var accessibilityLabel: String
    }

    static func badgePresentation(outstandingEssential: Int) -> BadgePresentation {
        guard outstandingEssential > 0 else {
            return BadgePresentation(visible: false, text: "", count: 0, accessibilityLabel: "")
        }
        let text = outstandingEssential > badgeCap ? "99+" : "\(outstandingEssential)"
        let label: String
        if outstandingEssential == 1 {
            label = "1 checklist item needs attention"
        } else {
            label = "\(outstandingEssential) checklist items need attention"
        }
        return BadgePresentation(
            visible: true,
            text: text,
            count: outstandingEssential,
            accessibilityLabel: label
        )
    }

    // MARK: - Progress

    static func progressFraction(done: Int, total: Int) -> Double {
        guard total > 0 else { return 0 }
        return min(1, max(0, Double(done) / Double(total)))
    }

    static func progressLabel(done: Int, total: Int) -> String {
        "\(done) of \(total) done"
    }

    static func outstandingCount(in category: ChecklistCategory) -> Int {
        category.items.filter { isOutstanding($0.status) }.count
    }

    static func visibleItems(
        in category: ChecklistCategory,
        showCompleted: Bool
    ) -> [ChecklistItem] {
        if showCompleted { return category.items }
        return category.items.filter { !isDone($0.status) }
    }

    // MARK: - Target resolution

    enum TargetKind: String, Equatable {
        case native
        case web
        case unresolved
    }

    enum NativeDestination: String, Equatable {
        case courseSettings = "CourseSettings"
        case syllabus = "Syllabus"
        case modules = "Modules"
        case courseFeed = "CourseFeed"
        case enrollments = "Enrollments"
        case discussions = "Discussions"
        case officeHours = "OfficeHours"
        case groups = "Groups"
        case files = "Files"
        case webOnly = "web-only"
    }

    struct ResolvedTarget: Equatable {
        var kind: TargetKind
        var native: NativeDestination?
        var workspaceSection: CourseWorkspaceSection?
        var webPath: String?
        var focusAnchor: String?
        var focusEntity: String?
    }

    /// Load the CC.8 native target table (anchor → platform destination).
    static func loadTargetTable(from data: Data) -> [String: String] {
        guard
            let root = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
            let targets = root["targets"] as? [[String: Any]]
        else { return [:] }
        var map: [String: String] = [:]
        for row in targets {
            guard let id = row["id"] as? String else { continue }
            // Prefer iOS key; fall back to shared when present.
            let dest = (row["ios"] as? String) ?? (row["android"] as? String) ?? "web-only"
            map[id] = dest
        }
        return map
    }

    /// Resolve a checklist nav target against the native table (FR-16).
    static func resolveTarget(
        _ target: ChecklistNavTarget?,
        courseCode: String,
        table: [String: String]
    ) -> ResolvedTarget {
        guard let target else {
            return ResolvedTarget(kind: .unresolved, native: nil, workspaceSection: nil, webPath: nil, focusAnchor: nil, focusEntity: nil)
        }

        let anchorKey = normalizeAnchorKey(target.anchor)
        let mapped = anchorKey.flatMap { table[$0] }
        let native = mapped.flatMap { NativeDestination(rawValue: $0) }

        if let native, native != .webOnly {
            return ResolvedTarget(
                kind: .native,
                native: native,
                workspaceSection: workspaceSection(for: native),
                webPath: nil,
                focusAnchor: target.anchor,
                focusEntity: target.entityKey
            )
        }

        if mapped == NativeDestination.webOnly.rawValue || mapped == nil {
            let path = webPath(route: target.route, courseCode: courseCode, anchor: target.anchor, entityKey: target.entityKey)
            if mapped == NativeDestination.webOnly.rawValue {
                return ResolvedTarget(
                    kind: .web,
                    native: .webOnly,
                    workspaceSection: fallbackSection(forRoute: target.route),
                    webPath: path,
                    focusAnchor: target.anchor,
                    focusEntity: target.entityKey
                )
            }
            // Unmapped: open closest course section, no highlight (FR-16).
            return ResolvedTarget(
                kind: .unresolved,
                native: nil,
                workspaceSection: fallbackSection(forRoute: target.route) ?? .overview,
                webPath: path,
                focusAnchor: nil,
                focusEntity: nil
            )
        }

        let path = webPath(route: target.route, courseCode: courseCode, anchor: target.anchor, entityKey: target.entityKey)
        return ResolvedTarget(
            kind: .web,
            native: .webOnly,
            workspaceSection: fallbackSection(forRoute: target.route),
            webPath: path,
            focusAnchor: target.anchor,
            focusEntity: target.entityKey
        )
    }

    static func normalizeAnchorKey(_ anchor: String?) -> String? {
        guard let anchor, !anchor.isEmpty else { return nil }
        // Entity anchors: `modules.item:{id}` → table key `modules.item`
        if let colon = anchor.firstIndex(of: ":") {
            return String(anchor[..<colon])
        }
        return anchor
    }

    static func workspaceSection(for destination: NativeDestination) -> CourseWorkspaceSection? {
        switch destination {
        case .courseSettings: return .settings
        case .syllabus: return .overview
        case .modules: return .modules
        case .courseFeed: return .feed
        case .enrollments: return .people
        case .discussions: return .discussions
        case .officeHours: return .officeHours
        case .groups: return .groups
        case .files: return .files
        case .webOnly: return nil
        }
    }

    static func fallbackSection(forRoute route: String) -> CourseWorkspaceSection? {
        let lower = route.lowercased()
        if lower.contains("settings") { return .settings }
        if lower.contains("modules") { return .modules }
        if lower.contains("syllabus") || lower.contains("overview") { return .overview }
        if lower.contains("enroll") || lower.contains("people") { return .people }
        if lower.contains("feed") || lower.contains("announce") { return .feed }
        if lower.contains("discussion") { return .discussions }
        if lower.contains("office") { return .officeHours }
        if lower.contains("group") { return .groups }
        if lower.contains("file") { return .files }
        if lower.contains("grading") { return .grading }
        return .overview
    }

    /// Build in-app browser path with `?focus=` (and optional `focusEntity`).
    static func webPath(
        route: String,
        courseCode: String,
        anchor: String?,
        entityKey: String?
    ) -> String {
        var path = route.replacingOccurrences(of: "{courseCode}", with: courseCode)
        if !path.hasPrefix("/") { path = "/\(path)" }
        var items: [URLQueryItem] = []
        if let anchor, !anchor.isEmpty {
            items.append(URLQueryItem(name: "focus", value: anchor))
        }
        if let entityKey, !entityKey.isEmpty {
            items.append(URLQueryItem(name: "focusEntity", value: entityKey))
        }
        guard !items.isEmpty else { return path }
        var components = URLComponents()
        components.path = path
        components.queryItems = items
        return components.url?.absoluteString ?? path
    }

    // MARK: - Summary memo

    struct SummaryCacheEntry: Equatable {
        var summary: CourseChecklistSummary
        var catalogVersion: String?
        var fetchedAt: Date
    }

    static func isSummaryFresh(_ entry: SummaryCacheEntry?, now: Date = Date()) -> Bool {
        guard let entry else { return false }
        return now.timeIntervalSince(entry.fetchedAt) < summaryMemoSeconds
    }

    static func shouldInvalidate(
        cachedCatalogVersion: String?,
        responseCatalogVersion: String
    ) -> Bool {
        guard let cachedCatalogVersion, !cachedCatalogVersion.isEmpty else { return false }
        return cachedCatalogVersion != responseCatalogVersion
    }

    // MARK: - Dismiss note

    static func clampedNote(_ note: String?) -> String {
        let trimmed = (note ?? "").trimmingCharacters(in: .whitespacesAndNewlines)
        if trimmed.count <= maxDismissNoteLength { return trimmed }
        return String(trimmed.prefix(maxDismissNoteLength))
    }

    // MARK: - Presentation snapshot (parity tests)

    struct ItemPresentation: Equatable {
        var id: String
        var title: String
        var status: String
        var accessibilityValue: String
        var isDone: Bool
        var isOutstanding: Bool
        var targetKind: String
    }

    static func presentItem(
        _ item: ChecklistItem,
        courseCode: String,
        table: [String: String]
    ) -> ItemPresentation {
        let resolved = resolveTarget(item.target, courseCode: courseCode, table: table)
        return ItemPresentation(
            id: item.id,
            title: item.title,
            status: normalizeStatus(item.status).rawValue,
            accessibilityValue: accessibilityStatusValue(item.status),
            isDone: isDone(item.status),
            isOutstanding: isOutstanding(item.status),
            targetKind: resolved.kind.rawValue
        )
    }

    static func presentChecklist(
        _ checklist: CourseChecklist,
        table: [String: String]
    ) -> [ItemPresentation] {
        var out: [ItemPresentation] = []
        for cat in checklist.categories {
            for item in cat.items {
                out.append(presentItem(item, courseCode: checklist.courseCode, table: table))
            }
        }
        for item in checklist.dismissed {
            out.append(presentItem(item, courseCode: checklist.courseCode, table: table))
        }
        return out
    }

    // MARK: - Rate limit copy

    static let rateLimitedMessage = "Just checked — try again in a moment"
    static let offlineMessage = "Connect to see your checklist"
}
